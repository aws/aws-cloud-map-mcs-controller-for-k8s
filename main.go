package main

import (
	"context"
	"flag"
	"os"

	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/common"

	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/cloudmap"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/version"
	"github.com/aws/aws-sdk-go-v2/config"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	multiclusterv1alpha1 "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/apis/multicluster/v1alpha1"
	multiclustercontrollers "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/controllers/multicluster"
	// +kubebuilder:scaffold:imports
)

var (
	scheme = runtime.NewScheme()
	log    = ctrl.Log.WithName("main")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(multiclusterv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	// Add the zap logger flag set to the CLI. The flag set must
	// be added before calling flag.Parse().
	opts := zap.Options{}
	opts.BindFlags(flag.CommandLine)

	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	v := version.GetVersion()
	log.Info("starting AWS Cloud Map MCS Controller for K8s", "version", v)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "db692913.x-k8s.io",
	})
	if err != nil {
		log.Error(err, "unable to start manager")
		os.Exit(1)
	}
	log.Info("configuring AWS session")
	// GO sdk will look for region in order 1) AWS_REGION env var, 2) ~/.aws/config file, 3) EC2 IMDS
	awsCfg, err := config.LoadDefaultConfig(context.TODO(), config.WithEC2IMDSRegion())

	if err != nil || awsCfg.Region == "" {
		log.Error(err, "unable to configure AWS session", "AWS_REGION", awsCfg.Region)
		os.Exit(1)
	}

	log.Info("Running with AWS region", "AWS_REGION", awsCfg.Region)

	serviceDiscoveryClient := cloudmap.NewDefaultServiceDiscoveryClient(&awsCfg)
	if err = (&multiclustercontrollers.ServiceExportReconciler{
		Client:   mgr.GetClient(),
		Log:      common.NewLogger("controllers", "ServiceExport"),
		Scheme:   mgr.GetScheme(),
		CloudMap: serviceDiscoveryClient,
	}).SetupWithManager(mgr); err != nil {
		log.Error(err, "unable to create controller", "controller", "ServiceExport")
		os.Exit(1)
	}

	cloudMapReconciler := &multiclustercontrollers.CloudMapReconciler{
		Client:   mgr.GetClient(),
		Cloudmap: serviceDiscoveryClient,
		Log:      common.NewLogger("controllers", "Cloudmap"),
	}

	if err = mgr.Add(cloudMapReconciler); err != nil {
		log.Error(err, "unable to create controller", "controller", "CloudMap")
		os.Exit(1)
	}

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		log.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		log.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	log.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Error(err, "problem running manager")
		os.Exit(1)
	}
}
