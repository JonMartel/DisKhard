package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"regexp"

	"github.com/bwmarrin/discordgo"
)

// ReactionHandler selectively Reactions on keywords
type ReactionHandler struct {
	reactionMap map[*regexp.Regexp]string
}

const reactionDataFile = "./reactionData.json"

type reactionData struct {
	TriggerWord string `json:"TriggerWord"`
	Reaction    string `json:"Reaction"`
}

// Init read in our configured Reaction keywords
func (rh *ReactionHandler) Init() {
	rh.reactionMap = make(map[*regexp.Regexp]string)

	// Get configuration
	fileData, err := ioutil.ReadFile(reactionDataFile)

	//Load up in-memory cache of this info
	if err == nil {
		fmt.Println("Reading Reaction notification data")
		var data []reactionData
		err = json.Unmarshal(fileData, &data)
		if err == nil {
			for _, reactionDef := range data {
				regex := regexp.MustCompile(`^.*` + reactionDef.TriggerWord + `.*$`)
				rh.reactionMap[regex] = reactionDef.Reaction
			}
		}
	}
}

func (rh *ReactionHandler) GetApplicationCommand() *discordgo.ApplicationCommand {
	return nil
}

func (rh *ReactionHandler) Handler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	//Empty, unused
}

// HandleMessage echoes the messages seen to stdout
func (rh *ReactionHandler) Message(s *discordgo.Session, m *discordgo.MessageCreate) {
	for regex, reaction := range rh.reactionMap {
		if regex.MatchString(m.Content) {
			MessageSender.React(m.ChannelID, m.ID, reaction)
		}
	}
}
