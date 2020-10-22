package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

//Configuration Struct used to store config info from json
type Configuration struct {
	Token string `json:"Token"`
}

func main() {

	configuration := Init()
	dg, err := discordgo.New("Bot " + configuration.Token)
	if err != nil {
		fmt.Println("Error creating Discord session: ", err)
		return
	} else {
		fmt.Println("Using token: " + configuration.Token)
	}

	//dg.addHandler(ready)
	//dg.addHandler(messageCreate)

	defer dg.Close()

	err = dg.Open()
	if err != nil {
		fmt.Println("Error opening Discord session: ", err)
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Diskhard is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()

}

//Init Reads in the configuration
func Init() Configuration {

	var configuration Configuration

	// Get configuration
	config, err := ioutil.ReadFile("./diskhard.json")
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println("Error reading configuration:")
		fmt.Println(err)
	} else {
		fmt.Println("Read configuration")
		json.Unmarshal(config, &configuration)
	}

	return configuration
}
