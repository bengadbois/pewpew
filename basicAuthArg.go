package main

import (
	"fmt"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	"strings"
)

type BasicAuthValue struct {
	User     string
	Password string
}

func (b *BasicAuthValue) Set(value string) error {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("expected USER:PASSWORD got '%s'", value)
	}
	b.User = parts[0]
	b.Password = parts[1]
	return nil
}

func (b *BasicAuthValue) String() string {
	if b.User == "" {
		return ""
	}
	return b.User + ":" + b.Password
}

func BasicAuth(s kingpin.Settings) (target *BasicAuthValue) {
	target = &BasicAuthValue{}
	s.SetValue((*BasicAuthValue)(target))
	return
}
