package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/prashanta0234/vpsmyth/internal/api"
	"github.com/prashanta0234/vpsmyth/internal/auth"
	cronrunner "github.com/prashanta0234/vpsmyth/internal/cron"
	"github.com/prashanta0234/vpsmyth/internal/db"
	"github.com/prashanta0234/vpsmyth/internal/monitoring"
)

func setupWizard() {
	hasUsers, err := db.HasUsers()
	if err != nil {
		log.Fatalf("Failed to check for users: %v", err)
	}

	if hasUsers {
		return
	}

	// Check for environment variables for non-interactive setup
	envUser := os.Getenv("ADMIN_USERNAME")
	envPass := os.Getenv("ADMIN_PASSWORD")

	if envUser != "" && envPass != "" {
		fmt.Println("Creating admin user from environment variables...")
		if len(envPass) < 8 {
			log.Fatal("Environment variable ADMIN_PASSWORD must be at least 8 characters.")
		}
		hash, err := auth.HashPassword(envPass)
		if err != nil {
			log.Fatalf("Failed to hash password: %v", err)
		}
		if err := db.CreateUser(envUser, hash); err != nil {
			log.Fatalf("Failed to create user: %v", err)
		}
		fmt.Println("Admin account created successfully from environment variables.")
		return
	}

	fmt.Println("\n" + strings.Repeat("=", 40))
	fmt.Println("   VPSMyth First-Run Setup Wizard   ")
	fmt.Println(strings.Repeat("=", 40))
	fmt.Println("No users found. Please create an admin account.")

	reader := bufio.NewReader(os.Stdin)

	var username string
	for {
		fmt.Print("Enter Admin Username: ")
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal("No interactive terminal detected. Set ADMIN_USERNAME and ADMIN_PASSWORD environment variables to create the admin account non-interactively.")
		}
		username = strings.TrimSpace(line)
		if username != "" {
			break
		}
		fmt.Println("Username cannot be empty.")
	}

	var password string
	for {
		fmt.Print("Enter Admin Password: ")
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal("No interactive terminal detected. Set ADMIN_USERNAME and ADMIN_PASSWORD environment variables to create the admin account non-interactively.")
		}
		password = strings.TrimSpace(line)
		if len(password) >= 8 {
			break
		}
		fmt.Println("Password must be at least 8 characters.")
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	if err := db.CreateUser(username, hash); err != nil {
		log.Fatalf("Failed to create user: %v", err)
	}

	fmt.Println("\nAdmin account created successfully!")
	fmt.Println(strings.Repeat("=", 40) + "\n")
}

func main() {
	// Initialize database
	if err := db.InitDB("vpsmyth.db"); err != nil {
		log.Fatal(err)
	}

	// Run setup wizard
	setupWizard()

	// Start background monitoring collector
	monitoring.Start()

	// Start cron scheduler
	cronrunner.Start()

	// Register all routes
	mux := http.DefaultServeMux
	api.RegisterRoutes(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "2026"
	}

	addr := "127.0.0.1:" + port

	panelPath := os.Getenv("PANEL_PATH")
	if panelPath != "" {
		fmt.Printf("VPSMyth server starting — Login: http://YOUR_SERVER_IP/%s/login.html\n", panelPath)
	} else {
		fmt.Printf("VPSMyth server starting on http://%s\n", addr)
	}

	log.Fatal(http.ListenAndServe(addr, api.IPWhitelistMiddleware(api.AuthMiddleware(mux))))
}
