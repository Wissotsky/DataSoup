package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf16"

	"golang.org/x/text/encoding/charmap"

	"github.com/saintfish/chardet"
)

type File struct {
	Success bool       `json:"success"`
	Result  FileResult `json:"result"`
}

type FileResult struct {
	Count   int              `json:"count"`
	Results []FileResultItem `json:"results"`
}

type FileResultItem struct {
	Frequency              string       `json:"frequency"`
	Update                 string       `json:"update"`
	Author                 string       `json:"author"`
	AuthorEmail            string       `json:"author_email"`
	CreatorUserId          string       `json:"creator_user_id"`
	Id                     string       `json:"id"`
	IsOpen                 bool         `json:"is_open"`
	LicenseId              string       `json:"license_id"`
	LicenseTitle           string       `json:"license_title"`
	MailBox                string       `json:"mail_box"`
	Maintainer             string       `json:"maintainer"`
	MaintainerEmail        string       `json:"maintainer_email"`
	MetadataCreated        string       `json:"metadata_created"`
	MetadataModified       string       `json:"metadata_modified"`
	Name                   string       `json:"name"`
	Notes                  string       `json:"notes"`
	NumResources           int          `json:"num_resources"`
	NumTags                int          `json:"num_tags"`
	Organization           Organization `json:"organization"`
	OwnerOrg               string       `json:"owner_org"`
	Private                bool         `json:"private"`
	Remark                 string       `json:"remark"`
	State                  string       `json:"state"`
	Title                  string       `json:"title"`
	TitleLink              string       `json:"title_link"`
	Type                   string       `json:"type"`
	Url                    string       `json:"url"`
	UrlLink                string       `json:"url_link"`
	Version                string       `json:"version"`
	Extras                 string       `json:"extras"`
	Resources              []Resource   `json:"resources"`
	Tags                   []Tag        `json:"tags"`
	Groups                 string       `json:"groups"`
	RelationshipsAsObject  string       `json:"relationships_as_object"`
	RelationshipsAsSubject string       `json:"relationships_as_subject"`
}

type Organization struct {
	Id             string `json:"id"`
	Name           string `json:"name"`
	Title          string `json:"title"`
	Type           string `json:"type"`
	Description    string `json:"description"`
	ImageUrl       string `json:"image_url"`
	Created        string `json:"created"`
	IsOrganization bool   `json:"is_organization"`
	ApprovalStatus string `json:"approval_status"`
	State          string `json:"state"`
}

type Resource struct {
	Frequency                               string `json:"frequency"`
	Category                                string `json:"category"`
	CkanUrl                                 string `json:"ckan_url"`
	Created                                 string `json:"created"`
	DatastoreActive                         bool   `json:"datastore_active"`
	DatastoreContainsAllRecordsOfSourceFile bool   `json:"datastore_contains_all_records_of_source_file"`
	Format                                  string `json:"format"`
	HowUpdate                               string `json:"how_update"`
	Id                                      string `json:"id"`
	LastModified                            string `json:"last_modified"`
	MetadataModified                        string `json:"metadata_modified"`
	Mimetype                                string `json:"mimetype"`
	Name                                    string `json:"name"`
	Notes                                   string `json:"notes"`
	OriginalUrl                             string `json:"original_url"`
	PackageId                               string `json:"package_id"`
	Position                                int    `json:"position"`
	ResourceId                              string `json:"resource_id"`
	Size                                    int    `json:"size"`
	State                                   string `json:"state"`
	TaskCreated                             string `json:"task_created"`
	Url                                     string `json:"url"`
	UrlType                                 string `json:"url_type"`
}

type Tag struct {
	DisplayName string `json:"display_name"`
	Id          string `json:"id"`
	Name        string `json:"name"`
	State       string `json:"state"`
}

type MessageEntity struct {
	Type   string `json:"type"`
	Offset int    `json:"offset"`
	Length int    `json:"length"`
	Url    string `json:"url"`
}

type SendMessagePayload struct {
	ChatId   string          `json:"chat_id"`
	Text     string          `json:"text"`
	Entities []MessageEntity `json:"entities"`
}

