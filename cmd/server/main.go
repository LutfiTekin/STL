package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
	"tram-predictor/internal/gtfs"
	"tram-predictor/internal/predictor"
	"tram-predictor/internal/storage"
)

const (
	stopID      = "665806" // Rathaus Schönefeld
	gtfsURL     = "https://download.gtfs.de/germany/free/latest.zip"
	redisAddr   = "localhost:6379"
	tmpDir      = "./tmp_gtfs"
)

func main() {
	fmt.Println("Tram Predictor starting...")

	redisHost := os.Getenv("REDIS_ADDR")
	if redisHost == "" {
		redisHost = redisAddr
	}

	store := storage.NewStorage(redisHost)
	ctx := context.Background()

	// Start weekly fetcher
	go runWeeklyFetcher(store)

	// Serve static files
	fs := http.FileServer(http.Dir("web"))
	http.Handle("/", fs)

	http.HandleFunc("/api/v1/tram-status", func(w http.ResponseWriter, r *http.Request) {
		var departures []gtfs.Departure
		err := store.GetDepartures(ctx, stopID, &departures)
		if err != nil {
			http.Error(w, "Failed to get departures: "+err.Error(), http.StatusInternalServerError)
			return
		}

		result := predictor.Predict(departures, time.Now())
		
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"safeToLeave": %t, "nextTramDeparture": "%s", "message": "%s"}`, 
			result.SafeToLeave, result.NextTramDeparture, result.Message)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "9030"
	}

	fmt.Printf("Server listening on port %s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func runWeeklyFetcher(store *storage.Storage) {
	ticker := time.NewTicker(7 * 24 * time.Hour)
	defer ticker.Stop()

	// Initial fetch
	fetchAndStore(store)

	for range ticker.C {
		fetchAndStore(store)
	}
}

func fetchAndStore(store *storage.Storage) {
	fmt.Println("Starting GTFS fetch and store...")
	
	err := os.MkdirAll(tmpDir, os.ModePerm)
	if err != nil {
		log.Printf("Failed to create tmp dir: %v", err)
		return
	}

	zipPath := tmpDir + "/latest.zip"
	fmt.Println("Downloading GTFS feed...")
	err = gtfs.DownloadFeed(gtfsURL, zipPath)
	if err != nil {
		log.Printf("Failed to download feed: %v", err)
		return
	}

	fmt.Println("Extracting GTFS feed...")
	extractDir := tmpDir + "/extracted"
	err = gtfs.ExtractFeed(zipPath, extractDir)
	if err != nil {
		log.Printf("Failed to extract feed: %v", err)
		return
	}

	fmt.Println("Parsing GTFS data...")
	parser := &gtfs.Parser{Dir: extractDir}
	routeID, err := parser.GetRouteID()
	if err != nil || routeID == "" {
		log.Printf("Failed to find route ID for STR 1: %v", err)
		return
	}

	tripHeadsigns, err := parser.GetTripIDs(routeID)
	if err != nil {
		log.Printf("Failed to get trip IDs: %v", err)
		return
	}

	departures, err := parser.GetDepartures(stopID, tripHeadsigns)
	if err != nil {
		log.Printf("Failed to get departures: %v", err)
		return
	}

	fmt.Printf("Storing %d departures in Redis...\n", len(departures))
	err = store.StoreDepartures(context.Background(), stopID, departures)
	if err != nil {
		log.Printf("Failed to store departures in Redis: %v", err)
	}

	fmt.Println("GTFS update complete.")
}
