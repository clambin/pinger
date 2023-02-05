package configuration_test

import (
	"bytes"
	"github.com/clambin/pinger/configuration"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

const body = `
debug: true
addr: :8080
targets:
  - name: foo
    host: foo
  - host: bar
  - name: localhost
    host: 127.0.0.1
`

func TestUnmarshal(t *testing.T) {
	v := viper.New()
	v.SetConfigType("yaml")

	err := v.ReadConfig(bytes.NewBufferString(body))
	require.NoError(t, err)

	var cfg configuration.Configuration
	err = v.Unmarshal(&cfg)
	require.NoError(t, err)

	assert.Equal(t, configuration.Configuration{
		Debug: true,
		Addr:  ":8080",
		Targets: []configuration.Target{
			{Name: "foo", Host: "foo"},
			{Name: "", Host: "bar"},
			{Name: "localhost", Host: "127.0.0.1"},
		},
	}, cfg)
}

func TestGetTargets(t *testing.T) {
	tests := []struct {
		name     string
		hosts    string
		args     []string
		expected configuration.Targets
		logEntry string
	}{
		{
			name:  "environment variable (spaces)",
			hosts: "127.0.0.1 google.com",
			expected: configuration.Targets{
				{Host: "127.0.0.1"},
				{Host: "google.com"},
			},
			logEntry: "127.0.0.1,google.com",
		},
		{
			name:  "environment variable (commas)",
			hosts: "127.0.0.1,google.com",
			expected: configuration.Targets{
				{Host: "127.0.0.1"},
				{Host: "google.com"},
			},
			logEntry: "127.0.0.1,google.com",
		},
		{
			name: "args",
			args: []string{"google.com", "127.0.0.1"},
			expected: configuration.Targets{
				{Host: "google.com"},
				{Host: "127.0.0.1"},
			},
			logEntry: "google.com,127.0.0.1",
		},
		{
			name: "config file",
			expected: configuration.Targets{
				{Name: "foo", Host: "foo"},
				{Name: "", Host: "bar"},
				{Name: "localhost", Host: "127.0.0.1"},
			},
			logEntry: "foo,bar,localhost",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := viper.New()
			v.SetConfigType("yaml")

			err := v.ReadConfig(bytes.NewBufferString(body))
			require.NoError(t, err)

			if tt.hosts != "" {
				require.NoError(t, os.Setenv("HOSTS", tt.hosts))
				defer func() { require.NoError(t, os.Setenv("HOSTS", "")) }()
			}

			targets := configuration.GetTargets(v, tt.args)
			assert.Equal(t, tt.expected, targets)
			assert.Equal(t, tt.logEntry, targets.LogValue().String())
		})
	}
}

func TestTarget_GetName(t *testing.T) {
	assert.Equal(t, "foo", configuration.Target{Name: "foo", Host: "bar"}.GetName())
	assert.Equal(t, "bar", configuration.Target{Name: "", Host: "bar"}.GetName())
}