func fetchResource(resource Resource, datapackage FileResultItem, waitGroup *sync.WaitGroup, client *http.Client, backoff int) {
	defer waitGroup.Done()
	dirpath := filepath.Join("data", datapackage.Organization.Name, datapackage.Id)
	// create all directories
	err := os.MkdirAll(dirpath, 0666)
	if err != nil {
		log.Fatalln(err)
	}
	// create request with custom UA datagov-external-client
	req, err := http.NewRequest("GET", resource.Url, nil)
	if err != nil {
		log.Fatalln(err)
	}
	req.Header.Set("User-Agent", "github.com/wissotsky#datagov-external-client")
	// send request
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return
	}
	// create file
	file, err := os.Create(filepath.Join(dirpath, resource.Id+".csv"))
	if err != nil {
		log.Fatalln(err)
	}
	// copy from request and close
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		// Panics here if the server closes the socket before we finish reading
		log.Println(resource.Url, resource.Name, err)
		// Backoff and retry
		file.Close()
		resp.Body.Close()

		// sleep for backoff time + random jitter of half backoff time to prevent crowding
		chosenBackoff := backoff + rand.IntN(backoff/2)
		time.Sleep(time.Duration(chosenBackoff) * time.Second)
		log.Println("Retrying", resource.Name, "after backoff", chosenBackoff)
		waitGroup.Add(1)
		go fetchResource(resource, datapackage, waitGroup, client, (backoff * 2))
		return
	}
	file.Close()
	resp.Body.Close()
	// print success
	fmt.Println("Downloaded", resource.Name)
}

// find the maximum amount of rows we can fit in from the diff to be under telegram's character limits
func findSubSliceOfMaxLen(slice []string, maxlen int, prefixLen int) ([]string, int) {
	var subSlice []string
	curLen := prefixLen
	for _, s := range slice {
		if curLen+len(s) < maxlen {
			subSlice = append(subSlice, s)
			curLen += len(s)
		} else {
			break
		}
	}
	remainingCount := len(slice) - len(subSlice)
	return subSlice, remainingCount
}

func isResourceExempt(resourceId string) bool {
	switch resourceId {
	case
		"053cea08-09bc-40ec-8f7a-156f0677aff3",
		"aba233c2-6a5a-487d-b0a8-9413ef849f15":
		return true
	}
	return false
}

