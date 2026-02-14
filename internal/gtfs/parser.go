package gtfs

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"sort"
	"time"
)

type Parser struct {
	Dir string
}

func (p *Parser) GetRouteID() (string, error) {
	f, err := os.Open(p.Dir + "/routes.txt")
	if err != nil {
		return "", err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	header, err := reader.Read()
	if err != nil {
		return "", err
	}

	headerMap := make(map[string]int)
	for i, h := range header {
		headerMap[h] = i
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		if record[headerMap["route_short_name"]] == "1" && record[headerMap["route_type"]] == "0" && record[headerMap["agency_id"]] == "287" {
			return record[headerMap["route_id"]], nil
		}
	}
	return "", nil
}

func (p *Parser) GetActiveServiceIDs(date time.Time) (map[string]bool, error) {
	activeServices := make(map[string]bool)
	dateStr := date.Format("20060102")
	weekday := date.Weekday() // 0 is Sunday

	// 1. Check calendar.txt
	f, err := os.Open(p.Dir + "/calendar.txt")
	if err == nil {
		defer f.Close()
		reader := csv.NewReader(f)
		header, err := reader.Read()
		if err == nil {
			headerMap := make(map[string]int)
			for i, h := range header {
				headerMap[h] = i
			}

			for {
				record, err := reader.Read()
				if err == io.EOF {
					break
				}
				if err != nil {
					break
				}

				start := record[headerMap["start_date"]]
				end := record[headerMap["end_date"]]

				if dateStr >= start && dateStr <= end {
					isActive := false
					switch weekday {
					case time.Monday:
						isActive = record[headerMap["monday"]] == "1"
					case time.Tuesday:
						isActive = record[headerMap["tuesday"]] == "1"
					case time.Wednesday:
						isActive = record[headerMap["wednesday"]] == "1"
					case time.Thursday:
						isActive = record[headerMap["thursday"]] == "1"
					case time.Friday:
						isActive = record[headerMap["friday"]] == "1"
					case time.Saturday:
						isActive = record[headerMap["saturday"]] == "1"
					case time.Sunday:
						isActive = record[headerMap["sunday"]] == "1"
					}
					if isActive {
						activeServices[record[headerMap["service_id"]]] = true
					}
				}
			}
		}
	}

	// 2. Check calendar_dates.txt (exceptions)
	f2, err := os.Open(p.Dir + "/calendar_dates.txt")
	if err == nil {
		defer f2.Close()
		reader := csv.NewReader(f2)
		header, err := reader.Read()
		if err == nil {
			headerMap := make(map[string]int)
			for i, h := range header {
				headerMap[h] = i
			}

			for {
				record, err := reader.Read()
				if err == io.EOF {
					break
				}
				if err != nil {
					break
				}

				if record[headerMap["date"]] == dateStr {
					if record[headerMap["exception_type"]] == "1" {
						activeServices[record[headerMap["service_id"]]] = true
					} else if record[headerMap["exception_type"]] == "2" {
						delete(activeServices, record[headerMap["service_id"]])
					}
				}
			}
		}
	}

	return activeServices, nil
}

func (p *Parser) GetTripIDs(routeID string, activeServices map[string]bool) (map[string]bool, error) {
	f, err := os.Open(p.Dir + "/trips.txt")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	header, err := reader.Read()
	if err != nil {
		return nil, err
	}

	headerMap := make(map[string]int)
	for i, h := range header {
		headerMap[h] = i
	}

	tripIDs := make(map[string]bool)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if record[headerMap["route_id"]] == routeID {
			serviceID := record[headerMap["service_id"]]
			if activeServices[serviceID] {
				tripIDs[record[headerMap["trip_id"]]] = true
			}
		}
	}
	return tripIDs, nil
}

type Departure struct {
	Time    string `json:"time"`
	TripID  string `json:"tripId"`
	StopSeq int    `json:"stopSeq"`
}

func (p *Parser) GetDeparturesToHbf(rsPlatforms []string, hbfStopID string, tripIDs map[string]bool) ([]Departure, error) {
	f, err := os.Open(p.Dir + "/stop_times.txt")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	header, err := reader.Read()
	if err != nil {
		return nil, err
	}

	headerMap := make(map[string]int)
	for i, h := range header {
		headerMap[h] = i
	}

	type tripInfo struct {
		RSSeq  int
		RSTime string
		HbfSeq int
	}
	trips := make(map[string]*tripInfo)

	platformMap := make(map[string]bool)
	for _, p := range rsPlatforms {
		platformMap[p] = true
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		tripID := record[headerMap["trip_id"]]
		if _, ok := tripIDs[tripID]; !ok {
			continue
		}

		if _, ok := trips[tripID]; !ok {
			trips[tripID] = &tripInfo{RSSeq: -1, HbfSeq: -1}
		}

		stopID := record[headerMap["stop_id"]]
		var seq int
		fmt.Sscanf(record[headerMap["stop_sequence"]], "%d", &seq)

		if platformMap[stopID] {
			trips[tripID].RSSeq = seq
			trips[tripID].RSTime = record[headerMap["departure_time"]]
		} else if stopID == hbfStopID {
			trips[tripID].HbfSeq = seq
		}
	}

	var results []Departure
	for tripID, info := range trips {
		if info.RSSeq != -1 && info.HbfSeq != -1 && info.HbfSeq > info.RSSeq {
			results = append(results, Departure{
				Time:    info.RSTime,
				TripID:  tripID,
				StopSeq: info.RSSeq,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Time < results[j].Time
	})

	return results, nil
}
