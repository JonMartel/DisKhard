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
}

type channelImageData struct {
	ChannelID string       `json:"channelID"`
	ImageData []*imageData `json:"imageData"`
}

const iCommand string = "/i"
const imageDataFile = "./imageData.json"

//Init compiles regexp and loads in saved information
func (ih *ImageHandler) Init(m chan *discordgo.MessageCreate) {
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

	//schedule to check for when we need to do stuff!
	minuteSchedule := time.NewTicker(time.Minute)
	defer minuteSchedule.Stop()

	go func() {
		for {
			select {
			case message := <-m:
				if message != nil {
					ih.handleMessage(message)
				} else {
					return
				}
			case <-minuteSchedule.C:
				ih.scheduledTask()
			}
		}
	}()
}

//GetName returns our name
func (ih *ImageHandler) GetName() string {
	return "Image Handler"
}

//HandleMessage echoes the messages seen to stdout
func (ih *ImageHandler) handleMessage(m *discordgo.MessageCreate) {
	submatches := ih.matcher.FindStringSubmatch(m.Content)
	if submatches != nil {
		command := submatches[1]
		switch command {
		case "start":
			ih.start(m.ChannelID, submatches[2])
		case "next":
			ih.next(m.ChannelID, submatches[2])
		case "list":
			ih.list(m.ChannelID)
		case "help":
			ih.help(m.ChannelID)
		default:
			ih.help(m.ChannelID)
		}

		//DeleteMessage(m.ChannelID, m.ID)
	}
}

//ScheduledTask Handle our scheduled release notifications
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

//Help Gets info about this handler
func (ih *ImageHandler) Help() string {
	return "/i : Image Reader - Reads images into chat from disk on a set schedule"
}

func (ih *ImageHandler) help(channelID string) {
	helpMessage := "The following commands are supported by /i:\n"
	helpMessage += "/i start <dir> <frequency> <hour> <pages-per-post> <repeat>\n"
	helpMessage += "  Starts automatic posting of the images in the specified server dir\n"
	helpMessage += "  Frequency is when posts are automatically made (manual|daily|monday|tuesday|wednesday|thursday|friday|saturday|sunday)\n"
	helpMessage += "  Hour is hour of the day to post at (0-23)\n"
	helpMessage += "  Pages per post is how many pages to display at once\n"
	helpMessage += "  Repeat allows the image block to repeat once it has finished (true|false)\n"
	helpMessage += "/i list - lists out all currently configured image blocks and their progress\n"

	MessageSender.SendMessage(channelID, helpMessage)
}

func (ih *ImageHandler) list(channelID string) {
	if channelData, exists := ih.imageMap[channelID]; exists {
		if len(channelData.ImageData) > 0 {
			message := "Image Block Data\n"
			for _, imageBlock := range channelData.ImageData {
				if fileList, err := ih.listFiles(imageBlock.Dir); err == nil {
					total := strconv.Itoa(len(fileList))
					page := strconv.Itoa(imageBlock.Current + 1) //Switch from 0-index to 1-index
					message += imageBlock.Dir + " Page: " + page + " / " + total + "\n"
				}
			}
			MessageSender.SendMessage(channelID, message)
			return
		}
	}

	//If we got here, no channel data exists!
	MessageSender.SendMessage(channelID, "No image block data exists for this channel!")
}

func (ih *ImageHandler) start(channelID string, command string) {
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
			MessageSender.SendMessage(channelID, schedule+" is not a valid schedule")
		}
	} else {
		MessageSender.SendMessage(channelID, "Invalid start usage. See help for details")
	}
}

func (ih *ImageHandler) next(channelID string, command string) {
	submatches := ih.nextMatcher.FindStringSubmatch(command)
	if submatches != nil {
		imageGroupDir := submatches[1]
		if imageGroup, ok := ih.imageMap[channelID]; ok {
			for _, data := range imageGroup.ImageData {
				if data.Dir == imageGroupDir {
					if imageList, err := ih.listFiles(data.Dir); err == nil {
						ih.displayMultiple(channelID, data, imageList)
						if len(imageList) <= data.Current {
							if data.Repeat {
								data.Current = 0
							} else {
								MessageSender.SendMessage(channelID, "Done! Completed all images for image block: "+data.Dir)
							}
						}
						ih.writeData()
					}
					return
				}
			}
			MessageSender.SendMessage(channelID, "Specified image group does not exist!")
		} else {
			MessageSender.SendMessage(channelID, "No image groups on this channel!")
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
	imageSlice := (imageList)
	showCount := len(imageSlice) - (data.Current)
	if showCount > (data.Multiplier) {
		showCount = (data.Multiplier)

		for i := 0; i < showCount; i++ {
			MessageSender.SendFile(channelID, imageList[data.Current])
			data.Current++
		}
	}

	//Cleanup is handled by the respective callers, as they need to handle clean-up differently
}
