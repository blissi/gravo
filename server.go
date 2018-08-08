package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Server is the http endpoint used by Grafana's SimpleJson plugin
type Server struct {
	api         *Api
	entityCache map[string]string
	debug       bool
}

func newServer(api *Api, debug bool) *Server {
	server := &Server{
		api:         api,
		entityCache: make(map[string]string),
		debug:       debug,
	}

	// get entity map on startup
	server.getPublicEntites()

	return server
}

func (server *Server) rootHandler(w http.ResponseWriter, r *http.Request) {
	body, _ := ioutil.ReadAll(r.Body)
	log.Printf("%v", string(body))
	fmt.Fprintf(w, "ok\n")
}

func (server *Server) annotationsHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	switch r.Method {
	case http.MethodOptions:
	case http.MethodPost:
		ar := AnnotationsRequest{}
		if err := json.NewDecoder(r.Body).Decode(&ar); err != nil {
			http.Error(w, fmt.Sprintf("json decode failed: %v", err), http.StatusBadRequest)
			return
		}

		resp := []AnnotationResponse{}

		if server.debug {
			j, _ := json.Marshal(resp)
			log.Println(string(j))
		}

		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("json encode failed: %v", err)
			http.Error(w, fmt.Sprintf("json encode failed: %v", err), http.StatusInternalServerError)
			return
		}
	default:
		http.Error(w, "Bad method; supported OPTIONS, POST", http.StatusBadRequest)
		return
	}

	duration := time.Now().Sub(start)
	log.Printf("%v %v (took %s)", r.Method, r.URL.Path, duration.String())
}

func (server *Server) tagKeysHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	switch r.Method {
	case http.MethodOptions:
	case http.MethodPost:
		resp := []TagKeyResponse{
			TagKeyResponse{
				Type: "string",
				Text: "group"},
			// TagKeyResponse{
			// 	Type: "string",
			// 	Text: "mode"}
		}

		if server.debug {
			j, _ := json.Marshal(resp)
			log.Println(string(j))
		}

		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("json encode failed: %v", err)
			http.Error(w, fmt.Sprintf("json encode failed: %v", err), http.StatusInternalServerError)
			return
		}
	default:
		http.Error(w, "Bad method; supported OPTIONS, POST", http.StatusBadRequest)
		return
	}

	duration := time.Now().Sub(start)
	log.Printf("%v %v (took %s)", r.Method, r.URL.Path, duration.String())
}

func (server *Server) tagValuesHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	switch r.Method {
	case http.MethodOptions:
	case http.MethodPost:
		resp := []TagValueResponse{
			TagValueResponse{"Current"},
			TagValueResponse{"Consumption"},
		}

		if server.debug {
			j, _ := json.Marshal(resp)
			log.Println(string(j))
		}

		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("json encode failed: %v", err)
			http.Error(w, fmt.Sprintf("json encode failed: %v", err), http.StatusInternalServerError)
			return
		}
	default:
		http.Error(w, "Bad method; supported OPTIONS, POST", http.StatusBadRequest)
		return
	}

	duration := time.Now().Sub(start)
	log.Printf("%v %v (took %s)", r.Method, r.URL.Path, duration.String())
}

func (server *Server) searchHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	switch r.Method {
	case http.MethodOptions:
	case http.MethodPost:
		sr := SearchRequest{}
		if err := json.NewDecoder(r.Body).Decode(&sr); err != nil {
			log.Printf("json decode failed: %v", err)
			http.Error(w, fmt.Sprintf("json decode failed: %v", err), http.StatusBadRequest)
			return
		}

		resp := server.executeSearch(sr)

		if server.debug {
			j, _ := json.Marshal(resp)
			log.Println(string(j))
		}

		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("json encode failed: %v", err)
			http.Error(w, fmt.Sprintf("json encode failed: %v", err), http.StatusInternalServerError)
			return
		}
	default:
		http.Error(w, "Bad method; supported OPTIONS, POST", http.StatusBadRequest)
		return
	}

	duration := time.Now().Sub(start)
	log.Printf("%v %v (took %s)", r.Method, r.URL.Path, duration.String())
}

