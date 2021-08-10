package main

import (
	"testing"
)

func TestConfigLoad(t *testing.T) {
	const src = "test-files/joonos.json"
	const expectedCacert = "/etc/joonos/ca.cert.pem"
	const expectedProvcert = "/etc/joonos/provisioning.cert.pem"
	const expectedProvkey = "/etc/joonos/provisioning.key.pem"
	const expectedDatadir = "/var/lib/joonos-sysmgr"

	conf, err := configLoad(src)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", src, err)
	}

	if conf.Cacert != expectedCacert {
		t.Errorf(
			"Expected ca-cert == \"%s\", got \"%s\"",
			expectedCacert,
			conf.Cacert,
		)
	}

	if conf.Provcert != expectedProvcert {
		t.Errorf(
			"Expected provisioning-cert == \"%s\", got \"%s\"",
			expectedProvcert,
			conf.Provcert,
		)
	}

	if conf.Provkey != expectedProvkey {
		t.Errorf(
			"Expected provisioning-key == \"%s\", got \"%s\"",
			expectedProvkey,
			conf.Provkey,
		)
	}

	if conf.Datadir != expectedDatadir {
		t.Errorf(
			"Expected data-dir == \"%s\", got \"%s\"",
			expectedDatadir,
			conf.Datadir,
		)
	}
}

func TestConfigLoadMissing(t *testing.T) {
	const src = "test-files/joonos.jsonn"

	_, err := configLoad(src)
	if err == nil {
		t.Errorf("Expected to fail to read from %s", src)
	}
}

func TestConfigLoadArray(t *testing.T) {
	const src = "test-files/array.json"

	_, err := configLoad(src)
	if err == nil {
		t.Errorf("Expected to fail to load from %s", src)
	}
}

func TestConfigLoadBad(t *testing.T) {
	const src = "test-files/bad.json"

	_, err := configLoad(src)
	if err == nil {
		t.Errorf("Expected to fail to load from %s", src)
	}
}
