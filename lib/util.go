package pewpew

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	reggen "github.com/lucasjones/reggen"
	http2 "golang.org/x/net/http2"
)

//splits on delim into parts and trims whitespace
//delim1 splits the pairs, delim2 splits amongst the pairs
//like parseKeyValString("key1: val2, key3 : val4,key5:val6 ", ",", ":") becomes
//["key1"]->"val2"
//["key3"]->"val4"
//["key5"]->"val6"
func parseKeyValString(keyValStr, delim1, delim2 string) (map[string]string, error) {
	m := make(map[string]string)
	if delim1 == delim2 {
		return m, errors.New("delimiters can't be equal")
	}
	pairs := strings.SplitN(keyValStr, delim1, -1)
	for _, pair := range pairs {
		parts := strings.SplitN(pair, delim2, 2)
		if len(parts) != 2 {
			return m, errors.New("failed to parse into two parts")
		}
		key, val := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		if key == "" || val == "" {
			return m, errors.New("key or value is empty")
		}
		m[key] = val
	}
	return m, nil
}

//build the http request out of the target's config
func buildRequest(t Target) (http.Request, error) {
	if t.URL == "" {
		return http.Request{}, errors.New("empty URL")
	}
	if len(t.URL) < 8 {
		return http.Request{}, errors.New("URL too short")
	}
	//prepend "http://" if scheme not provided
	//maybe a cleaner way to do this via net.url?
	if t.URL[:7] != "http://" && t.URL[:8] != "https://" {
		t.URL = "http://" + t.URL
	}
	var urlStr string
	var err error
	//when regex set, generate urls
	if t.RegexURL {
		urlStr, err = reggen.Generate(t.URL, 10)
		if err != nil {
			return http.Request{}, fmt.Errorf("failed to parse regex: %w", err)
		}
	} else {
		urlStr = t.URL
	}
	URL, err := url.Parse(urlStr)
	if err != nil {
		return http.Request{}, fmt.Errorf("failed to parse URL %s: %w", urlStr, err)
	}
	if URL.Host == "" {
		return http.Request{}, errors.New("empty hostname")
	}

	if t.Options.DNSPrefetch {
		addrs, err := net.LookupHost(URL.Hostname())
		if err != nil {
			return http.Request{}, fmt.Errorf("failed to prefetch host %s", URL.Host)
		}
		if len(addrs) == 0 {
			return http.Request{}, fmt.Errorf("no addresses found for %s", URL.Host)
		}
		URL.Host = addrs[0]
	}

	//setup the request
	var req *http.Request
	if t.Options.BodyFilename != "" {
		fileContents, fileErr := ioutil.ReadFile(t.Options.BodyFilename)
		if fileErr != nil {
			return http.Request{}, fmt.Errorf("failed to read contents of file %s: %w", t.Options.BodyFilename, fileErr)
		}
		req, err = http.NewRequest(t.Options.Method, URL.String(), bytes.NewBuffer(fileContents))
	} else if t.Options.Body != "" {
		bodyStr := t.Options.Body
		if t.Options.RegexBody {
			bodyStr, err = reggen.Generate(t.Options.Body, 10)
			if err != nil {
				return http.Request{}, fmt.Errorf("failed to parse regex: %w", err)
			}
		}
		req, err = http.NewRequest(t.Options.Method, URL.String(), bytes.NewBuffer([]byte(bodyStr)))
	} else {
		req, err = http.NewRequest(t.Options.Method, URL.String(), nil)
	}
	if err != nil {
		return http.Request{}, fmt.Errorf("failed to create request: %w", err)
	}
	//add headers
	if t.Options.Headers != "" {
		headerMap, err := parseKeyValString(t.Options.Headers, ",", ":")
		if err != nil {
			return http.Request{}, fmt.Errorf("could not parse headers: %w", err)
		}
		for key, val := range headerMap {
			req.Header.Add(key, val)
		}
	}

	req.Header.Set("User-Agent", t.Options.UserAgent)

	//add cookies
	if t.Options.Cookies != "" {
		cookieMap, err := parseKeyValString(t.Options.Cookies, ";", "=")
		if err != nil {
			return http.Request{}, fmt.Errorf("could not parse cookies: %w", err)
		}
		for key, val := range cookieMap {
			req.AddCookie(&http.Cookie{Name: key, Value: val})
		}
	}

	if t.Options.BasicAuth != "" {
		authMap, err := parseKeyValString(t.Options.BasicAuth, ",", ":")
		if err != nil {
			return http.Request{}, fmt.Errorf("could not parse basic auth: %w", err)
		}
		for key, val := range authMap {
			req.SetBasicAuth(key, val)
			break
		}
	}
	return *req, nil
}

func createClient(target Target) *http.Client {
	tr := &http.Transport{}
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: !target.Options.EnforceSSL}
	tr.DisableCompression = !target.Options.Compress
	tr.DisableKeepAlives = !target.Options.KeepAlive
	if target.Options.NoHTTP2 {
		tr.TLSNextProto = make(map[string](func(string, *tls.Conn) http.RoundTripper))
	} else {
		_ = http2.ConfigureTransport(tr)
	}
	var timeout time.Duration
	if target.Options.Timeout != "" {
		timeout, _ = time.ParseDuration(target.Options.Timeout)
	} else {
		timeout = time.Duration(0)
	}
	client := &http.Client{Timeout: timeout, Transport: tr}
	if !target.Options.FollowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}
	return client
}
