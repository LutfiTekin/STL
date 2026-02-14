# Tram Departure Predictor for Leipzig (STL)

## 1. Project Goal

To provide users with a simple web interface to answer: "Is it safe to leave the house now and still catch the tram?" for `STR1` towards "Hauptbahnhof" from "Rathaus Schönefeld, Leipzig".

## 2. Architecture Overview

A consolidated Go backend serves both the JSON API and the static web frontend. The system runs on the `quiet` server using Docker.

```mermaid
graph TD
    User --> App[Go Web App (on quiet)]
    App -- API Call --> App
    App -- Reads/Writes --> Redis[Redis (on quiet)]
    App -- Weekly Download --> GTFS_Source[GTFS Data Source (download.gtfs.de)]
    GoTicker(Go Ticker) --> App
```

## 3. Components

### 3.1. Go Backend (`quiet`)
- **GTFS Module:** Weekly download and parsing of German GTFS data.
- **Predictor Module:** Logic based on:
    - 3 minutes travel time to stop.
    - 3 minutes maximum wait time.
- **API:** `/api/v1/tram-status` returns JSON status.
- **Web Server:** Serves `index.html` at `/`.

### 3.2. Data Storage (`quiet`)
- **Redis:** Stores parsed departures for the specific stop and route.

## 4. Deployment

The application is containerized using `docker-compose`.

### Steps to deploy on `quiet`:
1. Copy the project files to `quiet`.
2. Run `docker-compose up -d --build`.

## 5. Safe to Leave Criteria
- **Departure in < 3m:** NO (Not enough time to reach stop).
- **Departure in 3m - 6m:** YES (Safe to leave).
- **Departure in > 6m:** NO (Waiting too long is futile).

## 6. Access
The application will be accessible at `http://localhost:9030`.
