package main

import (
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

// webserver config
type Config struct {
	Binding struct {
		IP   string
		Port int
	}
	Responses []struct {
		Url      string
		Response ResponseStruct
	}
}
type ResponseStruct struct {
	HttpStatusCode int
	Headers        []struct {
		Key   string
		Value string
	}
	Body string
}

// initialize handlers etc
func main() {
	// Open config.
	// Response URL format:
	// https://godoc.org/github.com/julienschmidt/httprouter
	bytes, err := ioutil.ReadFile("./dummyserver.conf")
	if err != nil {
		log.Println("Error reading ./dummyserver.conf: " + err.Error())
		return
	}
	// Parse config.
	cfg := &Config{}
	err = json.Unmarshal(bytes, cfg)
	if err != nil {
		log.Println("Config parser error: " + err.Error())
		return
	}

	// Create router.
	router := httprouter.New()

	// Create handlers.
	for _, response := range cfg.Responses {
		handler := createHandler(response.Response)
		router.GET(response.Url, handler)
	}

	// Bind to ip and port.
	ip := cfg.Binding.IP
	port := cfg.Binding.Port
	addr := fmt.Sprintf("%s:%d", ip, port)
	log.Println("Binding to: " + addr)
	log.Fatalln(http.ListenAndServe(addr, router))
}

func createHandler(response ResponseStruct) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	var body []byte
	bodySet := false
	for _, header := range response.Headers {
		if header.Key == "Content-Type" {
			if len(header.Value) >= 9 && header.Value[0:9] == "text/json" {
				body = []byte(strings.Replace(response.Body, "'", "\"", -1))
				bodySet = true
			}
		}
	}
	if !bodySet {
		body = []byte(response.Body)
	}

	return func(rw http.ResponseWriter, rr *http.Request, ps httprouter.Params) {
		log.Printf("Serving: %s [size=%d]\n", rr.RequestURI, rr.ContentLength)
		if rr.Body != nil {
			rr.Body.Close()
		}
		for _, header := range response.Headers {
			rw.Header().Set(header.Key, header.Value)
		}
		rw.WriteHeader(response.HttpStatusCode)
		rw.Write(body)
	}
}
