package http

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/byxorna/jot/pkg/runtime"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	// This is sourced from setting up the oauth client somewhere like
	// https://developers.google.com/calendar/caldav/v2/guide?hl=en_US
	// TODO: idk whether its ok to package this into the repo or not!!!!
	//go:embed credentials.json
	credentialsJSON []byte
)

type Client struct {
	sync.RWMutex
}

/*func GetHTTPClientFromGoogleCreds(ctx context.Context, creds *google.Credentials, tokenStorageFileName string) (*http.Client, error) {
	client, err := google.DefaultTokenSourcejk
	return client, nil
}
*/

// Retrieve a token, saves the token, then returns the generated client.
func GetHTTPClientFromOAuth2Creds(ctx context.Context, oauth2cfg *oauth2.Config, tokenStorageFileName string, tokenScope string) (*http.Client, error) {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile, err := runtime.File(tokenStorageFileName)
	if err != nil {
		return nil, fmt.Errorf("unable to determine token storage file %s: %w", tokenStorageFileName, err)
	}
	fmt.Printf("loading token in %s\n", tokFile)

	tok, err := TokenFromFile(tokFile)
	if err != nil {
		tok, err := GetTokenFromWeb(ctx, oauth2cfg, tokenScope)
		if err != nil {
			return nil, err
		}
		SaveToken(tokFile, tok)
	}

	return oauth2cfg.Client(ctx, tok), nil
}

// Request a token from the web, then returns the retrieved token.
func GetTokenFromWeb(ctx context.Context, oauth2cfg *oauth2.Config, tokenScope string) (*oauth2.Token, error) {
	authURL := oauth2cfg.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("You must authenticate via OAuth2 to Google to allow Jot access to manage your notes. Please visit the following URL and then enter the auth code below:\n%v\n", authURL)
	for _, scope := range oauth2cfg.Scopes {
		fmt.Printf("* %s\n", scope)
	}

	// TODO: exec "open ..."
	fmt.Printf("\n\nPlease enter the auth code here: ")
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
func TokenFromFile(file string) (*oauth2.Token, error) {
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
func SaveToken(path string, token *oauth2.Token) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("unable to cache oauth token: %w", err)
	}
	defer f.Close()
	fmt.Printf("caching token in %s\n", path)
	return json.NewEncoder(f).Encode(token)
}

// New creates a new http client that is authorized to use a given set of google API scopes
// If modifying these scopes, delete your previously saved tokenStorageFile
// tokenStorageFile will be used as a cache for the saved token
func NewClientWithGoogleAuthedScopes(ctx context.Context, tokenScope string, scope ...string) (*http.Client, error) {
	cfg, err := google.ConfigFromJSON(credentialsJSON, scope...)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %w", err)
	}
	httpclient, err := GetHTTPClientFromOAuth2Creds(ctx, cfg, fmt.Sprintf("%s_google_credentials.json", tokenScope), tokenScope)
	if err != nil {
		return nil, fmt.Errorf("unable to create client: %w", err)
	}

	return httpclient, nil
}

func NewDefaultClient(ctx context.Context, scope ...string) (*http.Client, error) {
	return NewClientWithGoogleAuthedScopes(ctx, "default", scope...)
}
