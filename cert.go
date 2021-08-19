package main

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
)

func certCheckKeyMatch(cert *x509.Certificate, key interface{}) error {
	if key == nil {
		return fmt.Errorf("Key is not set")
	}

	if cert == nil {
		return fmt.Errorf("Main cert is not set")
	}

	rsakey, isRsakey := key.(*rsa.PrivateKey)
	if !isRsakey {
		return fmt.Errorf("Check is only implemented for RSA keys")
	}

	switch pub := cert.PublicKey.(type) {
	case *rsa.PublicKey:
		if pub.N.Cmp(rsakey.N) != 0 {
			return fmt.Errorf("Private key does not match")
		}
		break
	default:
		return fmt.Errorf("Expected cert with an RSA key")
	}

	return nil
}

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

func certShowRaw(certificates [][]byte) error {
	for _, cert := range certificates {
		parsed, err := x509.ParseCertificate(cert)
		if err != nil {
			return err
		}
		certShow(parsed)
	}

	return nil
}

func certLoadFromPath(path string) ([]*x509.Certificate, error) {
	pemBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return certDecodePem(pemBytes)
}

func certLoadOneFromPath(path string) (*x509.Certificate, error) {
	certs, err := certLoadFromPath(path)
	if err != nil {
		return nil, err
	}

	certCnt := len(certs)

	if certCnt != 1 {
		return nil, fmt.Errorf(
			"Expected %s to contain one certificate, got %d",
			path,
			certCnt,
		)
	}

	return certs[0], nil
}

func certShowFromPath(path string) error {
	certificates, err := certLoadFromPath(path)
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

func certWriteChain(dest string, certs []*x509.Certificate) error {
	certfile, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf(
			"Failed to create %s: %w",
			dest,
			err,
		)
	}

	defer certfile.Close()

	for _, cert := range certs {
		certblock := pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Raw,
		}

		err = pem.Encode(certfile, &certblock)
	}

	return nil
}

func certWriteKey(dest string, key interface{}) error {
	keybytes, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		fmt.Errorf("Failed to marshal key: %w", err)
	}

	keyblock := pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: keybytes,
	}

	keyfile, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf(
			"Failed to create %s: %w",
			dest,
			err,
		)
	}

	defer keyfile.Close()

	err = pem.Encode(keyfile, &keyblock)
	if err != nil {
		return fmt.Errorf("Failed to encode key to %s: %w", dest, err)
	}

	return nil
}
