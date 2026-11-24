package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	dmmgo "codeberg.org/kita/dmm-go"
)

var (
	config *Config
	client *dmmgo.Client
)

func ListGames() {
	if err := CheckAccessToken(); err != nil {
		log.Fatal(err)
	}

	games, err := client.GetUserGames()
	if err != nil {
		log.Fatal("Could not get games: ", err)
	}

	fmt.Println("List of games:")

	for i, v := range games {
		st := strconv.Itoa(i+1) + ". " + v.Title + " - " + v.ProductID
		if CheckIfInstalled(v.ProductID) == true {
			st = st + " [INSTALLED]"
		}
		fmt.Println(st)
	}
}

func main() {
	auth := flag.Bool("auth", false, "Authenticate to DMM GAMES PLAYER")
	launch := flag.String("launch", "", "Launch a game")
	install := flag.String("install", "", "Install a game")
	list_games := flag.Bool("list-games", false, "List all games owned by user")
	handle_protocol := flag.String("handle-protocol", "", "Used to handle the dmmgameplayer mime protocol")
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

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if *handle_protocol != "" {
		HandleProtocol(*handle_protocol)
		return
	}

	if *auth == true {
		DoAuthenticationFlow()
		return
	}
	if *launch != "" {
		LaunchGame(*launch)
		return
	}

	if *install != "" {
		InstallGame(ctx, *install)
		return
	}

	if *list_games == true {
		ListGames()
		return
	}

	flag.Usage()
}
