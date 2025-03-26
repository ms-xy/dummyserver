## DummyServer

DummyServer provides mock application interfaces.

It's main purpose is to be able to mock APIs without even having developed your
actual server yet.
This can be very useful when rapidly prototyping client software, e.g. for
presentations.

Additionally it provides the ability to mock calls for testing purposes,
including advanced template functionality.

## Usage

Download and install the DummyServer into `$GOPATH/bin` (don't forget to add
this directory to your `$PATH`, if you haven't already).

```shell
$> go get github.com/ms-xy/dummyserver
```

Create a configuration file `dummyserver.yaml`, open a terminal at the
location and start DummyServer.

```shell
$> ./dummyserver
```

### Configuration

```yaml
# Server address configuration
server:
    ip: "127.0.0.1"
    port: 8080
#
# Endpoint definitions (method + url combinations must not conflict)
endpoints:
    # Simple static mock GET endpoint
    - url: /hello/:world
      method: GET
      actions:
        - type: response
          params:
            status: 200
            body:
                Hello {{.params.world}}
            delay: 2000

    #
    # Endpoints may perform any number of input processing or neutral actions.
    # However, only one response action may be specified.
    #
    # All actions have access to all global and local variables currently defined.
    # If actions define new variables, these are only available to actions
    # executed at a later point in time.
    #
    # Actions are executed sequentially blocking.
    #
    - url: /submit-form/:name
      method: POST
      #
      # Available action types are:
      #
      actions:
        #
        # Parse-Form Action:
        #   Processes a HTML multi-part form.
        #   Form values are accessible via `.form.key`
        #
        - type: parse-form
          params:
            contextKey: <context-key [default=form]>

        #
        # Parse-JSON Action:
        #   Parses and stores the sent request body as JSON.
        #   Start path for value retrieval is `.data`
        #
        - type: parse-json
          params:
            contextTarget: <context-key [default=data]>
        #   Start path for value retrieval is `.data`

        #
        # Parse-Yaml Action:
        #   Parses and stores the sent request body as Yaml
        #
        - type: parse-yaml
          params:
            contextKey: <context-key [default=data]>

        #
        # Multi-part Form Action:
        #   Reads the POST or PUT file upload identified by `attachementId`.
        #   If `attachementId` is undefined, the first found file is used.
        #   May use custom request bodies and headers.
        #
        - type: parse-multi-part-form
          params:
            # max parsing memory
            maxMemory: <max-memory-in-MB [default=50MB]>
            contextKey: <context-key [default=multi-part]>

        #
        # Cache Action:
        #   Store any number of context variables in the global cache under the
        #   respective specified keys.
        #   Cache times out after the given timeout duration.
        #   If the specified timeout is less or equal to 0, no timeout applies.
        #
        - type: cache
          params:
            mapping:
              [<context-path>: <cache-key>]*
            timeout: <cache timeout in seconds [default=300]>

        #
        # Cache Files Action:
        #   Store any number of files in the file cache for later retrieval in
        #   response actions.
        #
        #   Context is enriched by the following result mapping:
        #     __cached_files__:
        #       <file-cache-key>: ['success' | <error-msg>]
        #
        #   Cache times out after the given timeout duration.
        #   If the specified timeout is less or equal to 0, no timeout applies.
        #
        #   Note: parse-multi-part-forms must(!) be called before.
        #
        - type: cache-files
          params:
            mapping:
              [<multi-part-filename>: <file-cache-key>]*
            timeout: <cache timeout in seconds [default=300]>

        #
        # Request Action:
        #   Perform an HTTP/HTTPS request to any target.
        #   May use custom request bodies and headers.
        #
        - type: request
          params:
            method: GET
            url: http://127.0.0.1:8080/hello/{{.data.name}}

        #
        # Response Action:
        #
        - type: response
          params:
            status: "{{.__request__.status}}"
            headers:
              "Content-Type": "application/x-www-form-urlencoded"
            #
            # The response body can be specified as a HTML string.
            body: "{{.__request__.body}}"
            #
            # Alternatively you can also specify a local file or a file that was
            # uploaded and cached via `parse-multi-part-form` instead.
            #
            # Note that Windows users may need to use backslashes instead of
            # forward slashes for local file paths.
            localFile: ./path/to/my/file/relative/to/dummyserver/executable
            cachedFile: <file-cache-key>
```

Note that strings may need quotation marks if left empty, otherwise the yaml
parser may get confused.

You can try the above hello world endpoint by starting the server with it as
the only configured endpoint and then running in a terminal:

```shell
$>  curl --request GET 127.0.0.1:8080/hello/earth
```

### Template Engine

The template functionality is exactly Go's `text/template` with one additional
function: `byName` which allows searching the path slice for a specific URL
param.

Default context variables available to the template engine:
```
# map of all request params
params: map[string]string

# form params
form: map[string]any

# multipart
multi-part: map[string]any

# last request-action result
__request__: map[string]any{
        "status": response.StatusCode,
        "body": responseBody,
        "data": parsedJsonOrYamlResponseData,
        "headers": response.Header.Clone(),
    }

# cache access
__global__: map[string]any{
        "<key>": <value>
    }
```

### File Server Example

```yaml
server:
  ip: "127.0.0.1"
  port: 8080

endpoints:
  - url: /upload/:filename
    method: GET
    actions:
      - type: response
        params:
          body: '
<html>
  <body>
    <form action="/submitUpload/{{ .params.filename }}" method="post" enctype="multipart/form-data">
      Select file to upload:
      <input type="file" name="file">
      <input type="submit" value="Upload File" name="submit">
    </form>
  </body>
</html>
'

  - url: /submitUpload/:filename
    method: POST
    actions:
      - type: parse-multi-part-form
      - type: cache-files
        params:
          mapping:
            file: '{{ .params.filename }}'
      - type: response
        params:
          body: '
<html>
  <body>
    <p>Upload result:</p>
    <ul>
{{ range $key, $result := .__cached_files__ }}
      <li><a href=/download/{{ $key }}>{{ $key }}</a>: {{ $result }}</li>
{{ end }}
    </ul>
  </body>
</html>
'

  - url: /download/:filename
    method: GET
    actions:
      - type: response
        params:
          cachedFile: '{{ .params.filename }}'
```

## License

The DummyServer is licensed under GNU GPLv3.
Please see the attached License.txt file for details.
