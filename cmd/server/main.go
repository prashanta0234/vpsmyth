package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/prashanta0234/vpsmyth/internal/api"
	"github.com/prashanta0234/vpsmyth/internal/auth"
	"github.com/prashanta0234/vpsmyth/internal/db"
	"bufio"
	"strings"
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
		username, _ = reader.ReadString('\n')
		username = strings.TrimSpace(username)
		if username != "" {
			break
		}
		fmt.Println("Username cannot be empty.")
	}

	var password string
	for {
		fmt.Print("Enter Admin Password: ")
		password, _ = reader.ReadString('\n')
		password = strings.TrimSpace(password)
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

	// Register all routes
	mux := http.DefaultServeMux
	api.RegisterRoutes(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("VPSMyth server starting on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, api.AuthMiddleware(mux)))
}
