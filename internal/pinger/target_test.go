package pinger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTargets_LogValue(t *testing.T) {
	targets := Targets{
		{Name: "localhost", Host: "127.0.0.1"},
		{Name: "example.com", Host: "www.example.com"},
	}
	assert.Equal(t, "localhost,example.com", targets.LogValue().String())
}
