package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

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
    Url string
    Method string // Accepts GET and POST
    Actions []ActionStruct
    Params struct {
        Parser string // optional, "json" or "yaml", default is none
    }
}

type ActionStruct struct {
    Type string
    Params map[string]interface{}
    /*
    Examples:
        type: "response"
        params:
            status: 200
            headers:
                - key: "x-api"
                  value: "sdfsddflksdjflsdfkj"
            body: "{\"msg\":\"your request was process successfully\"}"
            delay: "3000" // value is in millis
        
        type: "request"
        params:
            method: "POST"
            url: "https://127.0.0.1/my-api"
            headers:
                - key: "key"
                  value: "value"
            body: '{"fieldA": "{{byName path paramName}}", "fieldB": "{{data.some.path.in.yaml.or.json}}"'
            delay: "10s"
    */
}

// initialize handlers etc
func main() {
    bytes, err := os.ReadFile("./dummyserver.yaml")
    if err != nil {
        log.Fatalf("Error reading ./dummyserver.yaml: %s", err.Error())
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

type ActionHandler func(requestId string, response http.ResponseWriter, request *http.Request, params httprouter.Params, body string, data map[string]interface{})

func newEndpointHandler(endpoint EndpointStruct) func(http.ResponseWriter, *http.Request, httprouter.Params) {
    actionHandlers := createActionHandlers(endpoint.Actions)
    return func(response http.ResponseWriter, request *http.Request, params httprouter.Params) {
        requestId := uuid.Must(uuid.NewRandom()).String()
        var body string = ""
        var data map[string]interface{} = make(map[string]interface{})
        log.Printf("[%s] %s %s [size=%d]", requestId, request.Method, request.RequestURI, request.ContentLength)
        if request.Method == "POST" || request.Method == "PUT" {
            if request.Body != nil {
                defer request.Body.Close()
                if bytesBody, err := io.ReadAll(request.Body); err != nil {
                    log.Fatal(err)
                } else {
                    body = string(bytesBody)
                    switch endpoint.Params.Parser {
                    case "json":
                        if err := json.Unmarshal(bytesBody, &data); err != nil {
                            log.Fatal(err)
                        }
                    case "yaml":
                        if err := yaml.Unmarshal(bytesBody, &data); err != nil {
                            log.Fatal(err)
                        }
                    default:
                    }
                }
            }
        }
        for _, action := range actionHandlers {
            action(requestId, response, request, params, body, data)
        }
    }
}

func createActionHandlers(actions []ActionStruct) []ActionHandler {
    actionHandlers := make([]ActionHandler, 0, len(actions))
    for _, action := range actions {
        switch action.Type {
        case "response":
            actionHandlers = append(actionHandlers, newResponseActionHandler(action.Params))
        case "request":
            actionHandlers = append(actionHandlers, newRequestActionHandler(action.Params))
        default:
            log.Fatalf("Unsupported action type '%s'", action.Type)
        }
    }
    return actionHandlers
}

var templateFuncs = template.FuncMap{
    "byName": func(params httprouter.Params, name string) string {
        return params.ByName(name)
    },
}

func fromTemplate(tpl string, data map[string]interface{}) string {
    buf := bytes.NewBuffer([]byte{})
    if err := template.Must(template.New("tmp").Funcs(templateFuncs).Parse(tpl)).Execute(buf, data); err != nil {
        log.Fatal(err)
    }
    return buf.String()
}

func newResponseActionHandler(config map[string]interface{}) ActionHandler {
    var (
        configMap    = PathAccessor{config: config}
        status       = configMap.Get("status", 200)
        headers      = configMap.Get("headers", map[string]interface{}{}).(map[string]interface{})
        responseBody = configMap.Get("body", "HTTP 200 OK").(string)
        delay        = configMap.Get("delay", 0).(int)
    )
    log.Printf("| {action:response=[%v]%v/%s/%v}", status, headers, responseBody, delay)
    return func(requestId string, response http.ResponseWriter, request *http.Request, params httprouter.Params, requestBody string, data map[string]interface{}) {
        mergedData := map[string]interface{}{
            "path": params,
            "data": data,
        }
        for key, value := range headers {
            response.Header().Set(fromTemplate(key, mergedData), fromTemplate(value.(string), mergedData))
        }
        if delay > 0 {
            time.Sleep(time.Duration(delay) * time.Millisecond)
        }
        body := fromTemplate(responseBody, mergedData)
        log.Printf("[%s] action:response [%v] %v / %s", requestId, status, response.Header(), body)
        if statusInt, ok := status.(int); ok {
            response.WriteHeader(statusInt)
        } else if statusInt, err := strconv.ParseInt(fromTemplate(status.(string), mergedData), 10, 32); err != nil {
            log.Fatalf("[%s] %v", requestId, err)
        } else {
            response.WriteHeader(int(statusInt))
        }
        response.Write([]byte(body))
    }
}

func newRequestActionHandler(configMap map[string]interface{}) ActionHandler {
    var (
        config       = PathAccessor{config: configMap}
        method       = config.Get("method", "GET").(string)
        url          = config.Get("url", "").(string)
        headers      = config.Get("headers", map[string]interface{}{}).(map[string]interface{})
        bodyTemplate = config.Get("body", "HTTP 200 OK").(string)
        delay        = config.Get("delay", 0).(int)
    )
    if url == "" {
        log.Fatal("config error: cannot have an empty request url for a request action")
    }
    log.Printf("| {action:request=%v/%v/%v/%v/%v}", method, url, headers, bodyTemplate, delay)
    return func(requestId string, response http.ResponseWriter, request *http.Request, params httprouter.Params, origRequestBody string, data map[string]interface{}) {
        mergedData := map[string]interface{}{
            "path": params,
            "data": data,
        }
        requestBody := fromTemplate(bodyTemplate, mergedData)
        if request, err := http.NewRequest(method, fromTemplate(url, mergedData), strings.NewReader(requestBody)); err != nil {
            log.Fatalf("[%s] %v", requestId, err)
        } else {
            for key, value := range headers {
                request.Header.Add(fromTemplate(key, mergedData), fromTemplate(value.(string), mergedData))
            }
            if delay > 0 {
                time.Sleep(time.Duration(delay) * time.Millisecond)
            }
            log.Printf("[%s] %s %s: %v", requestId, request.Method, request.URL, request)
            if response, err := http.DefaultClient.Do(request); err != nil {
                log.Fatalf("[%s] ERROR %s %s: %v", requestId, request.Method, request.URL, response)
            } else {
                log.Printf("[%s] Result %s %s: %v", requestId, request.Method, request.URL, response)
                responseBody := ""
                responseData := make(map[string]interface{})
                if response.Body != nil {
                    defer response.Request.Body.Close()
                    if bytes, err := io.ReadAll(response.Body); err != nil {
                        log.Fatalf("[%s] %v", requestId, err)
                    } else {
                        responseBody = string(bytes)
                    }
                    contentType := response.Header.Get("Content-Type")
                    switch contentType {
                    case "text/json":
                        if err := json.Unmarshal([]byte(responseBody), &responseData); err != nil {
                            log.Fatalf("[%s] %v", requestId, err)
                        }
                    case "text/yaml":
                        if err := yaml.Unmarshal([]byte(responseBody), &responseData); err != nil {
                            log.Fatalf("[%s] %v", requestId, err)
                        }
                    }
                }
                data["__request__"] = map[string]interface{}{
                    "status": response.StatusCode,
                    "body": responseBody,
                    "data": responseData,
                    "headers": response.Header.Clone(),
                }
            }
        }
    }
}

type PathAccessor struct {
    config map[string]interface{}
}

func (pathAccessor *PathAccessor) Get(path string, defaultValue interface{}) interface{} {
    var getRecursive func(map[string]interface{}, []string) interface{}
    getRecursive = func (_map map[string]interface{}, _parts []string) interface{} {
        if _value, _ok := _map[_parts[0]]; !_ok {
            //log.Printf("config %s not found, defaulting to %v", path, defaultValue)
            return defaultValue
        } else if len(_parts) == 1 {
            return _value
        } else if _actMap, _ok := _value.(map[string]interface{}); !_ok {
            //log.Printf("config %s is not an object/map -> error accessing %s", _parts[0], path)
            return defaultValue
        } else {
            return getRecursive(_actMap, _parts[1:])
        }
    }
    return getRecursive(pathAccessor.config, strings.Split(path, "."))
}
