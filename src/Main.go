package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/hugolgst/rich-go/client"
	"github.com/shirou/gopsutil/process"
	//"strconv"
)

var (
	placeId string
	reset   = false
)

func GetProcessByName(targetProcessName string) *process.Process {
	processes, _ := process.Processes()

	for _, proc := range processes {
		name, _ := proc.Name()

		if name == targetProcessName {
			return proc
		}
	}

	return nil
}

// https://mholt.github.io/json-to-go/
type Creator struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type GameInfo struct {
	ID          int64   `json:"id"` // This is UniverseId
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Creator     Creator `json:"creator"`
}

type GameResponse struct {
	Data []GameInfo `json:"data"`
}

func GetGameInfoByUniverseId(universeId string) *GameInfo {
	url := "https://games.roblox.com/v1/games?universeIds=" + universeId

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		return nil
	}

	req.Header.Set("User-Agent", "Mozilla/5.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("HTTP request failed: %v", err)
		return nil
	}
	defer resp.Body.Close()

	var response GameResponse
	body, _ := io.ReadAll(resp.Body) // Optional: for debugging
	// fmt.Println(string(body))

	if err := json.Unmarshal(body, &response); err != nil {
		log.Printf("JSON decoding failed: %v", err)
		return nil
	}

	if len(response.Data) == 0 {
		log.Println("No game data found.")
		return nil
	}

	return &response.Data[0]
}
func GetPlaceInfoByPlaceId(placeId string) *GameInfo {
	universeId := GetUniverseIdFromPlaceId(placeId)
	if universeId == "" {
		log.Println("Universe ID not found for placeId:", placeId)
		return nil
	}

	url := "https://games.roblox.com/v1/games?universeIds=" + universeId

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		return nil
	}

	req.Header.Set("User-Agent", "Mozilla/5.0")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("HTTP request failed: %v", err)
		return nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var response GameResponse
	if err := json.Unmarshal(body, &response); err != nil {
		log.Printf("JSON decoding failed: %v", err)
		log.Println("Body was:", string(body))
		return nil
	}

	if len(response.Data) == 0 {
		log.Println("No game data found for universeId:", universeId)
		return nil
	}

	return &response.Data[0]
}

type UniverseInfo struct {
	UniverseId int64 `json:"universeId"`
}

func GetUniverseIdFromPlaceId(placeId string) string {
	url := "https://apis.roblox.com/universes/v1/places/" + placeId + "/universe"

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Failed to get universeId:", err)
		return ""
	}
	defer resp.Body.Close()

	var u UniverseInfo
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		log.Println("Decode error:", err)
		return ""
	}
	return fmt.Sprint(u.UniverseId)
}

func UpdateRobloxPresence() {
	roblox := GetProcessByName("RobloxPlayerBeta.exe")

	for roblox == nil {
		roblox = GetProcessByName("RobloxPlayerBeta.exe")

		if reset == false {
			reset = true

			client.Logout()
			fmt.Println("reset client activity")
		}
	}

	err := client.Login("") //Client/Application ID from https://discord.com/developers/applications

	if err != nil {
		panic(err)
	}

	reset = false
	args, _ := roblox.Cmdline()
	pattern := `placeId(?:=|%3D)(\d+)`
	placePattern := regexp.MustCompile(pattern)
	StringSubmatch := placePattern.FindStringSubmatch(args)
	placeMatch := ""
	if len(StringSubmatch) > 0 {
		placeMatch = StringSubmatch[1]
		fmt.Println("string match: " + StringSubmatch[1])
	}

	// timePattern := regexp.MustCompile(`launchtime=(\d+)`)
	// timeMatch := timePattern.FindStringSubmatch(args)[1]

	// startTime, _ := strconv.ParseInt(timeMatch, 10, 64)

	now := time.Now()

	if placeMatch != placeId && placeMatch != "" {
		placeId = placeMatch
		fmt.Println("Getting place info")
		place := GetPlaceInfoByPlaceId(placeId)
		fmt.Println("setting activity...")
		err = client.SetActivity(client.Activity{
			State:      "by " + place.Creator.Name,
			Details:    place.Name,
			LargeImage: "roblox_logo",
			LargeText:  place.Name,
			SmallText:  place.Name,
			Timestamps: &client.Timestamps{
				Start: &now,
			},
			Buttons: []*client.Button{
				&client.Button{
					Label: place.Name,
					Url:   "https://www.roblox.com/games/" + placeId,
				},
			},
		})
		if err != nil {
			panic(err)
		}
		fmt.Println("set activity: " + place.Name)
		fmt.Println("by: " + place.Creator.Name)
	}
}

func main() {
	for true {
		UpdateRobloxPresence()

		time.Sleep(time.Second * 5)
	}
}
