package google

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type Auth struct {
	config    *oauth2.Config
	client    *http.Client
	tokenPath string
}

func NewAuth(credentialsPath, tokenPath, redirectURL string) (*Auth, error) {
	b, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read client secret file: %v", err)
	}

	config, err := google.ConfigFromJSON(b, sheets.SpreadsheetsScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %v", err)
	}

	// Set redirect URL from config
	config.RedirectURL = redirectURL

	return &Auth{
		config:    config,
		tokenPath: tokenPath,
	}, nil
}

func (a *Auth) GetClient() (*http.Client, error) {
	if a.client != nil {
		return a.client, nil
	}

	tok, err := a.tokenFromFile()
	if err != nil {
		tok, err = a.getTokenFromWeb()
		if err != nil {
			return nil, err
		}
		a.saveToken(tok)
	}

	a.client = a.config.Client(context.Background(), tok)
	return a.client, nil
}

func (a *Auth) GetSheetsService() (*sheets.Service, error) {
	client, err := a.GetClient()
	if err != nil {
		return nil, err
	}

	srv, err := sheets.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Sheets client: %v", err)
	}

	return srv, nil
}

func (a *Auth) getTokenFromWeb() (*oauth2.Token, error) {
	// Channel to receive the authorization code
	codeChan := make(chan string)
	
	// Extract port from redirect URL
	port := ":8080"
	if strings.Contains(a.config.RedirectURL, "localhost:") {
		parts := strings.Split(a.config.RedirectURL, ":")
		if len(parts) >= 3 {
			portPart := strings.Split(parts[2], "/")[0]
			port = ":" + portPart
		}
	}
	
	// Start local server to handle OAuth callback
	server := &http.Server{Addr: port}
	
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			fmt.Fprintf(w, "Error: No authorization code received")
			return
		}
		
		fmt.Fprintf(w, `
			<html>
				<head><title>Authentication Successful</title></head>
				<body>
					<h1>Authentication Successful!</h1>
					<p>You can close this window and return to the terminal.</p>
					<script>window.setTimeout(function(){window.close();}, 2000);</script>
				</body>
			</html>
		`)
		
		codeChan <- code
	})
	
	// Start server in goroutine
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
		}
	}()
	
	// Generate auth URL and open in browser
	authURL := a.config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Opening browser for authentication...\n")
	fmt.Printf("If browser doesn't open automatically, visit:\n%v\n", authURL)
	
	// Try to open browser automatically
	openBrowser(authURL)
	
	// Wait for authorization code
	fmt.Println("Waiting for authentication...")
	authCode := <-codeChan
	
	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
	
	// Exchange code for token
	tok, err := a.config.Exchange(context.Background(), authCode)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve token from web: %v", err)
	}
	
	return tok, nil
}

func (a *Auth) tokenFromFile() (*oauth2.Token, error) {
	f, err := os.Open(a.tokenPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

func (a *Auth) saveToken(token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", a.tokenPath)
	f, err := os.OpenFile(a.tokenPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

// openBrowser tries to open the URL in a browser
func openBrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}

	if err != nil {
		log.Printf("Failed to open browser: %v", err)
	}
}
