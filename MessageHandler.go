package main

import "github.com/bwmarrin/discordgo"

//MessageHandler Defines the functions all handlers should implement
type MessageHandler interface {
	HandleMessage(s *discordgo.Session, m *discordgo.MessageCreate)
}
