package pinger

import (
	"log/slog"
	"strings"
)

type Target struct {
	Name string
	Host string
}

var _ slog.LogValuer = Targets{}

type Targets []Target

func (t Targets) LogValue() slog.Value {
	values := make([]string, len(t))
	for i, target := range t {
		val := target.Name
		if val == "" {
			val = target.Host
		}
		values[i] = val
	}
	return slog.StringValue(strings.Join(values, ","))
}
