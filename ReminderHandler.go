package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
)

// ReminderHandler Echoes messages to stdout
type ReminderHandler struct {
	matcher              regexp.Regexp
	addMatcher           regexp.Regexp
	addRemoveUserMatcher regexp.Regexp
	editMatcher          regexp.Regexp
	deleteMatcher        regexp.Regexp

	channelReminders map[string]*channelReminderData
	dayMap           map[rune]time.Weekday
}

type Reminder struct {
	Name      string   `json:"n"`
	Hour      int      `json:"h"`
	Minute    int      `json:"m"`
	Days      []int    `json:"d"`
	Notifyees []string `json:"notifyees"`
}

type channelReminderData struct {
	ChannelID string
	Reminders []*Reminder
}

const remindCommand string = "/remind"
const reminderDataFile = "./ReminderData.json"

// Init compiles regexp and loads in saved information
func (rh *ReminderHandler) Init() {
	rh.matcher = *regexp.MustCompile(`^\` + remindCommand + `\s+(\w+)\s*(.*)$`)
	rh.addMatcher = *regexp.MustCompile(`^(\d{1,2}):(\d\d) ([U日]?[M月]?[T火]?[W水]?[R木]?[F金]?[S土]?) (.*)$`)
	rh.addRemoveUserMatcher = *regexp.MustCompile(`^(\d+)$`)
	rh.editMatcher = *regexp.MustCompile(`^(\d+) ([\w-/]+)`)
	rh.deleteMatcher = *regexp.MustCompile(`^(\d+)`)

	rh.channelReminders = make(map[string]*channelReminderData)
	rh.dayMap = make(map[rune]time.Weekday)

	//populate our daymap
	rh.dayMap['U'] = time.Sunday
	rh.dayMap['日'] = time.Sunday
	rh.dayMap['M'] = time.Monday
	rh.dayMap['月'] = time.Monday
	rh.dayMap['T'] = time.Tuesday
	rh.dayMap['火'] = time.Tuesday
	rh.dayMap['W'] = time.Wednesday
	rh.dayMap['水'] = time.Wednesday
	rh.dayMap['R'] = time.Thursday
	rh.dayMap['木'] = time.Thursday
	rh.dayMap['F'] = time.Friday
	rh.dayMap['金'] = time.Friday
	rh.dayMap['S'] = time.Saturday
	rh.dayMap['土'] = time.Saturday

	//Need to read in stored json info as well!
	var data []channelReminderData

	// Get configuration
	fileData, err := ioutil.ReadFile(reminderDataFile)

	//Load up in-memory cache of this info
	if err == nil {
		fmt.Println("Reading saved Reminder data")
		err = json.Unmarshal(fileData, &data)
		if err == nil {
			for _, channelData := range data {
				channelCopy := channelData
				rh.channelReminders[channelData.ChannelID] = &channelCopy
			}
		}
	}

	go func() {
		minuteSchedule := time.NewTicker(time.Minute)
		defer minuteSchedule.Stop()
		for {
			<-minuteSchedule.C
			rh.scheduledTask()
		}
	}()
}

func (rh *ReminderHandler) GetApplicationCommand() *discordgo.ApplicationCommand {
	zero := (float64)(0)
	command := discordgo.ApplicationCommand{
		Name:        "reminder",
		Description: "Create and register for reminders",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "list",
				Description: "Display all current reminders",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
			},
			{
				Name:        "create",
				Description: "Create a new reminder",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "name",
						Description: "Name of the reminder",
						Required:    true,
					},
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "days",
						Description: "Days to send reminder (Any combination of UMTWRFS日月火水木金土)",
						Required:    true,
					},
					{
						Type:        discordgo.ApplicationCommandOptionInteger,
						Name:        "hour",
						Description: "Hour (0-23)",
						Required:    true,
						MinValue:    &zero,
						MaxValue:    23,
					},
					{
						Type:        discordgo.ApplicationCommandOptionInteger,
						Name:        "minute",
						Description: "Minute (0-59)",
						Required:    true,
						MinValue:    &zero,
						MaxValue:    59,
					},
				},
			},
			{
				Name:        "subscribe",
				Description: "Subscribe to a reminder",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionInteger,
						Name:        "index",
						Description: "Reminder index",
						Required:    true,
						MinValue:    &zero,
					},
				},
			},
			{
				Name:        "unsubscribe",
				Description: "Unsubscribe from a reminder",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionInteger,
						Name:        "index",
						Description: "Reminder index",
						Required:    true,
						MinValue:    &zero,
					},
				},
			},
		},
	}

	return &command
}

func (rh *ReminderHandler) Handler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	command := i.ApplicationCommandData().Options[0]
	options := command.Options
	content := ""

	switch command.Name {
	case "create":
		content = rh.add(i, options)
	case "subscribe":
		content = rh.addUser(i, options)
	case "unsubscribe":
		content = rh.removeUser(i, options)
	case "list":
		content = rh.list(i)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	})
}

/*
// HandleMessage Adds/Edits/Removes reminders based on message input
func (rh *ReminderHandler) handleMessage(m *discordgo.MessageCreate) {
	submatches := rh.matcher.FindStringSubmatch(m.Content)
	if submatches != nil {
		command := submatches[1]
		switch command {
		case "add":
			rh.add(m.ChannelID, m.Author.ID, submatches[2])
		case "addme":
			rh.addUser(m.ChannelID, m.Author.ID, submatches[2])
		case "removeme":
			rh.removeUser(m.ChannelID, m.Author.ID, submatches[2])
		case "list":
			rh.list(m.ChannelID)
	}
}
*/

func (rh *ReminderHandler) scheduledTask() {
	currentTime := time.Now()

	for _, channelData := range rh.channelReminders {
		for _, rem := range channelData.Reminders {

			//Are we at the allotted time?
			if currentTime.Hour() == rem.Hour && currentTime.Minute() == rem.Minute {
				//Correct day?
				for _, weekday := range rem.Days {
					if weekday == (int)(currentTime.Weekday()) {
						//Send it out!
						message := rem.Name
						for _, user := range rem.Notifyees {
							message += " " + rh.userPingString(user)
						}
						MessageSender.SendMessage(channelData.ChannelID, message)
					}
				}
			}
		}
	}
}

func (rh *ReminderHandler) writeData() {
	//join all the Reminders into a single slice...
	channelDataSlice := make([]*channelReminderData, 0)

	for _, channelData := range rh.channelReminders {
		channelDataSlice = append(channelDataSlice, channelData)
	}
	//..which we then byte-ify and write to disk
	jsonBytes, err := json.Marshal(channelDataSlice)
	if err == nil {
		ioutil.WriteFile(reminderDataFile, jsonBytes, 0644)
	}
}

func (rh *ReminderHandler) add(i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) string {
	name := options[0].StringValue()
	days := options[1].StringValue()
	hour := options[2].IntValue()
	minute := options[3].IntValue()
	user := i.Member.User.ID

	message := ""

	//Let's validate these hour/minute values
	if hour < 0 || hour > 23 {
		return "Hour must be between 0 and 23"
	}

	if minute < 0 || minute > 59 {
		return "Minutes must be between 0 and 59"
	}

	channel, ok := rh.channelReminders[i.ChannelID]
	if !ok {
		channel = rh.initChannel(i.ChannelID)
	}

	reminder := Reminder{}
	reminder.Hour = (int)(hour)
	reminder.Minute = (int)(minute)
	reminder.Name = name
	reminder.Days = make([]int, 0)

	for _, letter := range days {
		if day, ok := rh.dayMap[letter]; ok {
			reminder.Days = append(reminder.Days, (int)(day))
		}
	}

	reminder.Notifyees = make([]string, 0)
	reminder.Notifyees = append(reminder.Notifyees, user)

	channel.Reminders = append(channel.Reminders, &reminder)

	rh.writeData()

	message = rh.userPingString(user) + " added " + reminder.Name + " reminder"

	return message
}

func (rh *ReminderHandler) list(i *discordgo.InteractionCreate) string {
	formattedChannelReminder := rh.formatChannelReminders(i.ChannelID)
	return formattedChannelReminder
}

func (rh *ReminderHandler) addUser(i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) string {
	user := i.Member.User.ID
	index := (int)(options[0].IntValue())

	if channelData, ok := rh.channelReminders[i.ChannelID]; ok {
		if len(channelData.Reminders) > index && index >= 0 {
			reminder := channelData.Reminders[index]
			for _, notifyee := range reminder.Notifyees {
				if user == notifyee {
					//Hey, you're already here!
					return "You're already subscribed to this reminder!"
				}
			}

			//Not here already, lets add you!
			reminder.Notifyees = append(reminder.Notifyees, user)
			rh.writeData()
			return "Subscribed user " + rh.userPingString(user) + " to reminder"
		} else {
			return "That's not a valid reminder!"
		}
	} else {
		return "No reminders for this channel!"
	}
}

func (rh *ReminderHandler) removeUser(i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) string {
	user := i.Member.User.ID
	index := (int)(options[0].IntValue())

	if channelData, ok := rh.channelReminders[i.ChannelID]; ok {

		if len(channelData.Reminders) > index && index >= 0 {
			reminder := channelData.Reminders[index]
			var removeIndex int = -1
			for x, notifyee := range reminder.Notifyees {
				if notifyee == user {
					removeIndex = x
					break
				}
			}

			if removeIndex == -1 {
				//You're not in this notification list!
				return "You're not registered as a subscriber of this reminder!"
			} else {
				currentLength := len(reminder.Notifyees)
				//Swap the last element to this element's position (may be the same element)
				//and then set our array to everything but that last element
				reminder.Notifyees[removeIndex] = reminder.Notifyees[currentLength-1]
				reminder.Notifyees = reminder.Notifyees[:currentLength-1]

				rh.writeData()
				return "Unsubscribed " + rh.userPingString(user) + " from notification list"
			}
		} else {
			return "Invalid reminder specified!"
		}

	} else {
		return "This channel does not have reminders!"
	}
}

func (rh *ReminderHandler) formatChannelReminders(channelID string) string {
	message := "```"
	columns := [10]string{"ID", "Name", "Time", "U", "M", "T", "W", "R", "F", "S"}
	columnRequiredSize := [10]int{2, 4, 5, 1, 1, 1, 1, 1, 1, 1}
	if channelData, ok := rh.channelReminders[channelID]; ok {
		if channelData.Reminders != nil && len(channelData.Reminders) > 0 {
			//First, determine the maximum size name
			for _, reminder := range channelData.Reminders {
				nameLength := len(reminder.Name)
				if columnRequiredSize[1] < nameLength {
					columnRequiredSize[1] = nameLength
				}
			}

			//ID is minimum 2 chars, but potentially more
			totalEntries := len(channelData.Reminders)
			columnRequiredSize[0] = len(strconv.FormatInt(int64(totalEntries), 10))
			if columnRequiredSize[0] < 2 {
				columnRequiredSize[0] = 2
			}

			//Build up our column headers
			//[ ID ][ Name ][ Time  ][ U ][ M ][ T ][ W ][ R ][ F ][ S ]
			//[ 1  ][ Hmm  ][ 10:30 ][   ][   ][ X ][   ][   ][   ][   ]
			for x := 0; x < len(columns); x++ {
				message += rh.formatName(columns[x], columnRequiredSize[x])
			}
			message += "\n"

			//Calculations out of the way, let's format this sucker
			for x, reminder := range channelData.Reminders {
				message += rh.formatName(strconv.FormatInt(int64(x), 10), columnRequiredSize[0])
				message += rh.formatName(reminder.Name, columnRequiredSize[1])
				timeString := strconv.FormatInt(int64(reminder.Hour), 10) + ":" + strconv.FormatInt(int64(reminder.Minute), 10)
				message += rh.formatName(timeString, columnRequiredSize[2])

				daySlice := make([]bool, 7)
				for _, weekday := range reminder.Days {
					daySlice[weekday] = true
				}
				for x := 0; x < len(daySlice); x++ {
					message += rh.formatDayActive(daySlice[x])
				}
				message += "\n"
			}
			message += "```"
		} else {
			return "```<No reminders>```"
		}
	} else {
		return "```<No reminders>```"
	}

	return message
}

func (rh *ReminderHandler) formatName(value string, requiredLength int) string {
	message := "[ " + value

	extraWhitespace := requiredLength - len(value)
	for x := 0; x < extraWhitespace; x++ {
		message += " "
	}

	message += " ]"

	return message
}

func (rh *ReminderHandler) formatDayActive(day bool) string {
	if day {
		return "[ X ]"
	}

	return "[   ]"
}

func (rh *ReminderHandler) help(channelID string) {
	helpMessage := "The following commands are supported by /rw:\n"
	helpMessage += remindCommand + " add <time> <days> <Reminder> - Adds the following Reminder for tracking.\n"
	helpMessage += "\t<time> is in HH:MM format using 24-hour time\n"
	helpMessage += "\t<days> is a string with any of MTWRF\n"
	helpMessage += "\teg: " + remindCommand + " add 20:45 TWRF Anime Time\n"
	helpMessage += remindCommand + " list - Lists all channel reminders\n"
	helpMessage += remindCommand + " addme <id> - Add yourself as a notifyee of the specified reminder\n"
	helpMessage += "\t<id> can be obtained from /rw list\n"
	helpMessage += "\teg: " + remindCommand + " addme 12\n"
	helpMessage += remindCommand + " removeme <id> - Remove yourself as a notifyee of the specified reminder\n"
	helpMessage += remindCommand + " help - This output here!"

	MessageSender.SendMessage(channelID, helpMessage)
}

func (rh *ReminderHandler) initChannel(channelID string) *channelReminderData {
	//Spin up our channel and return it
	channel := &channelReminderData{}
	channel.ChannelID = channelID
	channel.Reminders = make([]*Reminder, 0)

	rh.channelReminders[channelID] = channel

	return channel
}

func (rh *ReminderHandler) userPingString(user string) string {
	return "<@!" + user + ">"
}
