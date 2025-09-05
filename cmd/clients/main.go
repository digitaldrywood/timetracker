package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/digitaldrywood/timetracker/internal/config"
	"github.com/digitaldrywood/timetracker/internal/google"
)

type ClientMappings struct {
	Repos   map[string]RepoInfo   `json:"repos"`
	Clients map[string]ClientInfo `json:"clients"`
}

type RepoInfo struct {
	Client      string `json:"client"`
	Description string `json:"description"`
}

type ClientInfo struct {
	Active         bool    `json:"active"`
	SpreadsheetTab string `json:"spreadsheet_tab"`
	Rate           float64 `json:"rate"`
}

const mappingsFile = ".local/client_mappings.json"

func main() {
	var (
		list   = flag.Bool("list", false, "List all clients from spreadsheet")
		sync   = flag.Bool("sync", false, "Sync clients from spreadsheet tabs")
		map_   = flag.String("map", "", "Map a repo to a client (format: repo=client)")
		show   = flag.Bool("show", false, "Show current mappings")
		repo   = flag.String("repo", "", "Show or update mapping for specific repo")
		client = flag.String("client", "", "Client to map repo to")
	)
	flag.Parse()

	if *list || *sync {
		listOrSyncClients(*sync)
	} else if *map_ != "" {
		mapRepo(*map_)
	} else if *repo != "" && *client != "" {
		updateMapping(*repo, *client)
	} else if *show || *repo != "" {
		showMappings(*repo)
	} else {
		// Interactive mode - ask about unmapped repos from today's activity
		askAboutUnmappedRepos()
	}
}

func listOrSyncClients(sync bool) {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	auth, err := google.NewAuth(cfg.CredentialsPath, cfg.TokenPath, cfg.OAuthRedirectURL)
	if err != nil {
		log.Fatalf("Failed to create auth client: %v", err)
	}

	service, err := auth.GetSheetsService()
	if err != nil {
		log.Fatalf("Failed to get Sheets service: %v", err)
	}

	// Get spreadsheet to list tabs
	spreadsheet, err := service.Spreadsheets.Get(cfg.SpreadsheetID).Do()
	if err != nil {
		log.Fatalf("Failed to get spreadsheet: %v", err)
	}

	fmt.Println("📊 Spreadsheet Tabs (Clients):")
	fmt.Println("================================")
	
	mappings := loadMappings()
	
	for _, sheet := range spreadsheet.Sheets {
		tabName := sheet.Properties.Title
		isActive := !strings.HasSuffix(tabName, "- complete")
		isInvoices := tabName == "Invoices"
		
		if isInvoices {
			fmt.Printf("  💰 %s (invoices)\n", tabName)
			continue
		}
		
		status := "✅ Active"
		if !isActive {
			status = "⏸️  Complete"
		}
		
		fmt.Printf("  %s %s\n", status, tabName)
		
		if sync && !isInvoices {
			// Update client info in mappings
			clientName := strings.TrimSuffix(tabName, " - complete")
			if mappings.Clients == nil {
				mappings.Clients = make(map[string]ClientInfo)
			}
			mappings.Clients[clientName] = ClientInfo{
				Active:         isActive,
				SpreadsheetTab: tabName,
				Rate:           mappings.Clients[clientName].Rate, // Preserve rate
			}
		}
	}
	
	if sync {
		saveMappings(mappings)
		fmt.Println("\n✅ Client list synced to", mappingsFile)
	}
}

func mapRepo(mapping string) {
	parts := strings.Split(mapping, "=")
	if len(parts) != 2 {
		log.Fatalf("Invalid format. Use: repo=client")
	}
	
	updateMapping(parts[0], parts[1])
}

func updateMapping(repo, client string) {
	mappings := loadMappings()
	
	if mappings.Repos == nil {
		mappings.Repos = make(map[string]RepoInfo)
	}
	
	mappings.Repos[repo] = RepoInfo{
		Client:      client,
		Description: mappings.Repos[repo].Description,
	}
	
	saveMappings(mappings)
	fmt.Printf("✅ Mapped %s → %s\n", repo, client)
}

func showMappings(specificRepo string) {
	mappings := loadMappings()
	
	if specificRepo != "" {
		if info, exists := mappings.Repos[specificRepo]; exists {
			fmt.Printf("%s → %s\n", specificRepo, info.Client)
		} else {
			fmt.Printf("%s → (unmapped)\n", specificRepo)
		}
		return
	}
	
	fmt.Println("📊 Client Mappings")
	fmt.Println("==================")
	
	// Group repos by client
	byClient := make(map[string][]string)
	for repo, info := range mappings.Repos {
		byClient[info.Client] = append(byClient[info.Client], repo)
	}
	
	for client, repos := range byClient {
		clientInfo := mappings.Clients[client]
		status := "✅"
		if !clientInfo.Active {
			status = "⏸️"
		}
		fmt.Printf("\n%s %s", status, client)
		if clientInfo.Rate > 0 {
			fmt.Printf(" ($%.0f/hr)", clientInfo.Rate)
		}
		fmt.Println()
		for _, repo := range repos {
			fmt.Printf("  • %s\n", repo)
		}
	}
}

func askAboutUnmappedRepos() {
	// This would integrate with the tracker to find unmapped repos
	// For now, just show instructions
	fmt.Println("To map repositories to clients:")
	fmt.Println()
	fmt.Println("  clients -map repo=client")
	fmt.Println("  clients -repo REPO -client CLIENT")
	fmt.Println()
	fmt.Println("To sync clients from spreadsheet:")
	fmt.Println("  clients -sync")
	fmt.Println()
	fmt.Println("To show current mappings:")
	fmt.Println("  clients -show")
}

func loadMappings() ClientMappings {
	var mappings ClientMappings
	
	data, err := os.ReadFile(mappingsFile)
	if err != nil {
		return ClientMappings{
			Repos:   make(map[string]RepoInfo),
			Clients: make(map[string]ClientInfo),
		}
	}
	
	json.Unmarshal(data, &mappings)
	return mappings
}

func saveMappings(mappings ClientMappings) {
	data, err := json.MarshalIndent(mappings, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal mappings: %v", err)
	}
	
	// Ensure .local directory exists
	os.MkdirAll(filepath.Dir(mappingsFile), 0755)
	
	if err := os.WriteFile(mappingsFile, data, 0644); err != nil {
		log.Fatalf("Failed to save mappings: %v", err)
	}
}