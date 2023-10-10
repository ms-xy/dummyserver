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

The template functionality is exactly Go's `text/template` with one additional
function: `byName` which allows searching the path slice for a specific URL
param.

Local variables exported to the template are:
```
path: map[string]string
data: map[string]any
__request__: map[string]any{
        "status": response.StatusCode,
        "body": responseBody,
        "data": parsedJsonOrYamlResponseData,
        "headers": response.Header.Clone(),
    }
```

Example config showing some of the available flexibility:

```yaml
server:
    ip: "127.0.0.1"
    port: 8080
endpoints:
    - url: /hello/:world
      method: GET
      actions:
        - type: response
          params:
            status: 200
            body:
                Hello {{.path.world}}
            delay: 2000
    - url: /whoami
      method: POST
      params:
        parser: json
      actions:
        - type: request
          params:
            method: GET
            url: http://127.0.0.1:8080/hello/{{.data.name}}
        - type: response
          params:
            status: "{{.__request__.status}}"
            body: "{{.__request__.body}}"

```

Note that strings may need quotation marks if their being missing could confuse
the yaml parser.

You can try the above example config by using a second terminal to start a
`curl` call to the `whoami` endpoint that has been started:

```shell
$>  curl --request POST --data '{"name": "Mary"}' --header "Content-Type: text/json" 127.0.0.1:8080/whoami
```

## License

The DummyServer is licensed under GNU GPLv3.
Please see the attached License.txt file for details.
