package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"gopkg.in/yaml.v3"
)

func init() {
	actionProviderMap["request"] = newActionRequest
}

func newActionRequest(endpoint EndpointStruct, configMap map[string]interface{}) ActionHandler {
	var (
		config       = PathAccessor{config: configMap}
		method       = config.Get("method", "GET").(string)
		url          = config.Get("url", "").(string)
		headers      = config.Get("headers", []interface{}{}).([]interface{})
		bodyTemplate = config.Get("body", "HTTP 200 OK").(string)
		delay        = config.Get("delay", 0).(int)
	)
	if url == "" {
		panic("config error: cannot have an empty request url for a request action")
	}
	log.Printf("| {action:request=%v/%v/%v/%v/%v}", method, url, headers, bodyTemplate, delay)
	return func(requestId string, response http.ResponseWriter, request *http.Request, params httprouter.Params, context map[string]interface{}) {
		requestBody := fromTemplate(bodyTemplate, context)
		if request, err := http.NewRequest(method, fromTemplate(url, context), strings.NewReader(requestBody)); err != nil {
			panic(fmt.Errorf("[%s] %v", requestId, err))
		} else {
			for _, header := range headers {
				if headerMap, ok := header.(map[string]interface{}); !ok || len(headerMap) > 1 {
					panic(fmt.Errorf("invalid header entry in config: %v", header))
				} else {
					for key, value := range headerMap {
						request.Header.Add(fromTemplate(key, context), fromTemplate(value.(string), context))
					}
				}
			}
			if delay > 0 {
				time.Sleep(time.Duration(delay) * time.Millisecond)
			}
			log.Printf("[%s] %s %s: %v", requestId, request.Method, request.URL, request)
			if response, err := http.DefaultClient.Do(request); err != nil {
				panic(fmt.Errorf("[%s] ERROR %s %s: %v", requestId, request.Method, request.URL, err))
			} else {
				log.Printf("[%s] Result %s %s: %v", requestId, request.Method, request.URL, response)
				responseBody := ""
				responseData := make(map[string]interface{})
				if response.Body != nil {
					defer response.Request.Body.Close()
					if bytes, err := io.ReadAll(response.Body); err != nil {
						panic(fmt.Errorf("[%s] %v", requestId, err))
					} else {
						responseBody = string(bytes)
					}
					contentType := response.Header.Get("Content-Type")
					switch contentType {
					case "text/json":
						if err := json.Unmarshal([]byte(responseBody), &responseData); err != nil {
							panic(fmt.Errorf("[%s] %v", requestId, err))
						}
					case "text/yaml":
						if err := yaml.Unmarshal([]byte(responseBody), &responseData); err != nil {
							panic(fmt.Errorf("[%s] %v", requestId, err))
						}
					}
				}
				context["__request__"] = map[string]interface{}{
					"status":  response.StatusCode,
					"body":    responseBody,
					"data":    responseData,
					"headers": response.Header.Clone(),
				}
			}
		}
	}
}
