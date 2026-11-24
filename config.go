package main

import (
	"encoding/json"
	"log"
	"os"

	dmmgo "codeberg.org/kita/dmm-go"
)

type Game struct {
	ProductID string `json:"product_id"`
	Location  string `json:"location"`
	Version   string `json:"version"`
	IgnoreFileList []string `json:"ignore_files"`
}

type Config struct {
	AccessToken       string `json:"access_token"`
	MotherboardSerial string `json:"motherboard_serial"`
	HDDSerial         string `json:"hdd_serial"`
	MACAddress        string `json:"mac_address"`
	Games             []Game `json:"games"`
}

func GetConfig() (*Config, error) {
	var x Config

	data, err := os.ReadFile("epiphaneia.json")
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, &x); err != nil {
		return nil, err
	}

	return &x, nil
}

func WriteConfig(config *Config) error {
	if config == nil {
		ident := dmmgo.NewDeviceIdentity("", "", "")
		config = &Config{
			HDDSerial:         ident.HDDSerial,
			MotherboardSerial: ident.MotherboardSerial,
			MACAddress:        ident.MACAddress,
		}
	}
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	log.Print("Writing configuration...")
	if err := os.WriteFile("epiphaneia.json", data, 0644); err != nil {
		return err
	}

	return nil
}
