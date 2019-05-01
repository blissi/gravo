package volkszaehler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

type Client interface {
	QueryPublicEntities() []Entity
	QueryData(uuid string, from time.Time, to time.Time, group string, options string, tuples int) []Tuple
	QueryPrognosis(uuid string, period string) Prognosis
}

// Client is the volkszaehler API client
type client struct {
	url    string
	client http.Client
	debug  bool
}

// NewClient creates new volkszaehler api client
func NewClient(url string, timeout *time.Duration, debug bool) Client {
	return &client{
		url: detectAPIEndpoint(url),
		client: http.Client{
			Timeout: *timeout,
		},
		debug: debug,
	}
}

func detectAPIEndpoint(url string) string {
	const probe = "/entity.json"

	url = strings.TrimRight(url, "/")
	log.Println("Validating API endpoint")

	resp, err := http.Get(url + probe)
	if err == nil {
		_ = resp.Body.Close() // close body after checking for error

		if resp.StatusCode == 200 {
			log.Println("API endpoint validated")
			return url
		}
	}

	if strings.HasSuffix(url, "/middleware.php") {
		log.Println("API endpoint not responding. Will keep retrying using configured uri")
		return url
	}

	// append middleware.php
	detectedURL := url + "/middleware.php"
	log.Println("API endpoint not responding. Trying " + detectedURL)

	resp, err = http.Get(detectedURL + probe)
	if err == nil {
		_ = resp.Body.Close() // close body after checking for error

		if resp.StatusCode == 200 {
			log.Println("API endpoint detected, using " + detectedURL)
			return detectedURL
		}
	}

	log.Println("API endpoint still not responding. Will keep retrying using configured uri")
	return url
}

func (api *client) get(endpoint string) (io.Reader, error) {
	url := api.url + endpoint

	start := time.Now()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Accept", "application/json")

	resp, err := api.client.Do(req)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close() // close body after checking for error
	}()

	duration := time.Since(start)
	log.Printf("GET %s (%dms)", url, duration.Nanoseconds()/1e6)

	// read body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Print(err)
	}

	if api.debug {
		log.Print(string(body))
	}

	return bytes.NewReader(body), nil
}

// QueryPublicEntities retrieves public entities from middleware
func (api *client) QueryPublicEntities() []Entity {
	r, err := api.get("/entity.json")
	if err != nil {
		return []Entity{}
	}

	er := EntityResponse{}
	if err := json.NewDecoder(r).Decode(&er); err != nil {
		log.Printf("json decode failed: %v", err)
		return []Entity{}
	}

	return er.Entities
}

// QueryData retrieves data for specified timeframe and parameters
func (api *client) QueryData(uuid string, from time.Time, to time.Time,
	group string, options string, tuples int,
) []Tuple {
	f := from.Unix()
	t := to.Unix()
	url := fmt.Sprintf("/data/%s.json?from=%d&to=%d", uuid, f*1000, t*1000)

	if tuples > 0 {
		url += fmt.Sprintf("&tuples=%d", tuples)
	}

	if group != "" {
		url += "&group=" + group
	}

	if options != "" {
		url += "&options=" + options
	}

	reader, err := api.get(url)
	if err != nil {
		return []Tuple{}
	}

	dr := DataResponse{}
	if err := json.NewDecoder(reader).Decode(&dr); err != nil {
		log.Printf("json decode failed: %v", err)
		return []Tuple{}
	}

	return dr.Data.Tuples
}

// QueryPrognosis retrieves prognosis from middleware
func (api *client) QueryPrognosis(uuid string, period string) Prognosis {
	url := fmt.Sprintf("/prognosis/%s.json?period=%s", uuid, period)

	r, err := api.get(url)
	if err != nil {
		return Prognosis{}
	}

	pr := PrognosisResponse{}
	if err := json.NewDecoder(r).Decode(&pr); err != nil {
		log.Printf("json decode failed: %v", err)
		return Prognosis{}
	}

	return pr.Prognosis
}