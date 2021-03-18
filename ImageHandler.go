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

//ImageHandler automatically posts images from specified directories on a schedule
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
	ImageList  []string
}

type channelImageData struct {
	ChannelID string       `json:"channelID"`
	ImageData []*imageData `json:"releaseData"`
}

const iCommand string = "/i"
const imageDataFile = "./imageData.json"

//Init compiles regexp and loads in saved information
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
}

//GetName returns our name
func (ih *ImageHandler) GetName() string {
	return "Image Handler"
}

//HandleMessage echoes the messages seen to stdout
func (ih *ImageHandler) HandleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	submatches := ih.matcher.FindStringSubmatch(m.Content)
	if submatches != nil {
		command := submatches[1]
		switch command {
		case "start":
			ih.start(s, m.ChannelID, submatches[2])
		case "next":
			ih.next(s, m.ChannelID, submatches[2])
		case "help":
			ih.help(s, m.ChannelID)
		default:
			ih.help(s, m.ChannelID)
		}

		//_ = s.ChannelMessageDelete(m.ChannelID, m.ID)
	}
}

//ScheduledTask Handle our scheduled release notifications
func (ih *ImageHandler) ScheduledTask(s *discordgo.Session) {
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
							ih.displayMultiple(s, channelData.ChannelID, imageBlock)
							updatedGlobally = true
							if len(imageBlock.ImageList) <= imageBlock.Current {
								if imageBlock.Repeat {
									imageBlock.Current = 0
								} else {
									keep = false
								}
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

//Help Gets info about this handler
func (ih *ImageHandler) Help() string {
	return "/i : Image Reader - Reads images into chat from disk on a set schedule"
}

func (ih *ImageHandler) help(s *discordgo.Session, channelID string) {
	helpMessage := "The following commands are supported by /i:\n"
	helpMessage += "/i start <dir> <frequency>\n"
	helpMessage += "Starts automatic posting of the images in the specified server dir\n"
	helpMessage += "Frequency is one of: manual|daily|monday|tuesday|wednesday|thursday|friday\n"

	_, _ = s.ChannelMessageSend(channelID, helpMessage)
}

func (ih *ImageHandler) start(s *discordgo.Session, channelID string, command string) {
	submatches := ih.startMatcher.FindStringSubmatch(command)
	if submatches != nil {
		dir := submatches[1]
		schedule := submatches[2]
		hourStr := submatches[3]
		multiplierStr := submatches[4]
		repeatStr := submatches[5]

		//Valid schedule?
		if _, valid := ih.scheduleEnum[schedule]; valid {

			//Parse the parse-needed values
			if hour, err := strconv.Atoi(hourStr); err == nil {
				if multiplier, err := strconv.Atoi(multiplierStr); err == nil {
					if repeat, err := strconv.ParseBool(repeatStr); err == nil {
						if newImageData, err := ih.buildImageData(dir, schedule, hour, multiplier, repeat); err == nil {

							//Create our channel map if needed
							if _, exists := ih.imageMap[channelID]; !exists {
								ih.imageMap[channelID] = &channelImageData{
									ChannelID: channelID,
								}
								ih.imageMap[channelID].ImageData = make([]*imageData, 0)
							}

							//Append our new image data
							ih.imageMap[channelID].ImageData = append(ih.imageMap[channelID].ImageData, newImageData)
							ih.writeData()
						} else {
							fmt.Println("failed to build up image data!")
						}
					}
				}
			}
		} else {
			_, _ = s.ChannelMessageSend(channelID, schedule+" is not a valid schedule")
		}
	} else {
		_, _ = s.ChannelMessageSend(channelID, "Invalid start usage. See help for details")
	}
}

func (ih *ImageHandler) next(s *discordgo.Session, channelID string, command string) {
	submatches := ih.nextMatcher.FindStringSubmatch(command)
	if submatches != nil {
		imageGroupDir := submatches[1]
		if imageGroup, ok := ih.imageMap[channelID]; ok {
			for _, data := range imageGroup.ImageData {
				if data.Dir == imageGroupDir {
					ih.displayMultiple(s, channelID, data)
					if len(data.ImageList) <= data.Current {
						if data.Repeat {
							data.Current = 0
						} else {
							_, _ = s.ChannelMessageSend(channelID, "Done! Completed all images for image block: "+data.Dir)
						}
					}
					ih.writeData()
					return
				}
			}
			_, _ = s.ChannelMessageSend(channelID, "Specified image group does not exist!")
		} else {
			_, _ = s.ChannelMessageSend(channelID, "No image groups on this channel!")
		}
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

	data := imageData{}
	data.Dir = dir
	data.ImageList = files
	data.Current = 0
	data.Schedule = ih.scheduleEnum[schedule]
	data.Repeat = repeat
	data.Multiplier = multiplier
	data.Hour = hour

	return &data, nil
}

func (ih *ImageHandler) displayMultiple(s *discordgo.Session, channelID string, data *imageData) {
	//Better than converting to float64's imo
	//but there must be a better way of getting the minimum, right?
	imageSlice := (data.ImageList)
	showCount := len(imageSlice) - (data.Current)
	if showCount > (data.Multiplier) {
		showCount = (data.Multiplier)

		for i := 0; i < showCount; i++ {
			ih.displayImage(s, channelID, data.ImageList[data.Current])
			data.Current++
		}
	}

	//Cleanup is handled by the respective callers, as they need to handle clean-up differently
}

func (ih *ImageHandler) displayImage(s *discordgo.Session, channelID string, path string) error {
	if img, err := os.Open(path); err == nil {
		dgoFiles := make([]*discordgo.File, 0)
		dgoFiles = append(dgoFiles, &discordgo.File{
			Name:   path,
			Reader: img,
		})
		_, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
			Files: dgoFiles,
		})

		if err != nil {
			fmt.Println("Error sending image: ", err)
			return err
		}
	} else {
		return err
	}

	return nil
}
