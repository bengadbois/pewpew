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
This makes one request to http://www.example.com

---

Complex example:
```
pewpew -X POST --body '{"hello": "world"}' -n 10 -c 5 -t 2s -H Accept-Encoding:gzip https://www.example.com:443/path
```
This makes 10 POST requests to https://www.example.com:443/path

Each request times out after 2 seconds, and 5 are running concurrently

The body is `{"hello": "world"}` and includes the header `Accept-Encoding:gzip`

---

For more options, run `pewpew help` or `pewpew help stress`

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
	"time"
	pewpew "github.com/bengadbois/pewpew/lib"
)

func main() {
	stressCfg := pewpew.NewStressConfig()

	//configure non-default settings
	stressCfg.URL = "https://127.0.0.1:443/uri"
	stressCfg.Count = 10000
	stressCfg.Concurrency = 10
	stressCfg.Timeout = time.Duration(2) * time.Second
	stressCfg.ReqMethod = "POST"
	stressCfg.ReqBody = `{"field": "data", "work": true}`

	err := stress.RunStress(*stressCfg)
	if err != nil {
		fmt.Println("pewpew stress failed:  %s", err.Error())
	}
}
```
Full package documentation at [godoc.org](https://godoc.org/github.com/bengadbois/pewpew/lib)

## Hints

If you receive a lot of "socket: too many open files" errors while running many concurrent requests, try increasing your ulimit.
