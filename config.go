package main

import (
	"encoding/json"
	"io/ioutil"
)

type config struct {
	cacert   string
	provcert string
	provkey  string
	datadir  string
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
