package main

import (
	"regexp"
	"unicode"

	"github.com/bwmarrin/discordgo"
)

// AlternatingCaseHandler Echoes messages to stdout
type AlternatingCaseHandler struct {
}

// Init Handles setting up our channel listener
func (ach *AlternatingCaseHandler) Init() {
	//Nothing to initialize
}

func (ach *AlternatingCaseHandler) GetApplicationCommand() *discordgo.ApplicationCommand {
	return nil
}

func (ach *AlternatingCaseHandler) Handler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	//nothing to do here
}

func (ach *AlternatingCaseHandler) Message(s *discordgo.Session, m *discordgo.MessageCreate) {
	match, _ := regexp.MatchString("^/ac (.+)", m.Content)
	if match {
		//Alternate case the important bits
		sliced := []rune(m.Content[4:len(m.Content)])
		MessageSender.SendMessage(m.ChannelID, ach.alternateCase(sliced))
		MessageSender.DeleteMessage(m.ChannelID, m.ID)
	}
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
