package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

// Configuration Struct used to store config info from json
type Configuration struct {
	Token string `json:"Token"`
	Name  string `json:"Name"`
}

var handlers []MessageHandler

var handlerMap map[string]MessageHandler

var MessageSender Messager

func main() {

	configuration := Init()
	session, err := discordgo.New("Bot " + configuration.Token)
	if err != nil {
		fmt.Println("Error creating Discord session: ", err)
		os.Exit(1)
	}

	fmt.Println("Using token: " + configuration.Token)

	handlers = setupHandlers()

	session.AddHandler(ready)
	session.AddHandler(interaction)

	session.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsAllWithoutPrivileged)
	MessageSender.Init(session)
	err = session.Open()
	defer session.Close()
	if err != nil {
		fmt.Println("Error opening Discord session: ", err)
		return
	}

	//Get all the guilds we belong to (I don't have more than a handful, so I'm not worried about
	//handling more than 100 at this time. Known limitation, y'know?)
	guilds, err := session.UserGuilds(100, "", "")
	appId := session.State.User.ID
	fmt.Println("My id is maybe: " + appId)
	if err == nil {
		fmt.Println("Adding commands...")
		for _, guild := range guilds {
			/*
				fmt.Println("Guild: " + guild.Name)
				emojis, err := session.GuildEmojis(guild.ID)
				if err == nil {
					for _, emoji := range emojis {
						fmt.Println(emoji.Name + " : " + emoji.ID)
					}
				}
			*/

			//Do I need to clean up any legacy commands that shouldn't exist anymore?
			//This will remove everything that's registered with us, prior to adding what's current
			if allCommands, err := session.ApplicationCommands(appId, guild.ID); err == nil {
				fmt.Println("Commands for guild " + guild.ID + ":")
				for _, singleCommand := range allCommands {
					fmt.Println(singleCommand.Name)
					err := session.ApplicationCommandDelete(appId, guild.ID, singleCommand.ID)
					if err != nil {
						fmt.Println("Failed to remove: " + singleCommand.Name)
					}
				}
			}

			//Register commands for all our handlers (that have commands)
			handlerMap = make(map[string]MessageHandler)
			for _, handler := range handlers {
				appData := handler.GetApplicationCommand()
				if appData != nil {
					handlerMap[appData.Name] = handler
					_, err := session.ApplicationCommandCreate(session.State.User.ID, guild.ID, appData)
					if err != nil {
						log.Panicf("Cannot create '%v' command: %v\n", appData.Name, err)
					}
				}
			}

		}
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("DisKhard is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	//Run until we're done!
	<-sc
}

// Init Reads in the configuration
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
		&ReactionHandler{},
		&ImageHandler{},
		&ReminderHandler{},
		//&FortuneHandler{},
		&IPHandler{},
	}

	for _, handler := range slices {
		handler.Init()
	}

	return slices
}

func ready(s *discordgo.Session, event *discordgo.Ready) {
	s.UpdateGameStatus(0, "可笑しいな")
}

func interaction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if h, ok := handlerMap[i.ApplicationCommandData().Name]; ok {
		h.Handler(s, i)
	}
}
