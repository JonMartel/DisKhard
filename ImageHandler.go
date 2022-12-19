package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
)

// ImageHandler automatically posts images from specified directories on a schedule
type ImageHandler struct {
	matcher      regexp.Regexp
	startMatcher regexp.Regexp
	nextMatcher  regexp.Regexp
	imageMap     map[string]*channelImageData
	scheduleEnum map[string]time.Weekday
}

type imageData struct {
	Dir        string `json:"dir"`
	Current    int    `json:"current"`
	Schedule   time.Weekday
	Repeat     bool
	Hour       int
	Multiplier int
}

type channelImageData struct {
	ChannelID string       `json:"channelID"`
	ImageData []*imageData `json:"imageData"`
}

const iCommand string = "/i"
const imageDataFile = "./imageData.json"

func (ih *ImageHandler) GetApplicationCommand() *discordgo.ApplicationCommand {
	zero := (float64)(0)
	one := (float64)(1)
	command := discordgo.ApplicationCommand{
		Name:        "image",
		Description: "Handles publishing images from a directory on a schedule",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "list",
				Description: "Displays image schedule",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
			},
			{
				Name:        "schedule",
				Description: "Schedule automatic posting of images",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "directory",
						Description: "Directory containing images to schedule",
						Required:    true,
					},
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "frequency",
						Description: "Frequency of posting",
						Required:    true,
						Choices: []*discordgo.ApplicationCommandOptionChoice{
							{
								Name:  "manual",
								Value: "manual",
							},
							{
								Name:  "daily",
								Value: "daily",
							},
							{
								Name:  "monday",
								Value: "monday",
							},
							{
								Name:  "tuesday",
								Value: "tuesday",
							},
							{
								Name:  "wednesday",
								Value: "wednesday",
							},
							{
								Name:  "thursday",
								Value: "thursday",
							},
							{
								Name:  "friday",
								Value: "friday",
							},
							{
								Name:  "saturday",
								Value: "saturday",
							},
							{
								Name:  "sunday",
								Value: "sunday",
							},
						},
					},
					{
						Type:        discordgo.ApplicationCommandOptionInteger,
						Name:        "hour",
						Description: "Hour to post images at",
						MinValue:    &zero,
						Required:    true,
					},
					{
						Type:        discordgo.ApplicationCommandOptionInteger,
						Name:        "count",
						Description: "How many images to display each time",
						MinValue:    &one,
						Required:    true,
					},
					{
						Type:        discordgo.ApplicationCommandOptionBoolean,
						Name:        "repeat",
						Description: "Enable to loop displaying images",
						Required:    true,
					},
				},
			},
			{
				Name:        "next",
				Description: "Manually display next set of images",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "name",
						Description: "Image Block name (directory)",
						Required:    true,
					},
				},
			},
		},
	}

	return &command
}

