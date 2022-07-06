/*
 * Copyright 2021 - now, the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *       https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"flag"
	"log"

	"github.com/monimesl/istio-virtualservice-merger/api/v1alpha1"
	"github.com/monimesl/istio-virtualservice-merger/controller"
	"istio.io/client-go/pkg/apis/networking/v1alpha3"
	versionedclient "istio.io/client-go/pkg/clientset/versioned"

	"github.com/monimesl/operator-helper/config"
	"github.com/monimesl/operator-helper/reconciler"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	// +kubebuilder:scaffold:imports
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1alpha3.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var namespaceToMonitor string
	var leaderElectionID string
	var pspRestrictedCM string
	var pspRestrictedDDCM string
	var pspBaselineCM string
	var pspPrivilegedCM string
	var defaultConfig string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.StringVar(&namespaceToMonitor, "managed-namespace", "kube-configs", "Select which namespace to monitor. Defaults to all namespaces.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&leaderElectionID, "leader-election-id", "managed-namespace-leader.rokt.com", "Unique configmap id to lock leader election")
	flag.StringVar(&pspRestrictedCM, "psp-restricted", "psp-restricted-config", "Configmap to load template for restricted rolebinding")
	flag.StringVar(&pspRestrictedDDCM, "psp-restricted-datadog", "psp-restricted-datadog-config", "Configmap to load template for restricted datadog rolebinding")
	flag.StringVar(&pspBaselineCM, "psp-baseline", "psp-baseline-config", "Configmap to load template for baseline rolebinding")
	flag.StringVar(&pspPrivilegedCM, "psp-privileged", "psp-privileged-config", "Configmap to load template for privileged rolebinding")
	flag.StringVar(&defaultConfig, "default-config", "default-config", "Configmap to load defualt config from")
	opts := zap.Options{
		Development: true,
		// Level:       zapcore.DebugLevel,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	cfg, options := config.GetManagerParams(scheme,
		"istio-virtualservice-merger",
		"istiomerger.monime.sl")
	mgr, err := manager.New(cfg, options)
	if err != nil {
		log.Fatalf("manager create error: %s", err)
	}
	ic, err := versionedclient.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Failed to create istio client: %s", err)
	}
	if err = reconciler.Configure(mgr,
		&controllers.VirtualServicePatchReconciler{IstioClient: ic}); err != nil {
		log.Fatalf("reconciler cfg error: %s", err)
	}
	if err = mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Fatalf("operator start error: %s", err)
	}
}
