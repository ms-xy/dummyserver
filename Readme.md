## DummyServer

DummyServer is a server that provides mock API interfaces as provided via the
json configuration.

It's main purpose is to be able to mock server side APIs, without even having
developed your real server yet. This can be very useful when rapidly prototyping
client software, e.g. for presentations.

## Usage

Download and install the DummyServer into `$GOPATH/bin` (don't forget to add
this directory to your `$PATH`, if you haven't already).

```shell
$> go get github.com/ms-xy/DummyServer
```

Configure it in **any** working directory (it will always read from the current
working directory, so where you put this config is wholly up to you).

You may specify as many responses as you like. However, be aware that clashes
of URL configurations will result in runtime panics, thrown by the httprouter
package.

The config file name is `dummyserver.conf`.

```json
{
    "Binding": {
        "IP": "0.0.0.0",
        "Port": 7779
    },
    "Responses": [
        {
            "comment": "See https://godoc.org/github.com/julienschmidt/httprouter for infos on the url format",
            "Url": "/*anything",
            "Response": {
                "HttpStatusCode": 200,
                "Headers": [
                    {
                        "Key": "Content-Type",
                        "Value": "text/json; charset=utf-8"
                    }
                ],
                "Body": "{}"
            }
        }
    ]
}
```

Run the server. It will automatically use the configuration in the current
directory.

```shell
$> DummyServer
```

## License

The DummyServer is licensed under GNU GPLv3.
Please see the attached License.txt file for details.
Different license terms can be arranged on request.
