package model

import "time"

type Fare struct {
	FType     string `json:"type"`
	BaseMiles int    `json:"baseMiles"`
}

type Airline struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type flightDetail struct {
	Date    string  `json:"date"`
	Airport Airport `json:"airport"`
}

type Leg struct {
	Cabin     string       `json:"cabin"`
	Departure flightDetail `json:"departure"`
	Arrival   flightDetail `json:"arrival"`
}
type Flight struct {
	Cabin     string       `json:"cabin"`
	Stops     int          `json:"stops"`
	Departure flightDetail `json:"departure"`
	Arrival   flightDetail `json:"arrival"`
	Airline   Airline      `json:"airline"`
	LegList   []Leg        `json:"legList"`
	FareList  []Fare       `json:"fareList"`
}

type BestPricing struct {
	Miles      int    `json:"miles"`
	SourceFare string `json:"sourceFare"`
	Fare       Fare   `json:"fare"`
}

type Segment struct {
	SegmentType string      `json:"type"`
	FlightList  []Flight    `json:"flightList"`
	BestPricing BestPricing `json:"bestPricing"`
	Airports    Airports    `json:"airports"`
}

type Airport struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	City    string `json:"city"`
	Country string `json:"country"`
}

type Airports struct {
	DepartureAirports []Airport `json:"departureAirportList"`
	ArrivalAirports   []Airport `json:"arrivalAirportList"`
}
type Data struct {
	RequestedFlightSegmentList []Segment `json:"requestedFlightSegmentList"`
}

type Result struct {
	Data      Data
	QueryDate time.Time
}
