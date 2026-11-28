package main

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"path"

	dmmgo "codeberg.org/kita/dmm-go"
)

type Game struct {
	ProductID      string            `json:"product_id"`
	Location       string            `json:"location"`
	Version        string            `json:"version"`
	IsInstalled    bool              `json:"is_installed"`
	IgnoreFileList []string          `json:"ignore_files"`
	ProtonPath     string            `json:"proton_path"`
	WinePrefix     string            `json:"wineprefix"`
	Environment    map[string]string `json:"environment"`
}

type Config struct {
	AccessToken       string `json:"access_token"`
	MotherboardSerial string `json:"motherboard_serial"`
	HDDSerial         string `json:"hdd_serial"`
	MACAddress        string `json:"mac_address"`
	Games             []Game `json:"games"`
	ProtonPath        string `json:"proton_path"`
}

func GetConfigFilePath() (string, error) {
	d, err := os.UserConfigDir()
	if err != nil {
		return "epiphaneia.json", nil
	}

	_, err = os.Stat(path.Join(d, "epiphaneia"))
	if errors.Is(err, os.ErrNotExist) {
		if err := os.Mkdir(path.Join(d, "epiphaneia"), 0750); err != nil {
			return "", err
		}
	}

	return path.Join(d, "epiphaneia", "epiphaneia.json"), nil
}

func GetConfig() (*Config, error) {
	var x Config

	f, err := GetConfigFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(f)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, &x); err != nil {
		return nil, err
	}

	return &x, nil
}

func WriteConfig(config *Config) error {
	f, err := GetConfigFilePath()
	if err != nil {
		return err
	}

	if config == nil {
		ident := dmmgo.NewDeviceIdentity("", "", "")
		config = &Config{
			HDDSerial:         ident.HDDSerial,
			MotherboardSerial: ident.MotherboardSerial,
			MACAddress:        ident.MACAddress,
		}
	}
	data, err := json.MarshalIndent(config, "", "	")
	if err != nil {
		return err
	}
	log.Print("Writing configuration...")
	if err := os.WriteFile(f, data, 0644); err != nil {
		return err
	}

	return nil
}
