package pewpew

import (
	"testing"
)

func TestValidateTarget(t *testing.T) {
	tests := []struct {
		name      string
		t         Target
		expectErr bool
	}{
		{
			name:      "uninitialized target",
			t:         Target{},
			expectErr: true,
		},
		{
			name: "empty method",
			t: Target{
				URL:     DefaultURL,
				Timeout: DefaultTimeout,
				Method:  "",
			},
			expectErr: true,
		},
		{
			name: "valid empty timeout",
			t: Target{
				URL:     DefaultURL,
				Timeout: "",
				Method:  DefaultMethod,
			},
			expectErr: false,
		},
		{
			name: "unparseable string",
			t: Target{
				URL:     DefaultURL,
				Timeout: "unparseable",
				Method:  DefaultMethod,
			},
			expectErr: true,
		},
		{
			name: "timeout too short",
			t: Target{
				URL:     DefaultURL,
				Timeout: "1ms",
				Method:  DefaultMethod,
			},
			expectErr: true,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateTarget(tc.t)
			if (err != nil) != tc.expectErr {
				t.Errorf("got error: %t, wanted: %t", (err != nil), tc.expectErr)
			}
		})
	}
}
