package caldav

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/arran4/golang-ical"
	"github.com/studio-b12/gowebdav"
)

type Client struct {
	webdavClient *gowebdav.Client
	baseURL      string
}

type Config struct {
	URL      string
	Username string
	Password string
}

func NewClient(config Config) *Client {
	client := gowebdav.NewClient(config.URL, config.Username, config.Password)

	return &Client{
		webdavClient: client,
		baseURL:      config.URL,
	}
}

func (c *Client) GetEvents() ([]*ics.VEvent, error) {
	files, err := c.webdavClient.ReadDir("/")
	if err != nil {
		return nil, fmt.Errorf("failed to list calendar files: %w", err)
	}

	var events []*ics.VEvent

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".ics") {
			continue
		}

		content, err := c.webdavClient.Read(file.Name())
		if err != nil {
			fmt.Printf("Warning: failed to read file %s: %v\n", file.Name(), err)
			continue
		}

		calendar, err := ics.ParseCalendar(strings.NewReader(string(content)))
		if err != nil {
			fmt.Printf("Warning: failed to parse calendar %s: %v\n", file.Name(), err)
			continue
		}

		for _, event := range calendar.Events() {
			events = append(events, event)
		}
	}

	return events, nil
}

func (c *Client) TestConnection() error {
	resp, err := http.Get(c.baseURL)
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("server returned error status: %d", resp.StatusCode)
	}

	return nil
}