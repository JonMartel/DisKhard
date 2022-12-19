package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/bwmarrin/discordgo"
)

// IPHandler Echoes the public IP
type IPHandler struct {
}

// Init Nothing to do here
func (iph *IPHandler) Init() {
	//nothin'!
}

func (iph *IPHandler) GetApplicationCommand() *discordgo.ApplicationCommand {
	command := discordgo.ApplicationCommand{
		Name:        "whats-my-ip",
		Description: "Asks the bot to disclose it's publicly-facing IP",
	}

	return &command
}

func (iph *IPHandler) Handler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ip := iph.getIP()
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "My publicly accessible IP is: " + ip,
		},
	})
}

func (iph *IPHandler) Message(s *discordgo.Session, i *discordgo.MessageCreate) {
}

/*
{
  "ip": "173.48.36.225",
  "hostname": "pool-173-48-36-225.bstnma.fios.verizon.net",
  "city": "Methuen",
  "region": "Massachusetts",
  "country": "US",
  "loc": "42.7262,-71.1909",
  "org": "AS701 MCI Communications Services, Inc. d/b/a Verizon Business",
  "postal": "01844",
  "timezone": "America/New_York",
  "readme": "https://ipinfo.io/missingauth"
}
*/

type Message struct {
	IP       string `json:"ip"`
	Hostname string `json:"hostname"`
	City     string `json:"city"`
	Region   string `json:"region"`
	Country  string `json:"country"`
	Loc      string `json:"loc"`
	Org      string `json:"org"`
	Postal   string `json:"postal"`
	Timezone string `json:"timezone"`
	Readme   string `json:"readme"`
}

func (iph *IPHandler) getIP() string {
	var myClient = &http.Client{Timeout: 1 * time.Second}
	resp, err := myClient.Get("http://ipinfo.io")
	if err == nil {
		defer resp.Body.Close()
		message := new(Message)

		decoder := json.NewDecoder(resp.Body)
		err = decoder.Decode(message)
		if err != nil {
			print("Failed to extract ipinfo")
		} else {
			return message.IP
		}
	}
	return ""
}
