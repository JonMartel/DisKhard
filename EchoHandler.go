package main

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

//EchoHandler Echoes messages to stdout
type EchoHandler struct {
}

//Init Spins up our channel handling
func (eh *EchoHandler) Init(m chan *discordgo.MessageCreate) {
	go func() {
		for {
			select {
			case message := <-m:
				if message != nil {
					fmt.Printf("Message: %s\n", message.Content)
				} else {
					return
				}
			}
		}
	}()
}

//GetName returns name of handler
func (eh *EchoHandler) GetName() string {
	return "Echo Handler"
}

//Help Gets info about this release handler
func (eh *EchoHandler) Help() string {
	return "(Echo Handler Active)"
}
