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
	tests := []struct {
		name         string
		stressConfig StressConfig
		writer       io.Writer
		expectErr    bool
	}{
		{
			name:         "empty config",
			stressConfig: StressConfig{},
			writer:       ioutil.Discard,
			expectErr:    true,
		},
		{
			name:         "empty writer",
			stressConfig: StressConfig{},
			writer:       nil,
			expectErr:    true,
		},
		{
			name:         "invalid target",
			stressConfig: StressConfig{Targets: []Target{{}}},
			writer:       ioutil.Discard,
			expectErr:    true,
		},
		{
			name:         "invalid regex",
			stressConfig: StressConfig{Count: 10, Concurrency: 1, Targets: []Target{{URL: "*(", RegexURL: true, Method: "GET"}}},
			writer:       ioutil.Discard,
			expectErr:    true,
		},
		{
			name:         "invalid url",
			stressConfig: StressConfig{Count: 10, Concurrency: 1, Targets: []Target{{URL: ":::fail", Method: "GET"}}},
			writer:       ioutil.Discard,
			expectErr:    true,
		},
		{
			name:         "valid single targets",
			stressConfig: StressConfig{Count: 1, Concurrency: 1, Targets: []Target{{URL: "http://localhost", Method: "GET"}}},
			writer:       ioutil.Discard,
			expectErr:    false,
		},
		{
			name:         "valid multiple targets",
			stressConfig: StressConfig{Count: 1, Concurrency: 1, Targets: []Target{{URL: "http://localhost", Method: "GET"}, {URL: "http://localhost", Method: "GET"}}},
			writer:       ioutil.Discard,
			expectErr:    false,
		},
		{
			name:         "valid config, handleable http error",
			stressConfig: StressConfig{Count: 1, Concurrency: 1, Targets: []Target{{URL: "http://localhost:999999999", Method: "GET"}}},
			writer:       ioutil.Discard,
			expectErr:    false,
		},
		{
			name:         "valid no HTTP2",
			stressConfig: StressConfig{Count: 1, Concurrency: 1, Targets: []Target{{URL: "http://localhost", Method: "GET"}}, NoHTTP2: true},
			writer:       ioutil.Discard,
			expectErr:    false,
		},
		{
			name:         "valid timeout",
			stressConfig: StressConfig{Count: 1, Concurrency: 1, Targets: []Target{{URL: "http://localhost", Method: "GET"}}, Timeout: "2s"},
			writer:       ioutil.Discard,
			expectErr:    false,
		},
		{
			name:         "valid following redirects",
			stressConfig: StressConfig{Count: 1, Concurrency: 1, Targets: []Target{{URL: "http://localhost", Method: "GET"}}, FollowRedirects: true},
			writer:       ioutil.Discard,
			expectErr:    false,
		},
		{
			name:         "valid not following redirects",
			stressConfig: StressConfig{Count: 1, Concurrency: 1, Targets: []Target{{URL: "http://localhost", Method: "GET"}}, FollowRedirects: false},
			writer:       ioutil.Discard,
			expectErr:    false,
		},
		{
			name:         "valid verbose",
			stressConfig: StressConfig{Count: 1, Concurrency: 1, Targets: []Target{{URL: "http://localhost", Method: "GET"}}, Verbose: true},
			writer:       ioutil.Discard,
			expectErr:    false,
		},
		{
			name:         "valid quiet",
			stressConfig: StressConfig{Count: 1, Concurrency: 1, Targets: []Target{{URL: "http://localhost", Method: "GET"}}, Quiet: true},
			writer:       ioutil.Discard,
			expectErr:    false,
		},
		{
			name:         "valid body file",
			stressConfig: StressConfig{Count: 1, Concurrency: 1, Targets: []Target{{URL: "http://localhost", Method: "GET"}}, BodyFilename: tempFilename},
			writer:       ioutil.Discard,
			expectErr:    false,
		},
		{
			name:         "valid stressConfig constructor",
			stressConfig: *NewStressConfig(),
			writer:       ioutil.Discard,
			expectErr:    false,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := RunStress(tc.stressConfig, tc.writer)
			if (err != nil) != tc.expectErr {
				t.Errorf("got error: %t, wanted: %t", (err != nil), tc.expectErr)
			}
		})
	}
}

func TestValidateStressConfig(t *testing.T) {
	tests := []struct {
		name      string
		s         StressConfig
		expectErr bool
	}{
		{
			name:      "uninitialized",
			s:         StressConfig{},
			expectErr: true,
		},
		{
			name: "count is zero",
			s: StressConfig{
				Count:       0,
				Concurrency: DefaultConcurrency,
				Targets: []Target{
					{
						URL:     DefaultURL,
						Timeout: DefaultTimeout,
						Method:  DefaultMethod,
					},
				},
			},
			expectErr: true,
		},
		{
			name: "concurrency is zero",
			s: StressConfig{
				Count:       DefaultCount,
				Concurrency: 0,
				Targets: []Target{
					{
						URL:     DefaultURL,
						Timeout: DefaultTimeout,
						Method:  DefaultMethod,
					},
				},
			},
			expectErr: true,
		},
		{
			name: "concurrency > count",
			s: StressConfig{
				Count:       10,
				Concurrency: 20,
				Targets: []Target{
					{
						URL:     DefaultURL,
						Timeout: DefaultTimeout,
						Method:  DefaultMethod,
					},
				},
			},
			expectErr: true,
		},
		{
			name: "empty method",
			s: StressConfig{
				Count:       DefaultCount,
				Concurrency: DefaultConcurrency,
				Targets: []Target{
					{
						URL:     DefaultURL,
						Timeout: DefaultTimeout,
						Method:  "",
					},
				},
			},
			expectErr: true,
		},
		{
			name: "valid empty timeout string",
			s: StressConfig{
				Count:       DefaultCount,
				Concurrency: DefaultConcurrency,
				Targets: []Target{
					{
						URL:     DefaultURL,
						Timeout: "",
						Method:  DefaultMethod,
					},
				},
			},
			expectErr: false,
		},
		{
			name: "invalid time string",
			s: StressConfig{
				Count:       DefaultCount,
				Concurrency: DefaultConcurrency,
				Targets: []Target{
					{
						URL:     DefaultURL,
						Timeout: "unparseable",
						Method:  DefaultMethod,
					},
				},
			},
			expectErr: true,
		},
		{
			name: "timeout too short",
			s: StressConfig{
				Count:       DefaultCount,
				Concurrency: DefaultConcurrency,
				Targets: []Target{
					{
						URL:     DefaultURL,
						Timeout: "1ms",
						Method:  DefaultMethod,
					},
				},
			},
			expectErr: true,
		},
		{
			name:      "valid",
			s:         *NewStressConfig(),
			expectErr: false,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateStressConfig(tc.s)
			if (err != nil) != tc.expectErr {
				t.Errorf("got error: %t, wanted: %t", (err != nil), tc.expectErr)
			}
		})
	}
}
