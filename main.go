package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"regexp"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

//Configuration Struct used to store config info from json
type Configuration struct {
	Token string `json:"Token"`
	Name  string `json:"Name"`
}

var handlers []MessageHandler
var nameRegex regexp.Regexp

func main() {

	configuration := Init()
	dg, err := discordgo.New("Bot " + configuration.Token)
	if err != nil {
		fmt.Println("Error creating Discord session: ", err)
		os.Exit(1)
	}

	fmt.Println("Using token: " + configuration.Token)

	handlers = setupHandlers()
	regexPattern := "\\!" + configuration.Name
	nameRegex = *regexp.MustCompile(regexPattern)

	//Daily Checks
	hourSchedule := time.NewTicker(time.Minute)
	defer hourSchedule.Stop()

	dg.AddHandler(messageCreate)
	dg.AddHandler(guildCreate)

	dg.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildVoiceStates)
	err = dg.Open()
	defer dg.Close()
	if err != nil {
		fmt.Println("Error opening Discord session: ", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Diskhard is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)

	//Run until we're done! Handle scheduled tasks as needed
	for {
		select {
		case <-sc:
			return
		case <-hourSchedule.C:
			scheduledTask(dg)
		}
	}
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
		&ReleaseHandler{},
	}

	for _, handler := range slices {
		handler.Init()
		fmt.Println("Initialized ", handler.GetName())
	}

	return slices
}

//messageCreate handles passing messages to our interested handlers
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore ourselves
	if m.Author.ID == s.State.User.ID {
		return
	}

	if nameRegex.MatchString(m.Content) {
		showHandlerInfo(s, m.ChannelID)
	} else {
		for _, handler := range handlers {
			handler.HandleMessage(s, m)
		}
	}
}

//scheduledTask handles passing our scheduled tasks to our handlers
func scheduledTask(s *discordgo.Session) {
	for _, handler := range handlers {
		handler.ScheduledTask(s)
	}
}

func guildCreate(s *discordgo.Session, event *discordgo.GuildCreate) {

	if event.Guild.Unavailable {
		return
	}

	for _, channel := range event.Guild.Channels {
		if channel.ID == event.Guild.ID {
			fmt.Printf("Guild ID: %s\n", channel.GuildID)
		}
	}
}

func showHandlerInfo(s *discordgo.Session, channelID string) {
	helpMessage := "Hey there! I currently support the following options:\n"

	for _, handler := range handlers {
		handlerHelp := handler.Help()
		if len(handlerHelp) > 0 {
			helpMessage += handlerHelp + "\n"
		}
	}

	_, _ = s.ChannelMessageSend(channelID, helpMessage)
}
