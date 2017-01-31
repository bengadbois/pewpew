# Pewpew [![Travis](https://img.shields.io/travis/bengadbois/pewpew.svg?branch=master&style=flat-square)](https://travis-ci.org/bengadbois/pewpew) [![Go Report Card](https://goreportcard.com/badge/github.com/bengadbois/pewpew?style=flat-square)](https://goreportcard.com/report/github.com/bengadbois/pewpew) [![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](https://godoc.org/github.com/bengadbois/pewpew/lib)

Flexible HTTP stress tester

## Disclaimer
Pewpew is designed as a tool to help those developing web services and websites. Please use responsibly.

## Status
Pewpew is under active development. Building from master should generally work, but the API is not solidified yet. Don't rely on it for anything important yet.

## Usage
Simple example:
```
pewpew http://www.example.com
```
This makes ten requests to http://www.example.com

---

Complex example with multiple options and multiple targets:
```
pewpew -X POST --body '{"hello": "world"}' -n 100 -c 5 -t 2.5 -H Accept-Encoding:gzip -H Content-Type:application/json https://www.example.com:443/path localhost 123.456.78.9/api
```
Each of the three targets https://www.example.com:443/path, http://localhost, http://123.456.78.9/api
 - 100 requests total requests per target (300 total)
 - 5 concurrent requests per target (15 simultaneous)
 - POST with body `{"hello": "world"}`
 - Two headers: `Accept-Encoding:gzip` and `Content-Type:application/json`
 - Each request times out after 2.5 seconds

---

For more configuration options, run `pewpew help` or `pewpew help stress`

## Installing
Requires Golang 1.6+

If your `$GOPATH` is set correctly, you can just

```
go get github.com/bengadbois/pewpew
```

Will publish prebuilt binaries once first release is ready

## Using as a Golang library
```go
package main

import (
    "fmt"
    "net/url"
    "time"

    pewpew "github.com/bengadbois/pewpew/lib"
)   

func main() { 
    stressCfg := pewpew.NewStressConfig()
   
    //global settings 
	stressCfg.Quiet = true
    //setup one target
    parsedURL, _ := url.Parse("https://127.0.0.1:443/uri")
    stressCfg.Targets[0].URL = *parsedURL
    stressCfg.Targets[0].Count = 10000 
    stressCfg.Targets[0].Concurrency = 32
    stressCfg.Targets[0].Timeout = time.Duration(2) * time.Second
    stressCfg.Targets[0].ReqMethod = "POST"
    stressCfg.Targets[0].ReqBody = `{"field": "data", "work": true}`
   
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
