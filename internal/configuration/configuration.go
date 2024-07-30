package configuration

import (
	"github.com/clambin/pinger/internal/pinger"
	"github.com/spf13/viper"
	"os"
	"strings"
)

type Configuration struct {
	Addr    string
	Targets pinger.Targets
	Debug   bool
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

func getTargetsFromEnv(hosts string) pinger.Targets {
	sep := " "
	if strings.Contains(hosts, ",") {
		sep = ","
	}
	var targetList pinger.Targets
	for _, host := range strings.Split(hosts, sep) {
		targetList = append(targetList, pinger.Target{Host: host})
	}
	return targetList
}

func getTargetsFromArgs(args []string) pinger.Targets {
	var targetList pinger.Targets
	for _, arg := range args {
		targetList = append(targetList, pinger.Target{Host: arg})
	}
	return targetList
}

func getTargetsFromViper(v *viper.Viper) pinger.Targets {
	var targetList pinger.Targets
	viperVal := v.Get("targets")
	for _, t := range viperVal.([]any) {
		entry := t.(map[string]any)
		var host, name string
		if e := entry["name"]; e != nil {
			name = e.(string)
		}
		if e := entry["host"]; e != nil {
			host = e.(string)
		}
		targetList = append(targetList, pinger.Target{Name: name, Host: host})
	}
	return targetList
}
