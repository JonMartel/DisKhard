package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
)

// ReleaseHandler Echoes messages to stdout
type ReleaseHandler struct {
	matcher       regexp.Regexp
	addMatcher    regexp.Regexp
	editMatcher   regexp.Regexp
	deleteMatcher regexp.Regexp
	dateMatcher   regexp.Regexp

	releases map[string]*channelReleaseData
}

type releaseData struct {
	Name        string `json:"name"`
	ReleaseDate string `json:"releasedate"`
	ParsedDate  *time.Time
}

type channelReleaseData struct {
	ChannelID       string        `json:"channelID"`
	PinnedMessageID string        `json:"pinnedMessageID"`
	Releases        []releaseData `json:"releaseData"`
}

type byReleaseDate []releaseData

func (s byReleaseDate) Len() int {
	return len(s)
}
func (s byReleaseDate) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s byReleaseDate) Less(i, j int) bool {
	//First - can we simply sort by parsed date?
	if s[i].ParsedDate != nil {
		if s[j].ParsedDate == nil {
			return true
		}
		return s[i].ParsedDate.Before(*s[j].ParsedDate)
	} else if s[j].ParsedDate != nil {
		return false
	}

	//ParsedDate failed us - next, see if release is QXYYYY format
	//If so, prioritize whichever is earliest
	quarterRegex := regexp.MustCompile(`^Q(\d)(\d\d\d\d)$`)
	iMatches := quarterRegex.FindStringSubmatch(s[i].ReleaseDate)
	jMatches := quarterRegex.FindStringSubmatch(s[j].ReleaseDate)
	if iMatches != nil && jMatches != nil {
		iQuarter, iYear, iErr := extractQuarterInfo(iMatches[1], iMatches[2])
		jQuarter, jYear, jErr := extractQuarterInfo(jMatches[1], jMatches[2])

		if iErr == nil && jErr == nil {
			return (iYear < jYear ||
				(iYear == jYear && iQuarter < jQuarter) ||
				(iYear == jYear && iQuarter == jQuarter && s[i].Name < s[j].Name))
		} else if iErr != nil || jErr != nil {
			return iErr == nil
		}

	} else if iMatches != nil || jMatches != nil {
		return iMatches != nil
	}

	//Final fallback, release name
	return s[i].Name < s[j].Name
}

func extractQuarterInfo(quarter, year string) (int, int, error) {
	iQuarter, err := strconv.Atoi(quarter)
	if err == nil {
		iYear, err := strconv.Atoi(year)
		if err == nil {
			return iQuarter, iYear, nil
		}
	}

	return -1, -1, err
}

const rwCommand string = "/rw"
const dataFile = "./releaseData.json"

