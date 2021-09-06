package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"os"
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

	nodecert, nodecerterr := tls.LoadX509KeyPair(
		config.Nodecert(),
		config.Nodekey(),
	)

	var nodecertptr *tls.Certificate = nil
	if nodecerterr == nil {
		nodecertptr = &nodecert

		nodecertLeaf, err := x509.ParseCertificate(nodecert.Certificate[0])
		if err != nil {
			return res, fmt.Errorf(
				"Failed to parse leaf certificate from %s: %w",
				config.Nodecert(),
				err,
			)
		}
		nodecertptr.Leaf = nodecertLeaf
	}

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
	res.nodecert = nodecertptr
	res.nodecerterr = nodecerterr
	res.nodename = nodename

	return res, nil
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
	if s.nodecert != nil {
		return s.nodecert
	}

	return &s.provcert
}
