package calendar

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/byxorna/jot/pkg/runtime"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

var (
	// This is sourced from setting up the oauth client somewhere like
	// https://developers.google.com/calendar/caldav/v2/guide?hl=en_US
	// TODO: idk whether its ok to package this into the repo or not!!!!
	//go:embed credentials.json
	credentialsJSON []byte

	// PluginName is required to allow the package to be enabled
	PluginName = "calendar"

	tokenStorageFile = "google_calendar_token.json"
)

type Client struct {
	*calendar.Service
}

// Retrieve a token, saves the token, then returns the generated client.
func getClient(ctx context.Context, oauth2cfg *oauth2.Config) (*http.Client, error) {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile, err := runtime.File(tokenStorageFile)
	if err != nil {
		return nil, fmt.Errorf("unable to determine token storage file %s: %w", tokenStorageFile, err)
	}

	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok, err := getTokenFromWeb(ctx, oauth2cfg)
		if err != nil {
			return nil, err
		}
		saveToken(tokFile, tok)
	}

	return oauth2cfg.Client(ctx, tok), nil
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(ctx context.Context, oauth2cfg *oauth2.Config) (*oauth2.Token, error) {
	authURL := oauth2cfg.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("[plugin:%s] Go to the following link in your browser then type the authorization code: \n%v\n", PluginName, authURL)

	fmt.Printf("[plugin:%s] Please enter the auth code here: ", PluginName)
	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		return nil, fmt.Errorf("unable to read authorization code: %w", err)
	}

	tok, err := oauth2cfg.Exchange(ctx, authCode)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve token from web: %w", err)
	}
	return tok, nil
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) error {
	fmt.Printf("[plugin:%s] Saving credential file to: %s\n", PluginName, path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("unable to cache oauth token: %w", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
	return nil
}

func New(ctx context.Context) (*Client, error) {
	// If modifying these scopes, delete your previously saved token.json.
	cfg, err := google.ConfigFromJSON(credentialsJSON, calendar.CalendarReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %w", err)
	}
	client, err := getClient(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to create client: %w", err)
	}

	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Calendar client: %w", err)
	}

	c := Client{Service: srv}
	return &c, nil
}

func (c *Client) Run() error {
	t := time.Now().Format(time.RFC3339)
	events, err := c.Service.Events.List("primary").ShowDeleted(false).SingleEvents(true).TimeMin(t).MaxResults(10).OrderBy("startTime").Do()
	if err != nil {
		return fmt.Errorf("unable to retrieve next ten of the user's events: %w", err)
	}

	fmt.Println("Upcoming events:")
	if len(events.Items) == 0 {
		fmt.Println("No upcoming events found.")
	} else {
		for _, item := range events.Items {
			date := item.Start.DateTime
			if date == "" {
				date = item.Start.Date
			}
			fmt.Printf("%v (%v)\n", item.Summary, date)
		}
	}

	return nil
}
