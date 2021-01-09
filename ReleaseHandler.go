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
	releases      map[string][]releaseData
}

type releaseData struct {
	Name        string `json:"name"`
	ReleaseDate string `json:"releasedate"`
	ChannelID   string `json:"channelID"`
	ParsedDate  *time.Time
}

type byReleaseDate []releaseData

func (s byReleaseDate) Len() int {
	return len(s)
}
func (s byReleaseDate) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s byReleaseDate) Less(i, j int) bool {
	if s[i].ParsedDate != nil {
		if s[j].ParsedDate == nil {
			return true
		}
		return s[i].ParsedDate.Before(*s[j].ParsedDate)
	} else if s[j].ParsedDate != nil {
		return false
	}

	return s[i].Name < s[j].Name
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
	rh.releases = make(map[string][]releaseData)

	//Need to read in stored json info as well!
	var data []releaseData

	// Get configuration
	fileData, err := ioutil.ReadFile(dataFile)

	//Load up in-memory cache of this info
	if err == nil {
		fmt.Println("Reading saved release data")
		err = json.Unmarshal(fileData, &data)
		if err == nil {
			for _, release := range data {
				//Try to update this release's ParsedDate
				rh.updateReleaseTime(&release)

				slice := rh.releases[release.ChannelID]
				if slice == nil {
					slice = make([]releaseData, 0)
				}
				slice = append(slice, release)
				rh.releases[release.ChannelID] = slice
			}

			//Sort our slices now
			for channel := range rh.releases {
				sort.Stable(byReleaseDate(rh.releases[channel]))
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
	}
}

//ScheduledTask empty function to comply with interface reqs
func (rh *ReleaseHandler) ScheduledTask(s *discordgo.Session) {
	currentTime := time.Now()
	cdate := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, time.Local)
	changed := false
	if currentTime.Hour() == 11 && currentTime.Minute() == 0 {
		for channel, channelSlice := range rh.releases {
			tempReleases := channelSlice[:0]
			for _, release := range channelSlice {
				//Does this release have a notifiable release date specified?
				if release.ParsedDate != nil {

					nextWeek := cdate.AddDate(0, 0, 7)
					tomorrow := cdate.AddDate(0, 0, 1)

					if cdate == *release.ParsedDate {
						_, _ = s.ChannelMessageSend(channel, release.Name+" released today!")
					} else {
						//Regardless if we notify, add to the new list
						tempReleases = append(tempReleases, release)

						//Notify if appropriate!
						if nextWeek == *release.ParsedDate {
							_, _ = s.ChannelMessageSend(channel, release.Name+" is releasing next week!")
						} else if tomorrow == *release.ParsedDate {
							_, _ = s.ChannelMessageSend(channel, release.Name+" is releasing tomorrow!")
						}
					}
				} else {
					tempReleases = append(tempReleases, release)
				}
			}

			if len(tempReleases) != len(channelSlice) {
				rh.releases[channel] = tempReleases
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
	releaseSlice := make([]releaseData, 0)

	for _, channelSlice := range rh.releases {
		for _, channelRelease := range channelSlice {
			releaseSlice = append(releaseSlice, channelRelease)
		}
	}
	//..which we then byte-ify and write to disk
	jsonBytes, err := json.Marshal(releaseSlice)
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
		releaseInfo.ChannelID = channelID
		rh.updateReleaseTime(&releaseInfo)

		channelSlice := rh.releases[channelID]
		if channelSlice == nil {
			channelSlice = make([]releaseData, 0)
		}
		channelSlice = append(channelSlice, releaseInfo)
		sort.Stable(byReleaseDate(channelSlice))
		rh.releases[channelID] = channelSlice

		rh.writeData()

		_, _ = s.ChannelMessageSend(channelID, "Added "+releaseInfo.Name+" to releases, releasing "+releaseInfo.ReleaseDate)
	} else {
		_, _ = s.ChannelMessageSend(channelID, "Invalid add syntax")
	}

}

func (rh *ReleaseHandler) list(s *discordgo.Session, channelID string) {
	list := "Here are my currently tracked releases:\n"

	slice := rh.releases[channelID]
	if slice != nil {
		for x, release := range slice {
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
	_, _ = s.ChannelMessageSend(channelID, list)
}

func (rh *ReleaseHandler) edit(s *discordgo.Session, channelID string, data string) {
	match := rh.editMatcher.FindStringSubmatch(data)
	if match != nil {
		index, err := strconv.Atoi(match[1])
		if err == nil {
			newReleaseDate := match[2]
			slice := rh.releases[channelID]
			if slice != nil {
				if len(slice) > index || index < 0 {
					entry := &slice[index]
					entry.ReleaseDate = newReleaseDate
					rh.updateReleaseTime(entry)
					_, _ = s.ChannelMessageSend(channelID, "Successfully updated release date for "+entry.Name)
					sort.Stable(byReleaseDate(slice))
					rh.writeData()
				} else {
					//invalid index provided
					_, _ = s.ChannelMessageSend(channelID, "Invalid ID specified")
				}
			} else {
				//Channel has no releases?
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
			slice := rh.releases[channelID]
			if len(slice) > index || index < 0 {
				rh.releases[channelID] = append(slice[:index], slice[index+1:]...)
				rh.writeData()
				_, _ = s.ChannelMessageSend(channelID, "Successfully removed release")
			} else {
				_, _ = s.ChannelMessageSend(channelID, "Invalid ID specified")
			}
		}
	}
}

func (rh *ReleaseHandler) help(s *discordgo.Session, channelID string) {
	helpMessage := "The following commands are supported by /rw:\n"
	helpMessage += "/rw add <date> <release> - Adds the following release for tracking.\n"
	helpMessage += "\teg: /rw add 10/20/30 Persona 8 Dancing All 'Night\n"
	helpMessage += "/rw list - Lists all currently tracked releases\n"
	helpMessage += "/rw edit <id> <date> - Change the specified release's release date.\n\tID can be obtained from /rw list\n"
	helpMessage += "\teg: /rw edit 12 5/6/20\n"
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
