package main

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func init() {
	actionProviderMap["parse-json"] = newActionParseJSON
}

func newActionParseJSON(endpoint EndpointStruct, config map[string]interface{}) ActionHandler {
	var (
		__action__ = "parse-json"
		doPanic = makeActionExecutionPanicFn(endpoint, __action__)
		configMap  = PathAccessor{config: config}
		contextPath = configMap.Get("contextTarget", "form").(string)
		allowedMethods = map[string]any{"POST": 1, "PUT": 1}
	)

	if _, contains := allowedMethods[endpoint.Method]; !contains {
		actionSetupPanic(endpoint, __action__,
			"Invalid endpoint method to use this action")
	}

	return func(
		requestId string,
		response http.ResponseWriter,
		request *http.Request,
		params httprouter.Params,
		context map[string]interface{},
	) {
		var (
			jsonData map[string]any = make(map[string]any)
		)
		if request.Body == nil || request.ContentLength <= 0 {
			doPanic(requestId, "No request body received")
		}

		defer request.Body.Close()
		if bytesBody, err := io.ReadAll(request.Body); err != nil {
			doPanic(requestId, "%v", err)
		} else {
			if err := json.Unmarshal(bytesBody, &jsonData); err != nil {
				doPanic(requestId, "%v:\n%v", err, string(bytesBody))
			}
			context[contextPath] = jsonData
		}
	}
}
