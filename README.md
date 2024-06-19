# Pewpew [![Workflow](https://img.shields.io/github/actions/workflow/status/bengadbois/pewpew/build-and-test.yml?branch=master&style=flat-square)](https://github.com/bengadbois/pewpew/actions/workflows/build-and-test.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/bengadbois/pewpew?style=flat-square)](https://goreportcard.com/report/github.com/bengadbois/pewpew) [![Coveralls branch](https://img.shields.io/coveralls/bengadbois/pewpew/master.svg?style=flat-square)](https://coveralls.io/github/bengadbois/pewpew?branch=master) [![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](https://godoc.org/github.com/bengadbois/pewpew/lib)

Pewpew is a flexible command line HTTP stress tester. Unlike other stress testers, it can hit multiple targets with multiple configurations, simulating real world load and bypassing caches.

**Disclaimer**: Pewpew is designed as a tool to help those developing web services and websites. Please use responsibly.

![Demo](screencast.gif)

## Features
- Multiple modes for measuring servers
- Regular expression defined targets
- Multiple simultaneous targets
- No runtime dependencies, single binary file
- Statistics on timing, data transferred, status codes, and more
- Export raw data as TSV and/or JSON for analysis, graphs, etc.
- HTTP2 support
- IPV6 support
- Tons of command line and/or config file options (arbitrary headers, cookies, User-Agent, timeouts, ignore SSL certs, HTTP authentication, Keep-Alive, DNS prefetch, and more)

## Installing
Pre-compiled binaries for Windows, Mac, Linux, and BSD are available on [Releases](https://github.com/bengadbois/pewpew/releases).

If you want to get the latest or build from source: install Go 1.11+ and either `go get github.com/bengadbois/pewpew` or git clone this repo.

## Modes
Pewpew features two independent modes: stress and benchmark.

Stress mode (`pewpew stress`) sends requests as fast as the server can respond (limited by concurrency). This mode is usually best for answering questions such as "how fast can the server return 1000 requests?", "will the server ever OOM?", "can I get the server to 503?", and more related to overloading.

Benchmark mode (`pewpew benchmark`) sends requests at a fixed rate (requests per second). This mode is usually best for anwering questions such as "how much traffic can the server handle before latency surprasses 1 second?", "if traffic to the server is rate limited to 100 rps, will there by any 503s?", and other measurable controlled traffic tests.

## Examples
```
pewpew stress -n 50 www.example.com
```
Make 50 requests to http://www.example.com

```
pewpew benchmark --rps 100 --duration 60 www.example.com
```
For 60 seconds, send 100 requests each second to www.example.com

```
pewpew stress -X POST --body '{"hello": "world"}' -n 100 -c 5 -t 2.5s -H "Accept-Encoding:gzip, Content-Type:application/json" https://www.example.com:443/path localhost 127.0.0.1/api
```
Make request to each of the three targets https://www.example.com:443/path, http://localhost, http://127.0.0.1/api
 - 100 requests total requests per target (300 total)
 - 5 concurrent requests per target (15 simultaneous)
 - POST with body `{"hello": "world"}`
 - Two headers: `Accept-Encoding:gzip` and `Content-Type:application/json`
 - Each request times out after 2.5 seconds

For the full list of command line options, run `pewpew help` or `pewpew help stress`

### Using Regular Expression Targets
Pewpew supports using regular expressions (Perl syntax) to nondeterministically generate targets.
```
pewpew stress -r "localhost/pages/[0-9]{1,3}"
```
This example will generate target URLs such as:
```
http://localhost/pages/309
http://localhost/pages/390
http://localhost/pages/008
http://localhost/pages/8
http://localhost/pages/39
http://localhost/pages/104
http://localhost/pages/642
http://localhost/pages/479
http://localhost/pages/82
http://localhost/pages/3
```

```
pewpew stress -r "localhost/pages/[0-9]+\?cache=(true|false)(\&referrer=[0-9]{3})?"
```
This example will generate target URLs such as:
```
http://localhost/pages/278613?cache=false
http://localhost/pages/736?cache=false
http://localhost/pages/255?cache=false
http://localhost/pages/25042766?cache=false
http://localhost/pages/61?cache=true
http://localhost/pages/4561?cache=true&referrer=966
http://localhost/pages/7?cache=false&referrer=048
http://localhost/pages/01?cache=true
http://localhost/pages/767911706?cache=false&referrer=642
http://localhost/pages/68780?cache=true
```

Note: dots in IP addresses must be escaped, such as `pewpew stress -r "http://127\.0\.0\.1:8080/api/user/[0-9]{1,3}"`

### Using Config Files

Pewpew supports complex configurations more easily managed with a config file. You can define one or more targets each with their own settings.

By default, Pewpew looks for a config file in the current directory and named `pewpew.json` or `pewpew.toml`. If found, Pewpew can be run like:
```
pewpew stress
```

There are examples config files in `examples/`.

Pewpew allows combining config file and command line settings, to maximize flexibility. Pewpew uses [https://github.com/spf13/viper](Viper) and follows its rules of config precedence.

### Other Options

The full list of options for each command can be viewed by running Pewpew with the `--help` flag.

## Using as a Go library
```go
package main

import (
    "fmt"
    "os"

    pewpew "github.com/bengadbois/pewpew/lib"
)

func main() {
    stressCfg := pewpew.StressConfig{
        Count:       1,
        Concurrency: 1,
        Verbose:     false,
        Targets: []pewpew.Target{{
            URL: "https://127.0.0.1:443/home",
            Options: pewpew.TargetOptions{
                Timeout: "2s",
                Method:  "GET",
                Body:    `{"field": "data", "work": true}`,
            },
        }},
    }

    output := os.Stdout
    stats, err := pewpew.RunStress(stressCfg, output)
    if err != nil {
        fmt.Printf("pewpew stress failed:  %s", err.Error())
    }

    fmt.Printf("%+v", stats)
}
```
Full package documentation at [godoc.org](https://godoc.org/github.com/bengadbois/pewpew/lib)

## Hints

If you receive a lot of "socket: too many open files" errors while running many concurrent requests, try increasing your ulimit.
