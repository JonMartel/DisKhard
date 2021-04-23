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

//ReleaseHandler Echoes messages to stdout
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
			return (iYear < jYear || (iYear == jYear && iQuarter < jQuarter) || s[i].Name < s[j].Name)
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

//Init compiles regexp and loads in saved information
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
}

//GetName returns our name
func (rh *ReleaseHandler) GetName() string {
	return "Release Handler"
}

//HandleMessage echoes the messages seen to stdout
func (rh *ReleaseHandler) HandleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	submatches := rh.matcher.FindStringSubmatch(m.Content)
	if submatches != nil {
		command := submatches[1]
		switch command {
		case "add":
			rh.add(s, m.ChannelID, submatches[2])
		case "list":
			rh.list(s, m.ChannelID)
		case "edit":
			rh.edit(s, m.ChannelID, submatches[2])
		case "delete":
			rh.delete(s, m.ChannelID, submatches[2])
		case "help":
			rh.help(s, m.ChannelID)
		default:
			rh.help(s, m.ChannelID)
		}

		_ = s.ChannelMessageDelete(m.ChannelID, m.ID)
	}
}

//ScheduledTask Handle our scheduled release notifications
func (rh *ReleaseHandler) ScheduledTask(s *discordgo.Session) {
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
						_, _ = s.ChannelMessageSend(channelData.ChannelID, release.Name+" released today!")
					} else {
						//Regardless if we notify, add to the new list
						tempChannelReleases = append(tempChannelReleases, release)

						//Notify if appropriate!
						if nextWeek == *release.ParsedDate {
							_, _ = s.ChannelMessageSend(channelData.ChannelID, release.Name+" is releasing next week!")
						} else if tomorrow == *release.ParsedDate {
							_, _ = s.ChannelMessageSend(channelData.ChannelID, release.Name+" is releasing tomorrow!")
						}
					}
				} else {
					tempChannelReleases = append(tempChannelReleases, release)
				}
			}

			if len(tempChannelReleases) != len(channelData.Releases) {
				channelData.Releases = tempChannelReleases
				rh.updateChannelPin(s, channelData.ChannelID)
				changed = true
			}
		}
	}

	if changed {
		rh.writeData()
	}
}

