package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"smiles/model"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/schollz/progressbar/v3"
)

// input parameters
var (
	departureDateStr       = "2022-09-10" // primer día para la ida
	returnDateStr          = "2022-09-20" // primer día para la vuelta
	originAirportCode      = "EZE"        // aeropuerto de origen
	destinationAirportCode = "PUJ"        // aeropuerto de destino
	daysToQuery            = 1            // días corridos para buscar ida y vuelta
)

const (
	// only used for dev
	readFromFile            = false
	useCommandLineArguments = true
	mockResponseFilePath    = "data/response.json"

	dateLayout        = "2006-01-02"
	bigMaxMilesNumber = 9_999_999
)

func main() {

	if useCommandLineArguments {
		if len(os.Args) != 6 {
			fmt.Println("Forma de Uso: Origen Destino Fecha Ida Fecha Vuelta Cantidad de días a consultar")
			fmt.Println("Ejemplo: smiles BUE PUJ 2023-01-10 2023-01-20 5")
			os.Exit(1)
		}

		validateParameters()
	}

	c := http.Client{}

	startingDepartureDate, err := time.Parse(dateLayout, departureDateStr)
	startingReturningDate, err := time.Parse(dateLayout, returnDateStr)
	if err != nil {
		log.Fatal("Error parsing starting date")
	}

	fmt.Printf("Primer día de búsqueda para la ida: %s\n", departureDateStr)
	fmt.Printf("Primer día de búsqueda para la vuelta: %s\n", returnDateStr)
	fmt.Printf("Desde: %s\n", originAirportCode)
	fmt.Printf("Hasta: %s\n", destinationAirportCode)

	departuresCh := make(chan model.Result, daysToQuery)
	returnsCh := make(chan model.Result, daysToQuery)

	bar := progressbar.NewOptions(daysToQuery*2,
		progressbar.OptionSetDescription("Consultando vuelos en las fechas y tramos seleccionados.."),
		progressbar.OptionSetWidth(40),
		progressbar.OptionSetRenderBlankState(true),
	)

	start := time.Now()
	var wg sync.WaitGroup

	for i := 0; i < daysToQuery; i++ {
		departureDate := startingDepartureDate.AddDate(0, 0, i)
		returnDate := startingReturningDate.AddDate(0, 0, i)

		wg.Add(2)
		go makeRequest(&wg, departuresCh, &c, departureDate, originAirportCode, destinationAirportCode, bar)
		// inverting airports and changing date to query returns
		go makeRequest(&wg, returnsCh, &c, returnDate, destinationAirportCode, originAirportCode, bar)
	}

	wg.Wait()
	close(departuresCh)
	close(returnsCh)

	elapsed := time.Since(start).Round(time.Second).String()
	fmt.Printf("\nLas consultas tomaron %s\n", elapsed)

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

	fmt.Println("VUELOS DE IDA")
	processResults(&c, departureResults)

	fmt.Println("VUELOS DE VUELTA")
	processResults(&c, returnResults)
}

func sortResults(r []model.Result) {
	sort.Slice(r, func(i, j int) bool {
		return r[i].QueryDate.Before(r[j].QueryDate)
	})
}

