package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os/exec"

	"github.com/bwmarrin/discordgo"
)

//FortuneHandler Echoes messages to stdout
type FortuneHandler struct {
	active     bool
	channelIDs []string
}

const fortuneFile = "./fortuneData.json"

//Init Nothing to do here
func (fh *FortuneHandler) Init() {
	//If fortune's usable...
	_, err := exec.Command("fortune").Output()
	if err != nil {
		fmt.Println("fortune is not accessible; disabling.", err)
	} else {
		var chans []string

		fileData, err := ioutil.ReadFile(fortuneFile)

		//Load up in-memory cache of this info
		if err == nil {
			fmt.Println("Reading saved release data")
			err = json.Unmarshal(fileData, &chans)
			fh.channelIDs = chans
		}
	}

	fh.active = (err == nil)
}

//GetName returns name of handler
func (fh *FortuneHandler) GetName() string {
	return "Fortune Handler"
}

//HandleMessage echoes the messages seen to stdout
func (fh *FortuneHandler) HandleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	//Nothin!
}

//ScheduledTask enmpty func to comply with interface reqs
func (fh *FortuneHandler) ScheduledTask(s *discordgo.Session) {
	if fh.active {
		current := time.Now()

		//At 9, let's
		if current.Hour() == 9 && current.Minute() == 0 {
			out, err := exec.Command("fortune").Output()
			output := string(out)
			if err == nil {
				for _, channelID := range fh.channelIDs {
					_, _ = s.ChannelMessageSend(channelID, output)
				}
			}
		}
	}
}

//Help Gets info about this release handler
func (fh *FortuneHandler) Help() string {
	return "(Fortune Handler Active)"
}
