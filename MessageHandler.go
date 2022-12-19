package main

import "github.com/bwmarrin/discordgo"

//MessageHandler Defines the functions all handlers should implement
type MessageHandler interface {
	Init()
	GetApplicationCommand() *discordgo.ApplicationCommand
	Handler(s *discordgo.Session, i *discordgo.InteractionCreate)
}
