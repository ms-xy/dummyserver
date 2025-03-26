package main

import (
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

func init() {
	actionProviderMap["cache"] = newActionCache
}

func newActionCache(endpoint EndpointStruct, config map[string]interface{}) ActionHandler {
	var (
		__action__ = "cache"
		doPanic = makeActionExecutionPanicFn(endpoint, __action__)
		configMap  = PathAccessor{config: config}
		mapping = configMap.Get("mapping", make(map[string]string)).(map[string]string)
		cacheTimeout = time.Duration(configMap.Get("timeout", 5*60).(int)) * time.Second
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
		contextAccessor := &PathAccessor{context}
		for path, cacheKey := range mapping {
			path = fromTemplate(path, context)
			if value, err := contextAccessor.Must(path); err != nil {
				doPanic(
					requestId,
					"failed to cache context[%s], context-path does not exist",
					path)
			} else {
				globalContext.Add(
					fromTemplate(cacheKey, context),
					value,
					cacheTimeout)
			}
		}
	}
}
