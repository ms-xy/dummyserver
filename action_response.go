package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/julienschmidt/httprouter"
)

func init() {
	actionProviderMap["response"] = newActionResponse
}

func newActionResponse(endpoint EndpointStruct, config map[string]interface{}) ActionHandler {
	var (
		__action__ = "response"
		doPanic = makeActionExecutionPanicFn(endpoint, __action__)
		configMap    = PathAccessor{config: config}
		status       = configMap.Get("status", 200)
		headers      = configMap.Get("headers", []interface{}{}).([]interface{})
		responseBody = configMap.Get("body", "").(string)
		responseLocalFile = configMap.Get("localFile", "").(string)
		responseCachedFile = configMap.Get("cachedFile", "").(string)
		responseWriter ActionHandler
		delay        = configMap.Get("delay", 0).(int)
	)

	log.Printf("| {action:response=[%v]%v/%s/%v}", status, headers, responseBody, delay)

	selectedResponses := 0
	if responseBody != "" {
		selectedResponses ++
	}
	if responseLocalFile != "" {
		selectedResponses ++
	}
	if responseCachedFile != "" {
		selectedResponses ++
	}
	if selectedResponses != 1 {
		actionSetupPanic(endpoint, __action__, "Must specify exactly one of the following options:\n\t- body\n\t- localFile\n\t- cachedFile")
	}

	statusWriter := func(requestId string, response http.ResponseWriter, context map[string]any) {
		if statusInt, ok := status.(int); ok {
			response.WriteHeader(statusInt)
		} else if statusInt, err := strconv.ParseInt(fromTemplate(status.(string), context), 10, 32); err != nil {
			panic(fmt.Errorf("[%s] %v", requestId, err))
		} else {
			response.WriteHeader(int(statusInt))
		}
	}

	if responseBody != "" {
		responseWriter = func(requestId string, response http.ResponseWriter, request *http.Request, params httprouter.Params, context map[string]interface{}) {
			statusWriter(requestId, response, context)
			response.Write([]byte(fromTemplate(responseBody, context)))
		}

	} else if responseLocalFile != "" {
		responseWriter = func(requestId string, response http.ResponseWriter, request *http.Request, params httprouter.Params, context map[string]interface{}) {
			resolvedLocalPath := fromTemplate(responseLocalFile, context)
			if file, err := os.Open(resolvedLocalPath); err != nil {
				doPanic(requestId, "Error opening local file '%s': %v",
					resolvedLocalPath, err)
			} else {
				finfo, err := file.Stat()
				if err != nil {
					log.Panicf("[%s] action:error: %v", requestId, err)
				}
				response.Header().Add("Content-Type", "application/octet-stream")
				response.Header().Add("Content-Disposition", "attachment; filename="+finfo.Name())
				response.Header().Add("Content-Transfer-Encoding", "binary")
				response.Header().Add("Content-Length", fmt.Sprintf("%d", finfo.Size()))
				statusWriter(requestId, response, context)
				io.Copy(response, file)
			}
		}

	} else {
		responseWriter = func(requestId string, response http.ResponseWriter, _ *http.Request, _ httprouter.Params, context map[string]interface{}) {
			resolvedCachePath := fromTemplate(responseCachedFile, context)
			if cachedFile := fileCache.Get(resolvedCachePath, nil); cachedFile == nil {
				doPanic(requestId, "Error retrieving cached file '%s': not found", resolvedCachePath)
			} else {
				cachedFile.(CacheFile).AddHeaders(response)
				statusWriter(requestId, response, context)
				if err := cachedFile.(CacheFile).Copy(response); err != nil {
					doPanic(requestId, "Error reading cached file '%s': %v", resolvedCachePath, err)
				}
			}
		}
	}

	return func(requestId string, response http.ResponseWriter, request *http.Request, params httprouter.Params, context map[string]interface{}) {
		for _, header := range headers {
			if headerMap, ok := header.(map[string]interface{}); !ok || len(headerMap) > 1 {
				log.Panicf("invalid header entry in config: %v", header)
			} else {
				for key, value := range headerMap {
					response.Header().Set(fromTemplate(key, context), fromTemplate(value.(string), context))
				}
			}
		}
		if delay > 0 {
			time.Sleep(time.Duration(delay) * time.Millisecond)
		}
		responseWriter(requestId, response, request, params, context)
	}
}
