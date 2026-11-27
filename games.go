package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"path"
	"strings"
	"bufio"
	"os"
	"runtime"

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

func LaunchGame(product_id string) {
	if runtime.GOOS != "windows" {
		log.Fatal("Launching games on other platforms is not yet supported.")
	}

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
	cmd := path.Join(c.Location, data.ProductID, data.ExecFilename)
	args := strings.Split(data.ExecuteArgs, " ")

	log.Printf("Product: %s", data.ProductID)
	log.Printf("Version: %s (local: %s)", data.LatestVersion, c.Version)
	log.Printf("Executable: %s", cmd)
	log.Printf("Args: %s", data.ExecuteArgs)

	log.Printf("Launching...")

	_cmd := exec.Command(cmd, args...)
	_cmd.Start()
	_cmd.Wait()

	log.Print("Process exited")
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
