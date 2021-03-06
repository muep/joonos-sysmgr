package main

import (
	"encoding/json"
	"io/ioutil"
)

type config struct {
	Cacert   string   `json:"ca-cert"`
	Provcert string   `json:"provisioning-cert"`
	Provkey  string   `json:"provisioning-key"`
	Datadir  string   `json:"data-directory"`
	Mqttsrv  string   `json:"mqtt-server"`
	Nodename string   `json:"node-name"`
	Upgrade  []string `json:"upgrade"`
}

func configLoad(path string) (config, error) {
	var conf config

	jsonBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return conf, err
	}

	err = json.Unmarshal(jsonBytes, &conf)
	if err != nil {
		return conf, err
	}

	return conf, nil
}

func (c config) Nodecert() string {
	return c.Datadir + "/node.cert.pem"
}

func (c config) Nodekey() string {
	return c.Datadir + "/node.key.pem"
}