// Init compiles regexp and loads in saved information
func (ih *ImageHandler) Init() {
	ih.matcher = *regexp.MustCompile(`^\` + iCommand + `\s+(\w+)\s*(.*)`)
	//dir, schedule, hour, repeat
	ih.startMatcher = *regexp.MustCompile(`^(\w+)\s(\w+)\s(\d+)\s(\d+)\s(\w+)$`)
	ih.nextMatcher = *regexp.MustCompile(`^(\w+)$`)

	ih.scheduleEnum = make(map[string]time.Weekday)
	ih.scheduleEnum["sunday"] = time.Sunday
	ih.scheduleEnum["monday"] = time.Monday
	ih.scheduleEnum["tuesday"] = time.Tuesday
	ih.scheduleEnum["wednesday"] = time.Wednesday
	ih.scheduleEnum["thursday"] = time.Thursday
	ih.scheduleEnum["friday"] = time.Friday
	ih.scheduleEnum["saturday"] = time.Saturday

	//not real days of the week, but used for validating input
	ih.scheduleEnum["daily"] = 7
	ih.scheduleEnum["manual"] = -1

	ih.imageMap = make(map[string]*channelImageData)

	//Need to read in stored json info as well!
	var data []*channelImageData

	// Get configuration
	fileData, err := ioutil.ReadFile(imageDataFile)

	//Load up in-memory cache of this info
	if err == nil {
		fmt.Println("Reading saved image data")
		err = json.Unmarshal(fileData, &data)
		if err == nil {
			for _, channelData := range data {
				ih.imageMap[channelData.ChannelID] = channelData
			}
		}
	}

	go func() {
		//schedule to check for when we need to do stuff!
		minuteSchedule := time.NewTicker(time.Minute)
		defer minuteSchedule.Stop()
		for {
			<-minuteSchedule.C
			ih.scheduledTask()
		}
	}()
}

func (ih *ImageHandler) Handler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	command := i.ApplicationCommandData().Options[0]
	options := command.Options
	content := ""

	switch command.Name {
	case "schedule":
		content = ih.start(i, options)
	case "next":
		content = ih.next(i, options)
	case "list":
		content = ih.list(i)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	})
}

// ScheduledTask Handle our scheduled release notifications
func (ih *ImageHandler) scheduledTask() {
	currentTime := time.Now()
	updatedGlobally := false

	//We only ever kick off when the time value is on the hour
	if currentTime.Minute() == 0 {
		//For each channel...
		for _, channelData := range ih.imageMap {
			//For each group of images...
			afterIterationSlice := make([]*imageData, 0)
			for _, imageBlock := range channelData.ImageData {
				keep := true
				//Does this image block have a schedule specified?
				//-1 == manual, aka no schedule
				if imageBlock.Schedule > 0 {
					//7 == daily, so we do it always.
					if imageBlock.Schedule == 7 || (imageBlock.Schedule) == currentTime.Weekday() {
						if imageBlock.Hour == currentTime.Hour() {
							if imageList, err := ih.listFiles(imageBlock.Dir); err == nil {
								ih.displayMultiple(channelData.ChannelID, imageBlock, imageList)
								updatedGlobally = true
								if len(imageList) <= imageBlock.Current {
									if imageBlock.Repeat {
										imageBlock.Current = 0
									} else {
										keep = false
									}
								}
							} else {
								MessageSender.SendMessage(channelData.ChannelID, "Could not list out files for image block")
							}
						}
					}
				}

				if keep {
					afterIterationSlice = append(afterIterationSlice, imageBlock)
				}
			}

			channelData.ImageData = afterIterationSlice
		}
	}

	if updatedGlobally {
		ih.writeData()
	}
}

func (ih *ImageHandler) list(i *discordgo.InteractionCreate) string {
	if channelData, exists := ih.imageMap[i.ChannelID]; exists {
		if len(channelData.ImageData) > 0 {
			message := "Image Block Data\n"
			for _, imageBlock := range channelData.ImageData {
				if fileList, err := ih.listFiles(imageBlock.Dir); err == nil {
					total := strconv.Itoa(len(fileList))
					page := strconv.Itoa(imageBlock.Current + 1) //Switch from 0-index to 1-index
					message += imageBlock.Dir + " Page: " + page + " / " + total + "\n"
				}
			}

			return message
		}
	}

	//If we got here, no channel data exists!
	return "No image block data exists for this channel!"
}

func (ih *ImageHandler) start(i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) string {
	message := ""
	dir := options[0].StringValue()
	schedule := options[1].StringValue()
	hour := (int)(options[2].IntValue())
	multiplier := (int)(options[3].IntValue())
	repeat := options[4].BoolValue()

	//Valid schedule?
	if _, valid := ih.scheduleEnum[schedule]; valid {

		//Does the directory exist and have contents?
		if _, err := ih.listFiles(dir); err == nil {

			if newImageData, err := ih.buildImageData(dir, schedule, hour, multiplier, repeat); err == nil {

				//Create our channel map if needed
				if _, exists := ih.imageMap[i.ChannelID]; !exists {
					ih.imageMap[i.ChannelID] = &channelImageData{
						ChannelID: i.ChannelID,
					}
					ih.imageMap[i.ChannelID].ImageData = make([]*imageData, 0)
				}

				//Append our new image data
				ih.imageMap[i.ChannelID].ImageData = append(ih.imageMap[i.ChannelID].ImageData, newImageData)
				ih.writeData()

				message = "Scheduled image rotation"
			} else {
				fmt.Println("Failed to build up image data!")
			}
		} else {
			message = dir + " is not a valid image directory"
		}

	} else {
		message = schedule + " is not a valid schedule"
	}

	return message
}

func (ih *ImageHandler) next(i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) string {
	message := ""
	imageGroupDir := options[0].StringValue()

	if imageGroup, ok := ih.imageMap[i.ChannelID]; ok {
		for _, data := range imageGroup.ImageData {
			if data.Dir == imageGroupDir {
				go ih.display(i.ChannelID, data)
				return "Displaying next image(s) for " + imageGroupDir
			}
		}
		message = "Specified image group does not exist!"
	} else {
		message = "No image groups on this channel!"
	}

	return message
}

func (ih *ImageHandler) display(channelID string, data *imageData) {

	if imageList, err := ih.listFiles(data.Dir); err == nil {
		ih.displayMultiple(channelID, data, imageList)
		if len(imageList) <= data.Current {
			if data.Repeat {
				data.Current = 0
			} else {
				//"Done! Completed all images for image block: " + data.Dir
			}
		}
		ih.writeData()
	}
}

func (ih *ImageHandler) writeData() {
	//join all the releases into a single slice...

	channelDataSlice := make([]channelImageData, 0)

	for _, channelData := range ih.imageMap {
		channelDataSlice = append(channelDataSlice, *channelData)
	}

	//..which we then byte-ify and write to disk
	jsonBytes, err := json.Marshal(channelDataSlice)
	if err == nil {
		ioutil.WriteFile(imageDataFile, jsonBytes, 0644)
	}
}

func (ih *ImageHandler) buildImageData(dir string, schedule string, hour int, multiplier int, repeat bool) (*imageData, error) {

	data := imageData{}
	data.Dir = dir
	data.Current = 0
	data.Schedule = ih.scheduleEnum[schedule]
	data.Repeat = repeat
	data.Multiplier = multiplier
	data.Hour = hour

	return &data, nil
}

func (ih *ImageHandler) listFiles(dir string) ([]string, error) {
	path, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	path = filepath.FromSlash(path + "/reader/" + dir)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, err
	}

	files := make([]string, 0)

	err = filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

func (ih *ImageHandler) displayMultiple(channelID string, data *imageData, imageList []string) {
	//Better than converting to float64's imo
	//but there must be a better way of getting the minimum, right?
	showCount := len(imageList) - (data.Current)
	if showCount > (data.Multiplier) {
		showCount = (data.Multiplier)
	}

	for i := 0; i < showCount; i++ {
		MessageSender.SendFile(channelID, imageList[data.Current])
		data.Current++
	}

	//Cleanup is handled by the respective callers, as they need to handle clean-up differently
}
