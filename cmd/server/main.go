package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/prashanta0234/vpsmyth/internal/api"
	"github.com/prashanta0234/vpsmyth/internal/db"
)

func main() {
	// Initialize database
	if err := db.InitDB("vpsmyth.db"); err != nil {
		log.Fatal(err)
	}

	// Register all routes
	mux := http.DefaultServeMux
	api.RegisterRoutes(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("VPSMyth server starting on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
