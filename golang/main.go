package main

import (
	"context"
	"encoding/gob"
	"flag"
	"fmt"
	"hash/fnv"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

func main() {
	backDaysPtr := flag.Int("backDays", 720, "How many days back to process events")
	forwardDaysPtr := flag.Int("forwardDays", 365, "How many days forward to process events")
	portPtr := flag.Int("port", 3000, "The port to run the callback server on localhost")
	storePtr := flag.String("store", "$HOME/.local/share/gcal-to-org", "Directory to store tokens")

	flag.Parse()

	if len(flag.Args()) != 1 {
		fmt.Println("Expected FILE to output to")
		fmt.Printf("Usage: %s <flags> FILE", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	fmt.Println("backDays:", *backDaysPtr)
	fmt.Println("forwardDays:", *forwardDaysPtr)
	fmt.Println("port:", *portPtr)
	fmt.Println("store:", *storePtr)

	store, err := storeDir(*storePtr)
	if err != nil {
		fmt.Printf("Unable to create %s directory\n", *storePtr)
		os.Exit(1)
	}
	fmt.Println("store:", store)
	fmt.Println("dest:", flag.Arg(0))

	config := &oauth2.Config{
		ClientID:     MY_CLIENT_ID,
		ClientSecret: MY_CLIENT_SECRET,
		Endpoint:     google.Endpoint,
		Scopes:       []string{calendar.CalendarScope},
	}

	ctx := context.Background()
	client := newOAuthClient(ctx, config, store)
	// svc, err := calendar.New(client)
	svc, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to create Calendar service: %v", err)
	}

	// get all the events in the correct time range
	events, err := svc.Events.List("primary").
		SingleEvents(true).
		TimeMin(time.Now().AddDate(0, 0, -*backDaysPtr).Format(time.RFC3339)).
		TimeMax(time.Now().AddDate(0, 0, *forwardDaysPtr).Format(time.RFC3339)).
		Do()
	if err != nil {
		log.Fatalf("Unable to retrieve calendar events: %v", err)
	}

	fmt.Printf("Found %d events\n", len(events.Items))
	for _, event := range events.Items {
		if relevantEvent(event) {
			fmt.Printf("%s %s (%d attendees)\n", event.Status, event.Summary, len(event.Attendees))
		}
	}
}

func relevantEvent(event *calendar.Event) bool {
	if event.Status == "cancelled" ||
		event.Summary == "" ||
		event.Start.DateTime == "" ||
		len(event.Attendees) == 0 {
		return false
	}

	// check that I accepted the event
	for _, attendee := range event.Attendees {
		if attendee.Self && attendee.ResponseStatus != "accepted" {
			return false
		}
	}

	return true
}

func storeDir(unexanded string) (string, error) {
	expanded := os.ExpandEnv(unexanded)
	fmt.Printf("Ensuring that %s exists\n", expanded)
	err := os.MkdirAll(expanded, 0700)
	if err != nil {
		return "", err
	}
	return expanded, nil
}

func newOAuthClient(ctx context.Context, config *oauth2.Config, storeDir string) *http.Client {
	cacheFile := tokenCacheFile(config, storeDir)
	token, err := tokenFromFile(cacheFile)
	if err != nil {
		token = tokenFromWeb(ctx, config)
		saveToken(cacheFile, token)
	}

	return config.Client(ctx, token)
}

func tokenCacheFile(config *oauth2.Config, storeDir string) string {
	hash := fnv.New32a()
	hash.Write([]byte(config.ClientID))
	hash.Write([]byte(config.ClientSecret))
	hash.Write([]byte(strings.Join(config.Scopes, " ")))
	fn := fmt.Sprintf("gcal-to-org-tok%v", hash.Sum32())
	return filepath.Join(storeDir, url.QueryEscape(fn))
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	t := new(oauth2.Token)
	err = gob.NewDecoder(f).Decode(t)
	return t, err
}

func tokenFromWeb(ctx context.Context, config *oauth2.Config) *oauth2.Token {
	ch := make(chan string)
	randState := fmt.Sprintf("st%d", time.Now().UnixNano())
	ts := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/favicon.ico" {
			http.Error(rw, "", 404)
			return
		}
		if req.FormValue("state") != randState {
			log.Printf("State doesn't match: req = %#v", req)
			http.Error(rw, "", 500)
			return
		}
		if code := req.FormValue("code"); code != "" {
			fmt.Fprintf(rw, "<h1>Success</h1>Authorized.")
			rw.(http.Flusher).Flush()
			ch <- code
			return
		}
		log.Printf("no code")
		http.Error(rw, "", 500)
	}))
	defer ts.Close()

	config.RedirectURL = ts.URL
	authURL := config.AuthCodeURL(randState)
	go openURL(authURL)
	log.Printf("Authorize this app at: %s", authURL)
	code := <-ch
	log.Printf("Got code: %s", code)

	token, err := config.Exchange(ctx, code)
	if err != nil {
		log.Fatalf("Token exchange error: %v", err)
	}
	return token
}

func openURL(url string) {
	try := []string{"xdg-open", "google-chrome", "open"}
	for _, bin := range try {
		err := exec.Command(bin, url).Run()
		if err == nil {
			return
		}
	}
	log.Printf("Error opening URL in browser.")
}

func saveToken(file string, token *oauth2.Token) {
	f, err := os.Create(file)
	if err != nil {
		log.Printf("Warning: failed to cache oauth token: %v", err)
		return
	}
	defer f.Close()
	gob.NewEncoder(f).Encode(token)
}
