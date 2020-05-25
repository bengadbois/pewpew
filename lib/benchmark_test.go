package pewpew

import (
	"io"
	"io/ioutil"
	"testing"
)

func TestRunBenchmark(t *testing.T) {
	cases := []struct {
		benchmarkConfig BenchmarkConfig
		writer          io.Writer
		hasErr          bool
	}{
		{BenchmarkConfig{}, ioutil.Discard, true},                      //invalid config
		{BenchmarkConfig{}, nil, true},                                 //empty writer
		{BenchmarkConfig{Targets: []Target{{}}}, ioutil.Discard, true}, //invalid target
		{BenchmarkConfig{RPS: 10, Duration: 1, Targets: []Target{{URL: "*(", RegexURL: true, Method: "GET"}}}, ioutil.Discard, true}, //error building target, invalid regex
		{BenchmarkConfig{RPS: 10, Duration: 1, Targets: []Target{{URL: ":::fail", Method: "GET"}}}, ioutil.Discard, true},            //error building target, invalid url

		//good cases
		{BenchmarkConfig{RPS: 1, Duration: 1, Targets: []Target{{URL: "http://localhost", Method: "GET"}, {URL: "http://localhost", Method: "GET"}}}, ioutil.Discard, false}, //multiple targets
		{BenchmarkConfig{RPS: 1, Duration: 1, Targets: []Target{{URL: "http://localhost", Method: "GET"}}}, ioutil.Discard, false},                                           //single target
		{BenchmarkConfig{RPS: 1, Duration: 1, Targets: []Target{{URL: "http://localhost:999999999", Method: "GET"}}}, ioutil.Discard, false},                                 //request that should cause an http err that will get handled
		{BenchmarkConfig{RPS: 1, Duration: 1, Targets: []Target{{URL: "http://localhost", Method: "GET"}}, NoHTTP2: true}, ioutil.Discard, false},                            //noHTTP
		{BenchmarkConfig{RPS: 1, Duration: 1, Targets: []Target{{URL: "http://localhost", Method: "GET"}}, Timeout: "2s"}, ioutil.Discard, false},                            //timeout
		{BenchmarkConfig{RPS: 1, Duration: 1, Targets: []Target{{URL: "http://localhost", Method: "GET"}}, FollowRedirects: true}, ioutil.Discard, false},                    //follow redirects
		{BenchmarkConfig{RPS: 1, Duration: 1, Targets: []Target{{URL: "http://localhost", Method: "GET"}}, FollowRedirects: false}, ioutil.Discard, false},                   //don't follow redirects
		{BenchmarkConfig{RPS: 1, Duration: 1, Targets: []Target{{URL: "http://localhost", Method: "GET"}}, Verbose: true}, ioutil.Discard, false},                            //verbose
		{BenchmarkConfig{RPS: 1, Duration: 1, Targets: []Target{{URL: "http://localhost", Method: "GET"}}, Quiet: true}, ioutil.Discard, false},                              //quiet
		{BenchmarkConfig{RPS: 1, Duration: 1, Targets: []Target{{URL: "http://localhost", Method: "GET"}}, BodyFilename: tempFilename}, ioutil.Discard, false},               //body file
		{*NewBenchmarkConfig(), ioutil.Discard, false},
	}
	for _, c := range cases {
		_, err := RunBenchmark(c.benchmarkConfig, c.writer)
		if (err != nil) != c.hasErr {
			t.Errorf("RunBenchmark(%+v, %q) err: %t wanted %t", c.benchmarkConfig, c.writer, (err != nil), c.hasErr)
		}
	}
}

func TestValidateBenchmarkConfig(t *testing.T) {
	cases := []struct {
		s      BenchmarkConfig
		hasErr bool
	}{
		//multiple things uninitialized
		{BenchmarkConfig{}, true},
		//zero rps
		{BenchmarkConfig{
			RPS:      0,
			Duration: DefaultDuration,
			Targets: []Target{
				{
					URL:     DefaultURL,
					Timeout: DefaultTimeout,
					Method:  DefaultMethod,
				},
			},
		}, true},
		//zero duration
		{BenchmarkConfig{
			RPS:      DefaultRPS,
			Duration: 0,
			Targets: []Target{
				{
					URL:     DefaultURL,
					Timeout: DefaultTimeout,
					Method:  DefaultMethod,
				},
			},
		}, true},
		//empty method
		{BenchmarkConfig{
			RPS:      DefaultRPS,
			Duration: DefaultDuration,
			Targets: []Target{
				{
					URL:     DefaultURL,
					Timeout: DefaultTimeout,
					Method:  "",
				},
			},
		}, true},
		//empty timeout string okay
		{BenchmarkConfig{
			RPS:      DefaultRPS,
			Duration: DefaultDuration,
			Targets: []Target{
				{
					URL:     DefaultURL,
					Timeout: "",
					Method:  DefaultMethod,
				},
			},
		}, false},
		//invalid time string
		{BenchmarkConfig{
			RPS:      DefaultRPS,
			Duration: DefaultDuration,
			Targets: []Target{
				{
					URL:     DefaultURL,
					Timeout: "unparseable",
					Method:  DefaultMethod,
				},
			},
		}, true},
		//timeout too short
		{BenchmarkConfig{
			RPS:      DefaultRPS,
			Duration: DefaultDuration,
			Targets: []Target{
				{
					URL:     DefaultURL,
					Timeout: "1ms",
					Method:  DefaultMethod,
				},
			},
		}, true},

		//good cases
		{*NewBenchmarkConfig(), false},
	}
	for _, c := range cases {
		err := validateBenchmarkConfig(c.s)
		if (err != nil) != c.hasErr {
			t.Errorf("validateBenchmarkConfig(%+v) err: %t wanted %t", c.s, (err != nil), c.hasErr)
		}
	}
}
