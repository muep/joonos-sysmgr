package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"flag"
	"fmt"
	"os"
	"time"
)

type state struct {
	config      config
	cacert      *x509.Certificate
	provcert    tls.Certificate
	nodename    string
	nodecert    *tls.Certificate
	nodecerterr error
	csrkey      interface{}
}

func stateLoad(config config) (state, error) {
	res := state{
		config: config,
	}

	err := fsCheckDirPresent(config.Datadir)
	if err != nil {
		return res, fmt.Errorf(
			"Failed to ensure presence of %s: %w",
			config.Datadir,
			err,
		)
	}

	cacert, err := certLoadOneFromPath(config.Cacert)
	if err != nil {
		return res, fmt.Errorf(
			"Failed to load CA root certificate from %s: %w",
			config.Cacert,
			err,
		)
	}

	provcert, err := tls.LoadX509KeyPair(config.Provcert, config.Provkey)
	if err != nil {
		return res, fmt.Errorf(
			"Failed to load provisioning certificate from %s, %s: %w",
			config.Provcert,
			config.Provkey,
			err,
		)
	}
	provcertLeaf, err := x509.ParseCertificate(provcert.Certificate[0])
	if err != nil {
		return res, fmt.Errorf(
			"Failed to parse leaf certificate from %s: %w",
			config.Provcert,
			err,
		)
	}
	provcert.Leaf = provcertLeaf

	nodecert, nodecerterr := stateLoadNodecert(
		config.Nodecert(),
		config.Nodekey(),
	)
	// Nodecerterr is something that is not handled at this time.
	// The caller is expected to check it and make do without a
	// good node cert, if necessary

	nodename := config.Nodename
	if len(nodename) == 0 {
		hostname, err := os.Hostname()
		if err != nil {
			return res, err
		}

		nodename = hostname
	}

	res.cacert = cacert
	res.provcert = provcert
	res.nodecert = nodecert
	res.nodecerterr = nodecerterr
	res.nodename = nodename

	return res, nil
}

func stateLoadNodecert(certpath string, keypath string) (*tls.Certificate, error) {
	nodecert, err := tls.LoadX509KeyPair(
		certpath,
		keypath,
	)

	if err != nil {
		return &nodecert, err
	}

	leaf, err := x509.ParseCertificate(nodecert.Certificate[0])
	if err != nil {
		// This is not really expected to happen, given how
		// the tls package already managed to parse it. The
		// certificate is returned along with the error, just
		// in case it turns out to be useful. The remaining
		// checks here can not be done.
		return &nodecert, fmt.Errorf(
			"Failed to parse leaf certificate from %s: %w",
			certpath,
			err,
		)
	}

	// At least MQTT code expects to use Subject from Leaf
	// for producing the user name
	nodecert.Leaf = leaf

	notAfter := nodecert.Leaf.NotAfter
	if time.Now().After(notAfter) {
		return &nodecert, fmt.Errorf("Certificate expired on %s", notAfter)
	}

	return &nodecert, nil
}

func (s *state) csr() (*x509.CertificateRequest, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	subject := pkix.Name{
		CommonName: s.nodename,
	}
	template := x509.CertificateRequest{
		Subject: subject,
	}

	csrb, err := x509.CreateCertificateRequest(rand.Reader, &template, key)
	if err != nil {
		return nil, err
	}

	csr, err := x509.ParseCertificateRequest(csrb)
	if err != nil {
		return nil, err
	}

	s.csrkey = key

	return csr, nil
}

func (s *state) setCertificate(cert *x509.Certificate, intermediates []*x509.Certificate) error {
	chain, err := certVerifyLeafIntermediatesCa(cert, intermediates, s.cacert)
	if err != nil {
		return fmt.Errorf("Failed to verify the supplied certificate: %w", err)
	}

	err = certCheckKeyMatch(cert, s.csrkey)
	if err != nil {
		return fmt.Errorf("Could not match certificate to CSR key: %w", err)
	}

	err = certWriteKey(s.config.Nodekey(), s.csrkey)
	if err != nil {
		return fmt.Errorf("Failed to store key: %w", err)
	}

	err = certWriteChain(s.config.Nodecert(), chain[:len(chain)-1])
	if err != nil {
		return fmt.Errorf("Failed to store cert chain: %w", err)
	}

	return nil
}

func (s state) tlscert() *tls.Certificate {
	cert := s.nodecert
	if cert == nil {
		cert = &s.provcert
	}

	if s.nodecerterr != nil {
		cert = &s.provcert
	}

	return cert
}

func (s state) tlsconfig() *tls.Config {
	rootCAs := x509.NewCertPool()
	rootCAs.AddCert(s.cacert)

	config := &tls.Config{
		Certificates: []tls.Certificate{*s.tlscert()},
		RootCAs:      rootCAs,
	}

	return config
}

func stateShow(configpath string) error {
	config, err := configLoad(configpath)
	if err != nil {
		return fmt.Errorf("Failed to load config from %s: %w", configpath, err)
	}

	state, err := stateLoad(config)
	if err != nil {
		return fmt.Errorf("Failed to load state using config from %s: %w", configpath, err)
	}

	fmt.Printf("State for %s [%s]\n", state.nodename, configpath)

	fmt.Printf("  CA certificate [%s]:\n", state.config.Cacert)
	fmt.Println("   ", certDesc(state.cacert))

	fmt.Printf("  Provisioning certificate [%s]:\n", state.config.Provcert)
	fmt.Println("   ", certDesc(state.provcert.Leaf))

	fmt.Printf("  Node certificate [%s]:\n", state.config.Nodecert())
	if state.nodecert == nil {
		fmt.Println("    (absent)")
	} else {
		fmt.Println("   ", certDesc(state.nodecert.Leaf))
	}

	if state.nodecerterr != nil {
		fmt.Printf("    Error: %s\n", state.nodecerterr)
	}

	return nil
}

func stateShowSubcommand() *subcommand {
	flagset := flag.NewFlagSet("state-show", flag.ExitOnError)
	args := commonArgs{}
	commonFlags(flagset, &args)

	run := func() error {
		return stateShow(args.config)
	}

	opCommand := subcommand{
		flagset: flagset,
		run:     run,
	}
	return &opCommand
}
