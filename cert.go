package main

import (
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
)

func certDecodePem(pemBytes []byte) ([]*x509.Certificate, error) {
	const expectedType = "CERTIFICATE"
	res := make([]*x509.Certificate, 0, 5)

	for len(pemBytes) > 0 {
		block, rest := pem.Decode(pemBytes)
		if block == nil {
			break
		}

		if block.Type != expectedType {
			return nil, fmt.Errorf(
				"Expected PEM block type %s, got %s",
				expectedType,
				block.Type)
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse certificate: %w", err)
		}

		res = append(res, cert)
		pemBytes = rest
	}

	return res, nil
}

func certShow(cert *x509.Certificate) {
	fmt.Printf("Certificate of %s, issued by %s\n", cert.Subject, cert.Issuer)
}

func certShowFromPath(path string) error {
	fmt.Printf("Loading cert from %s\n", path)
	pemBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	certificates, err := certDecodePem(pemBytes)
	if err != nil {
		return err
	}

	for _, cert := range certificates {
		certShow(cert)
	}

	return nil
}

func certShowSubcommand() *subcommand {
	flagset := flag.NewFlagSet("cert-show", flag.ExitOnError)
	args := commonArgs{}
	commonFlags(flagset, &args)

	certIn := flagset.String("in", "", "path to PEM file")

	run := func() error {
		if len(*certIn) == 0 {
			return fmt.Errorf("The -in parameter is required")
		}
		return certShowFromPath(*certIn)
	}

	certShowCommand := subcommand{
		flagset: flagset,
		run:     run,
	}
	return &certShowCommand
}
