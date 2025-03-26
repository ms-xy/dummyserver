package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func init() {
	actionProviderMap["parse-form"] = newActionParseForm
}

func newActionParseForm(endpoint EndpointStruct, config map[string]interface{}) ActionHandler {
	var (
		__action__ = "parse-form"
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
		defer request.Body.Close()
		if err := request.ParseForm(); err != nil {
			doPanic(requestId, "%v", err)
		}
		context[contextPath] = request.Form
	}
}
