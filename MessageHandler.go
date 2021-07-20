package main

import "github.com/bwmarrin/discordgo"

//MessageHandler Defines the functions all handlers should implement
type MessageHandler interface {
	Init(m chan *discordgo.MessageCreate)
	GetName() string
	Help() string
}
