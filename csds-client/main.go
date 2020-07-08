package main

import (
	"envoy-tools/csds-client/client"
	"fmt"
)

func main() {
	_, error := client.New()
	if error != nil {
		fmt.Println(fmt.Errorf("%v", error).Error())
	}
}