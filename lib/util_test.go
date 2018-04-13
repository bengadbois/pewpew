package pewpew

import (
	"reflect"
	"testing"
)

func TestParseKeyValString(t *testing.T) {
	cases := []struct {
		str    string
		delim1 string
		delim2 string
		want   map[string]string
		hasErr bool
	}{
		{"", "", "", map[string]string{}, true},
		{"", ":", ";", map[string]string{}, true},
		{"", ":", ":", map[string]string{}, true},
		{"abc:123;", ";", ":", map[string]string{"abc": "123"}, true},
		{"abc:123", ";", ":", map[string]string{"abc": "123"}, false},
		{"key1: val2, key3 : val4,key5:val6", ",", ":", map[string]string{"key1": "val2", "key3": "val4", "key5": "val6"}, false},
	}
	for _, c := range cases {
		result, err := parseKeyValString(c.str, c.delim1, c.delim2)
		if (err != nil) != c.hasErr {
			t.Errorf("parseKeyValString(%q, %q, %q) err: %t wanted %t", c.str, c.delim1, c.delim2, (err != nil), c.hasErr)
			continue
		}
		if err == nil && !reflect.DeepEqual(result, c.want) {
			t.Errorf("parseKeyValString(%q, %q, %q) == %v wanted %v", c.str, c.delim1, c.delim2, result, c.want)
		}
	}
}

func TestBuildRequest(t *testing.T) {
	cases := []struct {
		target Target
		hasErr bool
	}{
		{Target{}, true},                                 //empty url
		{Target{URL: ""}, true},                          //empty url
		{Target{URL: "", RegexURL: true}, true},          //empty regex url
		{Target{URL: "h"}, true},                         //hostname too short
		{Target{URL: "http://(*", RegexURL: true}, true}, //invalid regex
		{Target{URL: "http://///"}, true},                //invalid hostname
		{Target{URL: "http://%%%"}, true},                //net/url will fail parsing
		{Target{URL: "http://"}, true},                   //empty hostname
		{Target{URL: "http://localhost",
			BodyFilename: "/thisfiledoesnotexist"}, true}, //bad file
		{Target{URL: "http://localhost",
			Headers: ",,,"}, true}, //invalid headers
		{Target{URL: "http://localhost",
			Headers: "a:b,c,d"}, true}, //invalid headers
		{Target{URL: "http://localhost",
			Cookies: ";;;"}, true}, //invalid cookies
		{Target{URL: "http://localhost",
			Cookies: "a=b;c;d"}, true}, //invalid cookies
		{Target{URL: "http://localhost",
			BasicAuth: "user:"}, true}, //invalid basic auth
		{Target{URL: "http://localhost",
			BasicAuth: ":pass"}, true}, //invalid basic auth
		{Target{URL: "http://localhost",
			BasicAuth: "::"}, true}, //invalid basic auth
		{Target{URL: "http://localhost",
			Method: "@"}, true}, //invalid method
		{Target{URL: "https://invaliddomain.invalidtld",
			DNSPrefetch: true}, true},

		//good cases
		{Target{URL: "localhost"}, false}, //missing scheme (http://) should be auto fixed
		{Target{URL: "http://localhost:80"}, false},
		{Target{URL: "http://localhost",
			Method: "POST",
			Body:   "data"}, false},
		{Target{URL: "https://www.github.com"}, false},
		{Target{URL: "http://github.com"}, false},
		{Target{URL: "https://www.github.com",
			DNSPrefetch: true}, false},
		{Target{URL: "https://www.github.com:80",
			DNSPrefetch: true}, false},
		{Target{URL: "https://www.github.com:80/path/",
			DNSPrefetch: true}, false},
		{Target{URL: "http://localhost",
			BodyFilename: ""}, false},
		{Target{URL: "http://localhost",
			BodyFilename: tempFilename}, false},
		{Target{URL: "http://localhost:80/path/?param=val&another=one",
			Headers:   "Accept-Encoding:gzip, Content-Type:application/json",
			Cookies:   "a=b;c=d",
			UserAgent: "pewpewpew",
			BasicAuth: "user:pass"}, false},
	}
	for _, c := range cases {
		_, err := buildRequest(c.target)
		if (err != nil) != c.hasErr {
			t.Errorf("buildRequest(%+v) err: %t wanted: %t", c.target, (err != nil), c.hasErr)
		}
	}
}

func TestCreateClient(t *testing.T) {
	cases := []struct {
		target Target
	}{
		{Target{}}, //empty
		{Target{EnforceSSL: true}},
		{Target{EnforceSSL: false}},
		{Target{Compress: true}},
		{Target{Compress: false}},
		{Target{KeepAlive: true}},
		{Target{KeepAlive: false}},
		{Target{NoHTTP2: true}},
		{Target{NoHTTP2: false}},
		{Target{Timeout: ""}},
		{Target{Timeout: "1s"}},
		{Target{FollowRedirects: true}},
		{Target{FollowRedirects: false}},
	}
	for _, c := range cases {
		createClient(c.target)
	}
}
