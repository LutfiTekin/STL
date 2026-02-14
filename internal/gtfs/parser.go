package gtfs

import (
	"encoding/csv"
	"io"
	"os"
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

	// Map headers to indices
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

		if record[headerMap["route_short_name"]] == "1" && record[headerMap["route_type"]] == "0" {
			return record[headerMap["route_id"]], nil
		}
	}
	return "", nil
}

func (p *Parser) GetTripIDs(routeID string) (map[string]string, error) {
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

	tripHeadsigns := make(map[string]string)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if record[headerMap["route_id"]] == routeID {
			tripHeadsigns[record[headerMap["trip_id"]]] = record[headerMap["trip_headsign"]]
		}
	}
	return tripHeadsigns, nil
}

type Departure struct {
	Time      string
	Headsign  string
	TripID    string
}

func (p *Parser) GetDepartures(stopID string, tripHeadsigns map[string]string) ([]Departure, error) {
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

	var departures []Departure
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		tripID := record[headerMap["trip_id"]]
		if headsign, ok := tripHeadsigns[tripID]; ok {
			if record[headerMap["stop_id"]] == stopID {
				departures = append(departures, Departure{
					Time:     record[headerMap["departure_time"]],
					Headsign: headsign,
					TripID:   tripID,
				})
			}
		}
	}
	return departures, nil
}
