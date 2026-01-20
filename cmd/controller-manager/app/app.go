package app

import (
	"crypto/tls"
	"os"

	rbaccontrollerv1 "github.com/GGh41th/rbac-controller/api/v1alpha1"
	"github.com/GGh41th/rbac-controller/cmd/controller-manager/app/options"
	"github.com/GGh41th/rbac-controller/internal/controller"
	rbaccontrollerv1webhook "github.com/GGh41th/rbac-controller/internal/webhook/v1alpha1"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

const (
	controllerName = "rbac-controller"
)

var (
	setupLog = ctrl.Log.WithName("setup")
)

func NewControllerManagerCommand() *cobra.Command {
	opts := &options.ControllerManagerOptions{}
	fs := pflag.NewFlagSet(controllerName, pflag.ExitOnError)
	opts.Addflags(fs)

	cmd := &cobra.Command{
		Use:   controllerName,
		Short: "Controller manager for rbac-controller",
		RunE: func(cmd *cobra.Command, args []string) error {
			// parse flags from args
			if err := fs.Parse(args); err != nil {
				return err
			}
			return runControllerManager(opts)
		},
	}
	cmd.Flags().AddFlagSet(fs)
	return cmd
}

func runControllerManager(opts *options.ControllerManagerOptions) error {

	var tlsOpts []func(*tls.Config)
	logOpts := zap.Options{
		Development: true,
	}

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&logOpts)))

	// if the enable-http2 flag is false (the default), http/2 should be disabled
	// due to its vulnerabilities. More specifically, disabling http/2 will
	// prevent from being vulnerable to the HTTP/2 Stream Cancellation and
	// Rapid Reset CVEs. For more information see:
	// - https://github.com/advisories/GHSA-qppj-fm5r-hxr3
	// - https://github.com/advisories/GHSA-4374-p667-p6c8
	disableHTTP2 := func(c *tls.Config) {
		setupLog.Info("disabling http/2")
		c.NextProtos = []string{"http/1.1"}
	}

	if !opts.EnableHTTP2 {
		tlsOpts = append(tlsOpts, disableHTTP2)
	}

	// Initial webhook TLS options
	webhookTLSOpts := tlsOpts
	webhookServerOptions := webhook.Options{
		TLSOpts: webhookTLSOpts,
	}

	if len(opts.WebhookCertPath) > 0 {
		setupLog.Info("Initializing webhook certificate watcher using provided certificates",
			"webhook-cert-path", opts.WebhookCertPath, "webhook-cert-name", opts.WebhookCertName, "webhook-cert-key", opts.WebhookCertKey)

		webhookServerOptions.CertDir = opts.WebhookCertPath
		webhookServerOptions.CertName = opts.WebhookCertName
		webhookServerOptions.KeyName = opts.WebhookCertKey
	}

	webhookServer := webhook.NewServer(webhookServerOptions)

	metricsServerOptions := metricsserver.Options{
		BindAddress:   opts.MetricsAddr,
		SecureServing: opts.SecureMetrics,
		TLSOpts:       tlsOpts,
	}
	// enable authN/authZ for metrics endpoint
	if opts.SecureMetrics {
		metricsServerOptions.FilterProvider = filters.WithAuthenticationAndAuthorization
	}

	// If the certificate is not specified, controller-runtime will automatically
	// generate self-signed certificates for the metrics server. While convenient for development and testing,
	// this setup is not recommended for production.
	// [TODO: Integrate with cert-manager]
	if len(opts.MetricsCertPath) > 0 {
		setupLog.Info("Initializing metrics certificate watcher using provided certificates",
			"metrics-cert-path", opts.MetricsCertPath, "metrics-cert-name", opts.MetricsCertName, "metrics-cert-key", opts.MetricsCertKey)

		metricsServerOptions.CertDir = opts.MetricsCertPath
		metricsServerOptions.CertName = opts.MetricsCertName
		metricsServerOptions.KeyName = opts.MetricsCertKey
	}

	electionName := controllerName
	cfg, err := ctrl.GetConfig()
	if err != nil {
		setupLog.Error(err, "Failed to get kubeconfig")
	}
	mgr, err := ctrl.NewManager(cfg, manager.Options{
		Metrics:          metricsServerOptions,
		LeaderElection:   opts.EnableLeaderElection,
		LeaderElectionID: electionName,
		PprofBindAddress: opts.ProbeBindAddress,
		WebhookServer:    webhookServer,
	})

	if err != nil {
		setupLog.Error(err, "Failed to create manager")
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "error adding healthz checker")
		return err
	}

	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "error adding Readyz checker")
		return err
	}

	if err := rbaccontrollerv1.AddToScheme(mgr.GetScheme()); err != nil {
		setupLog.Error(err, "unable to register scheme", "api", rbaccontrollerv1.GroupVersion.String())
		return err
	}

	// TODO(GGh41th) , wrap the registration with the manager in a helper (e.g Add)
	// this allows to pass a rawLogger (*logr.Logger) , from which we can
	// create a new logger at each reconcilation and add values (e.g RBACrule name)

	if err := (&controller.RBACRuleReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Log:    ctrl.Log.WithName("controllers").WithName("RBACRule"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "Failed to setup controller with manager")
		return err
	}
	if os.Getenv("ENABLE_WEBHOOK") != "false" {
		if err := rbaccontrollerv1webhook.SetupRBACRuleWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to register webhook with manager")
			return err
		}
	}

	rootCtx := signals.SetupSignalHandler()

	if err := mgr.Start(rootCtx); err != nil {
		setupLog.Error(err, "unable to start manager")
	}
	return nil
}
