package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"regexp"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

//Configuration Struct used to store config info from json
type Configuration struct {
	Token string `json:"Token"`
	Name  string `json:"Name"`
}

var handlers []MessageHandler
var handlerChannels []chan *discordgo.MessageCreate
var nameRegex regexp.Regexp

//var session *discordgo.Session

var MessageSender Messager

func main() {

	configuration := Init()
	session, err := discordgo.New("Bot " + configuration.Token)
	if err != nil {
		fmt.Println("Error creating Discord session: ", err)
		os.Exit(1)
	}

	fmt.Println("Using token: " + configuration.Token)

	handlers, handlerChannels = setupHandlers()
	regexPattern := "\\!" + configuration.Name
	nameRegex = *regexp.MustCompile(regexPattern)

	session.AddHandler(ready)
	session.AddHandler(messageCreate)

	session.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsAllWithoutPrivileged)
	MessageSender.Init(session)
	err = session.Open()
	defer session.Close()
	if err != nil {
		fmt.Println("Error opening Discord session: ", err)
		return
	}

	//Diagnostic info dump
	guilds, err := session.UserGuilds(100, "", "")
	if err == nil {
		for _, guild := range guilds {
			fmt.Println("Guild: " + guild.Name)
			emojis, err := session.GuildEmojis(guild.ID)
			if err == nil {
				for _, emoji := range emojis {
					fmt.Println(emoji.Name + " : " + emoji.ID)
				}
			}
		}
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("DisKhard is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)

	//Run until we're done!
	<-sc
	for _, handlerChannel := range handlerChannels {
		handlerChannel <- nil
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

func setupHandlers() ([]MessageHandler, []chan *discordgo.MessageCreate) {
	slices := []MessageHandler{
		//&EchoHandler{},
		&AlternatingCaseHandler{},
		&ReleaseHandler{},
		&ReactionHandler{},
		&ImageHandler{},
		&ReminderHandler{},
		//&FortuneHandler{},
		//&VoiceHandler{},
		&IPHandler{},
	}

	handlerChannels := make([]chan *discordgo.MessageCreate, 0)
	for _, handler := range slices {
		handlerChannel := make(chan *discordgo.MessageCreate)
		handler.Init(handlerChannel)
		handlerChannels = append(handlerChannels, handlerChannel)
		fmt.Println("Initialized ", handler.GetName())
	}

	return slices, handlerChannels
}

func ready(s *discordgo.Session, event *discordgo.Ready) {
	s.UpdateStatus(0, "Soul Eater Hungers")
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
		for _, handlerChannel := range handlerChannels {
			handlerChannel <- m
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
