package pewpew

import (
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

const tempFilename = "/tmp/testdata"

func TestMain(m *testing.M) {
	//setup
	//create a temp file on disk for use as post body filename
	err := ioutil.WriteFile(tempFilename, []byte(""), 0644)
	if err != nil {
		os.Exit(1)
	}

	retCode := m.Run()

	//teardown
	err = os.Remove(tempFilename)
	if err != nil {
		os.Exit(1)
	}

	os.Exit(retCode)
}

func TestValidateTargets(t *testing.T) {
	cases := []struct {
		s      StressConfig
		hasErr bool
	}{
		//multiple things uninitialized
		{StressConfig{}, true},
		//zero count
		{StressConfig{
			Targets: []Target{
				{
					URL:         DefaultURL,
					Count:       0,
					Concurrency: DefaultConcurrency,
					Timeout:     DefaultTimeout,
					Method:      DefaultMethod,
				},
			},
		}, true},
		//zero concurrency
		{StressConfig{
			Targets: []Target{
				{
					URL:         DefaultURL,
					Count:       DefaultCount,
					Concurrency: 0,
					Timeout:     DefaultTimeout,
					Method:      DefaultMethod,
				},
			},
		}, true},
		//concurrency > count
		{StressConfig{
			Targets: []Target{
				{
					URL:         DefaultURL,
					Count:       10,
					Concurrency: 20,
					Timeout:     DefaultTimeout,
					Method:      DefaultMethod,
				},
			},
		}, true},
		//empty method
		{StressConfig{
			Targets: []Target{
				{
					URL:         DefaultURL,
					Count:       DefaultCount,
					Concurrency: DefaultConcurrency,
					Timeout:     DefaultTimeout,
					Method:      "",
				},
			},
		}, true},
		//empty timeout string okay
		{StressConfig{
			Targets: []Target{
				{
					URL:         DefaultURL,
					Count:       DefaultCount,
					Concurrency: DefaultConcurrency,
					Timeout:     "",
					Method:      DefaultMethod,
				},
			},
		}, false},
		//invalid time string
		{StressConfig{
			Targets: []Target{
				{
					URL:         DefaultURL,
					Count:       DefaultCount,
					Concurrency: DefaultConcurrency,
					Timeout:     "unparseable",
					Method:      DefaultMethod,
				},
			},
		}, true},
		//timeout too short
		{StressConfig{
			Targets: []Target{
				{
					URL:         DefaultURL,
					Count:       DefaultCount,
					Concurrency: DefaultConcurrency,
					Timeout:     "1ms",
					Method:      DefaultMethod,
				},
			},
		}, true},

		//good cases
		{*NewStressConfig(), false},
	}
	for _, c := range cases {
		err := validateTargets(c.s)
		if (err != nil) != c.hasErr {
			t.Errorf("validateTargets(%+v) err: %t wanted %t", c.s, (err != nil), c.hasErr)
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

		//good cases
		{Target{URL: "localhost"}, false}, //missing scheme (http://) should be auto fixed
		{Target{URL: "http://localhost:80"}, false},
		{Target{URL: "http://localhost",
			Method: "POST",
			Body:   "data"}, false},
		{Target{URL: "https://www.github.com"}, false},
		{Target{URL: "http://github.com"}, false},
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

func TestRunStress(t *testing.T) {
	cases := []struct {
		stressConfig StressConfig
		writer       io.Writer
		hasErr       bool
	}{
		{StressConfig{}, ioutil.Discard, true},                                                                                         //invalid config
		{StressConfig{}, nil, true},                                                                                                    //empty writer
		{StressConfig{Targets: []Target{{}}}, ioutil.Discard, true},                                                                    //invalid target
		{StressConfig{Targets: []Target{{URL: "*(", RegexURL: true, Method: "GET", Count: 10, Concurrency: 1}}}, ioutil.Discard, true}, //error building target, invalid regex
		{StressConfig{Targets: []Target{{URL: ":::fail", Method: "GET", Count: 10, Concurrency: 1}}}, ioutil.Discard, true},            //error building target

		//good cases
		{StressConfig{Targets: []Target{{URL: "http://localhost", Method: "GET", Count: 1, Concurrency: 1}, {URL: "http://localhost", Method: "GET", Count: 1, Concurrency: 1}}}, ioutil.Discard, false}, //multiple targets
		{StressConfig{Targets: []Target{{URL: "http://localhost", Method: "GET", Count: 1, Concurrency: 1}}}, ioutil.Discard, false},                                                                     //single target
		{StressConfig{Targets: []Target{{URL: "http://localhost:999999999", Method: "GET", Count: 1, Concurrency: 1}}}, ioutil.Discard, false},                                                           //request that should cause an http err that will get handled
		{StressConfig{Targets: []Target{{URL: "http://localhost", Method: "GET", Count: 1, Concurrency: 1}}, NoHTTP2: true}, ioutil.Discard, false},                                                      //noHTTP
		{StressConfig{Targets: []Target{{URL: "http://localhost", Method: "GET", Count: 1, Concurrency: 1}}, Timeout: "2s"}, ioutil.Discard, false},                                                      //timeout
		{StressConfig{Targets: []Target{{URL: "http://localhost", Method: "GET", Count: 1, Concurrency: 1}}, FollowRedirects: true}, ioutil.Discard, false},                                              //follow redirects
		{StressConfig{Targets: []Target{{URL: "http://localhost", Method: "GET", Count: 1, Concurrency: 1}}, FollowRedirects: false}, ioutil.Discard, false},                                             //don't follow redirects
		{StressConfig{Targets: []Target{{URL: "http://localhost", Method: "GET", Count: 1, Concurrency: 1}}, Verbose: true}, ioutil.Discard, false},                                                      //verbose
		{StressConfig{Targets: []Target{{URL: "http://localhost", Method: "GET", Count: 1, Concurrency: 1}}, Quiet: true}, ioutil.Discard, false},                                                        //quiet
		{StressConfig{Targets: []Target{{URL: "http://localhost", Method: "GET", Count: 1, Concurrency: 1}}, BodyFilename: tempFilename}, ioutil.Discard, false},                                         //body file
		{*NewStressConfig(), ioutil.Discard, false},
	}
	for _, c := range cases {
		_, err := RunStress(c.stressConfig, c.writer)
		if (err != nil) != c.hasErr {
			t.Errorf("RunStress(%+v, %q) err: %t wanted %t", c.stressConfig, c.writer, (err != nil), c.hasErr)
		}
	}
}
