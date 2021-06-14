package pewpew

import (
	"net/http"
	"testing"
)

func TestRunRequest(t *testing.T) {
	invalidURLRequest, err := http.NewRequest("GET", "http://1234567890.0987654321", nil)
	if err != nil {
		t.Errorf("failed to create bad http request")
	}
	//TODO setup a local http server and request that instead of using github
	goodRequestWithNoBody, err := http.NewRequest("HEAD", "http://github.com", http.NoBody)
	if err != nil {
		t.Errorf("failed to create good http request with no body")
	}

	tests := []struct {
		name string
		r    http.Request
		c    *http.Client
	}{
		{
			name: "invalid url",
			r:    *invalidURLRequest,
			c:    &http.Client{},
		},
		{
			name: "valid",
			r:    *goodRequestWithNoBody,
			c:    &http.Client{},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			runRequest(tc.r, tc.c)
		})
	}
}
