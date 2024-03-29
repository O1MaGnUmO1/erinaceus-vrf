package main

import (
	"github.com/hashicorp/go-plugin"

	"github.com/O1MaGnUmO1/chainlink-common/pkg/loop"
	"github.com/O1MaGnUmO1/chainlink-common/pkg/loop/reportingplugins"
	"github.com/O1MaGnUmO1/chainlink-common/pkg/types"
	"github.com/O1MaGnUmO1/erinaceus-vrf/plugins/medianpoc"
)

const (
	loggerName = "PluginMedianPoc"
)

func main() {
	s := loop.MustNewStartedServer(loggerName)
	defer s.Stop()

	p := medianpoc.NewPlugin(s.Logger)
	defer s.Logger.ErrorIfFn(p.Close, "Failed to close")

	s.MustRegister(p)

	stop := make(chan struct{})
	defer close(stop)

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: reportingplugins.ReportingPluginHandshakeConfig(),
		Plugins: map[string]plugin.Plugin{
			reportingplugins.PluginServiceName: &reportingplugins.GRPCService[types.MedianProvider]{
				PluginServer: p,
				BrokerConfig: loop.BrokerConfig{
					Logger:   s.Logger,
					StopCh:   stop,
					GRPCOpts: s.GRPCOpts,
				},
			},
		},
		GRPCServer: s.GRPCOpts.NewServer,
	})
}
