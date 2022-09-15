package model

import (
	"encoding/json"
	"time"
)

type Fare struct {
	UId   string `json:"uid"`
	FType string `json:"type"`
	Miles int    `json:"miles"`
}

type Airline struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type FlightDetail struct {
	Date    time.Time `json:"date"`
	Airport Airport   `json:"airport"`
}

type Leg struct {
	Cabin     string       `json:"cabin"`
	Departure FlightDetail `json:"departure"`
	Arrival   FlightDetail `json:"arrival"`
}
type Flight struct {
	UId       string       `json:"uid"`
	Cabin     string       `json:"cabin"`
	Stops     int          `json:"stops"`
	Departure FlightDetail `json:"departure"`
	Arrival   FlightDetail `json:"arrival"`
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
type Totals struct {
	Total     Total `json:"total"`
	TotalFare Total `json:"totalFare"`
}

type Total struct {
	Miles int     `json:"miles"`
	Money float32 `json:"money"`
}

type BoardingTax struct {
	Totals Totals `json:"totals"`
}

// needed because the date expected has the format "2006-01-02T15:04:05"
func (f *FlightDetail) UnmarshalJSON(p []byte) error {
	var aux struct {
		Date    string  `json:"date"`
		Airport Airport `json:"airport"`
	}

	err := json.Unmarshal(p, &aux)
	if err != nil {
		return err
	}

	t, err := time.Parse("2006-01-02T15:04:05", aux.Date)
	if err != nil {
		return err
	}

	f.Date = t
	f.Airport = aux.Airport

	return nil
}