func processDiffToPayload(isNewResource bool, diff []string, datapackage FileResultItem, resource Resource) SendMessagePayload {
	// TODO: if diff length is equal to zero it means there is no difference between the files, therefore dont send a message
	var prefix string
	if isNewResource {
		fmt.Println("TODO Notification: Added New Resource ", resource.Name)
		prefix = "📗 New Resource: "
	} else {
		fmt.Println("TODO Notification: Updated Resource ", resource.Name)
		prefix = "📘 Update: "
	}

	// if its the flight data we will process it differently //TODO: make a more robust system for processing exceptional data
	isFlightData := false
	isRegularDiff := true
	if resource.Id == "e83f763b-b7d7-479e-b172-ae981ddc6de5" {
		isRegularDiff = false
		isFlightData = true
		prefix = "✈ Flights Update: "
	}

	datasetName := resource.Name
	var datasetDiff string

	if isRegularDiff {
		diffSlice, remainingCount := findSubSliceOfMaxLen(diff, 3800, len(utf16.Encode([]rune(datasetName))))
		datasetDiffJoined := strings.Join(diffSlice, "\n")

		if remainingCount == 0 {
			datasetDiff = datasetDiffJoined
		} else {
			datasetDiff = fmt.Sprintf("%s\n... and %d more", datasetDiffJoined, remainingCount)
		}
	}

	if isFlightData {
		departed := make(map[string]int)
		landed := make(map[string]int)
		cancelled := make(map[string]int)

		for _, flightEntry := range diff {
			splitFlightEntry := strings.Split(flightEntry, ",")

			countryName := splitFlightEntry[11]
			statusString := splitFlightEntry[15]

			switch statusString {
			case "LANDED":
				landed[countryName] += 1
			case "DEPARTED":
				departed[countryName] += 1
			case "CANCELED":
				cancelled[countryName] += 1
			}
		}

		var stringArrayToBuildMessage []string

		stringArrayToBuildMessage = append(stringArrayToBuildMessage, "🛫 Departures To:")

		for key, value := range departed {
			stringArrayToBuildMessage = append(stringArrayToBuildMessage, fmt.Sprintf("%s: %d", key, value))
		}

		stringArrayToBuildMessage = append(stringArrayToBuildMessage, "🛬 Arrived From:")

		for key, value := range landed {
			stringArrayToBuildMessage = append(stringArrayToBuildMessage, fmt.Sprintf("%s: %d", key, value))
		}

		stringArrayToBuildMessage = append(stringArrayToBuildMessage, "❌ Cancelled Flights:")

		for key, value := range cancelled {
			stringArrayToBuildMessage = append(stringArrayToBuildMessage, fmt.Sprintf("%s: %d", key, value))
		}

		datasetDiff = strings.Join(stringArrayToBuildMessage, "\n")
	}

	prefixLen := len(utf16.Encode([]rune(prefix)))
	datasetNameLen := len(utf16.Encode([]rune(datasetName)))
	datasetDiffLen := len(utf16.Encode([]rune(datasetDiff)))

	var tagNames []string
	//var tagOffsets []int
	//var tagLengths []int

	// parse datapackage tags
	for _, tag := range datapackage.Tags {
		tagName := strings.ReplaceAll(tag.DisplayName, " ", "_")
		tagNames = append(tagNames, fmt.Sprintf("#%s", tagName))
	}

	tagString := strings.Join(tagNames, " ")

	blockquoteMsgEntity := MessageEntity{
		Type:   "expandable_blockquote",
		Offset: prefixLen + datasetNameLen + 2,
		Length: datasetDiffLen,
	}

	urllinkMsgEntity := MessageEntity{
		Type:   "text_link",
		Url:    fmt.Sprintf("https://data.gov.il/dataset/%s/resource/%s", datapackage.Id, resource.Id),
		Offset: prefixLen + 1,
		Length: datasetNameLen,
	}
	// send message
	payload := SendMessagePayload{
		ChatId: "@datasoup",
		Text:   strings.Join([]string{prefix, datasetName, datasetDiff, tagString}, "\n"),
		Entities: []MessageEntity{
			blockquoteMsgEntity,
			urllinkMsgEntity,
		},
	}
	return payload
}

