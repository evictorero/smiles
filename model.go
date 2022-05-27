package main

import "time"

type fare struct {
	fareType string `json:"type"`
}

type BestPricing struct {
	Miles      int    `json:"miles"`
	SourceFare string `json:"sourceFare"`
	Fare       fare   `json:"fare"`
}

type Segment struct {
	SegmentType string `json:"type"`
	//FlightList  []flight    `json:"flightList"`
	BestPricing BestPricing `json:"bestPricing"`
	Airports    Airports    `json:"airports"`
}

type DepartureAirport struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	City     string `json:"city"`
	Country  string `json:"country"`
	Timezone string `json:"timezone"`
}

type Airports struct {
	DepartureAirports []DepartureAirport `json:"departureAirportList"`
	ArrivalAirports   []DepartureAirport `json:"arrivalAirportList"`
}
type Data struct {
	RequestedFlightSegmentList []Segment `json:"requestedFlightSegmentList"`
}

type Result struct {
	Data      Data
	QueryDate time.Time
}
