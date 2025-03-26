package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

func init() {
	actionProviderMap["cache-files"] = newActionCacheFile
}

func newActionCacheFile(endpoint EndpointStruct, config map[string]interface{}) ActionHandler {
	var (
		__action__ = "cache-files"
		doPanic = makeActionExecutionPanicFn(endpoint, __action__)
		doError = makeActionExecutionErrorFn(endpoint, __action__)
		configMap  = PathAccessor{config: config}
		mapping = configMap.Get("mapping", make(map[string]any)).(map[string]any)
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
		result := make(map[string]string)
		if request.MultipartForm == nil {
			doPanic(requestId, "Error: no parsed multi-part-form available, did you forget to add the corresponding action?")
		}
		for formKey, cacheKey := range mapping {
			formKey = fromTemplate(formKey, context)
			cacheKeyStr := fromTemplate(cacheKey.(string), context)
			if file, header, err := request.FormFile(formKey); err != nil {
				errMsg := fmt.Sprintf(
					"Error: failed extract form file '%s': %v", formKey, err)
				result[cacheKeyStr] = errMsg
				doError(requestId, errMsg)
			} else if cacheFile, err := NewCacheFile(file, header); err == nil {
				fileCache.Add(
					cacheKeyStr,
					cacheFile,
					cacheTimeout)
				result[cacheKeyStr] = fmt.Sprintf(
					"Success (key=%s; timeout=%s)",
					cacheKeyStr,
					time.Now().Add(cacheTimeout).String())
			} else {
				errMsg := fmt.Sprintf(
					"Error: failed to add cache file: %v", err)
				result[cacheKeyStr] = errMsg
				doError(requestId, errMsg)
			}
		}
		context["__cached_files__"] = result
	}
}
