package main

import (
	"encoding/json"
	"io/ioutil"
)

type config struct {
	Cacert   string `json:"ca-cert"`
	Provcert string `json:"provisioning-cert"`
	Provkey  string `json:"provisioning-key"`
	Datadir  string `json:"data-directory"`
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
