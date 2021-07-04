package pewpew

import (
	"io"
	"io/ioutil"
	"testing"
)

func TestRunBenchmark(t *testing.T) {
	tests := []struct {
		name            string
		benchmarkConfig BenchmarkConfig
		writer          io.Writer
		expectErr       bool
	}{
		{
			name:            "empty config",
			benchmarkConfig: BenchmarkConfig{},
			writer:          ioutil.Discard,
			expectErr:       true,
		},
		{
			name:            "empty writer",
			benchmarkConfig: BenchmarkConfig{},
			writer:          nil,
			expectErr:       true,
		},
		{
			name: "empty target",
			benchmarkConfig: BenchmarkConfig{
				Targets: []Target{{}},
			},
			writer:    ioutil.Discard,
			expectErr: true,
		},
		{
			name: "invalid regex",
			benchmarkConfig: BenchmarkConfig{
				RPS:      10,
				Duration: 1,
				Targets: []Target{
					{
						URL:      "*(",
						RegexURL: true,
						Options: TargetOptions{
							Method: "GET",
						},
					},
				},
			},
			writer:    ioutil.Discard,
			expectErr: true,
		},
		{
			name: "invalid url",
			benchmarkConfig: BenchmarkConfig{
				RPS:      10,
				Duration: 1,
				Targets: []Target{
					{
						URL: ":::fail",
						Options: TargetOptions{
							Method: "GET",
						},
					},
				},
			},
			writer:    ioutil.Discard,
			expectErr: true,
		},
		{
			name: "multiple targets",
			benchmarkConfig: BenchmarkConfig{
				RPS:      1,
				Duration: 1,
				Targets: []Target{
					{
						URL: "http://localhost",
						Options: TargetOptions{
							Method: "GET",
						},
					},
					{
						URL: "http://localhost",
						Options: TargetOptions{
							Method: "GET",
						},
					},
				},
			},
			writer:    ioutil.Discard,
			expectErr: false,
		},
		{
			name: "single target",
			benchmarkConfig: BenchmarkConfig{
				RPS:      1,
				Duration: 1,
				Targets: []Target{
					{
						URL: "http://localhost",
						Options: TargetOptions{
							Method: "GET",
						},
					},
				},
			},
			writer:    ioutil.Discard,
			expectErr: false,
		},
		{
			name: "expected http error",
			benchmarkConfig: BenchmarkConfig{
				RPS:      1,
				Duration: 1,
				Targets: []Target{
					{
						URL: "http://localhost:999999999",
						Options: TargetOptions{
							Method: "GET",
						},
					},
				},
			},
			writer:    ioutil.Discard,
			expectErr: false,
		},
		{
			name: "No HTTP2",
			benchmarkConfig: BenchmarkConfig{
				RPS:      1,
				Duration: 1,
				Targets: []Target{
					{
						URL: "http://localhost",
						Options: TargetOptions{
							Method: "GET",
						},
					},
				},
				Options: TargetOptions{
					NoHTTP2: true,
				},
			},
			writer:    ioutil.Discard,
			expectErr: false,
		},
		{
			name: "timeout",
			benchmarkConfig: BenchmarkConfig{
				RPS:      1,
				Duration: 1,
				Targets: []Target{
					{
						URL: "http://localhost",
						Options: TargetOptions{
							Method: "GET",
						},
					},
				},
				Options: TargetOptions{
					Timeout: "2s",
				},
			},
			writer:    ioutil.Discard,
			expectErr: false,
		},
		{
			name: "follow redirects",
			benchmarkConfig: BenchmarkConfig{
				RPS:      1,
				Duration: 1,
				Targets: []Target{
					{
						URL: "http://localhost",
						Options: TargetOptions{
							Method: "GET",
						},
					},
				},
				Options: TargetOptions{
					FollowRedirects: true,
				},
			},
			writer:    ioutil.Discard,
			expectErr: false,
		},
		{
			name: "don't follow redirects",
			benchmarkConfig: BenchmarkConfig{
				RPS:      1,
				Duration: 1,
				Targets: []Target{
					{
						URL: "http://localhost",
						Options: TargetOptions{
							Method: "GET",
						},
					},
				},
				Options: TargetOptions{
					FollowRedirects: false,
				},
			},
			writer:    ioutil.Discard,
			expectErr: false,
		},
		{
			name: "verbose",
			benchmarkConfig: BenchmarkConfig{
				RPS:      1,
				Duration: 1,
				Targets: []Target{
					{
						URL: "http://localhost",
						Options: TargetOptions{
							Method: "GET",
						},
					},
				},
				Verbose: true,
			},
			writer:    ioutil.Discard,
			expectErr: false,
		},
		{
			name: "quiet",
			benchmarkConfig: BenchmarkConfig{
				RPS:      1,
				Duration: 1,
				Targets: []Target{
					{
						URL: "http://localhost",
						Options: TargetOptions{
							Method: "GET",
						},
					},
				},
				Quiet: true,
			},
			writer:    ioutil.Discard,
			expectErr: false,
		},
		{
			name: "body file",
			benchmarkConfig: BenchmarkConfig{
				RPS:      1,
				Duration: 1,
				Targets: []Target{
					{
						URL: "http://localhost",
						Options: TargetOptions{
							Method: "GET",
						},
					},
				},
				Options: TargetOptions{
					BodyFilename: tempFilename,
				},
			},
			writer:    ioutil.Discard,
			expectErr: false,
		},
		{
			name:            "BenchmarkConfig constructor",
			benchmarkConfig: *NewBenchmarkConfig(),
			writer:          ioutil.Discard,
			expectErr:       false,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := RunBenchmark(tc.benchmarkConfig, tc.writer)
			if (err != nil) != tc.expectErr {
				t.Errorf("got error: %t, wanted: %t", (err != nil), tc.expectErr)
			}
		})
	}
}

func TestValidateBenchmarkConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    BenchmarkConfig
		expectErr bool
	}{
		{
			name:      "uninitialized",
			config:    BenchmarkConfig{},
			expectErr: true,
		},
		{
			name: "zero rps",
			config: BenchmarkConfig{
				RPS:      0,
				Duration: DefaultDuration,
				Targets: []Target{
					{
						URL: DefaultURL,
						Options: TargetOptions{
							Timeout: DefaultTimeout,
							Method:  DefaultMethod,
						},
					},
				},
			},
			expectErr: true,
		},
		{
			name: "zero duration",
			config: BenchmarkConfig{
				RPS:      DefaultRPS,
				Duration: 0,
				Targets: []Target{
					{
						URL: DefaultURL,
						Options: TargetOptions{
							Timeout: DefaultTimeout,
							Method:  DefaultMethod,
						},
					},
				},
			},
			expectErr: true,
		},
		{
			name: "empty method",
			config: BenchmarkConfig{
				RPS:      DefaultRPS,
				Duration: DefaultDuration,
				Targets: []Target{
					{
						URL: DefaultURL,
						Options: TargetOptions{
							Timeout: DefaultTimeout,
							Method:  "",
						},
					},
				},
			},
			expectErr: true,
		},
		{
			name: "empty timeout",
			config: BenchmarkConfig{
				RPS:      DefaultRPS,
				Duration: DefaultDuration,
				Targets: []Target{
					{
						URL: DefaultURL,
						Options: TargetOptions{
							Timeout: "",
							Method:  DefaultMethod,
						},
					},
				},
			},
			expectErr: false,
		},
		{
			name: "unparseable time string",
			config: BenchmarkConfig{
				RPS:      DefaultRPS,
				Duration: DefaultDuration,
				Targets: []Target{
					{
						URL: DefaultURL,
						Options: TargetOptions{
							Timeout: "unparseable",
							Method:  DefaultMethod,
						},
					},
				},
			},
			expectErr: true,
		},
		{
			name: "timeout too short",
			config: BenchmarkConfig{
				RPS:      DefaultRPS,
				Duration: DefaultDuration,
				Targets: []Target{
					{
						URL: DefaultURL,
						Options: TargetOptions{
							Timeout: "1ms",
							Method:  DefaultMethod,
						},
					},
				},
			},
			expectErr: true,
		},
		{
			name:      "valid",
			config:    *NewBenchmarkConfig(),
			expectErr: false,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateBenchmarkConfig(tc.config)
			if (err != nil) != tc.expectErr {
				t.Errorf("got error: %t, wanted: %t", (err != nil), tc.expectErr)
			}
		})
	}
}