func main() {
	bootstrapPtr := flag.Bool("bootstrap", false, "Bootstrap the data files")
	flag.Parse()
	fmt.Println("Hello, World!")
	if *bootstrapPtr {
		fmt.Println("Bootstrapping data files")
		// Ensure data dir exists
		err := os.MkdirAll("data", 0666)
		if err != nil {
			log.Fatalln(err)
		}
		// Get json from datagov
		resp, err := http.Post("https://data.gov.il/api/3/action/package_search", "application/json", strings.NewReader(`{"rows": 99999}`))
		if err != nil {
			log.Fatalln(err)
		}
		// write json response to file
		file, err := os.Create("data/packagedata.json")
		if err != nil {
			log.Fatalln(err)
		}

		_, err = io.Copy(file, resp.Body)
		if err != nil {
			log.Fatalln(err)
		}
		file.Close()
		resp.Body.Close()

		// unmarsal json to struct
		data, err := os.ReadFile("data/packagedata.json")
		if err != nil {
			log.Fatalln(err)
		}
		var datafile File
		json.Unmarshal(data, &datafile)

		client := &http.Client{Transport: &http.Transport{MaxConnsPerHost: 50}}

		var waitGroup sync.WaitGroup
		var packageCount int
		for _, datapackage := range datafile.Result.Results {
			for _, resource := range datapackage.Resources {
				if resource.Format == "CSV" {
					metadataTime, err := time.Parse("2006-01-02T15:04:05.000000", resource.MetadataModified)
					if err != nil {
						log.Fatalln(err)
					}
					if metadataTime.After(time.Now().AddDate(0, 0, -7)) { // if modified in the last 6 months
						waitGroup.Add(1)
						packageCount++
						go fetchResource(resource, datapackage, &waitGroup, client, 5)
					}
				}
			}
		}
		fmt.Println("Waiting for downloads to finish...")
		fmt.Println("Downloading", packageCount, "resources")
		waitGroup.Wait()
		fmt.Println("Downloads finished!")

	} else {
		fmt.Println("Running normally")
		// telegram bot
		// load token from environment variable
		token := os.Getenv("TELEGRAM_TOKEN")
		if token == "" {
			// fallback to reading from file for backward compatibility
			tokenBytes, err := os.ReadFile(".telegram_token")
			if err != nil {
				log.Fatalln("TELEGRAM_TOKEN environment variable not set and .telegram_token file not found")
			}
			token = string(tokenBytes)
		}
		endpointUrl := fmt.Sprint("https://api.telegram.org/bot", token)

		// check that it works
		respBotCheck, err := http.Get(fmt.Sprint(endpointUrl, "/getMe"))
		if err != nil {
			log.Fatalln(err)
		}
		// print body
		bodyBotCheck, err := io.ReadAll(respBotCheck.Body)
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Println(string(bodyBotCheck))
		respBotCheck.Body.Close()

		// Parse previous file for last modified date
		data, err := os.ReadFile("data/packagedata.json")
		if err != nil {
			log.Fatalln(err)
		}
		var datafile File
		json.Unmarshal(data, &datafile)
		refTime, err := time.Parse("2006-01-02T15:04:05.000000", datafile.Result.Results[0].MetadataModified) // wtf golang time parsing ಠ_ಠ
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Println(refTime)

		// Get json from datagov
		resp, err := http.Post("https://data.gov.il/api/3/action/package_search", "application/json", strings.NewReader(`{"rows": 99999}`))
		if err != nil {
			log.Fatalln(err)
		}
		// read response body into memory
		newDatafileBody, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatalln(err)
		}
		resp.Body.Close()

		var newDatafile File
		json.Unmarshal(newDatafileBody, &newDatafile)
		if newDatafile.Success == false {
			log.Fatalln("Failed to fetch new datafile")
		}

		client := &http.Client{Transport: &http.Transport{MaxConnsPerHost: 50}}
		charDetector := chardet.NewTextDetector()

		for _, datapackage := range newDatafile.Result.Results {
			for _, resource := range datapackage.Resources {
				if resource.Format == "CSV" && !isResourceExempt(resource.Id) && resource.Size < 200_000_000 { // is it csv, not excempt and less than 200 megabytes
					curTime, err := time.Parse("2006-01-02T15:04:05.000000", resource.MetadataModified)
					if err != nil {
						log.Fatalln(err)
					}
					dirpath := filepath.Join("data", datapackage.Organization.Name, datapackage.Id)
					filepath := filepath.Join("data", datapackage.Organization.Name, datapackage.Id, resource.Id+".csv")
					// check if the resource file was modified after refTime
					isFileOlderThanLastUpdate := false
					fileInfo, err := os.Stat(filepath)
					if err != nil {
						// if the file doesnt exist then we say that its older
						isFileOlderThanLastUpdate = true
					} else if fileInfo.ModTime().Before(refTime) {
						isFileOlderThanLastUpdate = true
					}

					if curTime.After(refTime) && isFileOlderThanLastUpdate {
						time.Sleep(2 * time.Second) // rate limit so telegram doesnt get mad
						fmt.Println(resource.Url)
						// fetch updated
						req, err := http.NewRequest("GET", resource.Url, nil)
						if err != nil {
							log.Fatalln(err)
						}
						req.Header.Set("User-Agent", "github.com/wissotsky#datagov-external-client")
						resp, err := client.Do(req) // TODO: Sometimes it returns an html page with 'Internal Server Error' must handle that as it currently corrupts the stored file
						if err != nil {
							log.Fatalln(err)
						}
						newfilebody, err := io.ReadAll(resp.Body)
						if err != nil {
							log.Fatalln(err)
						}
						resp.Body.Close()

						// detect encoding
						result, err := charDetector.DetectBest(newfilebody)
						if err != nil {
							log.Fatalln(err)
						}
						fmt.Println(result.Charset)
						// if charset is ISO-8859-8 or ISO-8859-8-I then convert from windows1255 to utf8
						if result.Charset != "UTF-8" {
							decoder := charmap.Windows1255.NewDecoder()
							newfilebody, err = decoder.Bytes(newfilebody)
							if err != nil {
								log.Fatalln(err)
							}
						}

						// check if file already exists
						if _, err := os.Stat(filepath); err == nil { // TODO: Redundant double check of file stat
							// file exists
							fmt.Println("File exists, diffing and overwriting")
							oldfile, err := os.ReadFile(filepath)
							if err != nil {
								log.Fatalln(err)
							}
							// run diffing TODO: Dont publish message if there is no difference in the resource
							oldlines := strings.Split(string(oldfile), "\n")
							newlines := strings.Split(string(newfilebody), "\n")
							var diff []string

							hashmap := make(map[string]struct{}, len(oldlines))
							for _, line := range oldlines {
								hashmap[line] = struct{}{}
							}

							for _, line := range newlines {
								if _, ok := hashmap[line]; !ok {
									diff = append(diff, line)
								}
							}

							payload := processDiffToPayload(false, diff, datapackage, resource)

							payloadJson, err := json.Marshal(payload)
							if err != nil {
								log.Fatalln(err)
							}
							fmt.Println(string(payloadJson))
							// TODO: send messsage to telegram
							resp, err = http.Post(fmt.Sprint(endpointUrl, "/sendMessage"), "application/json", bytes.NewBuffer(payloadJson))
							if err != nil {
								log.Fatalln(err)
							}
							fmt.Println(resp)
							// print body
							body, err := io.ReadAll(resp.Body)
							if err != nil {
								log.Fatalln(err)
							}
							fmt.Println(string(body))
							resp.Body.Close()

							// overwrite file

							file, err := os.Create(filepath)
							if err != nil {
								log.Fatalln(err)
							}
							_, err = file.Write(newfilebody)
							if err != nil {
								log.Fatalln(err)
							}
							file.Close()

						} else if os.IsNotExist(err) {
							// file does not exist
							fmt.Println("File does not exist, creating")
							err := os.MkdirAll(dirpath, 0666)
							if err != nil {
								log.Fatalln(err)
							}
							file, err := os.Create(filepath)
							if err != nil {
								log.Fatalln(err)
							}
							_, err = file.Write(newfilebody)
							if err != nil {
								log.Fatalln(err)
							}
							file.Close()

							diff := strings.Split(string(newfilebody), "\n")

							payload := processDiffToPayload(true, diff, datapackage, resource)

							payloadJson, err := json.Marshal(payload)
							if err != nil {
								log.Fatalln(err)
							}
							fmt.Println(string(payloadJson))
							// TODO: send messsage to telegram
							resp, err = http.Post(fmt.Sprint(endpointUrl, "/sendMessage"), "application/json", bytes.NewBuffer(payloadJson))
							if err != nil {
								log.Fatalln(err)
							}
							fmt.Println(resp)
							// print body
							body, err := io.ReadAll(resp.Body)
							if err != nil {
								log.Fatalln(err)
							}
							fmt.Println(string(body))
							resp.Body.Close()

						} else {
							log.Fatalln(err)
						}
					}
				}
			}

		}
		fmt.Println("Done updating, overwriting packagedata.json")
		// overwrite packagedata.json

		file, err := os.Create("data/packagedata.json")
		if err != nil {
			log.Fatalln(err)
		}
		_, err = file.Write(newDatafileBody)
		if err != nil {
			log.Fatalln(err)
		}
		file.Close()

	}

	fmt.Println("Done!")

	// may the operating system be our garbage collector and handle handler （￣︶￣）↗
}
