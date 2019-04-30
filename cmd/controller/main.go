/*
Copyright 2018 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"log"
	"os"

	"github.com/knative/eventing/pkg/reconciler/eventtype"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/rest"

	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	informers "github.com/knative/eventing/pkg/client/informers/externalversions"
	"github.com/knative/eventing/pkg/logconfig"
	"github.com/knative/eventing/pkg/logging"
	"github.com/knative/eventing/pkg/reconciler"
	"github.com/knative/eventing/pkg/reconciler/channel"
	"github.com/knative/eventing/pkg/reconciler/namespace"
	"github.com/knative/eventing/pkg/reconciler/subscription"
	"github.com/knative/eventing/pkg/reconciler/trigger"
	"github.com/knative/eventing/pkg/reconciler/v1alpha1/broker"
	"github.com/knative/pkg/configmap"
	kncontroller "github.com/knative/pkg/controller"
	"github.com/knative/pkg/logging/logkey"
	"github.com/knative/pkg/signals"
	"go.uber.org/zap"
	controllerruntime "sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	hardcodedLoggingConfig bool
)

func main() {
	flag.Parse()
	logf.SetLogger(logf.ZapLogger(false))

	logger, atomicLevel := setupLogger()
	defer logger.Sync()
	logger = logger.With(zap.String(logkey.ControllerType, logconfig.Controller))

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	cfg, err := controllerruntime.GetConfig()
	if err != nil {
		logger.Fatalf("Error building kubeconfig: %v", err)
	}

	logger.Info("Starting the controller")

	const numControllers = 6
	cfg.QPS = numControllers * rest.DefaultQPS
	cfg.Burst = numControllers * rest.DefaultBurst
	opt := reconciler.NewOptionsOrDie(cfg, logger, stopCh)

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(opt.KubeClientSet, opt.ResyncPeriod)
	eventingInformerFactory := informers.NewSharedInformerFactory(opt.EventingClientSet, opt.ResyncPeriod)

	// Eventing
	triggerInformer := eventingInformerFactory.Eventing().V1alpha1().Triggers()
	channelInformer := eventingInformerFactory.Eventing().V1alpha1().Channels()
	subscriptionInformer := eventingInformerFactory.Eventing().V1alpha1().Subscriptions()
	brokerInformer := eventingInformerFactory.Eventing().V1alpha1().Brokers()
	eventTypeInformer := eventingInformerFactory.Eventing().V1alpha1().EventTypes()

	// Kube
	serviceInformer := kubeInformerFactory.Core().V1().Services()
	namespaceInformer := kubeInformerFactory.Core().V1().Namespaces()
	configMapInformer := kubeInformerFactory.Core().V1().ConfigMaps()
	deploymentInformer := kubeInformerFactory.Apps().V1().Deployments()

	// Build all of our controllers, with the clients constructed above.
	// Add new controllers to this array.
	// You also need to modify numControllers above to match this.
	controllers := [...]*kncontroller.Impl{
		subscription.NewController(
			opt,
			subscriptionInformer,
		),
		namespace.NewController(
			opt,
			namespaceInformer,
		),
		channel.NewController(
			opt,
			channelInformer,
		),
		trigger.NewController(
			opt,
			triggerInformer,
			channelInformer,
			subscriptionInformer,
			brokerInformer,
			serviceInformer,
		),
		broker.NewController(
			opt,
			brokerInformer,
			subscriptionInformer,
			channelInformer,
			serviceInformer,
			deploymentInformer,
			broker.ReconcilerArgs{
				IngressImage:              getRequiredEnv("BROKER_INGRESS_IMAGE"),
				IngressServiceAccountName: getRequiredEnv("BROKER_INGRESS_SERVICE_ACCOUNT"),
				FilterImage:               getRequiredEnv("BROKER_FILTER_IMAGE"),
				FilterServiceAccountName:  getRequiredEnv("BROKER_FILTER_SERVICE_ACCOUNT"),
			},
		),
		eventtype.NewController(
			opt,
			eventTypeInformer,
			brokerInformer,
		),
	}
	// This line asserts at compile time that the length of controllers is equal to numControllers.
	// It is based on https://go101.org/article/tips.html#assert-at-compile-time, which notes that
	// var _ [N-M]int
	// asserts at compile time that N >= M, which we can use to establish equality of N and M:
	// (N >= M) && (M >= N) => (N == M)
	var _ [numControllers - len(controllers)][len(controllers) - numControllers]int

	// Watch the logging config map and dynamically update logging levels.
	opt.ConfigMapWatcher.Watch(logconfig.ConfigMapName(), logging.UpdateLevelFromConfigMap(logger, atomicLevel, logconfig.Controller))
	// TODO: Watch the observability config map and dynamically update metrics exporter.
	//opt.ConfigMapWatcher.Watch(metrics.ObservabilityConfigName, metrics.UpdateExporterFromConfigMap(component, logger))
	if err := opt.ConfigMapWatcher.Start(stopCh); err != nil {
		logger.Fatalw("failed to start configuration manager", zap.Error(err))
	}

	// Start all of the informers and wait for them to sync.
	logger.Info("Starting informers.")
	if err := kncontroller.StartInformers(
		stopCh,
		// Eventing
		brokerInformer.Informer(),
		channelInformer.Informer(),
		subscriptionInformer.Informer(),
		triggerInformer.Informer(),
		eventTypeInformer.Informer(),
		// Kube
		configMapInformer.Informer(),
		serviceInformer.Informer(),
		namespaceInformer.Informer(),
		deploymentInformer.Informer(),
	); err != nil {
		logger.Fatalf("Failed to start informers: %v", err)
	}

	// Start all of the controllers.
	logger.Info("Starting controllers.")

	kncontroller.StartAll(stopCh, controllers[:]...)
}

func init() {
	flag.BoolVar(&hardcodedLoggingConfig, "hardCodedLoggingConfig", false, "If true, use the hard coded logging config. It is intended to be used only when debugging outside a Kubernetes cluster.")
}

func setupLogger() (*zap.SugaredLogger, zap.AtomicLevel) {
	// Set up our logger.
	loggingConfigMap := getLoggingConfigOrDie()
	loggingConfig, err := logging.NewConfigFromMap(loggingConfigMap)
	if err != nil {
		log.Fatalf("Error parsing logging configuration: %v", err)
	}
	return logging.NewLoggerFromConfig(loggingConfig, logconfig.Controller)
}

func getLoggingConfigOrDie() map[string]string {
	if hardcodedLoggingConfig {
		return map[string]string{
			"loglevel.controller": "info",
			"zap-logger-config": `
				{
					"level": "info",
					"development": false,
					"outputPaths": ["stdout"],
					"errorOutputPaths": ["stderr"],
					"encoding": "json",
					"encoderConfig": {
					"timeKey": "ts",
					"levelKey": "level",
					"nameKey": "logger",
					"callerKey": "caller",
					"messageKey": "msg",
					"stacktraceKey": "stacktrace",
					"lineEnding": "",
					"levelEncoder": "",
					"timeEncoder": "iso8601",
					"durationEncoder": "",
					"callerEncoder": ""
				}`,
		}
	} else {
		cm, err := configmap.Load("/etc/config-logging")
		if err != nil {
			log.Fatalf("Error loading logging configuration: %v", err)
		}
		return cm
	}
}

func getRequiredEnv(envKey string) string {
	val, defined := os.LookupEnv(envKey)
	if !defined {
		log.Fatalf("required environment variable not defined '%s'", envKey)
	}
	return val
}
