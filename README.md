# Pewpew [![Travis](https://img.shields.io/travis/bengadbois/pewpew.svg?branch=master&style=flat-square)](https://travis-ci.org/bengadbois/pewpew) [![Go Report Card](https://goreportcard.com/badge/github.com/bengadbois/pewpew?style=flat-square)](https://goreportcard.com/report/github.com/bengadbois/pewpew) [![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](https://godoc.org/github.com/bengadbois/pewpew/lib)

Flexible HTTP stress tester

## Disclaimer
Pewpew is designed as a tool to help those developing web services and websites. Please use responsibly.

## Status
Pewpew is under active development. Building from master should generally work, but the API is not solidified yet. Don't rely on it for anything important yet.

## Installing
Requires Golang 1.6+

If your `$GOPATH` is set correctly, you can just

```
go get github.com/bengadbois/pewpew
```

Will publish prebuilt binaries once first release is ready

## Usage
Simple example:
```
pewpew stress http://www.example.com
```
This makes ten requests to http://www.example.com

Complex example with multiple options and multiple targets:
```
pewpew stress -X POST --body '{"hello": "world"}' -n 100 -c 5 -t 2.5 -H Accept-Encoding:gzip -H Content-Type:application/json https://www.example.com:443/path localhost 127.0.0.1/api
```
Each of the three targets https://www.example.com:443/path, http://localhost, http://127.0.0.1/api
 - 100 requests total requests per target (300 total)
 - 5 concurrent requests per target (15 simultaneous)
 - POST with body `{"hello": "world"}`
 - Two headers: `Accept-Encoding:gzip` and `Content-Type:application/json`
 - Each request times out after 2.5 seconds

For the full list of command line options, run `pewpew help` or `pewpew help stress`

---

Pewpew supports complex configurations using a config file. Pewpew expects the config file is in the current directory and named `config.json` or `config.toml`. There are more examples in `examples/`.
```toml
Quiet = false
GlobalCompress = true
GlobalUserAgent = "pewpewpewpewpew"
GlobalTimeout = "1.75s"
GlobalHeaders = "Accept-Encoding:gzip"

[[Targets]]
URL = "http://127.0.0.1/home"
Count = 15
Concurrency = 3
[[Targets]]
URL = "https://127.0.0.1/api/user"
Count = 1
Method = "POST"
Body = "{\"username\": \"newuser1\", \"email\": \"newuser1@domain.com\"}"
Headers = "Accept-Encoding:gzip, Content-Type:application/json"
Compress = true
Timeout = "500ms"
UserAgent = "notpewpew"
```
Pewpew's config allows for cascading settings.
Precedence (from high to low):
- Command line settings, such as `-n 40`
- Config file global settings, such as `GlobalCount: 30`
- Config file individual target settings, such as `Count: 20`
- Default global settings, such as `GlobalCount: 10`

All command line options are treated as global settings, and URLs specified on the command line overwrite all Targets set config files.

Not all settings are available per target, such as Verbose, which is only a global setting.

Global settings:
- NoHTTP2 (default false)
- EnforceSSL (default false)
- ResultFilenameJSON (default empty, so skipped)
- ResultFilenameCSV (default empty, so skipped)
- Quiet (default false)
- Verbose (default false)
- GlobalCount (default defer to Target)
- GlobalConcurrency (default defer to Target)
- GlobalTimeout (default defer to Target)
- GlobalMethod (default defer to Target)
- GlobalBody (default defer to Target)
- GlobalBodyFilename (default defer to Target)
- GlobalHeaders (default defer to Target)
- GlobalUserAgent (default defer to Target)
- GlobalBasicAuth (default defer to Target)
- GlobalCompress (default defer to Target)

Individual target settings:
- URL (default "http://localhost")
- GlobalCount (default 10)
- GlobalConcurrency (default 1)
- GlobalTimeout (default 10s)
- GlobalMethod (default GET)
- GlobalBody (default empty)
- GlobalBodyFilename (default none)
- GlobalHeaders (default none)
- GlobalUserAgent (default "pewpew")
- GlobalBasicAuth (default none)
- GlobalCompress (default false)

## Using as a Golang library
```go
package main

import (
    "fmt"

    pewpew "github.com/bengadbois/pewpew/lib"
)

func main() {
    stressCfg := pewpew.NewStressConfig()

    //global settings
    stressCfg.Quiet = true
    //setup one target
    stressCfg.Targets[0].URL = "https://127.0.0.1:443/home"
    stressCfg.Targets[0].Count = 100
    stressCfg.Targets[0].Concurrency = 32
    stressCfg.Targets[0].Timeout = "2s"
    stressCfg.Targets[0].Method = "POST"
    stressCfg.Targets[0].Body = `{"field": "data", "work": true}`

    //begin testing
    err := pewpew.RunStress(*stressCfg)
    if err != nil {
        fmt.Println("pewpew stress failed:  %s", err.Error())
    }
}
```
Full package documentation at [godoc.org](https://godoc.org/github.com/bengadbois/pewpew/lib)

## Hints

If you receive a lot of "socket: too many open files" errors while running many concurrent requests, try increasing your ulimit.
