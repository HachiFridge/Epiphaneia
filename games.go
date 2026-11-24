package main

import "log"

func CheckIfInstalled(product_id string) bool {
	for _, v := range config.Games {
		if product_id == v.ProductID {
			return true
		}
	}

	return false
}

func LaunchGame(product_id string) {
	if CheckIfInstalled(product_id) == false {
		log.Fatal(product_id, " is not installed")
	}
}
