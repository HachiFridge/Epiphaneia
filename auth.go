package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
)

func CheckAccessToken() error {
	if config.AccessToken == "" {
		return fmt.Errorf("No access token configured. Run with -auth first.")
	}
	log.Print("Checking access token...")
	if err := client.CheckAccessToken(config.AccessToken); err != nil {
		return fmt.Errorf("Access token error: %v", err)
	}
	log.Print("Access token verified")

	return nil
}

func IssueAccessToken(code string) error {
	log.Print("Issuing access token...")
	res, err := client.IssueAccessToken(code)
	if err != nil {
		return fmt.Errorf("Could not issue access token: %v", err)
	}
	config.AccessToken = res.AccessToken

	if err := WriteConfig(config); err != nil {
		return fmt.Errorf("Unable to write config")
	}

	log.Print("Recording login to DMM server...")
	client.RecordLogin()

	return nil
}

func DoAuthenticationFlow() {

	if err := CheckAccessToken(); err == nil {
		log.Print("Already authenticated, quitting...")
		return
	}

	u, err := client.LoginURL()
	if err != nil {
		log.Fatal("Could not request OAuth URL")
	}

	fmt.Println("Wait for the browser to open or visit this URL to authenticate:", u.URL)
	fmt.Println("After authenticating, do NOT open DMM GAMES PLAYER when prompted.")
	fmt.Println("The authentication token is the text after `code=` in the URL.")
	fmt.Println("")
	open_browser(u.URL)

	fmt.Print("Enter the authentication token: ")

	var token string

	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		token = scanner.Text()
	}

	if err := scanner.Err(); err != nil {
		log.Fatal("Could not initialize prompt:", err)
	}

	if err := IssueAccessToken(token); err != nil {
		log.Fatal(err)
	}

	user, err := client.GetUserInfo()
	if err != nil {
		log.Fatal("Couldn't retrieve user information")
	}

	log.Print("Logged in as", user.Profile.Nickname, user.Email)
}
