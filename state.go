package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
)

type state struct {
	config      config
	cacert      *x509.Certificate
	provcert    tls.Certificate
	nodecert    *tls.Certificate
	nodecerterr error
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

	nodecert, nodecerterr := tls.LoadX509KeyPair(
		config.Nodecert(),
		config.Nodekey(),
	)

	var nodecertptr *tls.Certificate = nil
	if nodecerterr == nil {
		nodecertptr = &nodecert
	}

	res.cacert = cacert
	res.provcert = provcert
	res.nodecert = nodecertptr
	res.nodecerterr = nodecerterr

	return res, nil
}

func (s state) tlscert() *tls.Certificate {
	if s.nodecert != nil {
		return s.nodecert
	}

	return &s.provcert
}
