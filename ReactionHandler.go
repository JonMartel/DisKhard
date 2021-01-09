package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"regexp"

	"github.com/bwmarrin/discordgo"
)

//ReactionHandler selectively Reactions on keywords
type ReactionHandler struct {
	reactionMap map[*regexp.Regexp]string
}

const reactionDataFile = "./reactionData.json"

type reactionData struct {
	TriggerWord string `json:"TriggerWord"`
	Reaction    string `json:"Reaction"`
}

//Init read in our configured Reaction keywords
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

//GetName returns name of handler
func (rh *ReactionHandler) GetName() string {
	return "Reaction Handler"
}

//HandleMessage echoes the messages seen to stdout
func (rh *ReactionHandler) HandleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	for regex, Reaction := range rh.reactionMap {
		if regex.MatchString(m.Content) {
			err := s.MessageReactionAdd(m.ChannelID, m.ID, Reaction)
			if err != nil {
				fmt.Println("Failed to add reaction", err)
			}
		}
	}
}

//ScheduledTask enmpty func to comply with interface reqs
func (rh *ReactionHandler) ScheduledTask(s *discordgo.Session) {
	//nothing
}

//Help Gets info about this release handler
func (rh *ReactionHandler) Help() string {
	return "(Reaction Handler Active)"
}
