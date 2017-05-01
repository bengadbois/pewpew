package pewpew

import (
	"testing"
)

func TestValidateTarget(t *testing.T) {
	cases := []struct {
		t      Target
		hasErr bool
	}{
		//multiple things uninitialized
		{Target{}, true},
		//empty method
		{Target{
			URL:     DefaultURL,
			Timeout: DefaultTimeout,
			Method:  "",
		}, true},
		//empty timeout string okay
		{Target{
			URL:     DefaultURL,
			Timeout: "",
			Method:  DefaultMethod,
		}, false},
		//invalid time string
		{Target{
			URL:     DefaultURL,
			Timeout: "unparseable",
			Method:  DefaultMethod,
		}, true},
		//timeout too short
		{Target{
			URL:     DefaultURL,
			Timeout: "1ms",
			Method:  DefaultMethod,
		}, true},
	}
	for _, c := range cases {
		err := validateTarget(c.t)
		if (err != nil) != c.hasErr {
			t.Errorf("validateTarget(%+v) err: %t wanted %t", c.t, (err != nil), c.hasErr)
		}
	}
}
