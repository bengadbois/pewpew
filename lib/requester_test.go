package pewpew

import (
	"bytes"
	"net/http"
	"testing"
)

func TestRunRequest(t *testing.T) {
	badRequest, err := http.NewRequest("GET", "http://1234567890.0987654321", nil)
	if err != nil {
		t.Errorf("failed to create bad http request")
	}
	//TODO setup a local http server and request that instead of using github
	goodRequestWithNoBody, err := http.NewRequest("HEAD", "http://github.com", http.NoBody)
	if err != nil {
		t.Errorf("failed to create good http request with no body")
	}
	goodRequestWithBody, err := http.NewRequest("HEAD", "http://github.com", bytes.NewBufferString("the body"))
	if err != nil {
		t.Errorf("failed to create good http request with body")
	}
	cases := []struct {
		r http.Request
		c *http.Client
	}{
		{*badRequest, &http.Client{}},
		{*goodRequestWithBody, &http.Client{}},
		{*goodRequestWithNoBody, &http.Client{}},
	}
	for _, c := range cases {
		runRequest(c.r, c.c)
	}
}
