package main

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// EchoHandler Echoes messages to stdout
type EchoHandler struct {
}

// Init Spins up our channel handling
func (eh *EchoHandler) Init() {

}

func (eh *EchoHandler) Handler(s *discordgo.Session, i *discordgo.InteractionCreate) {
}

func (eh *EchoHandler) Message(s *discordgo.Session, m *discordgo.MessageCreate) {
	fmt.Println(m.Content)
}
