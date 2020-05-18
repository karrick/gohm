package gohm

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	// moreSecure configures the HTTPS server to be more secure, but might cause
	// compatibility problems with older HTTP clients. Because this server is
	// meant to be consumed by modern browsers inside our engineering domain, it
	// is an acceptible tradeoff.
	moreSecure = true
)

var (
	// globalBaseDirname specifies the base directory for this instance of the
	// program.
	globalBaseDirname string

	// globalCertsDirname specifies the directory where TLS certificate files
	// are found. This global variable is assigned at the beginning of the
	// program prior to starting any concurrency and only read from thereafter.
	globalCertsDirname string // FIXME = "development-certs"

	// globalBaseDirname specifies the data directory for this instance of the
	// program.
	globalDataDirname string

	// globalHTTPSClient is used for queries that this server proxies to other
	// servers. This global variable is go-routine safe, and according to the
	// standard library, is intended to be used by many go-routines
	// concurrently.
	globalHTTPSClient *http.Client

	// globalHTTPClientRequestTimeout specifies the timeout duration for
	// incoming client requests. This is best made a large number, as a
	// fail-safe for the entire service, then allow each handler to impose
	// reasonable overrides.
	globalHTTPClientRequestTimeout time.Duration

	// globalTLSConfig is used to hold loaded certificates and configuration
	// parameters for TLS connections.
	globalTLSConfig *tls.Config

	// pathnameCABundle is used to specify the pathname of an additional
	// Certificate Authority (CA) bundle file. This file is loaded in addition
	// to the system CA bundle to validate identities of other hosts to this
	// server. This global variable is assigned at the beginning of the program
	// prior to starting any concurrency and only read from thereafter.
	pathnameCABundle = os.Getenv("CA_BUNDLE")
)

func init() {
	// When BASEDIR defined, read certs from $BASEDIR/certs. Otherwise read certs
	// from "devlopment-certs" subdirectory from working directory.
	if globalBaseDirname != "" {
		globalCertsDirname = filepath.Join(globalBaseDirname, "certs")
	}
	if pathnameCABundle == "" {
		// Unless overridden by environment, use system CA bundle.
		pathnameCABundle = "/etc/ca-bundle.crt"
	}
}

func initCerts() error {
	// Load the system CA Bundle, and on top of that, load in the CA bundle from
	// riddler.
	caCertPool, err := x509.SystemCertPool()
	if err != nil {
		return err
	}

	if _, err := os.Stat(pathnameCABundle); err == nil {
		log.Printf("[VERBOSE] using CA bundle: %q", pathnameCABundle)
		caCert, err := ioutil.ReadFile(pathnameCABundle)
		if err != nil {
			return err
		}
		caCertPool.AppendCertsFromPEM(caCert)
	}

	var certificates []tls.Certificate

	// First attempt to load custom certificates from DATADIR/tls. If that
	// directory does not exist, attempt to load LID installed certificates from
	// BASEDIR/var.

	certDir := filepath.Join(globalDataDirname, "tls")
	if _, err := os.Stat(certDir); err == nil {
		// Load manually installed certificates.
		log.Printf("[VERBOSE] adding custom server identity: %q", certDir)
		cf := filepath.Join(certDir, "identity.cert")
		kf := filepath.Join(certDir, "identity.key")
		cert, err := tls.LoadX509KeyPair(cf, kf)
		if err != nil {
			return err
		}
		certificates = append(certificates, cert)
	} else if _, err := os.Stat(globalCertsDirname); err == nil {
		// Load certificates installed by LID.
		log.Printf("[VERBOSE] adding lid server identity: %q", globalCertsDirname)
		cf := filepath.Join(globalCertsDirname, "identity.cert")
		kf := filepath.Join(globalCertsDirname, "identity.key")
		cert, err := tls.LoadX509KeyPair(cf, kf)
		if err != nil {
			return err
		}
		certificates = append(certificates, cert)
	}

	if l := len(certificates); l == 0 {
		return errors.New("cannot run service without at least one server identity certificate")
	}

	// Create TLS config structure with the previously loaded crypto.
	globalTLSConfig = &tls.Config{
		Certificates: certificates,
		RootCAs:      caCertPool,

		// Have server use Go's default ciphersuite preferences, which are tuned to
		// avoid attacks.
		PreferServerCipherSuites: true,

		// Only use curves which have assembly implementations.
		CurvePreferences: []tls.CurveID{
			tls.CurveP256,
			tls.X25519, // Go >= 1.8
		},
	}

	if moreSecure {
		globalTLSConfig.MinVersion = tls.VersionTLS12
		globalTLSConfig.CipherSuites = []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305, // Go 1.8 only
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,   // Go 1.8 only
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,

			// Best disabled, as they don't provide Forward Secrecy,
			// but might be necessary for some clients
			// tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			// tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		}
	}

	globalTLSConfig.BuildNameToCertificate() // only needed when using client certificates

	globalHTTPSClient = &http.Client{
		Timeout:   globalHTTPClientRequestTimeout,
		Transport: &http.Transport{TLSClientConfig: globalTLSConfig},
	}

	return nil
}
