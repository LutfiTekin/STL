package gtfs

type Stop struct {
	ID   string
	Name string
}

type Route struct {
	ID        string
	ShortName string
	Type      string
}

type Trip struct {
	ID        string
	RouteID   string
	Headsign  string
	Direction string
}

type StopTime struct {
	TripID        string
	ArrivalTime   string
	DepartureTime string
	StopID        string
	StopSequence  int
}
