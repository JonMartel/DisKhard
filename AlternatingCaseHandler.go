package main

import (
	"regexp"
	"unicode"

	"github.com/bwmarrin/discordgo"
)

//AlternatingCaseHandler Echoes messages to stdout
type AlternatingCaseHandler struct {
}

const acCommand = "/ac "

//Init Handles setting up our channel listener
func (ach *AlternatingCaseHandler) Init(m chan *discordgo.MessageCreate) {
	go func() {
		for {
			select {
			case message := <-m:
				if message != nil {
					ach.handleMessage(message)
				} else {
					return
				}
			}
		}
	}()
}

//GetName returns the name of this handler
func (ach *AlternatingCaseHandler) GetName() string {
	return "Alternating Case Handler"
}

//HandleMessage echoes the messages seen back, but aLtErNaTiNg CaSe
func (ach *AlternatingCaseHandler) handleMessage(m *discordgo.MessageCreate) {
	match, _ := regexp.MatchString("^"+acCommand+"(.+)", m.Content)
	if match == true {
		//Alternate case the important bits
		sliced := []rune(m.Content[4:len(m.Content)])
		MessageSender.SendMessage(m.ChannelID, ach.alternateCase(sliced))
		MessageSender.DeleteMessage(m.ChannelID, m.ID)
	}
}

//Help Gets info about this handler
func (ach *AlternatingCaseHandler) Help() string {
	return acCommand + ": Alternate Case - takes input string and aLtErNaTeS iT!"
}

func (ach *AlternatingCaseHandler) alternateCase(sliced []rune) string {
	uppered := false

	for i := 0; i < len(sliced); i++ {
		thisRuneLower := unicode.ToLower(sliced[i])
		thisRuneUpper := unicode.ToUpper(sliced[i])

		if uppered {
			sliced[i] = thisRuneUpper
		} else {
			sliced[i] = thisRuneLower
		}

		//If the upper and lower are different, this means we've 'made a change', so to speak
		//even if no change needed to be applied - swap the casing we want next!
		if thisRuneLower != thisRuneUpper {
			uppered = !uppered
		}
	}

	return string(sliced)
}
