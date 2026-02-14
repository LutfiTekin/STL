package predictor

import (
	"fmt"
	"time"
	"tram-predictor/internal/gtfs"
)

type Result struct {
	SafeToLeave        bool   `json:"safeToLeave"`
	NextTramDeparture  string `json:"nextTramDeparture"`
	Message            string `json:"message"`
}

func Predict(departures []gtfs.Departure, now time.Time) Result {
	for _, d := range departures {
		// Filter for Hauptbahnhof direction
		// Note: We might need to be more specific with the headsign check
		if d.Headsign != "Lausen" { // Based on earlier conversation, Lausen seems to be the direction
			continue
		}

		depTime, err := parseGTFSTime(d.Time, now)
		if err != nil {
			continue
		}

		diff := depTime.Sub(now).Minutes()

		if diff >= 3 && diff <= 6 {
			return Result{
				SafeToLeave:       true,
				NextTramDeparture: d.Time,
				Message:           fmt.Sprintf("Safe to leave! Next tram to Hbf in %.0f minutes.", diff),
			}
		}
	}

	return Result{
		SafeToLeave: false,
		Message:     "No suitable tram found soon. Either too late or too early.",
	}
}

func parseGTFSTime(gtfsTime string, now time.Time) (time.Time, error) {
	var h, m, s int
	fmt.Sscanf(gtfsTime, "%d:%d:%d", &h, &m, &s)
	
	// Handle cases where time is > 24h (GTFS specific)
	dayOffset := 0
	if h >= 24 {
		dayOffset = h / 24
		h = h % 24
	}

	t := time.Date(now.Year(), now.Month(), now.Day(), h, m, s, 0, now.Location())
	if dayOffset > 0 {
		t = t.AddDate(0, 0, dayOffset)
	}
	
	return t, nil
}