func makeRequest(wg *sync.WaitGroup, ch chan<- model.Result, c *http.Client, startingDate time.Time, originAirport string, destinationAirport string, bar *progressbar.ProgressBar) {
	defer wg.Done()
	defer bar.Add(1)

	var body []byte
	var err error
	data := model.Data{}

	u := createURL(startingDate.Format(dateLayout), originAirport, destinationAirport) // Encode and assign back to the original query.
	req := createRequest(u, "api-air-flightsearch-prd.smiles.com.br")

	//fmt.Println("Making request with URL: ", req.URL.String())
	//fmt.Printf("Consultando %s - %s para el día %s \n", originAirport, destinationAirport, startingDate.Format(dateLayout))

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

func createRequest(u url.URL, authority string) *http.Request {
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
	req.Header.Add("authority", authority)
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

func createTaxURL(departureFlight *model.Flight, departureFare *model.Fare) url.URL {
	u := url.URL{
		Scheme:   "https",
		Host:     "api-airlines-boarding-tax-prd.smiles.com.br",
		RawQuery: "adults=1&children=0&infants=0&highlightText=SMILES_CLUB",
		Path:     "/v1/airlines/flight/boardingtax",
	}
	q := u.Query()
	q.Add("type", "SEGMENT_1")
	q.Add("uid", departureFlight.UId)
	q.Add("fareuid", departureFare.UId)
	u.RawQuery = q.Encode()
	return u
}

func getSmilesClubFare(f *model.Flight) *model.Fare {
	for i, v := range f.FareList {
		if v.FType == "SMILES_CLUB" {
			return &f.FareList[i]
		}
	}
	fmt.Println("WARN: SMILES_CLUB fare not fund")
	// for the sake of simplicity returning ridiculous default big number when fare not found
	return &model.Fare{Miles: bigMaxMilesNumber}
}

func validateParameters() {
	originAirportCode = os.Args[1]
	if len(originAirportCode) != 3 {
		fmt.Fprintf(os.Stderr, "Error: El aeropuerto de origen %s no es válido\n", originAirportCode)
		os.Exit(1)
	}

	destinationAirportCode = os.Args[2]
	if len(destinationAirportCode) != 3 {
		fmt.Fprintf(os.Stderr, "Error: El aeropuerto de destino %s no es válido\n", destinationAirportCode)
		os.Exit(1)
	}

	departureDateStr = os.Args[3]
	_, err := time.Parse(dateLayout, departureDateStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: La fecha de salida %s no es válida. %v \n", departureDateStr, err)
		os.Exit(1)
	}

	returnDateStr = os.Args[4]
	_, err = time.Parse(dateLayout, returnDateStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: La fecha de regreso %s no es válida. %v \n", returnDateStr, err)
		os.Exit(1)
	}

	v, err := strconv.ParseInt(os.Args[5], 10, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: La cantidad de días %d no es válida. %v \n", v, err)
		os.Exit(1)
	}

	if v > 10 {
		fmt.Fprintf(os.Stderr, "Error: La cantidad de días no puede ser mayor a 10 \n")
		os.Exit(1)
	}
	daysToQuery = int(v)
}

func processResults(c *http.Client, r []model.Result) {
	// using the first flight as cheapest default
	var cheapestFlight *model.Flight
	cheapestFare := &model.Fare{
		Miles: bigMaxMilesNumber,
	}

	// loop through all results
	for _, v := range r {
		var cheapestFlightDay *model.Flight
		cheapestFareDay := &model.Fare{
			Miles: bigMaxMilesNumber,
		}

		// loop through all flights by day
		for _, f := range v.Data.RequestedFlightSegmentList[0].FlightList {
			smilesClubFare := getSmilesClubFare(&f)
			if cheapestFareDay.Miles > smilesClubFare.Miles {
				cheapestFlightDay = &f
				cheapestFareDay = smilesClubFare
			}
		}

		if cheapestFare.Miles > cheapestFareDay.Miles {
			cheapestFlight = cheapestFlightDay
			cheapestFare = cheapestFareDay
		}

		if cheapestFareDay.Miles != bigMaxMilesNumber {
			fmt.Printf("Vuelo más barato del día %s: %s - %s, %s, %s, %d escalas, %d millas\n",
				cheapestFlightDay.Departure.Date.Format(dateLayout),
				cheapestFlightDay.Departure.Airport.Code,
				cheapestFlightDay.Arrival.Airport.Code,
				cheapestFlightDay.Cabin,
				cheapestFlightDay.Airline.Name,
				cheapestFlightDay.Stops,
				cheapestFareDay.Miles,
			)
		}
	}

	fmt.Println()
	if cheapestFare.Miles != bigMaxMilesNumber {
		boardingTax := getTaxForFlight(c, cheapestFlight, cheapestFare)

		fmt.Printf("Vuelo más barato en estas fechas: %s, %s - %s, %s, %s, %d escalas, %d millas, %f de Tasas e impuestos\n",
			cheapestFlight.Departure.Date.Format(dateLayout),
			cheapestFlight.Departure.Airport.Code,
			cheapestFlight.Arrival.Airport.Code,
			cheapestFlight.Cabin,
			cheapestFlight.Airline.Name,
			cheapestFlight.Stops,
			cheapestFare.Miles,
			boardingTax.Totals.Total.Money,
		)

	}
	fmt.Println()
}

func getTaxForFlight(c *http.Client, flight *model.Flight, fare *model.Fare) *model.BoardingTax {
	u := createTaxURL(flight, fare)
	r := createRequest(u, "api-airlines-boarding-tax-prd.smiles.com.br")
	var body []byte
	var data model.BoardingTax

	res, err := c.Do(r)
	if err != nil {
		log.Fatal("Error making request ", err)
	}

	body, err = ioutil.ReadAll(res.Body)
	if body == nil {
		log.Fatal("Empty result")
	}

	if err := json.Unmarshal(body, &data); err != nil {
		log.Fatal("Error unmarshalling data ", err)
	}

	return &data
}
