package main

import (
	"regexp"

	"github.com/bwmarrin/discordgo"
)

//VoiceHandler Echoes messages to stdout
type VoiceHandler struct {
	command    string
	voiceRegex *regexp.Regexp
}

//Init Nothing to do here
func (vh *VoiceHandler) Init() {
	vh.command = "/v"

	vh.voiceRegex = regexp.MustCompile(`^` + vh.command + `\s*(.*)`)
}

//GetName returns name of handler
func (vh *VoiceHandler) GetName() string {
	return "Voice Handler"
}

//HandleMessage echoes the messages seen to stdout
func (vh *VoiceHandler) HandleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {

	commandMatches := vh.voiceRegex.FindStringSubmatch(m.Content)
	if commandMatches != nil {
		// Find the channel that the message came from.
		c, err := s.State.Channel(m.ChannelID)
		if err != nil {
			// Could not find channel.
			return
		}

		// Find the guild for that channel.
		g, err := s.State.Guild(c.GuildID)
		if err != nil {
			// Could not find guild.
			return
		}

		// Look for the message sender in that guild's current voice states.
		for _, vs := range g.VoiceStates {
			if vs.UserID == m.Author.ID {
				/*
					vc, err := s.ChannelVoiceJoin(c.GuildID, m.ChannelID, true, true)
					if err == nil {
						//nothin!
						vc.Speaking(false)
					} else {
						fmt.Println("Error joining channel", err)
					}
				*/
			}
		}
	}
}

//ScheduledTask enmpty func to comply with interface reqs
func (vh *VoiceHandler) ScheduledTask(s *discordgo.Session) {
	//nothing
}

//Help Gets info about this release handler
func (vh *VoiceHandler) Help() string {
	return "/v : Join a voice channel and do voice-related things"
}
