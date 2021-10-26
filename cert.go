package main

import (
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"
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

func certDesc(cert *x509.Certificate) string {
	return fmt.Sprintf(
		"%s, issued by %s%s",
		cert.Subject,
		cert.Issuer,
		certDescValidityPeriod(cert),
	)
}

func certDescValidityPeriod(cert *x509.Certificate) string {
	now := time.Now()

	if now.Before(cert.NotBefore) {
		return fmt.Sprintf(" (becomes valid on %s)", cert.NotBefore)
	}

	if now.After(cert.NotAfter) {
		return fmt.Sprintf(" (expired on %s)", cert.NotAfter)
	}

	remaining := cert.NotAfter.Sub(now)

	return fmt.Sprintf(
		" (Valid until %s, remaining %d days)",
		cert.NotAfter,
		int(remaining.Hours())/24,
	)
}

func certKeyEqual(keya crypto.PublicKey, keyb crypto.PublicKey) bool {
	rsakeya, isRsakey := keya.(*rsa.PublicKey)
	if isRsakey {
		return rsakeya.Equal(keyb)
	}

	return false
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

func certShow(cert *x509.Certificate) {
	fmt.Println("Certificate of", certDesc(cert))
}

func certShowFromPath(path string, cacertPath string) error {
	cacert, err := certLoadOneFromPath(cacertPath)
	if err != nil {
		return fmt.Errorf("Failed to load CA cert from %s: %w", cacertPath, err)
	}

	certificates, err := certLoadFromPath(path)
	if err != nil {
		return fmt.Errorf("Failed to load certs from %s: %w", path, err)
	}

	chain, err := certVerifyChain(certificates, cacert)
	if err != nil {
		return fmt.Errorf("Failed to verify certs from %s, %s: %w", path, cacertPath, err)
	}

	for _, cert := range chain {
		certShow(cert)
	}

	return nil
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

func certShowSubcommand() *subcommand {
	flagset := flag.NewFlagSet("cert-show", flag.ExitOnError)

	certIn := flagset.String("in", "", "path to PEM file")
	cacertIn := flagset.String("cacert", "", "path to PEM file")

	run := func() error {
		if len(*certIn) == 0 {
			return fmt.Errorf("The -in parameter is required")
		}
		return certShowFromPath(*certIn, *cacertIn)
	}

	certShowCommand := subcommand{
		flagset: flagset,
		run:     run,
	}
	return &certShowCommand
}

func certVerifyChain(
	certs []*x509.Certificate,
	cacert *x509.Certificate) ([]*x509.Certificate, error) {

	leaf := certs[0]
	intermediates := certs[1:]

	return certVerifyLeafIntermediatesCa(leaf, intermediates, cacert)
}

func certVerifyLeafIntermediatesCa(
	leaf *x509.Certificate,
	intermediates []*x509.Certificate,
	cacert *x509.Certificate) ([]*x509.Certificate, error) {

	caPool := x509.NewCertPool()
	caPool.AddCert(cacert)

	intermediatePool := x509.NewCertPool()
	for _, imdt := range intermediates {
		intermediatePool.AddCert(imdt)
	}

	opts := x509.VerifyOptions{
		Intermediates: intermediatePool,
		Roots:         caPool,
	}

	chains, err := leaf.Verify(opts)
	if err != nil {
		return nil, fmt.Errorf("Failed to verify the supplied certificate: %w", err)
	}

	if len(chains) != 1 {
		return nil, fmt.Errorf("Expected one chain, got %d", len(chains))
	}

	chain := chains[0]

	expectedChainLen := len(intermediates) + 2
	if len(chain) != expectedChainLen {
		for _, c := range chain {
			certShow(c)
		}
		return nil, fmt.Errorf(
			"Expected %d certs in the chain, got %d",
			expectedChainLen,
			len(chain),
		)
	}

	return chain, nil
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
		if err != nil {
			return fmt.Errorf(
				"failed to code PEM data to %s: %w",
				dest,
				err,
			)
		}
	}

	return nil
}

func certWriteKey(dest string, key interface{}) error {
	keybytes, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return fmt.Errorf("Failed to marshal key: %w", err)
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
