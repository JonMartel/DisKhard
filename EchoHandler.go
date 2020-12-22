package main

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

//EchoHandler Echoes messages to stdout
type EchoHandler struct {
}

//HandleMessage echoes the messages seen to stdout
func (eh *EchoHandler) HandleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	fmt.Printf("Message: %s\n", m.Content)
}
