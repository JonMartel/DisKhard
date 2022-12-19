package main

import (
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
	command := discordgo.ApplicationCommand{
		Name:        "ac",
		Description: "Display a message using alternating case",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "message",
				Description: "Message to alternate case",
				Required:    true,
			},
		},
	}

	return &command
}

func (ach *AlternatingCaseHandler) Handler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	message := i.ApplicationCommandData().Options[0].StringValue()
	runed := []rune(message)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: ach.alternateCase(runed),
		},
	})
}

func (ach *AlternatingCaseHandler) Message(s *discordgo.Session, i *discordgo.MessageCreate) {
	//Nothing here
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
