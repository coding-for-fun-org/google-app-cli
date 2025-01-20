package google_oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

type installed struct {
	ClientID            string   `json:"client_id"`
	ProjectID           string   `json:"project_id"`
	AuthURI             string   `json:"auth_uri"`
	TokenURI            string   `json:"token_uri"`
	AuthProviderCertURL string   `json:"auth_provider_x509_cert_url"`
	ClientSecret        string   `json:"client_secret"`
	RedirectURIs        []string `json:"redirect_uris"`
}

type credentials struct {
	Installed installed `json:"installed"`
}

type OAuth struct {
	creds credentials
}

// initializeCredentials method initializes the credentials for the OAuth client.
func (o *OAuth) initializeCredentials() {
	googleClientId := os.Getenv("GOOGLE_CLIENT_ID")

	if googleClientId == "" {
		log.Fatalln("GOOGLE_CLIENT_ID is not set")
	}

	googleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")

	if googleClientSecret == "" {
		log.Fatalln("GOOGLE_CLIENT_SECRET is not set")
	}

	o.creds = credentials{
		Installed: installed{
			ProjectID:           "jiyeol-tech",
			AuthURI:             "https://accounts.google.com/o/oauth2/auth",
			TokenURI:            "https://oauth2.googleapis.com/token",
			AuthProviderCertURL: "https://www.googleapis.com/oauth2/v1/certs",
			RedirectURIs:        []string{"http://localhost:8000/callback"},
			ClientID:            googleClientId,
			ClientSecret:        googleClientSecret,
		},
	}
}

// GetCalendarService method returns a new calendar service.
func (o *OAuth) GetCalendarService() (*calendar.Service, error) {
	o.initializeCredentials()

	ctx := context.Background()

	b, err := json.Marshal(o.creds)
	if err != nil {
		return nil, err
	}

	config, err := google.ConfigFromJSON(b, calendar.CalendarEventsScope)
	if err != nil {
		return nil, err
	}

	client, err := getClient(config)
	if err != nil {
		return nil, err
	}

	return calendar.NewService(ctx, option.WithHTTPClient(client))
}

// getClient function retrieves a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) (*http.Client, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	tokFile := home + "/token.json"
	tok, err := getTokenFromFile(tokFile)
	if err != nil {
		tok, err = getTokenFromWeb(config)
		if err != nil {
			return nil, err
		}
		err := saveToken(tokFile, tok)
		if err != nil {
			return nil, err
		}
	}
	return config.Client(context.Background(), tok), nil
}

// getTokenFromWeb function retrieves a token from the web.
func getTokenFromWeb(config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	log.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	err := exec.Command("open", authURL).Start()
	if err != nil {
		return nil, err
	}

	listenAddr := "localhost:8000"
	redirectURI := fmt.Sprintf("http://%s/callback", listenAddr)
	config.RedirectURL = redirectURI

	authCodeCh := make(chan string)

	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Authorization code not found", http.StatusBadRequest)
			return
		}

		authCodeCh <- code
		fmt.Fprintln(w, "Authorization code received. You can close this window.")
	})
	go func() {
		log.Fatal(http.ListenAndServe(listenAddr, nil))
	}()

	authCode := <-authCodeCh

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		return nil, err
	}
	return tok, nil
}

// getTokenFromFile function retrieves a token from a local file.
func getTokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// saveToken function saves a token to a local file.
func saveToken(path string, token *oauth2.Token) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)

	return nil
}
