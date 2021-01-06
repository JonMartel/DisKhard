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

//Init does nothing for this handler
func (ach *AlternatingCaseHandler) Init() {

}

//GetName returns the name of this handler
func (ach *AlternatingCaseHandler) GetName() string {
	return "Alternating Case Handler"
}

//HandleMessage echoes the messages seen to stdout
func (ach *AlternatingCaseHandler) HandleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	match, _ := regexp.MatchString(acCommand+"(.+)", m.Content)
	if match == true {
		//Alternate case the important bits
		sliced := []rune(m.Content[4:len(m.Content)])
		_, _ = s.ChannelMessageSend(m.ChannelID, alternateCase(sliced))
		_ = s.ChannelMessageDelete(m.ChannelID, m.ID)
	}
}

//Help Gets info about this handler
func (ach *AlternatingCaseHandler) Help() string {
	return acCommand + ": Alternate Case - takes input string and aLtErNaTeS iT!"
}

func alternateCase(sliced []rune) string {
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

//ScheduledTask empty function to comply with interface reqs
func (ach *AlternatingCaseHandler) ScheduledTask(s *discordgo.Session) {
	//nothing
}
