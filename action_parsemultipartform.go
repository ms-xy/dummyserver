package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func init() {
	actionProviderMap["parse-multi-part-form"] = newActionParseMultiPartForm
}

func newActionParseMultiPartForm(endpoint EndpointStruct, config map[string]interface{}) ActionHandler {
	var (
		__action__ = "parse-multi-part-form"
		doPanic = makeActionExecutionPanicFn(endpoint, __action__)
		configMap  = PathAccessor{config: config}
		maxMemory = configMap.Get("maxMemory", 50).(int) * 1024 * 1024
		contextKey = configMap.Get("contextKey", "multi-part").(string)
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
		defer request.Body.Close()
		if err := request.ParseMultipartForm(int64(maxMemory)); err != nil {
			doPanic(requestId, "%v", err)
		}
		context[contextKey] = request.MultipartForm
	}
}
