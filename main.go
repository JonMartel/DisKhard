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

var handlers []MessageHandler

func main() {

	configuration := Init()
	dg, err := discordgo.New("Bot " + configuration.Token)
	if err != nil {
		fmt.Println("Error creating Discord session: ", err)
		return
	}

	fmt.Println("Using token: " + configuration.Token)

	handlers = setupHandlers()

	dg.AddHandler(messageCreate)
	dg.AddHandler(guildCreate)

	dg.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildVoiceStates)
	err = dg.Open()
	defer dg.Close()
	if err != nil {
		fmt.Println("Error opening Discord session: ", err)
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Diskhard is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
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

func setupHandlers() []MessageHandler {
	slices := []MessageHandler{
		//&EchoHandler{},
		&AlternatingCaseHandler{},
	}
	return slices
}

//messageCreate handles passing messages to our interested handlers
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore ourselves
	if m.Author.ID == s.State.User.ID {
		return
	}

	for _, handler := range handlers {
		handler.HandleMessage(s, m)
	}
}

func guildCreate(s *discordgo.Session, event *discordgo.GuildCreate) {

	if event.Guild.Unavailable {
		return
	}

	for _, channel := range event.Guild.Channels {
		if channel.ID == event.Guild.ID {
			//_, _ = s.ChannelMessageSend(channel.ID, "Airhorn is ready! Type !airhorn while in a voice channel to play a sound.")
			//return
			fmt.Printf("Guild ID: %s\n", channel.GuildID)
		}
	}
}
