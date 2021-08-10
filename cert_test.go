package main

import (
	"io/ioutil"
	"testing"
)

func Test_certDecodePem(t *testing.T) {
	const src = "test-files/wikipedia.cert.pem"
	pemBytes, err := ioutil.ReadFile(src)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", src, err)
	}

	cert, err := certDecodePem(pemBytes)
	if err != nil {
		t.Fatalf("Failed to decode PEM: %v", err)
	}

	if cert == nil {
		t.Errorf("Expected non-nil cert")
	}
}
