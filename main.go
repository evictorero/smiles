package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"sync"
	"time"
)

const (
	DaysToQuery  = 5
	readFromFile = false

	departureDateStr       = "2022-09-10"
	returnDateStr          = "2022-09-20"
	originAirportCode      = "BUE"
	destinationAirportCode = "PUJ"
	mockResponseFilePath   = "response.json"
)

func main() {

	//var res *http.Response
	c := http.Client{}
	startingDepartureDate, err := time.Parse("2006-01-02", departureDateStr)
	startingReturningDate, err := time.Parse("2006-01-02", returnDateStr)
	if err != nil {
		log.Fatal("Error parsing starting date")
	}

	fmt.Printf("Departure starting date: %s\n", departureDateStr)
	fmt.Printf("Return starting date: %s\n", returnDateStr)
	fmt.Printf("From: %s", originAirportCode)
	fmt.Printf("To: %s\n", mockResponseFilePath)

	departuresCh := make(chan Result, DaysToQuery)
	returnsCh := make(chan Result, DaysToQuery)

	var wg sync.WaitGroup
	for i := 0; i < DaysToQuery; i++ {
		departureDate := startingDepartureDate.AddDate(0, 0, i)
		returnDate := startingReturningDate.AddDate(0, 0, i)

		wg.Add(2)
		go makeRequest(&wg, departuresCh, c, departureDate, originAirportCode, destinationAirportCode)
		// inverting airports and changing date to query returns
		go makeRequest(&wg, returnsCh, c, returnDate, destinationAirportCode, originAirportCode)
	}

	wg.Wait()
	close(departuresCh)
	close(returnsCh)

	var departureResults []Result
	var returnResults []Result

	for elem := range departuresCh {
		departureResults = append(departureResults, elem)
	}

	for elem := range returnsCh {
		returnResults = append(returnResults, elem)
	}

	sortResults(departureResults)
	sortResults(returnResults)

	fmt.Println("DEPARTURES")
	for _, r := range departureResults {
		printResult(r)
	}

	fmt.Println("RETURNS")
	for _, r := range returnResults {
		printResult(r)
	}

}

func sortResults(r []Result) {
	sort.Slice(r, func(i, j int) bool {
		return r[i].QueryDate.Before(r[j].QueryDate)
	})
}

func makeRequest(wg *sync.WaitGroup, ch chan<- Result, c http.Client, startingDate time.Time, originAirport string, destinationAirport string) {
	defer wg.Done()
	var body []byte
	var err error
	data := Data{}

	u := createURL(startingDate.Format("2006-01-02"), originAirport, destinationAirport) // Encode and assign back to the original query.
	req := createRequest(u)

	//fmt.Println("Making request with URL: ", req.URL.String())
	fmt.Printf("Making request %s - %s for day %s \n", originAirport, destinationAirport, startingDate.Format("2006-01-02"))

	// only for dev purposes
	if readFromFile {
		fmt.Println("Reading from file ", mockResponseFilePath)
		body, err = os.ReadFile(mockResponseFilePath)
		if err != nil {
			log.Fatal("error reading file")
		}
	} else {
		res, err := c.Do(req)
		if err != nil {
			log.Fatal("Error making request ", err)
		}

		body, err = ioutil.ReadAll(res.Body)
		if body == nil {
			log.Fatal("Empty result")
		}
	}

	if err := json.Unmarshal(body, &data); err != nil {
		log.Fatal("Error unmarshalling data ", err)
	}

	//printResult(data, startingDate)
	ch <- Result{Data: data, QueryDate: startingDate}
}

func printResult(result Result) {
	if result.Data.RequestedFlightSegmentList != nil {
		fmt.Printf("%s: %s - %s -> Best Price %d miles \n",
			result.Data.RequestedFlightSegmentList[0].Airports.DepartureAirports[0].Code,
			result.Data.RequestedFlightSegmentList[0].Airports.ArrivalAirports[0].Code,
			result.QueryDate.Format("2006-01-02"),
			result.Data.RequestedFlightSegmentList[0].BestPricing.Miles,
		)
	}
}

func createRequest(u url.URL) *http.Request {
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		log.Fatal("Error creating request ", err)
	}

	// headers
	req.Header.Add("x-api-key", "aJqPU7xNHl9qN3NVZnPaJ208aPo2Bh2p2ZV844tw")
	req.Header.Add("region", "ARGENTINA")
	req.Header.Add("origin", "https://www.smiles.com.ar")
	req.Header.Add("referer", "https://www.smiles.com.ar")
	req.Header.Add("channel", "web")
	req.Header.Add("authority", "api-air-flightsearch-prd.smiles.com.br")
	return req
}

func createURL(departureDate string, originAirport string, destinationAirport string) url.URL {
	u := url.URL{
		Scheme:   "https",
		Host:     "api-air-flightsearch-prd.smiles.com.br",
		RawQuery: "adults=1&cabinType=all&children=0&currencyCode=ARS&infants=0&isFlexibleDateChecked=false&tripType=2&forceCongener=false&r=ar",
		Path:     "/v1/airlines/search",
	}
	q := u.Query()
	q.Add("departureDate", departureDate)
	q.Add("originAirportCode", originAirport)
	q.Add("destinationAirportCode", destinationAirport)
	u.RawQuery = q.Encode()
	return u
}
