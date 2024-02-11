package pinger_test

import (
	"github.com/clambin/pinger/pkg/pinger"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTarget_GetName(t *testing.T) {
	assert.Equal(t, "foo", pinger.Target{Name: "foo", Host: "bar"}.GetName())
	assert.Equal(t, "bar", pinger.Target{Name: "", Host: "bar"}.GetName())
}

func TestTargets_LogValue(t *testing.T) {
	target := pinger.Targets{
		{Name: "foo", Host: "127.0.0.1"},
		{Host: "127.0.0.1"},
	}

	assert.Equal(t, "foo,127.0.0.1", target.LogValue().String())
}
