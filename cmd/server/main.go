package main

import (
	"context"
	"encoding/json"
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
	stopID      = "665806" // Parent Rathaus Schönefeld
	hbfStopID   = "218016" // Hauptbahnhof (Steig A)
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

	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	http.HandleFunc("/api/v1/tram-status", func(w http.ResponseWriter, r *http.Request) {
		var departures []gtfs.Departure
		dateStr := time.Now().Format("20060102")
		err := store.GetDepartures(ctx, stopID, dateStr, &departures)
		if err != nil {
			fmt.Printf("Error getting departures from Redis: %v\n", err)
			// Return a valid JSON even on error to avoid frontend 500
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"safeToLeave": false, "message": "Data not ready yet. Please try again later."}`)
			return
		}

		result := predictor.Predict(departures, time.Now())
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
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
	fmt.Printf("Found Route ID: %s\n", routeID)

	// Leipzig, Rathaus Schönefeld platforms
	platforms := []string{"223358", "517083"}

	// Process for the next 7 days
	for i := 0; i < 7; i++ {
		date := time.Now().AddDate(0, 0, i)
		dateStr := date.Format("20060102")

		activeServices, err := parser.GetActiveServiceIDs(date)
		if err != nil {
			log.Printf("Failed to get active service IDs for %s: %v", dateStr, err)
			continue
		}

		tripIDs, err := parser.GetTripIDs(routeID, activeServices)
		if err != nil {
			log.Printf("Failed to get trip IDs for %s: %v", dateStr, err)
			continue
		}

		allDepartures, err := parser.GetDeparturesToHbf(platforms, hbfStopID, tripIDs)
		if err != nil {
			log.Printf("Failed to get departures for %s: %v", dateStr, err)
			continue
		}

		fmt.Printf("Storing %d departures for %s to Hbf in Redis...\n", len(allDepartures), dateStr)
		err = store.StoreDepartures(context.Background(), stopID, dateStr, allDepartures)
		if err != nil {
			log.Printf("Failed to store departures for %s in Redis: %v", dateStr, err)
		}
	}

	fmt.Println("Cleaning up temporary GTFS files...")
	os.RemoveAll(tmpDir)

	fmt.Println("GTFS update complete.")
}
