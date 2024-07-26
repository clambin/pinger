package pinger

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTargets_LogValue(t *testing.T) {
	targets := Targets{
		{Name: "foo", Host: "example.com"},
		{Host: "example.com"},
	}
	assert.Equal(t, "foo,example.com", targets.LogValue().String())
}
