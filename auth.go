package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
)

func DoAuthenticationFlow() {

	if config.AccessToken != "" {
		log.Print("Checking access token...")
		if err := client.CheckAccessToken(config.AccessToken); err == nil {
			log.Print("Already authenticated, quitting...")
			return
		} else {
			log.Fatal("It looks like an error occurred trying to check the access token:", err)
		}
	}

	u, err := client.LoginURL()
	if err != nil {
		log.Fatal("Could not request OAuth URL")
	}

	fmt.Println("Visit this URL to authenticate:", u.URL)
	fmt.Println("After authenticating, do NOT open DMM GAMES PLAYER when prompted.")
	fmt.Println("The authentication token is the text after `code=` in the URL.")
	fmt.Println("")
	fmt.Print("Enter the authentication token: ")

	var token string

	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		token = scanner.Text()
	}

	if err := scanner.Err(); err != nil {
		log.Fatal("Could not initialize prompt:", err)
	}

	res, err := client.IssueAccessToken(token)
	if err != nil {
		log.Fatal("Could not issue access token:", err)
	}
	config.AccessToken = res.AccessToken

	if err := WriteConfig(config); err != nil {
		log.Fatal("Unable to write config")
	}

	client.RecordLogin()
	user, err := client.GetUserInfo()
	if err != nil {
		log.Fatal("Couldn't retrieve user information")
	}

	fmt.Printf("Logged in as %s (%s)\n", user.Profile.Nickname, user.Email)
}
