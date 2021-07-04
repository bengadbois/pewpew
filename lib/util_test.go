package pewpew

import (
	"reflect"
	"testing"
)

func TestParseKeyValString(t *testing.T) {
	tests := []struct {
		name      string
		str       string
		delim1    string
		delim2    string
		want      map[string]string
		expectErr bool
	}{
		{
			name:      "empty string, empty delimiters",
			str:       "",
			delim1:    "",
			delim2:    "",
			want:      map[string]string{},
			expectErr: true,
		},
		{
			name:      "empty string, delimiters set",
			str:       "",
			delim1:    ":",
			delim2:    ";",
			want:      map[string]string{},
			expectErr: true,
		},
		{
			name:      "empty string, matching delimiters",
			str:       "",
			delim1:    ":",
			delim2:    ":",
			want:      map[string]string{},
			expectErr: true,
		},
		{
			name:      "trailing delimiter; empty key-val",
			str:       "abc:123;",
			delim1:    ";",
			delim2:    ":",
			want:      map[string]string{"abc": "123"},
			expectErr: true,
		},
		{
			name:      "single key val pair",
			str:       "abc:123",
			delim1:    ";",
			delim2:    ":",
			want:      map[string]string{"abc": "123"},
			expectErr: false,
		},
		{
			name:      "multiple key val pairs, inconsistent whitespace",
			str:       "key1: val2, key3 : val4,key5:val6",
			delim1:    ",",
			delim2:    ":",
			want:      map[string]string{"key1": "val2", "key3": "val4", "key5": "val6"},
			expectErr: false,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := parseKeyValString(tc.str, tc.delim1, tc.delim2)
			if (err != nil) != tc.expectErr {
				t.Errorf("got error: %t, wanted: %t", (err != nil), tc.expectErr)
			}
			if err == nil && !reflect.DeepEqual(result, tc.want) {
				t.Errorf("got result: %v, wanted: %v", result, tc.want)
			}
		})
	}
}

