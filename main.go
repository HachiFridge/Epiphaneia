package main

import (
	"context"
	"embed"
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
	config       *Config
	client       *dmmgo.Client
	enable_fanza bool
	version      = "version unknown"
	commit       = "N/A"
)

//go:embed LICENSE
var efs embed.FS

func ListGames() {
	if err := CheckAccessToken(); err != nil {
		log.Fatal(err)
	}

	log.Print("Retrieving games in library...")
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
	fanza := flag.Bool("allow-fanza", false, "Allow FANZA GAMES content (does nothing)")
	install_dir := flag.String("install-dir", "", "Install destination path")
	print_license := flag.Bool("license", false, "Display license")
	do_write_config := flag.Bool("write-config", false, "Write configuration")
	flag.Parse()

	log.Printf("Epiphaneia %s (%s)", version, commit)

	if *print_license == true {
		data, err := efs.ReadFile("LICENSE")
		if err != nil {
			panic(err)
		}
		fmt.Println(string(data))
		return
	}

	enable_fanza = *fanza

	f, err := GetConfigFilePath()
	if err != nil {
		log.Fatal(err)
	}

	_, err = os.Stat(f)
	if errors.Is(err, os.ErrNotExist) {
		if err := WriteConfig(nil); err != nil {
			log.Fatalf("Could not write empty config file: %s", err)
		}
	} else if err != nil {
		log.Fatalf("Could not check config file: %s", err)
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

	if *do_write_config == true {
		if err := WriteConfig(config); err != nil {
			log.Fatalf("Could not write config file: %s", err)
		}
		log.Print("Wrote configuration file")
		return
	}

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
		if *install_dir == "" {
			home_dir, err := os.UserHomeDir()
			if err != nil {
				log.Fatal("Could not determine your home folder. Specify install path with -install-dir.")
			}
			install_dir = &home_dir
		}

		InstallGame(ctx, *install, *install_dir)
		return
	}

	if *list_games == true {
		ListGames()
		return
	}

	flag.Usage()
}
