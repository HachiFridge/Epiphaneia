package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"

	dmmgo "codeberg.org/kita/dmm-go"
)

func EULAAgreement(product_id string) {
	log.Printf("Agreement of license for %s is required to install", product_id)
	data, err := client.GetGameEULA(product_id)
	if err != nil {
		log.Fatalf("Unable to get license agreement for %s: %s", product_id, err)
	}

	for _, v := range data {
		fmt.Println(v.Title)
		fmt.Println(v.Agreement)
	}

	fmt.Print("Agree to terms [n]? ")

	var resp string

	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		resp = scanner.Text()
	}

	if err := scanner.Err(); err != nil {
		log.Fatal("Could not initialize prompt:", err)
	}

	if len(resp) == 0 {
		resp = "n"
	}

	if strings.ToLower(string([]rune(resp)[0])) != "y" {
		log.Fatal("You did not agree to the terms. Operation has been aborted.")
	}

	if err := client.AgreeToGameEULA(product_id); err != nil {
		log.Fatalf("Could not agree to terms: %s", err)
	}
}

func IsOwnGame(product_id string) bool {
	data, err := client.GetUserGames()
	if err != nil {
		log.Fatal(err)
	}

	for _, v := range data {
		if v.ProductID == product_id {
			return true
		}
	}

	return false
}

func CheckIfInstalled(product_id string) bool {
	for _, v := range config.Games {
		if product_id == v.ProductID {
			return true
		}
	}

	return false
}

func GetGameConfigIndex(product_id string) int {
	for i, v := range config.Games {
		if product_id == v.ProductID {
			return i
		}
	}

	return 0
}

func GetGameConfig(product_id string) *Game {
	for _, v := range config.Games {
		if product_id == v.ProductID {
			return &v
		}
	}

	return nil
}

func LaunchGameLinux(cfg Game, exe string, args []string) {
	umu, err := exec.LookPath("umu-run")
	if err != nil {
		log.Fatal("umu-run was not found. Please install umu-launcher first.")
	}
	log.Printf("umu-run found at %s", umu)

	for k := range cfg.Environment {
		for _, v := range []string{"PROTONPATH", "GAMEID", "WINEPREFIX"} {
			if v == k {
				log.Fatalf("Error: %s is a reserved environment variable", v)
			}
		}
	}

	var new_args []string

	new_args = append(new_args, exe)
	new_args = append(new_args, args...)

	cmd := exec.Command(umu, new_args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	for _, v := range os.Environ() {
		cmd.Env = append(cmd.Env, v)
	}

	cmd.Env = append(cmd.Env, "GAMEID=umu-default")
	if cfg.ProtonPath != "" {
		_, err := os.Stat(cfg.ProtonPath)
		if err != nil {
			log.Fatalf("Error with Proton path: %s", err)
		}
		log.Printf("Using Proton at %s (overridden by game-specific config)", cfg.ProtonPath)
		cmd.Env = append(cmd.Env, fmt.Sprintf("PROTONPATH=%s", cfg.ProtonPath))
	} else {
		if config.ProtonPath != "" {
			_, err := os.Stat(config.ProtonPath)
			if err != nil {
				log.Fatalf("Error with Proton path: %s", err)
			}
			log.Printf("Using Proton at %s", config.ProtonPath)
			cmd.Env = append(cmd.Env, fmt.Sprintf("PROTONPATH=%s", config.ProtonPath))
		} else {
			log.Print("Using default provided Proton by umu-launcher")
		}
	}
	if cfg.WinePrefix != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("WINEPREFIX=%s", cfg.WinePrefix))
	} else {
		h, err := os.UserHomeDir()
		if err != nil {
			log.Fatal("Error: Could not determine your home folder. Specify WINEPREFIX manually in configuration.")
		}
		p := path.Join(h, ".local", "share", "epihaneia", "games", cfg.ProductID)
		if err := os.MkdirAll(p, 0750); err != nil {
			log.Fatalf("Error: Unable to make directory %s", p)
		}
		cmd.Env = append(cmd.Env, fmt.Sprintf("WINEPREFIX=%s", p))
	}
	for k, v := range cfg.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	log.Print("Environment variables:")
	for _, v := range cmd.Env {
		log.Print(v)
	}

	log.Print("Launching...")

	if err := cmd.Run(); err != nil {
		log.Fatalf("%s: %s", cfg.ProductID, err)
	}
}

func LaunchGame(product_id string) {
	if err := CheckAccessToken(); err != nil {
		log.Fatal(err)
	}

	if CheckIfInstalled(product_id) == false {
		log.Fatal(product_id, " is not installed. Install it first with -install")
	}

	log.Printf("Getting game details...")
	data, err := client.LaunchGame(product_id, "GCL")
	if err != nil {
		log.Fatal(err)
	}

	c := GetGameConfig(product_id)
	executable := path.Join(c.Location, data.ProductID, data.ExecFilename)
	args := strings.Split(data.ExecuteArgs, " ")

	log.Printf("Product: %s", data.ProductID)
	log.Printf("Version: %s (local: %s)", data.LatestVersion, c.Version)
	log.Printf("Command: %s %s", executable, data.ExecuteArgs)

	switch runtime.GOOS {
	case "linux":
		LaunchGameLinux(*c, executable, args)
	case "windows":
		cmd := exec.Command(executable, args...)
		log.Print("Launching...")
		if err := cmd.Run(); err != nil {
			log.Fatalf("%s: %s", product_id, err)
		}
	default:
		log.Fatalf("Launching games on %s is not supported", runtime.GOOS)
	}
}

func InstallGame(ctx context.Context, product_id string, dest_dir string) {
	if err := CheckAccessToken(); err != nil {
		log.Fatal(err)
	}

	log.Printf("Determining ownership for %s...", product_id)
	if ok := IsOwnGame(product_id); !ok {
		log.Fatalf("You do not own %s. You need to add it to your library/buy it first from the official client.", product_id)
	}

	log.Printf("Requesting install information for %s...", product_id)

	var data dmmgo.InstallCLData

	for {
		d, err := client.GetInstallData(product_id, "GCL")

		if errors.Is(err, dmmgo.AgreementError) {
			EULAAgreement(product_id)
			continue
		} else if err != nil {
			log.Fatal(err)
		}

		data = *d
		break
	}

	fold := path.Join(dest_dir, product_id)
	log.Print("Destination directory:", fold)

	if CheckIfInstalled(product_id) != true {
		config.Games = append(config.Games, Game{
			ProductID: product_id,
			Location:  dest_dir,
			Version:   data.LatestVersion,
		})
		WriteConfig(config)
	} else {
		idx := GetGameConfigIndex(product_id)
		if config.Games[idx].Location != dest_dir {
			config.Games[idx].Location = dest_dir
			WriteConfig(config)
		}
	}

	gconf := GetGameConfig(product_id)
	DoGameDownload(ctx, data.FileListURL, fold, data.Sign, *gconf)

	log.Print("Finished downloading")
}
