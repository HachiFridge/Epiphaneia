package main

import (
	"log"
	"net/url"
)

func HandleProtocol(u string) {
	parsed, err := url.Parse(u)
	if err != nil {
		log.Fatal("Could not parse URL")
	}

	if parsed.Scheme != "dmmgameplayer" {
		log.Fatal("Invalid scheme", parsed.Scheme)
	}

	log.Print("Handling protocol: ", u)

	if parsed.Host == "view" && parsed.Path == "/page" {
		m, err := url.ParseQuery(parsed.RawQuery)
		if err != nil {
			log.Fatal(err)
		}

		if m.Has("code") != true {
			log.Fatal("missing `code` query")
		}

		IssueAccessToken(m["code"][0])
		return
	}

	log.Fatalf("Unhandled protocol %s", u)
}
