package pinger

import (
	"log/slog"
	"strings"
)

type Target struct {
	Host string
	Name string
}

func (t Target) GetName() string {
	if t.Name != "" {
		return t.Name
	}
	return t.Host
}

func (t Target) LogValue() slog.Value {
	return slog.StringValue(t.GetName())
}

type Targets []Target

func (t Targets) LogValue() slog.Value {
	var values []string
	for _, target := range t {
		values = append(values, target.LogValue().String())
	}
	return slog.StringValue(strings.Join(values, ","))
}
