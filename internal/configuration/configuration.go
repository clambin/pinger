package configuration

import (
	"github.com/clambin/pinger/pkg/pinger"
	"github.com/spf13/viper"
	"os"
	"strings"
)

type Configuration struct {
	Debug   bool
	Addr    string
	Targets []pinger.Target
}

func GetTargets(v *viper.Viper, args []string) pinger.Targets {
	if hosts := os.Getenv("HOSTS"); hosts != "" {
		return getTargetsFromEnv(hosts)
	}
	if len(args) > 0 {
		return getTargetsFromArgs(args)
	}
	return getTargetsFromViper(v)
}

func getTargetsFromEnv(hosts string) []pinger.Target {
	sep := " "
	if strings.Contains(hosts, ",") {
		sep = ","
	}
	var targets []pinger.Target
	for _, host := range strings.Split(hosts, sep) {
		targets = append(targets, pinger.Target{Host: host})
	}
	return targets
}

func getTargetsFromArgs(args []string) []pinger.Target {
	var targets []pinger.Target
	for _, arg := range args {
		targets = append(targets, pinger.Target{Host: arg})
	}
	return targets
}

func getTargetsFromViper(v *viper.Viper) []pinger.Target {
	var targets []pinger.Target
	for _, target := range v.Get("targets").([]interface{}) {
		entry := target.(map[string]interface{})
		var host, name string
		if e := entry["name"]; e != nil {
			name = e.(string)
		}
		if e := entry["host"]; e != nil {
			host = e.(string)
		}
		targets = append(targets, pinger.Target{Name: name, Host: host})
	}
	return targets
}
