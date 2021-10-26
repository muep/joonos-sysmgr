package main

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"io"
	"os"
)

func offlineReadUntilEof(f io.Reader) ([]byte, error) {
	buf := bytes.Buffer{}
	_, err := buf.ReadFrom(f)

	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func offlineReadCerts(f io.Reader) ([]*x509.Certificate, error) {
	pemBytes, err := offlineReadUntilEof(f)
	if err != nil {
		return nil, err
	}

	return certDecodePem(pemBytes)
}

func offlineProvision(configpath string) error {
	config, err := configLoad(configpath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	state, err := stateLoad(config)
	if err != nil {
		return fmt.Errorf("failed to initialize state: %w", err)
	}

	if state.nodecert != nil {
		fmt.Println("Node certificate is already set up.")
		return nil
	}

	if !os.IsNotExist(state.nodecerterr) {
		fmt.Println("Unexpected error for node cert:", state.nodecerterr)
	}

	csr, err := state.csr()
	if err != nil {
		return err
	}

	csrBlock := pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csr.Raw,
	}
	pem.Encode(os.Stdout, &csrBlock)

	certs, err := offlineReadCerts(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to read certificates from stdin: %w", err)
	}

	leaf := certs[0]
	intermediates := certs[1:]

	err = state.setCertificate(leaf, intermediates)
	if err != nil {
		return fmt.Errorf("failed to install certificates: %w", err)
	}

	return nil
}

func offlineSubcommand() *subcommand {
	flagset := flag.NewFlagSet("offline-provision", flag.ExitOnError)
	args := commonArgs{}
	commonFlags(flagset, &args)

	run := func() error {
		return offlineProvision(args.config)
	}

	opCommand := subcommand{
		flagset: flagset,
		run:     run,
	}
	return &opCommand

}
