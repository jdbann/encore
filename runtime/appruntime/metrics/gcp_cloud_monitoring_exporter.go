//go:build !encore_no_gcp

package metrics

import (
	"encore.dev/appruntime/config"
	"encore.dev/appruntime/metadata"
	"encore.dev/appruntime/metrics/gcp"
)

func init() {
	registerProvider(providerDesc{
		name: "gcp_cloud_monitoring",
		matches: func(cfg *config.Metrics) bool {
			return cfg.CloudMonitoring != nil
		},
		newExporter: func(mgr *Manager) exporter {
			containerMetadata, err := metadata.GetContainerMetadata(mgr.cfg.Runtime)
			if err != nil {
				mgr.rootLogger.Err(err).Msg("unable to initialize metrics exporter: error getting container metadata")
				return nil
			}

			metricsCfg := mgr.cfg.Runtime.Metrics
			nodeID, ok := metricsCfg.CloudMonitoring.MonitoredResourceLabels["node_id"]
			if !ok {
				mgr.rootLogger.Err(err).Msg("unable to initialize metrics exporter: missing node_id")
				return nil
			}
			metricsCfg.CloudMonitoring.MonitoredResourceLabels["node_id"] = nodeID + "-" + containerMetadata.InstanceID
			return gcp.New(mgr.cfg.Static.BundledServices, metricsCfg.CloudMonitoring, containerMetadata, mgr.rootLogger)
		},
	})
}
