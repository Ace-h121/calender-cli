package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

func loadConfig() (*oauth2.Config, error) {
	b, err := os.ReadFile(secretPath())
	if err != nil {
		return nil, fmt.Errorf("unable to read client secret file %s: %w", secretPath(), err)
	}
	config, err := google.ConfigFromJSON(b, calendar.CalendarScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file: %w", err)
	}
	return config, nil
}

// getClient returns an HTTP client backed by the stored token, triggering the
// OAuth flow the first time the token is missing.
func getClient(config *oauth2.Config, tokFile string) *http.Client {
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// getTokenFromWeb runs the OAuth loopback flow and returns the exchanged token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	ch := make(chan string)
	state := fmt.Sprintf("st%d", time.Now().UnixNano())

	ts := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/favicon.ico" {
			http.Error(rw, "", http.StatusNotFound)
			return
		}
		if req.FormValue("state") != state {
			log.Printf("state mismatch: %#v", req)
			http.Error(rw, "", http.StatusInternalServerError)
			return
		}
		if code := req.FormValue("code"); code != "" {
			fmt.Fprintln(rw, "<h1>Success</h1>Authorized.")
			ch <- code
			return
		}
		log.Printf("no code in request: %#v", req)
		http.Error(rw, "", http.StatusInternalServerError)
	}))
	defer ts.Close()

	config.RedirectURL = ts.URL
	authURL := config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	fmt.Printf("Go to the following link in your browser to authorize:\n%v\n", authURL)
	fmt.Println("Waiting for authorization...")

	tok, err := config.Exchange(context.TODO(), <-ch)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// tokenFromFile retrieves a token from a local file.
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

// saveToken writes a token to a local file.
func saveToken(path string, token *oauth2.Token) {
	if err := os.MkdirAll(configDir(), 0o700); err != nil {
		log.Fatalf("Unable to create config dir: %v", err)
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication with Google",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate (or re-authenticate) with Google",
	RunE: func(cmd *cobra.Command, args []string) error {
		os.Remove(tokenPath())
		config, err := loadConfig()
		if err != nil {
			return err
		}
		tok := getTokenFromWeb(config)
		saveToken(tokenPath(), tok)
		fmt.Println("Authenticated.")
		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove the stored token and revoke access",
	RunE: func(cmd *cobra.Command, args []string) error {
		if tok, err := tokenFromFile(tokenPath()); err == nil {
			httpc := &http.Client{Timeout: 10 * time.Second}
			httpc.Post("https://oauth2.googleapis.com/revoke?token="+tok.AccessToken,
				"application/x-www-form-urlencoded", nil)
		}
		if err := os.Remove(tokenPath()); err != nil && !os.IsNotExist(err) {
			return err
		}
		fmt.Println("Logged out.")
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		tok, err := tokenFromFile(tokenPath())
		if err != nil {
			fmt.Println("Not authenticated. Run `calender-cli auth login`.")
			return nil
		}
		exp := "unknown"
		if !tok.Expiry.IsZero() {
			exp = tok.Expiry.Local().Format(time.RFC3339)
			if time.Until(tok.Expiry) < 0 {
				exp += " (expired, will refresh)"
			}
		}
		fmt.Printf("Token file: %s\n", tokenPath())
		fmt.Printf("Expiry:     %s\n", exp)
		fmt.Printf("Scopes:     %s\n", grantedScopes(tok.AccessToken))
		return nil
	},
}

// grantedScopes queries Google's tokeninfo endpoint for the scopes attached to
// a token. Returns "unknown" when the call fails.
func grantedScopes(accessToken string) string {
	if accessToken == "" {
		return "unknown"
	}
	httpc := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpc.Get("https://www.googleapis.com/oauth2/v3/tokeninfo?access_token=" + accessToken)
	if err != nil {
		return "unknown"
	}
	defer resp.Body.Close()
	var info struct {
		Scope string `json:"scope"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "unknown"
	}
	if info.Scope == "" {
		return "unknown"
	}
	return info.Scope
}

func init() {
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
}
