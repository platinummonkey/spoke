package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/platinummonkey/spoke/pkg/api"
	"github.com/platinummonkey/spoke/pkg/storage"
)

func main() {
	// Parse command line flags
	port := flag.String("port", "8080", "Port to listen on")
	storageDir := flag.String("storage-dir", filepath.Join(os.TempDir(), "spoke"), "Directory to store protobuf files")
	flag.Parse()

	// Create storage directory if it doesn't exist
	if err := os.MkdirAll(*storageDir, 0755); err != nil {
		log.Fatalf("Failed to create storage directory: %v", err)
	}

	// Initialize storage
	store, err := storage.NewFileSystemStorage(*storageDir)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	log.Printf("Storage initialized in %s", *storageDir)

	// Create and start server
	server := api.NewServer(store)
	log.Printf("Starting Spoke Schema Registry server on port %s...", *port)
	if err := http.ListenAndServe(":"+*port, server); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
} 