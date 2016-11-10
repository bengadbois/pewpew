# Pewpew [![Travis](https://img.shields.io/travis/bengadbois/pewpew.svg?branch=master)](https://travis-ci.org/bengadbois/pewpew) [![Go Report Card](https://goreportcard.com/badge/github.com/bengadbois/pewpew)](https://goreportcard.com/report/github.com/bengadbois/pewpew)

Dead simple HTTP stress tester

## Disclaimer
Pewpew is designed as a tool to help those developing web services and websites. Please use responsibly.

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

The body is `{"hello": "world"}` and includes the header 'Accept-Encoding:gzip'

---

For more options, run `pewpew help` or `pewpew help stress`

## Installing
Requires Golang 1.6+

If your `$GOPATH` is set correctly, you can just

```go get github.com/bengadbois/pewpew```

Will publish prebuilt binaries once first release is ready
