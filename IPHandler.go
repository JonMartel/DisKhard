package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/bwmarrin/discordgo"
)

//IPHandler Echoes the public IP
type IPHandler struct {
}

//Init Nothing to do here
func (iph *IPHandler) Init(m chan *discordgo.MessageCreate) {
	go func() {
		for {
			select {
			case message := <-m:
				if message != nil {
					iph.handleMessage(message)
				} else {
					return
				}
			}
		}
	}()
}

//GetName returns name of handler
func (iph *IPHandler) GetName() string {
	return "IP Handler"
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

//handleMessage echoes the messages seen to stdout
func (iph *IPHandler) handleMessage(m *discordgo.MessageCreate) {
	if m.Content == "/ip" {
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
				MessageSender.SendMessage(m.ChannelID, "My publicly accessible IP is: "+message.IP)
				if err != nil {
					print("Error sending message")
				}
			}
			return
		}

		MessageSender.SendMessage(m.ChannelID, "Error obtaining publicly accessible IP")
	}

}

//Help Gets info about this release handler
func (iph *IPHandler) Help() string {
	return "/ip - Display the current publicly accessible IP"
}
