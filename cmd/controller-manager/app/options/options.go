package options

import (
	"github.com/spf13/pflag"
)

type ControllerManagerOptions struct {
	MetricsAddr          string
	MetricsCertPath      string
	MetricsCertName      string
	MetricsCertKey       string
	EnableLeaderElection bool
	SecureMetrics        bool
	EnableHTTP2          bool
	ProbeBindAddress     string
}

func (c *ControllerManagerOptions) Addflags(fs *pflag.FlagSet) {
	fs.StringVar(&c.MetricsAddr, "metricsAddr", ":8080", "the address that the metrics server should bind to")
	fs.StringVar(&c.MetricsCertPath, "metricsCertPath", "/tmp/k8s-metrics-server/serving-certs", "the directory that contains the metrics server key and certificate")
	fs.StringVar(&c.MetricsCertName, "metricsCertName", "tls.crt", "the metrics server certificate name")
	fs.StringVar(&c.MetricsCertKey, "metricsCertKey", "tls.key", "the metrics server key name")
	fs.StringVar(&c.ProbeBindAddress, "ProbeBindAddress", "", "the TCP address that the manager should bind to")
	fs.BoolVar(&c.EnableLeaderElection, "enableLeaderElection", false, "enable leader election for the controller manager")
	fs.BoolVar(&c.SecureMetrics, "secureMetrics", false, "enables serving metrics via https")
	fs.BoolVar(&c.EnableHTTP2, "enableHTTP2", false, "enable HTTP2")
}
