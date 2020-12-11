package main

import (
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/node_exporter/pkg/consul"
	"os"
	"strconv"
	"strings"

	"github.com/go-kit/kit/log/level"
)

func InitConsul(consulUrl, serviceName, ipPort string) {
	promlogConfig := &promlog.Config{}

	logger := promlog.New(promlogConfig)

	ip := strings.Split(ipPort, ":")
	if len(ip) != 2 {
		level.Error(logger).Log("ipPort not len 2")
		os.Exit(1)
	}

	port, err := strconv.Atoi(ip[1])
	if err != nil {
		level.Error(logger).Log("strconv: %s Atoi err: %v", ip[1], err)
		os.Exit(1)
	}

	sd, err := consul.NewServiceDiscovery(serviceName, ip[0], uint64(port), 10, 10, "ok", consulUrl)
	if err != nil {
		level.Error(logger).Log("consul new service discovery err: %v", err)
		os.Exit(1)
	}
	if err := sd.ServiceRegistr(); err != nil {
		level.Error(logger).Log("service register err: %v", err)
		os.Exit(1)
	}
}
