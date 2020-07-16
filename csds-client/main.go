package main

import (
	"envoy-tools/csds-client/client"
	"log"
)

func main() {
	c, err := client.New()
	if err != nil {
		log.Fatal(err)
	}

	if err := c.Run(); err != nil {
		log.Fatal(err)
	}
}