//Help Gets info about this release handler
func (rh *ReleaseHandler) Help() string {
	return "/rw : Release Watch - Tracks upcoming releases and notifies when they've arrived"
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

func (rh *ReleaseHandler) add(s *discordgo.Session, channelID string, data string) {
	match := rh.addMatcher.FindStringSubmatch(data)
	if match != nil {
		releaseInfo := releaseData{}
		releaseInfo.ReleaseDate = match[1]
		releaseInfo.Name = match[2]
		rh.updateReleaseTime(&releaseInfo)

		if releaseInfo.ParsedDate != nil {
			now := time.Now()
			if !now.Before(*releaseInfo.ParsedDate) {
				_, _ = s.ChannelMessageSend(channelID, "Error: Specified date \""+match[1]+"\" is in the past!")
				return
			}
		}

		channel, ok := rh.releases[channelID]
		if !ok {
			channel = rh.initChannel(channelID)
		}

		channel.Releases = append(channel.Releases, releaseInfo)
		sort.Stable(byReleaseDate(channel.Releases))

		rh.writeData()
		rh.updateChannelPin(s, channelID)
		_, _ = s.ChannelMessageSend(channelID, "Added "+releaseInfo.Name+" to releases, releasing "+releaseInfo.ReleaseDate)
	} else {
		_, _ = s.ChannelMessageSend(channelID, "Invalid add syntax")
	}

}

func (rh *ReleaseHandler) list(s *discordgo.Session, channelID string) {
	formattedChannelRelease := rh.formatChannelReleases(channelID)
	_, _ = s.ChannelMessageSend(channelID, formattedChannelRelease)
}

func (rh *ReleaseHandler) formatChannelReleases(channelID string) string {
	list := "Here are my currently tracked releases:\n"

	if channelData, ok := rh.releases[channelID]; ok {
		if channelData.Releases != nil && len(channelData.Releases) > 0 {
			for x, release := range channelData.Releases {
				list += strconv.FormatInt(int64(x), 10) + ") " + release.Name + " ("
				if release.ParsedDate != nil {
					list += release.ParsedDate.Format("01-02-2006")
				} else {
					list += release.ReleaseDate
				}
				list += ")\n"
			}
		} else {
			list += "<No tracked releases>"
		}
	} else {
		list += "<No tracked releases>"
	}

	return list
}

func (rh *ReleaseHandler) edit(s *discordgo.Session, channelID string, data string) {
	match := rh.editMatcher.FindStringSubmatch(data)
	if match != nil {
		index, err := strconv.Atoi(match[1])
		if err == nil {
			newReleaseDate := match[2]
			if channelData, ok := rh.releases[channelID]; ok {

				slice := channelData.Releases
				if slice != nil {
					if len(slice) > index || index < 0 {
						entry := &slice[index]
						entry.ReleaseDate = newReleaseDate
						rh.updateReleaseTime(entry)
						sort.Stable(byReleaseDate(slice))

						rh.updateChannelPin(s, channelData.ChannelID)
						rh.writeData()
						_, _ = s.ChannelMessageSend(channelID, "Successfully updated release date for "+entry.Name)
					} else {
						//invalid index provided
						_, _ = s.ChannelMessageSend(channelID, "Invalid ID specified")
					}
				} else {
					//Channel has no releases?
					_, _ = s.ChannelMessageSend(channelID, "No releases currently available to edit")
				}
			} else {
				//Channel data doesn't exist (yet), hence no releases to edit
				_, _ = s.ChannelMessageSend(channelID, "No releases currently available to edit")
			}
		} else {
			//error parsing index
			_, _ = s.ChannelMessageSend(channelID, "Could not parse ID: "+match[1])
		}
	} else {
		//Invalid command format!
		_, _ = s.ChannelMessageSend(channelID, "Invalid parameters for /rw edit, please see help")
	}
}

func (rh *ReleaseHandler) delete(s *discordgo.Session, channelID string, data string) {
	match := rh.deleteMatcher.FindStringSubmatch(data)
	if match != nil {
		index, err := strconv.Atoi(match[1])
		if err == nil {
			if channelData, ok := rh.releases[channelID]; ok {
				if channelData.Releases != nil {
					if len(channelData.Releases) > index || index < 0 {
						removedRelease := channelData.Releases[index]
						fmt.Println("Removing " + removedRelease.Name + " (" + removedRelease.ReleaseDate + ") from releases")
						channelData.Releases = append(channelData.Releases[:index], channelData.Releases[index+1:]...)
						rh.updateChannelPin(s, channelData.ChannelID)
						rh.writeData()
						_, _ = s.ChannelMessageSend(channelID, "Removed "+removedRelease.Name+" from releases")
					} else {
						_, _ = s.ChannelMessageSend(channelID, "Error: Invalid ID specified")
					}
				} else {
					_, _ = s.ChannelMessageSend(channelID, "Error: Channel does not have any releases to delete!")
				}
			} else {
				_, _ = s.ChannelMessageSend(channelID, "Error: Channel does not have any releases to delete!")
			}
		}
	}
}

func (rh *ReleaseHandler) help(s *discordgo.Session, channelID string) {
	helpMessage := "The following commands are supported by /rw:\n"
	helpMessage += "/rw add <date> <release> - Adds the following release for tracking.\n"
	helpMessage += "\t<date> can be in the following formats: MM/DD/YYYY MM-DD-YY\n"
	helpMessage += "\teg: /rw add 10/20/35 Persona 8 Dancing All 'Night\n"
	helpMessage += "/rw list - Lists all currently tracked releases\n"
	helpMessage += "/rw edit <id> <date> - Change the specified release's release date.\n"
	helpMessage += "\tID can be obtained from /rw list\n"
	helpMessage += "\teg: /rw edit 12 5/16/2024\n"
	helpMessage += "/rw delete <id> - Delete the specified release!\n\teg: /rw delete 5\n"
	helpMessage += "/rw help - This output here!"

	_, _ = s.ChannelMessageSend(channelID, helpMessage)
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

func (rh *ReleaseHandler) updateChannelPin(s *discordgo.Session, channelID string) {
	message := rh.formatChannelReleases(channelID)

	channel, ok := rh.releases[channelID]
	if !ok {
		channel = rh.initChannel(channelID)
	}

	if channel.PinnedMessageID != "" {
		_, _ = s.ChannelMessageEdit(channel.ChannelID, channel.PinnedMessageID, message)
	} else {
		//We need to blast out our release entries and then set our message id for this channel
		//If we error, do *not* set our pin message id
		sentMessage, error := s.ChannelMessageSend(channelID, message)
		if error == nil {
			id := sentMessage.ID
			pinError := s.ChannelMessagePin(channel.ChannelID, id)
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
