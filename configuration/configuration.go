package configuration

import (
	"github.com/spf13/viper"
	"log/slog"
	"os"
	"strings"
)

type Configuration struct {
	Debug   bool
	Addr    string
	Targets []Target
}

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

func GetTargets(v *viper.Viper, args []string) Targets {
	if hosts := os.Getenv("HOSTS"); hosts != "" {
		return getTargetsFromEnv(hosts)
	}
	if len(args) > 0 {
		return getTargetsFromArgs(args)
	}
	return getTargetsFromViper(v)
}

func getTargetsFromEnv(hosts string) []Target {
	sep := " "
	if strings.Contains(hosts, ",") {
		sep = ","
	}
	var targets []Target
	for _, host := range strings.Split(hosts, sep) {
		targets = append(targets, Target{Host: host})
	}
	return targets
}

func getTargetsFromArgs(args []string) []Target {
	var targets []Target
	for _, arg := range args {
		targets = append(targets, Target{Host: arg})
	}
	return targets
}

func getTargetsFromViper(v *viper.Viper) []Target {
	var targets []Target
	for _, target := range v.Get("targets").([]interface{}) {
		entry := target.(map[string]interface{})
		var host, name string
		if e := entry["name"]; e != nil {
			name = e.(string)
		}
		if e := entry["host"]; e != nil {
			host = e.(string)
		}
		targets = append(targets, Target{Name: name, Host: host})
	}
	return targets
}
