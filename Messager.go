package main

import (
	"fmt"
	"os"
	"sync"

	"github.com/bwmarrin/discordgo"
)

type Messager struct {
	session      *discordgo.Session
	messageMutex sync.Mutex
}

func (m *Messager) Init(sess *discordgo.Session) {
	m.session = sess
}

func (m *Messager) SendMessage(channelID string, message string) (*discordgo.Message, error) {
	m.messageMutex.Lock()
	mess, err := m.session.ChannelMessageSend(channelID, message)
	m.messageMutex.Unlock()
	if err != nil {
		fmt.Println("Error sending message", err)
	}

	return mess, err
}

func (m *Messager) SendFile(channelID string, filePath string) error {
	if img, err := os.Open(filePath); err == nil {
		dgoFiles := make([]*discordgo.File, 0)
		dgoFiles = append(dgoFiles, &discordgo.File{
			Name:   filePath,
			Reader: img,
		})

		m.messageMutex.Lock()
		_, err := m.session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
			Files: dgoFiles,
		})
		m.messageMutex.Unlock()

		if err != nil {
			fmt.Println("Error sending image: ", err)
			return err
		}
	} else {
		return err
	}

	return nil
}

func (m *Messager) DeleteMessage(channelID string, messageID string) error {
	m.messageMutex.Lock()
	err := m.session.ChannelMessageDelete(channelID, messageID)
	m.messageMutex.Unlock()
	if err != nil {
		fmt.Println("Error removing message", err)
	}

	return err
}

func (m *Messager) EditMessage(channelID string, messageID string, newMessage string) error {
	m.messageMutex.Lock()
	_, err := m.session.ChannelMessageEdit(channelID, messageID, newMessage)
	m.messageMutex.Unlock()
	if err != nil {
		fmt.Println("Error editing message", err)
	}

	return err
}

func (m *Messager) PinMessage(channelID string, messageID string) error {
	m.messageMutex.Lock()
	err := m.session.ChannelMessagePin(channelID, messageID)
	m.messageMutex.Unlock()
	if err != nil {
		fmt.Println("Error pinning message", err)
	}

	return err
}

func (m *Messager) React(channelID string, messageID string, reaction string) error {
	m.messageMutex.Lock()
	err := m.session.MessageReactionAdd(channelID, messageID, reaction)
	m.messageMutex.Unlock()
	if err != nil {
		fmt.Println("Error reacting to message", err)
	}

	return err
}
