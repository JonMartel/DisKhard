package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os/exec"
	"regexp"
	"time"

	"github.com/bwmarrin/discordgo"
)

//FortuneHandler Echoes messages to stdout
type FortuneHandler struct {
	active       bool
	channelIDs   []string
	fortuneRegex *regexp.Regexp
}

const fortuneFile = "./fortuneData.json"

//Init Nothing to do here
func (fh *FortuneHandler) Init(m chan *discordgo.MessageCreate) {
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
			if err != nil {
				fmt.Println("Error unmarshalling json", err)
			}
			fh.channelIDs = chans
		}

		//Set up our regexp
		fh.fortuneRegex = regexp.MustCompile(`^/fortune$`)
	}

	fh.active = (err == nil)

	go func() {
		minuteSchedule := time.NewTicker(time.Minute)
		defer minuteSchedule.Stop()
		for {
			select {
			case message := <-m:
				if message != nil {
					fh.handleMessage(message)
				} else {
					return
				}
			case <-minuteSchedule.C:
				fh.scheduledTask()
			}
		}
	}()

}

//GetName returns name of handler
func (fh *FortuneHandler) GetName() string {
	return "Fortune Handler"
}

//HandleMessage echoes the messages seen to stdout
func (fh *FortuneHandler) handleMessage(m *discordgo.MessageCreate) {
	if fh.active {
		if fh.fortuneRegex.MatchString(m.Content) {
			channelID := make([]string, 1)
			channelID[0] = m.ChannelID
			fh.generateFortune(channelID)
		}
	}
}

//ScheduledTask enmpty func to comply with interface reqs
func (fh *FortuneHandler) scheduledTask() {
	if fh.active {
		current := time.Now()

		//At 9, let's fortune!
		if current.Hour() == 9 && current.Minute() == 0 {
			fh.generateFortune(fh.channelIDs)
		}
	}
}

func (fh *FortuneHandler) generateFortune(channelIDs []string) {
	command := "fortune"

	//Special frogtime for wednesday?
	//command = "fortune | cowsay -f bud-frogs"

	out, err := exec.Command(command, "startrek").Output()
	output := string(out)
	if err == nil {
		for _, channelID := range channelIDs {
			MessageSender.SendMessage(channelID, output)
		}
	}
}

//Help Gets info about this release handler
func (fh *FortuneHandler) Help() string {
	return "(Fortune Handler Active)"
}
