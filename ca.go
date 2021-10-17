package main

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type ca struct {
	privkey rsa.PrivateKey
	pubkey  rsa.PublicKey
}

type caconfig struct {
	Cacert   string `json:"ca-cert"`
	Tlscert  string `json:"tls-cert"`
	Tlskey   string `json:"tls-key"`
	Signcert string `json:"sign-cert"`
	Signkey  string `json:"sign-key"`
	Mqttsrv  string `json:"mqtt-server"`
}

func caFlags(flagset *flag.FlagSet, args *commonArgs) {
	flagset.StringVar(
		&args.config,
		"config",
		"/etc/joonos/ca.conf",
		"path to config file")
}

func caLoadSigncert(signcertpath string, signkeypath string) (*x509.Certificate, interface{}, error) {
	signcertPem, err := ioutil.ReadFile(signcertpath)
	if err != nil {
		return nil, nil, err
	}

	signkeyPem, err := ioutil.ReadFile(signkeypath)
	if err != nil {
		return nil, nil, err
	}

	block, rest := pem.Decode(signcertPem)
	if len(rest) > 0 {
		return nil, nil, fmt.Errorf("Trailing garbage in %s", signcertpath)
	}

	signcert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, nil, err
	}

	block, rest = pem.Decode(signkeyPem)
	if len(rest) > 0 {
		return nil, nil, fmt.Errorf("Trailing garbage in %s", signkeypath)
	}

	signkey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, nil, err
	}

	return signcert, signkey, nil
}

func caRun(configpath string) error {
	configbytes, err := ioutil.ReadFile(configpath)
	if err != nil {
		return fmt.Errorf(
			"Failed to read %s: %w",
			configpath,
			err,
		)
	}

	var config caconfig
	err = json.Unmarshal(configbytes, &config)
	if err != nil {
		return fmt.Errorf(
			"Failed to parse JSON from %s: %w",
			configpath,
			err,
		)
	}

	tlscert, err := tls.LoadX509KeyPair(config.Tlscert, config.Tlskey)
	if err != nil {
		return fmt.Errorf(
			"Failed to load key pair for TLS from %s, %s: %w",
			config.Tlscert,
			config.Tlskey,
			err,
		)
	}

	tlsleaf, err := x509.ParseCertificate(tlscert.Certificate[0])
	if err != nil {
		return err
	}

	tlscert.Leaf = tlsleaf

	signcert, signkey, err := caLoadSigncert(config.Signcert, config.Signkey)
	if err != nil {
		return err
	}

	fmt.Println("loaded", signcert, "and", signkey)

	opts := mqtt.NewClientOptions()
	opts.AddBroker(config.Mqttsrv)
	opts.SetAutoReconnect(false)
	opts.SetUsername(tlsleaf.Subject.CommonName)

	client := mqtt.NewClient(opts)
	client.Connect().Wait()

	csrs := make(chan *x509.CertificateRequest)

	client.Subscribe("joonos/%/csr", 0, func(c mqtt.Client, m mqtt.Message) {
		csr, err := x509.ParseCertificateRequest(m.Payload())
		if err != nil {
			fmt.Printf("Failed to parse CSR: %v\n", err)
			return
		}
		csrs <- csr
	}).Wait()

	for {
		csr := <-csrs

		fmt.Println("Received CSR", csr)
	}
}

func caSubcommand() *subcommand {
	flagset := flag.NewFlagSet("ca", flag.ExitOnError)
	args := commonArgs{}
	caFlags(flagset, &args)
	run := func() error {
		return caRun(args.config)
	}

	runCommand := subcommand{
		flagset: flagset,
		run:     run,
	}
	return &runCommand
}
