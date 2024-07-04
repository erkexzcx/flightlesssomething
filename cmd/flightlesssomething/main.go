package main

import (
	"flightlesssomething"
	"fmt"
	"log"
)

var (
	version string
)

func main() {
	c, err := flightlesssomething.NewConfig()
	if err != nil {
		log.Fatalln("Failed to get config:", err)
	}

	if c.Version {
		fmt.Println("Version:", version)
		return
	}

	flightlesssomething.Start(c)
}
