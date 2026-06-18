package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/gob"
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

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

func newCalendarService(ctx context.Context, storeDirPath string) (*calendar.Service, error) {
	store, err := storeDir(storeDirPath)
	if err != nil {
		return nil, fmt.Errorf("create store directory %q: %w", storeDirPath, err)
	}

	config := &oauth2.Config{
		ClientID:     MY_CLIENT_ID,
		ClientSecret: MY_CLIENT_SECRET,
		Endpoint:     google.Endpoint,
		Scopes:       []string{calendar.CalendarScope},
	}

	client, err := newOAuthClient(ctx, config, store)
	if err != nil {
		return nil, fmt.Errorf("create OAuth client: %w", err)
	}

	svc, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("create Calendar service: %w", err)
	}
	return svc, nil
}

func storeDir(unexpanded string) (string, error) {
	expanded := os.ExpandEnv(unexpanded)
	err := os.MkdirAll(expanded, 0700)
	if err != nil {
		return "", err
	}
	return expanded, nil
}

func newOAuthClient(ctx context.Context, config *oauth2.Config, storeDir string) (*http.Client, error) {
	cacheFile := tokenCacheFile(config, storeDir)
	token, err := tokenFromFile(cacheFile)
	if err != nil {
		token, err = tokenFromWeb(ctx, config)
		if err != nil {
			return nil, err
		}
		if err := saveToken(cacheFile, token); err != nil {
			log.Printf("Warning: failed to cache oauth token: %v", err)
		}
	}

	return config.Client(ctx, token), nil
}

func tokenCacheFile(config *oauth2.Config, storeDir string) string {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(config.ClientID))
	_, _ = hash.Write([]byte(config.ClientSecret))
	_, _ = hash.Write([]byte(strings.Join(config.Scopes, " ")))
	fn := fmt.Sprintf("gcal-to-org-tok%v", hash.Sum32())
	return filepath.Join(storeDir, url.QueryEscape(fn))
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	t := new(oauth2.Token)
	err = gob.NewDecoder(f).Decode(t)
	return t, err
}

func tokenFromWeb(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	ch := make(chan string)
	randState, err := randomState()
	if err != nil {
		return nil, err
	}
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

	cfg := *config
	cfg.RedirectURL = ts.URL
	authURL := cfg.AuthCodeURL(randState)
	go openURL(authURL)
	log.Printf("Authorize this app at: %s", authURL)
	code := <-ch
	log.Printf("Got code: %s", code)

	token, err := cfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchange token: %w", err)
	}
	return token, nil
}

func randomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
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

func saveToken(file string, token *oauth2.Token) (err error) {
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := f.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()
	if err := gob.NewEncoder(f).Encode(token); err != nil {
		return err
	}
	return nil
}
