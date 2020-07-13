package main

import (
	"envoy-tools/csds-client/client"
	"fmt"
)

func main() {
	c, err := client.New()
	if err != nil {
		fmt.Printf("%v\n", err)
	}

	if err := c.Run(); err != nil {
		fmt.Printf("%v\n", err)
	}
}
