# Pewpew [![Travis](https://img.shields.io/travis/bengadbois/pewpew/master.svg?&style=flat-square)](https://travis-ci.org/bengadbois/pewpew) [![Go Report Card](https://goreportcard.com/badge/github.com/bengadbois/pewpew?style=flat-square)](https://goreportcard.com/report/github.com/bengadbois/pewpew) [![Coveralls branch](https://img.shields.io/coveralls/bengadbois/pewpew/master.svg?style=flat-square)](https://coveralls.io/github/bengadbois/pewpew?branch=master) [![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](https://godoc.org/github.com/bengadbois/pewpew/lib)

Pewpew is a flexible command line HTTP stress tester. Unlike other stress testers, it can hit multiple targets with multiple configurations, simulating real world load and bypassing caches.

**Disclaimer**: Pewpew is designed as a tool to help those developing web services and websites. Please use responsibly.

![Demo](http://i.imgur.com/4Hj6AuO.gif)

## Features
- Regular expression defined targets
- Multiple simultaneous targets
- No dependencies, single binary
- Statistics on timing, data transferred, status codes, and more
- Export raw data as TSV and/or JSON for analysis, graphs, etc.
- HTTP2 support
- IPV6 support
- Available as a Go library
- Tons of command line and/or config file options (arbitrary headers, cookies, User-Agent, timeouts, ignore SSL certs, HTTP authentication, Keep-Alive and more)

## Status
Pewpew is under active development. Since Pewpew is pre-1.0, minor version changes may be breaking. Tagged releases should be stable. Versioning follows [SemVer](http://semver.org/).

## Installing
Pre-compiled binaries are available on [Releases](https://github.com/bengadbois/pewpew/releases).

If you want to get the latest or build from source: install Go 1.8+, `go get github.com/bengadbois/pewpew`, and install dependencies with [Glide](http://glide.sh/).

## Examples
```
pewpew stress -n 50 www.example.com
```
Make 50 requests to http://www.example.com

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

### Using Config Files

Pewpew supports complex configurations more easily managed with a config file. You can define one or more targets each with their own settings.

Pewpew expects the config file is in the current directory and named `config.json` or `config.toml`. Then just run:
```
pewpew stress
```

Here is an example `config.toml`. There are more examples in `examples/`.
```toml
#Global settings
Count = 10
Quiet = false
Compress = true
UserAgent = "pewpewpewpewpew"
Timeout = "1.75s"
Headers = "Accept-Encoding:gzip"

#Settings for each of the three Targets
[[Targets]]
URL = "http://127.0.0.1/home"
Count = 15
Concurrency = 3
[[Targets]]
URL = "https://127.0.0.1/api/user"
Count = 1 #this overwrites the default global Count (10) for this target
Method = "POST"
Body = "{\"username\": \"newuser1\", \"email\": \"newuser1@domain.com\"}"
Headers = "Accept-Encoding:gzip, Content-Type:application/json"
Cookies = "data=123; session=456" #equivalent to adding "Cookie: data=123; session=456," to the Header option
Compress = true #redundant with the global which is fine
Timeout = "500ms" #this overwrites the explicitly set global Timeout for this target
UserAgent = "notpewpew"
[[Targets]]
URL = "https://127\\.0\\.0\\.1/api/user/[0-9]{1,4}" #double \\ to escape both the '.' and TOML
RegexURL = true #parse URL with Perl syntax regex
Count = 5
```
Pewpew allows for cascading settings, to maximize flexibility and readability.
Precedence (highest first):
- Individual target setting from config file
- Command line setting (which are global)
- Global setting from config file
- Default global setting

All command line options are treated as global settings, and URLs specified on the command line overwrite all Targets set config files.

Not all settings are available per target, such as Verbose, which is only a global setting.

Global settings:
- NoHTTP2 (default false)
- EnforceSSL (default false)
- Quiet (default false)
- Verbose (default false)
- Count (default defer to Target)
- Concurrency (default defer to Target)
- Timeout (default defer to Target)
- Method (default defer to Target)
- Body (default defer to Target)
- BodyFilename (default defer to Target)
- Headers (default defer to Target)
- Cookies (default defer to Target)
- UserAgent (default defer to Target)
- BasicAuth (default defer to Target)
- Compress (default defer to Target)
- KeepAlive (default defer to Target)
- FollowRedirects (default defer to Target)

Individual target settings:
- URL (default "http://localhost")
- RegexURL (default false)
- Count (default 10)
- Concurrency (default 1)
- Timeout (default 10s)
- Method (default GET)
- Body (default empty)
- BodyFilename (default none)
- Headers (default none)
- Cookies (default none)
- UserAgent (default "pewpew")
- BasicAuth (default none)
- Compress (default false)
- KeepAlive (default false)
- FollowRedirects (default true)

## Using as a Go library
```go
package main

import (
    "fmt"
    "os"

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
    output := os.Stdout //can be any io.Writer, such as a file
    stats, err := pewpew.RunStress(*stressCfg, output)
    if err != nil {
        fmt.Println("pewpew stress failed:  %s", err.Error())
    }
    
    //do whatever you want with the raw stats
    fmt.Printf("%+v", stats)
}
```
Full package documentation at [godoc.org](https://godoc.org/github.com/bengadbois/pewpew/lib)

## Hints

If you receive a lot of "socket: too many open files" errors while running many concurrent requests, try increasing your ulimit.
