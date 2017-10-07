package pewpew

import (
	"io"
	"io/ioutil"
	"os"
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

func TestRunStress(t *testing.T) {
	cases := []struct {
		stressConfig StressConfig
		writer       io.Writer
		hasErr       bool
	}{
		{StressConfig{}, ioutil.Discard, true},                                                                                         //invalid config
		{StressConfig{}, nil, true},                                                                                                    //empty writer
		{StressConfig{Targets: []Target{{}}}, ioutil.Discard, true},                                                                    //invalid target
		{StressConfig{Count: 10, Concurrency: 1, Targets: []Target{{URL: "*(", RegexURL: true, Method: "GET"}}}, ioutil.Discard, true}, //error building target, invalid regex
		{StressConfig{Count: 10, Concurrency: 1, Targets: []Target{{URL: ":::fail", Method: "GET"}}}, ioutil.Discard, true},            //error building target

		//good cases
		{StressConfig{Count: 1, Concurrency: 1, Targets: []Target{{URL: "http://localhost", Method: "GET"}, {URL: "http://localhost", Method: "GET"}}}, ioutil.Discard, false}, //multiple targets
		{StressConfig{Count: 1, Concurrency: 1, Targets: []Target{{URL: "http://localhost", Method: "GET"}}}, ioutil.Discard, false},                                           //single target
		{StressConfig{Count: 1, Concurrency: 1, Targets: []Target{{URL: "http://localhost:999999999", Method: "GET"}}}, ioutil.Discard, false},                                 //request that should cause an http err that will get handled
		{StressConfig{Count: 1, Concurrency: 1, Targets: []Target{{URL: "http://localhost", Method: "GET"}}, NoHTTP2: true}, ioutil.Discard, false},                            //noHTTP
		{StressConfig{Count: 1, Concurrency: 1, Targets: []Target{{URL: "http://localhost", Method: "GET"}}, Timeout: "2s"}, ioutil.Discard, false},                            //timeout
		{StressConfig{Count: 1, Concurrency: 1, Targets: []Target{{URL: "http://localhost", Method: "GET"}}, FollowRedirects: true}, ioutil.Discard, false},                    //follow redirects
		{StressConfig{Count: 1, Concurrency: 1, Targets: []Target{{URL: "http://localhost", Method: "GET"}}, FollowRedirects: false}, ioutil.Discard, false},                   //don't follow redirects
		{StressConfig{Count: 1, Concurrency: 1, Targets: []Target{{URL: "http://localhost", Method: "GET"}}, Verbose: true}, ioutil.Discard, false},                            //verbose
		{StressConfig{Count: 1, Concurrency: 1, Targets: []Target{{URL: "http://localhost", Method: "GET"}}, Quiet: true}, ioutil.Discard, false},                              //quiet
		{StressConfig{Count: 1, Concurrency: 1, Targets: []Target{{URL: "http://localhost", Method: "GET"}}, BodyFilename: tempFilename}, ioutil.Discard, false},               //body file
		{*NewStressConfig(), ioutil.Discard, false},
	}
	for _, c := range cases {
		_, err := RunStress(c.stressConfig, c.writer)
		if (err != nil) != c.hasErr {
			t.Errorf("RunStress(%+v, %q) err: %t wanted %t", c.stressConfig, c.writer, (err != nil), c.hasErr)
		}
	}
}

func TestValidateStressConfig(t *testing.T) {
	cases := []struct {
		s      StressConfig
		hasErr bool
	}{
		//multiple things uninitialized
		{StressConfig{}, true},
		//zero count
		{StressConfig{
			Count:       0,
			Concurrency: DefaultConcurrency,
			Targets: []Target{
				{
					URL:     DefaultURL,
					Timeout: DefaultTimeout,
					Method:  DefaultMethod,
				},
			},
		}, true},
		//zero concurrency
		{StressConfig{
			Count:       DefaultCount,
			Concurrency: 0,
			Targets: []Target{
				{
					URL:     DefaultURL,
					Timeout: DefaultTimeout,
					Method:  DefaultMethod,
				},
			},
		}, true},
		//concurrency > count
		{StressConfig{
			Count:       10,
			Concurrency: 20,
			Targets: []Target{
				{
					URL:     DefaultURL,
					Timeout: DefaultTimeout,
					Method:  DefaultMethod,
				},
			},
		}, true},
		//empty method
		{StressConfig{
			Count:       DefaultCount,
			Concurrency: DefaultConcurrency,
			Targets: []Target{
				{
					URL:     DefaultURL,
					Timeout: DefaultTimeout,
					Method:  "",
				},
			},
		}, true},
		//empty timeout string okay
		{StressConfig{
			Count:       DefaultCount,
			Concurrency: DefaultConcurrency,
			Targets: []Target{
				{
					URL:     DefaultURL,
					Timeout: "",
					Method:  DefaultMethod,
				},
			},
		}, false},
		//invalid time string
		{StressConfig{
			Count:       DefaultCount,
			Concurrency: DefaultConcurrency,
			Targets: []Target{
				{
					URL:     DefaultURL,
					Timeout: "unparseable",
					Method:  DefaultMethod,
				},
			},
		}, true},
		//timeout too short
		{StressConfig{
			Count:       DefaultCount,
			Concurrency: DefaultConcurrency,
			Targets: []Target{
				{
					URL:     DefaultURL,
					Timeout: "1ms",
					Method:  DefaultMethod,
				},
			},
		}, true},

		//good cases
		{*NewStressConfig(), false},
	}
	for _, c := range cases {
		err := validateStressConfig(c.s)
		if (err != nil) != c.hasErr {
			t.Errorf("validateStressConfig(%+v) err: %t wanted %t", c.s, (err != nil), c.hasErr)
		}
	}
}
