package predictor

import (
	"fmt"
	"time"
	"tram-predictor/internal/gtfs"
)

type Result struct {
	SafeToLeave        bool     `json:"safeToLeave"`
	NextTramDeparture  string   `json:"nextTramDeparture"`
	UpcomingDepartures []string `json:"upcomingDepartures"`
	Message            string   `json:"message"`
}

func Predict(departures []gtfs.Departure, now time.Time) Result {
	fmt.Printf("Predicting for %d departures at %v\n", len(departures), now.Format("15:04:05"))
	
	var upcoming []gtfs.Departure

	for _, d := range departures {
		depTime, err := parseGTFSTime(d.Time, now)
		if err != nil {
			continue
		}

		diff := depTime.Sub(now).Minutes()
		if diff < 0 {
			continue // Already departed
		}

		upcoming = append(upcoming, d)
		if len(upcoming) >= 3 {
			break
		}
	}

	if len(upcoming) == 0 {
		return Result{
			SafeToLeave: false,
			Message:     "No more trams found for today.",
		}
	}

	var upcomingTimes []string
	for _, u := range upcoming {
		upcomingTimes = append(upcomingTimes, u.Time)
	}

	firstDepTime, _ := parseGTFSTime(upcoming[0].Time, now)
	firstDiff := firstDepTime.Sub(now).Minutes()

	res := Result{
		NextTramDeparture:  upcoming[0].Time,
		UpcomingDepartures: upcomingTimes,
	}

	if firstDiff >= 3 && firstDiff <= 6 {
		res.SafeToLeave = true
		res.Message = fmt.Sprintf("Safe to leave! Next tram in %.0f minutes.", firstDiff)
	} else {
		res.SafeToLeave = false
		if firstDiff < 3 {
			res.Message = fmt.Sprintf("Too late! Next tram is in %.0f minutes.", firstDiff)
		} else {
			res.Message = fmt.Sprintf("Too early! Next tram is in %.0f minutes.", firstDiff)
		}
	}

	return res
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