// Init compiles regexp and loads in saved information
func (rh *ReleaseHandler) Init() {
	rh.matcher = *regexp.MustCompile(`^\` + rwCommand + `\s+(\w+)\s*(.*)`)
	rh.addMatcher = *regexp.MustCompile(`^([\w-/]+) (.*)`)
	rh.editMatcher = *regexp.MustCompile(`^(\d+) ([\w-/]+)`)
	rh.deleteMatcher = *regexp.MustCompile(`^(\d+)`)
	rh.dateMatcher = *regexp.MustCompile(`(\d+)[-\/](\d+)[-\/](\d+)`)
	rh.releases = make(map[string]*channelReleaseData)

	//Need to read in stored json info as well!
	var data []channelReleaseData

	// Get configuration
	fileData, err := ioutil.ReadFile(dataFile)

	//Load up in-memory cache of this info
	if err == nil {
		fmt.Println("Reading saved release data")
		err = json.Unmarshal(fileData, &data)
		if err == nil {
			for _, channelData := range data {
				for _, release := range channelData.Releases {
					//Try to update this release's ParsedDate
					//This will ensure we convert any releases missing parsed times
					rh.updateReleaseTime(&release)
				}

				//Sort our slices now, in case the ordering changed by updating
				//parsed dates above
				sort.Stable(byReleaseDate(channelData.Releases))
				channelCopy := channelData
				rh.releases[channelData.ChannelID] = &channelCopy
			}
		}
	}

	go func() {
		//Now, get our schedule ready
		minuteSchedule := time.NewTicker(time.Minute)
		defer minuteSchedule.Stop()
		for {
			<-minuteSchedule.C
			rh.scheduledTask()
		}
	}()
}

func (rh *ReleaseHandler) GetApplicationCommand() *discordgo.ApplicationCommand {
	command := discordgo.ApplicationCommand{
		Name:        "release-watch",
		Description: "Track upcoming releases",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "list",
				Description: "Displays all currently tracked releases",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
			},
			{
				Name:        "track",
				Description: "Add a new release to be tracked",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "name",
						Description: "Name of the thing being released",
						Required:    true,
					},
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "date",
						Description: "Release date (MM/DD/YYYY, QXYYYY, or Freeform)",
						Required:    true,
					},
				},
			},
			{
				Name:        "edit",
				Description: "Edit the release date for a specified release",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionInteger,
						Name:        "index",
						Description: "Index of the release to edit",
						Required:    true,
					},
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "date",
						Description: "Release date (MM/DD/YYYY, QXYYYY, or Freeform)",
						Required:    true,
					},
				},
			},
			{
				Name:        "delete",
				Description: "Remove a release from tracking",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionInteger,
						Name:        "index",
						Description: "Index of the release to untrack",
						Required:    true,
					},
				},
			},
		},
	}

	return &command
}

func (rh *ReleaseHandler) Handler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	command := i.ApplicationCommandData().Options[0]
	options := command.Options
	content := ""

	// As you can see, names of subcommands (nested, top-level)
	// and subcommand groups are provided through the arguments.
	switch command.Name {
	case "list":
		content = rh.list(i)
	case "track":
		content = rh.add(i, options)
	case "edit":
		content = rh.edit(i, options)
	case "delete":
		content = rh.delete(i, options)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	})
}

func (rh *ReleaseHandler) scheduledTask() {
	currentTime := time.Now()
	cdate := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, time.Local)
	changed := false
	if currentTime.Hour() == 11 && currentTime.Minute() == 0 {
		for _, channelData := range rh.releases {
			tempChannelReleases := channelData.Releases[:0]
			for _, release := range channelData.Releases {
				//Does this release have a notifiable release date specified?
				if release.ParsedDate != nil {

					nextWeek := cdate.AddDate(0, 0, 7)
					tomorrow := cdate.AddDate(0, 0, 1)

					if cdate == *release.ParsedDate {
						MessageSender.SendMessage(channelData.ChannelID, release.Name+" released today!")
					} else {
						//Regardless if we notify, add to the new list
						tempChannelReleases = append(tempChannelReleases, release)

						//Notify if appropriate!
						if nextWeek == *release.ParsedDate {
							MessageSender.SendMessage(channelData.ChannelID, release.Name+" is releasing next week!")
						} else if tomorrow == *release.ParsedDate {
							MessageSender.SendMessage(channelData.ChannelID, release.Name+" is releasing tomorrow!")
						}
					}
				} else {
					tempChannelReleases = append(tempChannelReleases, release)
				}
			}

			if len(tempChannelReleases) != len(channelData.Releases) {
				channelData.Releases = tempChannelReleases
				rh.updateChannelPin(channelData.ChannelID)
				changed = true
			}
		}
	}

	if changed {
		rh.writeData()
	}
}

func (rh *ReleaseHandler) writeData() {
	//join all the releases into a single slice...
	channelDataSlice := make([]channelReleaseData, 0)

	for _, channelData := range rh.releases {
		channelDataSlice = append(channelDataSlice, *channelData)
	}
	//..which we then byte-ify and write to disk
	jsonBytes, err := json.Marshal(channelDataSlice)
	if err == nil {
		ioutil.WriteFile(dataFile, jsonBytes, 0644)
	}
}

func (rh *ReleaseHandler) add(i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) string {

	// name string, date string
	releaseInfo := releaseData{}
	releaseInfo.Name = options[0].StringValue()
	releaseInfo.ReleaseDate = options[1].StringValue()

	rh.updateReleaseTime(&releaseInfo)

	message := ""
	if releaseInfo.ParsedDate != nil {
		now := time.Now()
		if !now.Before(*releaseInfo.ParsedDate) {
			message = "Error: Specified date \"" + options[1].StringValue() + "\" is in the past!"
		}
	}

	if message == "" {
		channel, ok := rh.releases[i.ChannelID]
		if !ok {
			channel = rh.initChannel(i.ChannelID)
		}

		channel.Releases = append(channel.Releases, releaseInfo)
		sort.Stable(byReleaseDate(channel.Releases))

		rh.writeData()
		rh.updateChannelPin(i.ChannelID)
		message = "Added " + releaseInfo.Name + " to releases, releasing " + releaseInfo.ReleaseDate
	}

	return message
}

func (rh *ReleaseHandler) list(i *discordgo.InteractionCreate) string {
	formattedChannelRelease := rh.formatChannelReleases(i.ChannelID)
	return formattedChannelRelease
}

func (rh *ReleaseHandler) formatChannelReleases(channelID string) string {
	list := "Here are my currently tracked releases:\n"

	if channelData, ok := rh.releases[channelID]; ok {
		if channelData.Releases != nil && len(channelData.Releases) > 0 {
			for x, release := range channelData.Releases {
				if release.ParsedDate != nil {
					list += release.ParsedDate.Format("01-02-2006")
				} else {
					list += release.ReleaseDate
				}
				list += " " + release.Name + " [" + strconv.FormatInt(int64(x), 10) + "]\n"
			}
		} else {
			list += "<No tracked releases>"
		}
	} else {
		list += "<No tracked releases>"
	}

	return list
}

func (rh *ReleaseHandler) edit(i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) string {
	message := ""
	index := (int)(options[0].IntValue())
	newReleaseDate := options[1].StringValue()

	if channelData, ok := rh.releases[i.ChannelID]; ok {

		slice := channelData.Releases
		if slice != nil {
			if len(slice) > index || index < 0 {
				entry := &slice[index]
				entryName := entry.Name
				entry.ReleaseDate = newReleaseDate
				rh.updateReleaseTime(entry)
				sort.Stable(byReleaseDate(slice))

				rh.updateChannelPin(channelData.ChannelID)
				rh.writeData()
				message = "Successfully updated release date for " + entryName
			} else {
				//invalid index provided
				message = "Invalid ID specified"
			}
		} else {
			//Channel has no releases?
			message = "No releases currently available to edit"
		}
	} else {
		//Channel data doesn't exist (yet), hence no releases to edit
		message = "No releases currently available to edit"
	}

	return message
}

func (rh *ReleaseHandler) delete(i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) string {
	message := ""
	index := (int)(options[0].IntValue())

	if channelData, ok := rh.releases[i.ChannelID]; ok {
		if channelData.Releases != nil {
			if len(channelData.Releases) > index || index < 0 {
				removedRelease := channelData.Releases[index]
				fmt.Println("Removing " + removedRelease.Name + " (" + removedRelease.ReleaseDate + ") from releases")
				channelData.Releases = append(channelData.Releases[:index], channelData.Releases[index+1:]...)
				rh.updateChannelPin(channelData.ChannelID)
				rh.writeData()
				message = "Removed " + removedRelease.Name + " from releases"
			} else {
				message = "Error: Invalid ID specified"
			}
		} else {
			message = "Error: Channel does not have any releases to delete!"
		}
	} else {
		message = "Error: Channel does not have any releases to delete!"
	}

	return message
}

func (rh *ReleaseHandler) updateReleaseTime(rel *releaseData) {

	dateMatch := rh.dateMatcher.FindStringSubmatch(rel.ReleaseDate)
	if dateMatch != nil {
		//It does, do we need to notify?
		day, err1 := strconv.Atoi(dateMatch[2])
		month, err2 := strconv.Atoi(dateMatch[1])
		year, err3 := strconv.Atoi(dateMatch[3])

		if err1 != nil || err2 != nil || err3 != nil {
			fmt.Println("Error parsing dates for release notification")
			return
		}

		//Must be using 2 digit year format!
		if year < 100 {
			year += 2000
		}

		parsed := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
		rel.ParsedDate = &parsed
	}
}

func (rh *ReleaseHandler) updateChannelPin(channelID string) {
	message := rh.formatChannelReleases(channelID)

	channel, ok := rh.releases[channelID]
	if !ok {
		channel = rh.initChannel(channelID)
	}

	if channel.PinnedMessageID != "" {
		MessageSender.EditMessage(channel.ChannelID, channel.PinnedMessageID, message)
	} else {
		//We need to blast out our release entries and then set our message id for this channel
		//If we error, do *not* set our pin message id
		sentMessage, error := MessageSender.SendMessage(channelID, message)
		if error == nil {
			id := sentMessage.ID
			pinError := MessageSender.PinMessage(channel.ChannelID, id)
			if pinError == nil {
				channel.PinnedMessageID = id
			}
		} else {
			fmt.Println("Error sending message: " + error.Error())
		}
	}
}

func (rh *ReleaseHandler) initChannel(channelID string) *channelReleaseData {
	//Spin up our channel and return it
	channel := &channelReleaseData{}
	channel.ChannelID = channelID
	channel.Releases = make([]releaseData, 0)

	rh.releases[channelID] = channel

	return channel
}
