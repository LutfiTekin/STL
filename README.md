# STL (Safe To Leave) - Leipzig Tram Predictor

STL is a lightweight web application designed to answer one simple question: **"Is it safe to leave the house now to catch the STR 1 tram towards Leipzig Hauptbahnhof?"**

## 🚀 How it Works

The application follows an automated lifecycle to ensure you have the most accurate static schedule data available.

### 1. Data Acquisition (Weekly)
- Every week, a background Go routine (Ticker) downloads the latest Germany-wide GTFS data from `download.gtfs.de`.
- The `~1GB` zip file is extracted and processed locally on the server.

### 2. Schedule Parsing & Filtering
The parser performs a multi-stage filter to pinpoint relevant trams:
- **Agency Filter:** Specifically targets **LVB** (Leipziger Verkehrsbetriebe).
- **Route Filter:** Extracts only **STR 1** (Tram Line 1).
- **Calendar Filter:** Identifies active services for the **current date** (handling weekdays, weekends, and specific holiday exceptions).
- **Direction Filter:** Only includes trips that stop at **Rathaus Schönefeld** and subsequently stop at **Leipzig Hauptbahnhof** (detecting direction via stop sequence).

### 3. Smart Storage
- Processed departures for the next **7 days** are stored in **Redis**.
- Data is indexed by date to ensure that requests at midnight or across different days always receive the correct schedule.

### 4. Prediction Logic
When you open the web interface, the "Safe To Leave" engine evaluates the next 3 trams against your personal criteria:
- **3 Minutes Travel Time:** It takes ~3 minutes to walk to the stop.
- **3 Minutes Max Wait:** You don't want to wait more than 3 minutes at the stop.

| Time until Departure | Status | Message |
| :--- | :--- | :--- |
| **< 3 mins** | 🔴 NO | Too late! You won't make it. |
| **3 - 6 mins** | 🟢 YES | Safe to leave! Catch it perfectly. |
| **> 6 mins** | 🔴 NO | Too early! You'll be waiting too long. |

### 5. Frontend
- A simple, responsive dashboard served on port `9030`.
- Automatically refreshes every minute to keep the prediction current.

## 🛠 Tech Stack
- **Backend:** Go (Golang)
- **Database:** Redis (In-memory storage)
- **Deployment:** Docker & Docker Compose
- **Data Source:** GTFS (Static Schedule Data)

## 📦 Deployment
Run the following on the server:
```bash
docker compose up -d --build
```
Access at: `http://localhost:9030`