func (server *Server) flattenEntities(result *[]Entity, entities []Entity, parent string) {
	for _, entity := range entities {
		if entity.Type == "group" {
			server.flattenEntities(result, entity.Children, entity.Title)
		} else {
			if parent != "" {
				entity.Title = fmt.Sprintf("%s (%s)", entity.Title, parent)
			}
			*result = append(*result, entity)
		}
	}
}

func (server *Server) populateCache(entities []Entity) {
	if len(entities) > 0 {
		server.entityCache = make(map[string]string)
	}

	// add to cache
	for _, entity := range entities {
		if _, ok := server.entityCache[entity.UUID]; !ok {
			server.entityCache[entity.UUID] = entity.Title
		}
	}
}

func (server *Server) getPublicEntites() []Entity {
	entities := make([]Entity, 0)
	server.flattenEntities(&entities, server.api.getEntities(), "")
	server.populateCache(entities)
	return entities
}

func (server *Server) executeSearch(sr SearchRequest) []SearchResponse {
	entities := server.getPublicEntites()

	res := []SearchResponse{}
	for _, entity := range entities {
		res = append(res, SearchResponse{
			Text: entity.Title,
			UUID: entity.UUID,
		})
	}

	return res
}

func (server *Server) queryHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	switch r.Method {
	case http.MethodOptions:
	case http.MethodPost:
		qr := QueryRequest{}
		if err := json.NewDecoder(r.Body).Decode(&qr); err != nil {
			log.Printf("json decode failed: %v", err)
			http.Error(w, fmt.Sprintf("json decode failed: %v", err), http.StatusBadRequest)
			return
		}

		resp := server.sortQueryResponse(qr, server.executeQuery(qr))

		if server.debug {
			j, _ := json.Marshal(resp)
			log.Println(string(j))
		}

		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("json encode failed: %v", err)
			http.Error(w, fmt.Sprintf("json encode failed: %v", err), http.StatusInternalServerError)
			return
		}
	default:
		http.Error(w, "Bad method; supported OPTIONS, POST", http.StatusBadRequest)
		return
	}

	duration := time.Now().Sub(start)
	log.Printf("%v %v (took %s)", r.Method, r.URL.Path, duration.String())
}

func (server *Server) sortQueryResponse(qr QueryRequest, resp []QueryResponse) (res []QueryResponse) {
	// sort by query targets
	for _, target := range qr.Targets {
		for _, metric := range resp {
			if metric.Target.(string) == target.Target {
				res = append(res, metric)
			}
		}
	}

	// substitute name
	for idx, metric := range res {
		if text, ok := server.entityCache[metric.Target.(string)]; ok {
			res[idx].Target = text
		}
	}

	return res
}

func roundTimestampMS(ts int64, group string) int64 {
	t := time.Unix(ts/1000, 0)

	switch group {
	case "hour":
		t.Truncate(time.Hour)
	case "day":
		t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
	case "month":
		t = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.Local)
	}

	return t.Unix() * 1000
}

func (server *Server) executeQuery(qr QueryRequest) []QueryResponse {
	res := []QueryResponse{}
	wg := &sync.WaitGroup{}

	for _, target := range qr.Targets {
		wg.Add(1)

		go func(wg *sync.WaitGroup, target Target) {
			var group, options string

			data := target.Data
			if grp, ok := data["group"]; ok {
				group = strings.ToLower(grp)
			}
			if opt, ok := data["options"]; ok {
				options = strings.ToLower(opt)
			}

			tuples := server.api.getData(
				target.Target,
				qr.Range.From,
				qr.Range.To,
				group,
				options,
				qr.MaxDataPoints)

			qtr := &QueryResponse{
				Target:     target.Target,
				Datapoints: []Tuple{},
			}

			for _, tuple := range tuples {
				ts := tuple[0]
				if group != "" {
					ts = float64(roundTimestampMS(int64(ts), group))
				}

				qtr.Datapoints = append(qtr.Datapoints, Tuple{tuple[1], ts})
			}

			res = append(res, *qtr)
			wg.Done()
		}(wg, target)
	}

	wg.Wait()
	return res
}
