package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"smiles/data/model"
	"sort"
	"sync"
	"time"
)

// input parameters
const (
	departureDateStr       = "2023-01-15" // primer día para la ida
	returnDateStr          = "2023-01-20" // primer día para la vuelta
	originAirportCode      = "EZE"        // aeropuerto de origen
	destinationAirportCode = "SAO"        // aeropuerto de destino
	daysToQuery            = 1            // días corridos para buscar ida y vuelta
)

// only used for dev
const (
	readFromFile         = false
	mockResponseFilePath = "data/response.json"
)

func main() {

	start := time.Now()
	c := http.Client{}

	startingDepartureDate, err := time.Parse("2006-01-02", departureDateStr)
	startingReturningDate, err := time.Parse("2006-01-02", returnDateStr)
	if err != nil {
		log.Fatal("Error parsing starting date")
	}

	fmt.Printf("Primer día de búsqueda para la ida: %s\n", departureDateStr)
	fmt.Printf("Primer día de búsqueda para la vuelta: %s\n", returnDateStr)
	fmt.Printf("Desde: %s\n", originAirportCode)
	fmt.Printf("Hasta: %s\n", destinationAirportCode)

	departuresCh := make(chan model.Result, daysToQuery)
	returnsCh := make(chan model.Result, daysToQuery)

	var wg sync.WaitGroup
	for i := 0; i < daysToQuery; i++ {
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

	elapsed := time.Since(start)
	fmt.Printf("Las consultas tomaron %s\n", elapsed)

	var departureResults []model.Result
	var returnResults []model.Result

	for elem := range departuresCh {
		departureResults = append(departureResults, elem)
	}

	for elem := range returnsCh {
		returnResults = append(returnResults, elem)
	}

	sortResults(departureResults)
	sortResults(returnResults)

	fmt.Println("IDAS")
	for _, r := range departureResults {
		printResult(r)
	}

	fmt.Println("VUELTAS")
	for _, r := range returnResults {
		printResult(r)
	}

	fmt.Println("El vuelo de ida más barato es: ")
	processResults(departureResults)

	fmt.Println("El vuelo de vuelta más barato es: ")
	processResults(returnResults)
}

func sortResults(r []model.Result) {
	sort.Slice(r, func(i, j int) bool {
		return r[i].QueryDate.Before(r[j].QueryDate)
	})
}

func makeRequest(wg *sync.WaitGroup, ch chan<- model.Result, c http.Client, startingDate time.Time, originAirport string, destinationAirport string) {
	defer wg.Done()
	var body []byte
	var err error
	data := model.Data{}

	u := createURL(startingDate.Format("2006-01-02"), originAirport, destinationAirport) // Encode and assign back to the original query.
	req := createRequest(u)

	fmt.Println("Making request with URL: ", req.URL.String())
	fmt.Printf("Consultando %s - %s para el día %s \n", originAirport, destinationAirport, startingDate.Format("2006-01-02"))

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

	ch <- model.Result{Data: data, QueryDate: startingDate}
}

func printResult(result model.Result) {
	if result.Data.RequestedFlightSegmentList != nil && len(result.Data.RequestedFlightSegmentList[0].FlightList) > 0 {
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
		RawQuery: "adults=1&cabinType=all&children=0&currencyCode=ARS&infants=0&isFlexibleDateChecked=false&tripType=2&forceCongener=true&r=ar",
		Path:     "/v1/airlines/search",
	}
	q := u.Query()
	q.Add("departureDate", departureDate)
	q.Add("originAirportCode", originAirport)
	q.Add("destinationAirportCode", destinationAirport)
	u.RawQuery = q.Encode()
	return u
}

func processResults(r []model.Result) {
	// using the first flight as cheapest default
	var cheapestFlight model.Flight
	cheapestFare := 9_999_999_999

	// loop through all results
	for _, v := range r {
		// loop through all flights by day
		for _, f := range v.Data.RequestedFlightSegmentList[0].FlightList {
			smilesClubFare := getSmilesClubFare(&f)
			if cheapestFare > smilesClubFare {
				cheapestFlight = f
				cheapestFare = smilesClubFare
			}
		}
	}

	if cheapestFare != 9_999_999_999 {
		fmt.Printf("%s, %s - %s, %s, %s, %d escalas, %d millas\n",
			cheapestFlight.Departure.Date.Format("2006-01-02"),
			cheapestFlight.Departure.Airport.Code,
			cheapestFlight.Arrival.Airport.Code,
			cheapestFlight.Cabin,
			cheapestFlight.Airline.Name,
			cheapestFlight.Stops,
			cheapestFare,
		)
	}
}

func getSmilesClubFare(f *model.Flight) int {
	for _, v := range f.FareList {
		if v.FType == "SMILES_CLUB" {
			return v.Miles
		}
	}
	fmt.Println("WARN: SMILES_CLUB fare not fund")
	// for the sake of simplicity returning ridiculous default big number when fare not found
	return 9_999_999_999
}
