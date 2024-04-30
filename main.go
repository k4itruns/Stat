package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/fatih/color"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Config struct {
		Token string `yaml:"token"`
	} `yaml:"config"`
}

func main() {
	fmt.Println("Enter the number of threads (goroutines):")
	var numThreads int
	_, err := fmt.Scanln(&numThreads)
	if err != nil {
		log.Fatalf("Error reading number of threads: %v\n", err)
	}

	os.Setenv("CLS", "cls")

	log.Println(blue("Loading statuses."))

	statusList, err := readStatusesFromFile("statuses.json")
	if err != nil {
		log.Fatalf(red("Failed to load statuses: %v\n"), err)
	}

	log.Printf(blue("Total %d statuses found.\n"), len(statusList))

	config, err := readConfigFromFile("config.yml")
	if err != nil {
		log.Fatalf(red("Failed to load config: %v\n"), err)
	}

	var wg sync.WaitGroup
	for i := 0; i < numThreads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			statuser := newStatuser(config.Config.Token, statusList)
			for {
				err := statuser.rotate()
				if err != nil {
					log.Printf(red("Error: %v\n"), err)
				}
				time.Sleep(5 * time.Second)
			}
		}()
	}
	wg.Wait()
}

func readStatusesFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var statuses struct {
		Statuses []string `json:"statuses"`
	}
	if err := json.NewDecoder(file).Decode(&statuses); err != nil {
		return nil, err
	}

	return statuses.Statuses, nil
}

func readConfigFromFile(filename string) (*Config, error) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(file, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

type statuser struct {
	Token  string
	Status []string
}

func newStatuser(token string, status []string) *statuser {
	return &statuser{
		Token:  token,
		Status: status,
	}
}

func (s *statuser) rotate() error {
	rand.Seed(time.Now().UnixNano())
	status := s.Status[rand.Intn(len(s.Status))]

	payload := map[string]interface{}{
		"custom_status": map[string]interface{}{
			"text":       status,
			"emoji_id":   nil,
			"emoji_name": nil,
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("JSON marshal error: %w", err)
	}

	req, err := http.NewRequest("PATCH", "https://discord.com/api/v9/users/@me/settings", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("HTTP request creation error: %w", err)
	}

	req.Header.Set("Authorization", s.Token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request execution error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
		log.Printf(green("Rotated status, Status: %s\n"), status)
	} else {
		log.Printf(red("Failed to rotate status. Status code: %d\n"), resp.StatusCode)
	}

	return nil
}

var (
	red    = color.New(color.FgRed, color.Bold).SprintFunc()
	blue   = color.New(color.FgBlue, color.Bold).SprintFunc()
	green  = color.New(color.FgGreen, color.Bold).SprintFunc()
	yellow = color.New(color.FgYellow, color.Bold).SprintFunc()
)
