package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
	"math/big"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type ca struct {
	privkey rsa.PrivateKey
	pubkey  rsa.PublicKey
}

type cacsr struct {
	from string
	csr  *x509.CertificateRequest
}

type caconfig struct {
	Cacert   string `json:"ca-cert"`
	Datadir  string `json:"data-directory"`
	Tlscert  string `json:"tls-cert"`
	Tlskey   string `json:"tls-key"`
	Signcert string `json:"sign-cert"`
	Signkey  string `json:"sign-key"`
	Mqttsrv  string `json:"mqtt-server"`
}

const certduration time.Duration = time.Hour * 24 * 30

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

	rootcert, err := certLoadOneFromPath(config.Cacert)
	if err != nil {
		return fmt.Errorf(
			"Failed to load root CA cert from %s: %w",
			config.Cacert,
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

	opts := mqtt.NewClientOptions()
	opts.AddBroker(config.Mqttsrv)
	opts.SetAutoReconnect(false)
	opts.SetUsername(tlsleaf.Subject.CommonName)
	opts.SetTLSConfig(caTlsConfig(rootcert, tlscert))

	client := mqtt.NewClient(opts)

	clientConnect := client.Connect()
	clientConnect.Wait()
	err = clientConnect.Error()
	if err != nil {
		fmt.Printf("Failed to connect: %s\n", err)
		return err
	}

	csrs := make(chan cacsr)

	csrSub := client.Subscribe("joonos/+/csr", 1, func(c mqtt.Client, m mqtt.Message) {
		sender, err := caSenderFromTopic(m.Topic())
		if err != nil {
			fmt.Printf("Failed to read sender: %v\n", err)
			return
		}

		csr, err := x509.ParseCertificateRequest(m.Payload())
		if err != nil {
			fmt.Printf("Failed to parse CSR: %v\n", err)
			return
		}
		csrs <- cacsr{
			from: sender,
			csr:  csr,
		}
	})
	csrSub.Wait()
	err = csrSub.Error()
	if err != nil {
		fmt.Printf("Failed to subscribe: %s\n", err)
		return err
	}

	serials := caSerialChan(config.Datadir + "/serial")

	for {
		csr := <-csrs

		commonName := csr.csr.Subject.CommonName

		fmt.Println("Received CSR for", commonName, "from", csr.from)

		cert, err := caSign(serials, signcert, signkey, csr.csr, certduration)

		if err != nil {
			fmt.Printf("Failed to generate certificate for %s: %v", commonName, err)
			continue
		}

		if !certKeyEqual(cert.PublicKey, csr.csr.PublicKey) {
			fmt.Println("Generated for the wrong key?")
			continue
		}

		certTopic := fmt.Sprintf("joonos/%s/cert", csr.from)

		fmt.Println("Publishing cert", cert.SerialNumber, "of", cert.Subject.CommonName, "on", certTopic)

		certbytes := make([]byte, len(cert.Raw))
		copy(certbytes, cert.Raw)
		certbytes = append(certbytes, signcert.Raw...)

		client.Publish(certTopic, 1, false, certbytes)
	}
}

func caSenderFromTopic(topic string) (string, error) {
	prefix := "joonos/"
	suffix := "/csr"
	if !strings.HasPrefix(topic, prefix) {
		return "", fmt.Errorf("Expected topic to start with %s", prefix)
	}

	if !strings.HasSuffix(topic, suffix) {
		return "", fmt.Errorf("Expected topic to end with %s", suffix)
	}

	return strings.TrimPrefix(strings.TrimSuffix(topic, suffix), prefix), nil
}

func caSerialChan(path string) <-chan uint64 {
	state := caSerialInit(path)
	buf := [8]byte{}
	serials := make(chan uint64)

	go func() {
		for {
			newSerial := state + 1
			serials <- newSerial
			state = newSerial
			binary.LittleEndian.PutUint64(buf[:], newSerial)
			ioutil.WriteFile(path, buf[:], 0600)
		}
	}()

	return serials
}

func caSerialInit(path string) uint64 {
	oldcontent, err := ioutil.ReadFile(path)
	if err != nil {
		return 1
	}

	if len(oldcontent) != 8 {
		return 1
	}

	return binary.LittleEndian.Uint64(oldcontent)
}

func caSign(
	serials <-chan uint64,
	signcert *x509.Certificate,
	signkey crypto.PrivateKey,
	csr *x509.CertificateRequest,
	duration time.Duration,
) (*x509.Certificate, error) {
	notBefore := time.Now()
	notAfter := notBefore.Add(certduration)

	serial := big.NewInt(int64(<-serials))
	subject := pkix.Name{
		CommonName: csr.Subject.CommonName,
	}
	template := &x509.Certificate{
		Subject:      subject,
		SerialNumber: serial,
		NotBefore:    notBefore,
		NotAfter:     notAfter,
	}

	newcert, err := x509.CreateCertificate(
		rand.Reader,
		template,
		signcert,
		csr.PublicKey,
		signkey,
	)

	if err != nil {
		return nil, fmt.Errorf("Failed to issue certificate: %w", err)
	}

	parsedCert, err := x509.ParseCertificate(newcert)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse freshly issued certificate: %w", err)
	}

	return parsedCert, nil
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

func caTlsConfig(rootCert *x509.Certificate, tlscert tls.Certificate) *tls.Config {
	rootCAs := x509.NewCertPool()
	rootCAs.AddCert(rootCert)

	config := &tls.Config{
		Certificates: []tls.Certificate{tlscert},
		RootCAs:      rootCAs,
	}

	return config
}
