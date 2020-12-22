package main

import (
	"fmt"
	"regexp"
	"unicode"

	"github.com/bwmarrin/discordgo"
)

//AlternatingCaseHandler Echoes messages to stdout
type AlternatingCaseHandler struct {
}

//HandleMessage echoes the messages seen to stdout
func (eh *AlternatingCaseHandler) HandleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	match, _ := regexp.MatchString("^/ac (.+)", m.Content)
	if match == true {
		//Alternate case the important bits
		sliced := []byte(m.Content[4:len(m.Content)])
		_, _ = s.ChannelMessageSend(m.ChannelID, alternateCase(sliced))
	}
	fmt.Printf("Message: %s\n", m.Content)
}

func alternateCase(sliced []byte) string {
	for i := 0; i < len(sliced); i += 2 {
		sliced[i] = byte(unicode.ToLower(rune(sliced[i])))
	}

	for i := 1; i < len(sliced); i += 2 {
		sliced[i] = byte(unicode.ToUpper(rune(sliced[i])))
	}

	return string(sliced)
}