func TestBuildRequest(t *testing.T) {
	tests := []struct {
		name      string
		target    Target
		expectErr bool
	}{
		{
			name:      "empty url",
			target:    Target{},
			expectErr: true,
		},
		{
			name:      "empty url",
			target:    Target{URL: ""},
			expectErr: true,
		},
		{
			name:      "empty regex url",
			target:    Target{URL: "", RegexURL: true},
			expectErr: true,
		},
		{
			name:      "hostname too short",
			target:    Target{URL: "h"},
			expectErr: true,
		},
		{
			name:      "invalid regex",
			target:    Target{URL: "http://(*", RegexURL: true},
			expectErr: true,
		},
		{
			name:      "invalid hostname",
			target:    Target{URL: "http://///"},
			expectErr: true,
		},
		{
			name:      "unparseable url",
			target:    Target{URL: "http://%%%"},
			expectErr: true,
		},
		{
			name:      "empty hostname",
			target:    Target{URL: "http://"},
			expectErr: true,
		},
		{
			name: "attached non-existent body file",
			target: Target{
				URL: "http://localhost",
				Options: TargetOptions{
					BodyFilename: "/thisfiledoesnotexist",
				},
			},
			expectErr: true,
		},
		{
			name: "invalid headers, empty key-values",
			target: Target{
				URL: "http://localhost",
				Options: TargetOptions{
					Headers: ",,,",
				},
			},
			expectErr: true,
		},
		{
			name: "invalid headers, invalid key-value format",
			target: Target{
				URL: "http://localhost",
				Options: TargetOptions{
					Headers: "a:b,c,d",
				},
			},
			expectErr: true,
		},
		{
			name: "invalid cookies, empty key-values",
			target: Target{
				URL: "http://localhost",
				Options: TargetOptions{
					Cookies: ";;;",
				},
			},
			expectErr: true,
		},
		{
			name: "invalid cookies, invalid key-value format",
			target: Target{
				URL: "http://localhost",
				Options: TargetOptions{
					Cookies: "a=b;c;d",
				},
			},
			expectErr: true,
		},
		{
			name: "invalid basic auth, missing password",
			target: Target{
				URL: "http://localhost",
				Options: TargetOptions{
					BasicAuth: "user:",
				},
			},
			expectErr: true,
		},
		{
			name: "invalid basic auth, missing user",
			target: Target{
				URL: "http://localhost",
				Options: TargetOptions{
					BasicAuth: ":pass",
				},
			},
			expectErr: true,
		},
		{
			name: "invalid basic auth, missing user and password",
			target: Target{
				URL: "http://localhost",
				Options: TargetOptions{
					BasicAuth: "::",
				},
			},
			expectErr: true,
		},
		{
			name: "invalid method",
			target: Target{
				URL: "http://localhost",
				Options: TargetOptions{
					Method: "@",
				},
			},
			expectErr: true,
		},
		{
			name: "invalid address",
			target: Target{
				URL: "https://invaliddomain.invalidtld",
				Options: TargetOptions{
					DNSPrefetch: true,
				},
			},
			expectErr: true,
		},
		{
			name: "valid omitted scheme",
			target: Target{
				URL: "localhost",
				Options: TargetOptions{
					DNSPrefetch: true,
				},
			},
			expectErr: false,
		},
		{
			name: "valid localhost and port",
			target: Target{
				URL: "http://localhost:80",
				Options: TargetOptions{
					DNSPrefetch: true,
				},
			},
			expectErr: false,
		},
		{
			name: "valid localhost without port",
			target: Target{
				URL: "http://localhost",
				Options: TargetOptions{
					DNSPrefetch: true,
				},
			},
			expectErr: false,
		},
		{
			name: "valid http address with www",
			target: Target{
				URL: "https://www.github.com",
				Options: TargetOptions{
					DNSPrefetch: true,
				},
			},
			expectErr: false,
		},
		{
			name: "valid http address without www",
			target: Target{
				URL: "http://github.com",
				Options: TargetOptions{
					DNSPrefetch: true,
				},
			},
			expectErr: false,
		},
		{
			name: "valid https address",
			target: Target{
				URL: "https://www.github.com",
				Options: TargetOptions{
					DNSPrefetch: true,
				},
			},
			expectErr: false,
		},
		{
			name: "valid https address with port",
			target: Target{
				URL: "https://www.github.com:80",
				Options: TargetOptions{
					DNSPrefetch: true,
				},
			},
			expectErr: false,
		},
		{
			name: "valid https address with port and path",
			target: Target{
				URL: "https://www.github.com:80/path/",
			},
			expectErr: false,
		},
		{
			name: "valid empty body file",
			target: Target{
				URL: "http://localhost",
				Options: TargetOptions{
					BodyFilename: "",
				},
			},
			expectErr: false,
		},
		{
			name: "valid non-empty body file",
			target: Target{
				URL: "http://localhost",
				Options: TargetOptions{
					BodyFilename: tempFilename,
				},
			},
			expectErr: false,
		},
		{
			name: "valid headers, cookies, useragent, and basicauth",
			target: Target{
				URL: "http://localhost:80/path/?param=val&another=one",
				Options: TargetOptions{
					Headers:   "Accept-Encoding:gzip, Content-Type:application/json",
					Cookies:   "a=b;c=d",
					UserAgent: "pewpewpew",
					BasicAuth: "user:pass",
				},
			},
			expectErr: false,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := buildRequest(tc.target)
			if (err != nil) != tc.expectErr {
				t.Errorf("got error: %t, wanted: %t", (err != nil), tc.expectErr)
			}
		})
	}
}

func TestCreateClient(t *testing.T) {
	tests := []struct {
		name   string
		target Target
	}{
		{
			name:   "empty",
			target: Target{},
		},
		{
			name: "enforce ssl",
			target: Target{
				Options: TargetOptions{
					EnforceSSL: true,
				},
			},
		},
		{
			name: "don't enforce ssl",
			target: Target{
				Options: TargetOptions{
					EnforceSSL: false,
				},
			},
		},
		{
			name: "compress",
			target: Target{
				Options: TargetOptions{
					Compress: true,
				},
			},
		},
		{
			name: "don't compress",
			target: Target{
				Options: TargetOptions{
					Compress: false,
				},
			},
		},
		{
			name: "keealive",
			target: Target{
				Options: TargetOptions{
					KeepAlive: true,
				},
			},
		},
		{
			name: "don't keepalive",
			target: Target{
				Options: TargetOptions{
					KeepAlive: false,
				},
			},
		},
		{
			name: "no HTTP2",
			target: Target{
				Options: TargetOptions{
					NoHTTP2: true,
				},
			},
		},
		{
			name: "allow HTTP2",
			target: Target{
				Options: TargetOptions{
					NoHTTP2: false,
				},
			},
		},
		{
			name: "empty timeout",
			target: Target{
				Options: TargetOptions{
					Timeout: "",
				},
			},
		},
		{
			name: "non-empty timeout",
			target: Target{
				Options: TargetOptions{
					Timeout: "1s",
				},
			},
		},
		{
			name: "follow redirects",
			target: Target{
				Options: TargetOptions{
					FollowRedirects: true,
				},
			},
		},
		{
			name: "don't follow redirects",
			target: Target{
				Options: TargetOptions{
					FollowRedirects: false,
				},
			},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			createClient(tc.target)
		})
	}
}
