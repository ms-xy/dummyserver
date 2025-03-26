package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"text/template"

	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"gopkg.in/yaml.v3"
)

// webserver config
type Config struct {
	Server struct {
		Ip   string
		Port int
	}
	Endpoints []EndpointStruct
}

type EndpointStruct struct {
	// Endpoint URL format:
	// https://godoc.org/github.com/julienschmidt/httprouter
	Url     string
	Method  string // Accepts GET and POST
	Actions []ActionStruct
	Params  struct {
		// Parser string // optional, "json" or "yaml", default is none
	}
}

type ActionStruct struct {
	Type   string
	Params map[string]interface{}
}

var (
	globalContext = NewCache(func(key string, value any) {})
	fileCache = NewCache(func(key string, value any) {
		value.(CacheFile).Remove()
	})
	actionProviderMap = map[string]ActionHandlerProvider{}
)

// initialize handlers etc
func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func(){
		<-c
		signal.Stop(c)
		fileCache.Clear()
		os.Exit(0)
	}()
	defer func(){
		// ensure temp files are cleaned up
		r := recover()
		fileCache.Clear()
		if r != nil {
			panic(r)
		}
	}()
	configFile := "dummyserver.yaml"
	if len(os.Args) >= 2 {
		configFile = os.Args[1]
	}
	bytes, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatalf("Error reading dummyserver.yaml: %s", err.Error())
	}
	// Parse config.
	cfg := &Config{}
	err = yaml.Unmarshal(bytes, cfg)
	if err != nil {
		log.Println("Config parser error: " + err.Error())
		return
	}

	// Create router.
	router := httprouter.New()

	// Create handlers.
	for _, endpoint := range cfg.Endpoints {
		log.Println(".")
		switch endpoint.Method {
		case "GET":
			router.GET(endpoint.Url, newEndpointHandler(endpoint))
		case "POST":
			router.POST(endpoint.Url, newEndpointHandler(endpoint))
		case "PUT":
			router.PUT(endpoint.Url, newEndpointHandler(endpoint))
		default:
			log.Fatalf("Unsupported endpoint method type '%s' for %s", endpoint.Method, endpoint.Url)
		}
		log.Printf(" `-> [%s] %s", endpoint.Method, endpoint.Url)
	}

	// Bind to ip and port.
	ip := cfg.Server.Ip
	port := cfg.Server.Port
	addr := fmt.Sprintf("%s:%d", ip, port)
	log.Println()
	log.Printf("Binding to: %s\n\n", addr)
	log.Fatalln(http.ListenAndServe(addr, router))
}

type ActionHandler func(requestId string, response http.ResponseWriter, request *http.Request, params httprouter.Params, context map[string]interface{})

func newEndpointHandler(endpoint EndpointStruct) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	actionHandlers := createActionHandlers(endpoint)
	return func(response http.ResponseWriter, request *http.Request, params httprouter.Params) {
		var (
			requestId  = uuid.Must(uuid.NewRandom()).String()
			paramMap   = make(map[string]string)
			context map[string]interface{}
		)
		log.Printf("[%s] %s %s [size=%d]", requestId, request.Method, request.RequestURI, request.ContentLength)
		for _, param := range params {
			paramMap[param.Key] = param.Value
		}
		context = map[string]interface{}{
			"params": paramMap,
		}
		for _, action := range actionHandlers {
			action(requestId, response, request, params, context)
		}
	}
}

type ActionHandlerProvider func(endpoint EndpointStruct, actionParams map[string]any) ActionHandler

func createActionHandlers(endpoint EndpointStruct) []ActionHandler {
	actionHandlers := make([]ActionHandler, 0, len(endpoint.Actions))
	for _, action := range endpoint.Actions {
		log.Printf("attempting to add action %v", action)
		if actionProvider, exists := actionProviderMap[action.Type]; exists {
			actionHandlers = append(actionHandlers, actionProvider(endpoint, action.Params))
		} else {
			log.Fatalf("Unsupported action type '%s'", action.Type)
		}
	}
	return actionHandlers
}

var templateFuncs = template.FuncMap{
	// "byName": func(params httprouter.Params, name string) string {
	//     return params.ByName(name)
	// },
}

func fromTemplate(tpl string, data map[string]interface{}) string {
	buf := bytes.NewBuffer([]byte{})
	if err := template.Must(template.New("tmp").Funcs(templateFuncs).Parse(tpl)).Execute(buf, data); err != nil {
		panic(err)
	}
	return buf.String()
}

func actionPanic(
	requestId string,
	endpoint EndpointStruct,
	action string,
	fmtString string,
	fmtParams ...interface{},
) {
	log.Panicf("[%s] >> ERROR << [%s|%s] action:%s\n%s",
		requestId,
		endpoint.Method,
		endpoint.Url,
		action,
		fmt.Sprintf(fmtString, fmtParams...))
}

func actionError(
	requestId string,
	endpoint EndpointStruct,
	action string,
	fmtString string,
	fmtParams ...interface{},
) {
	log.Printf("[%s] >> ERROR << [%s|%s] action:%s\n%s",
		requestId,
		endpoint.Method,
		endpoint.Url,
		action,
		fmt.Sprintf(fmtString, fmtParams...))
}

type ActionPanicFunc func(requestId string, fmtString string, fmtParams ...interface{})

func makeActionExecutionPanicFn(endpoint EndpointStruct, action string) ActionPanicFunc {
	return func(
		requestId string,
		fmtString string,
		fmtParams ...interface{},
	) {
		actionPanic(requestId, endpoint, action, fmtString, fmtParams...)
	}
}

func makeActionExecutionErrorFn(endpoint EndpointStruct, action string) ActionPanicFunc {
	return func(
		requestId string,
		fmtString string,
		fmtParams ...interface{},
	) {
		actionError(requestId, endpoint, action, fmtString, fmtParams...)
	}
}

func actionSetupPanic(
	endpoint EndpointStruct,
	action string,
	fmtString string,
	fmtParams ...interface{},
) {
	log.Fatalf("| >> SETUP ERROR << [%s|%s] action:%s\n -> %s",
		endpoint.Method, endpoint.Url,
		action,
		fmt.Sprintf(fmtString, fmtParams...))
}
