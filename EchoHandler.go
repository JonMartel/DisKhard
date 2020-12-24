package main

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

//EchoHandler Echoes messages to stdout
type EchoHandler struct {
}

//Init Nothing to do here
func (eh *EchoHandler) Init() {

}

//GetName returns name of handler
func (eh *EchoHandler) GetName() string {
	return "Echo Handler"
}

//HandleMessage echoes the messages seen to stdout
func (eh *EchoHandler) HandleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	fmt.Printf("Message: %s\n", m.Content)
}

//ScheduledTask enmpty func to comply with interface reqs
func (eh *EchoHandler) ScheduledTask(s *discordgo.Session) {
	//nothing
}

//Help Gets info about this release handler
func (eh *EchoHandler) Help() string {
	return "(Echo Handler Active)"
}
