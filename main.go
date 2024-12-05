package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"time"
)

type Feed struct {
	Entries []Entry `xml:"entry"`
}

type Entry struct {
	ID      string `xml:"id"`
	Updated string `xml:"updated"`
	Title   string `xml:"title"`
	Content string `xml:"content>div"`
	Link    struct {
		Href string `xml:"href,attr"`
	} `xml:"link"`
}

type Incident struct {
	ID      string `json:"id"`
	Updated string `json:"updated"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Link    string `json:"link"`
}

const (
	feedURL  = "https://status.hetzner.com/en.atom"
	dataFile = "data.json"
)

func main() {
	resp, err := http.Get(feedURL)
	if err != nil {
		fmt.Printf("Error fetching the feed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body: %v\n", err)
		os.Exit(1)
	}

	var feed Feed
	if err := xml.Unmarshal(body, &feed); err != nil {
		fmt.Printf("Error parsing the feed: %v\n", err)
		os.Exit(1)
	}

	existingIncidents := loadIncidents()

	newIncidents := []Incident{}
	for _, entry := range feed.Entries {
		if _, exists := existingIncidents[entry.ID]; !exists {
			incident := Incident{
				ID:      entry.ID,
				Updated: entry.Updated,
				Title:   entry.Title,
				Content: entry.Content,
				Link:    entry.Link.Href,
			}
			existingIncidents[entry.ID] = incident
			newIncidents = append(newIncidents, incident)
		}
	}

	if len(newIncidents) > 0 {
		saveIncidents(existingIncidents)
		commitToGit(newIncidents)
	} else {
		fmt.Println("No new incidents to commit.")
	}
}

func loadIncidents() map[string]Incident {
	file, err := os.Open(dataFile)
	if os.IsNotExist(err) {
		return make(map[string]Incident)
	} else if err != nil {
		fmt.Printf("Error opening data file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	var incidents map[string]Incident
	if err := json.NewDecoder(file).Decode(&incidents); err != nil {
		fmt.Printf("Error parsing JSON data: %v\n", err)
		os.Exit(1)
	}
	return incidents
}

func saveIncidents(incidents map[string]Incident) {
	file, err := os.Create(dataFile)
	if err != nil {
		fmt.Printf("Error creating data file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(incidents); err != nil {
		fmt.Printf("Error saving JSON data: %v\n", err)
		os.Exit(1)
	}
}

func commitToGit(newIncidents []Incident) {
	runCommand("git", "config", "--global", "user.email", "github-actions[bot]@users.noreply.github.com")
	runCommand("git", "config", "--global", "user.name", "GitHub Actions")

	runCommand("git", "add", dataFile)

	message := fmt.Sprintf("Update incidents: %d new (%s)", len(newIncidents), time.Now().Format(time.RFC3339))
	runCommand("git", "commit", "-m", message)
	runCommand("git", "push")
}

func runCommand(command string, args ...string) {
	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error running command '%s %v': %v\n", command, args, err)
		os.Exit(1)
	}
}
