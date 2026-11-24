package main

import (
	"errors"
	"flag"
	"log"
	"os"

	dmmgo "codeberg.org/kita/dmm-go"
)

var (
	config *Config
	client *dmmgo.Client
)

func main() {
	auth := flag.Bool("auth", false, "Authenticate to DMM GAMES PLAYER")
	launch := flag.String("launch", "", "Launch a game")
	flag.Parse()

	_, err := os.Stat("epiphaneia.json")
	if errors.Is(err, os.ErrNotExist) {
		if err := WriteConfig(nil); err != nil {
			log.Fatal("Could not write empty config file")
			os.Exit(1)
		}
	} else if err != nil {
		log.Fatal("Could not check config file:", err)
	}

	cfg, err := GetConfig()
	if err != nil {
		log.Fatal(err)
	}
	config = cfg

	log.Printf("MAC Address: %s", config.MACAddress)
	log.Printf("HDD Serial: %s", config.HDDSerial)
	log.Printf("Motherboard Serial: %s", config.MotherboardSerial)

	ident := dmmgo.NewDeviceIdentity(config.MACAddress, config.HDDSerial, config.MotherboardSerial)
	client = dmmgo.NewClient(ident, config.AccessToken)

	if *auth == true {
		DoAuthenticationFlow()
		return
	}
	if *launch != "" {
		LaunchGame(*launch)
		return
	}
}
