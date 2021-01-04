package main

import "github.com/bwmarrin/discordgo"

//MessageHandler Defines the functions all handlers should implement
type MessageHandler interface {
	Init()
	GetName() string
	GetCommand() string
	HandleMessage(s *discordgo.Session, m *discordgo.MessageCreate)
	ScheduledTask(s *discordgo.Session)
	Help() string
}
