package main

import (
	"encoding/json"
	"github.com/beancount-gs/script"
)

type Config struct {
	Title             string `json:"title"`
	DataPath          string `json:"dataPath"`
	OperatingCurrency string `json:"operatingCurrency"`
	StartDate         string `json:"startDate"`
	IsBak             bool   `json:"isBak"`
}

func LoadConfig(globalConfig Config) Config {
	err := json.Unmarshal(script.ReadFile("./config/config.json"), &globalConfig)
	if err != nil {
		script.LogError("config file (/config/config.json) unmarshall failed")
	}
	return globalConfig
}
