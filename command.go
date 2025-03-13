package main

import (
	"fmt"
	"github.com/kjkondratuk/kinetiq/listeners"
	"log"
)

func main() {
	//	call update here
	err := listeners.ListenForUpdates()
	if err != nil {
		fmt.Println("Error listening for updates")
		log.Fatal(err)
	}
}
