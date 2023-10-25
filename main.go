package main

import (
	"fmt"
	"log"
	"os"

	"github.com/pkg/errors"
	"github.com/vmware-tanzu/sonobuoy/pkg/client"
	"github.com/vmware-tanzu/sonobuoy/pkg/config"
	sonodynamic "github.com/vmware-tanzu/sonobuoy/pkg/dynamic"
	"github.com/vmware-tanzu/sonobuoy/pkg/errlog"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/manifest"
	ctrlConfig "sigs.k8s.io/controller-runtime/pkg/client/config"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
)

const (
	skipPreflight   = false
	sonobuoyVersion = "v0.56.16"
	k8sVersion      = "v1.27.3"
)

func getSonobuoyClient(cfg *rest.Config) (*client.SonobuoyClient, error) {
	var skc *sonodynamic.APIHelper
	var err error
	if cfg != nil {
		skc, err = sonodynamic.NewAPIHelperFromRESTConfig(cfg)
		if err != nil {
			return nil, errors.Wrap(err, "couldn't get sonobuoy api helper")
		}
	}
	return client.NewSonobuoyClient(cfg, skc)
}

func configureDockerLibraryRegistry(m *manifest.Manifest) error {
	if m.ConfigMap == nil {
		m.ConfigMap = map[string]string{}
	}
	m.ConfigMap["conformance-image-config.yaml"] = string("dockerLibraryRegistry: mirror.gcr.io/library\n")

	m.Spec.Env = append(m.Spec.Env, corev1.EnvVar{
		Name:  "KUBE_TEST_REPO_LIST",
		Value: fmt.Sprintf("/tmp/sonobuoy/config/%v", "conformance-image-config.yaml"),
	})
	return nil
}

func getGenConfig() client.GenConfig {
	return client.GenConfig{
		Config: &config.Config{
			Aggregation: plugin.AggregationConfig{
				BindAddress:    "0.0.0.0",
				BindPort:       8080,
				TimeoutSeconds: 21600,
			},
			Description: "DEFAULT",
			Version:     sonobuoyVersion,
			ResultsDir:  "/tmp/sonobuoy/results",
			Filters: config.FilterOptions{
				Namespaces:    ".*",
				LabelSelector: "",
			},
			Limits: config.LimitConfig{
				PodLogs: config.PodLogLimits{
					Namespaces:        "kube-system",
					SonobuoyNamespace: ptr.To(true),
					FieldSelectors:    []string{},
					LabelSelector:     "",
					Previous:          false,
					SinceSeconds:      nil,
					SinceTime:         nil,
					Timestamps:        false,
					TailLines:         nil,
					LimitBytes:        nil,
				},
			},
			QPS:              30,
			Burst:            50,
			PluginSelections: nil,
			PluginSearchPath: []string{
				"./plugins.d",
				"/etc/sonobuoy/plugins.d",
				"~/sonobuoy/plugins.d",
			},
			Namespace:                "sonobuoy",
			WorkerImage:              "sonobuoy/sonobuoy:" + sonobuoyVersion,
			ImagePullPolicy:          "IfNotPresent",
			ImagePullSecrets:         "",
			AggregatorPermissions:    "clusterAdmin",
			ServiceAccountName:       "sonobuoy-serviceaccount",
			NamespacePSAEnforceLevel: "privileged",
			ProgressUpdatesPort:      "8099",
			SecurityContextMode:      "nonroot",
		},
		EnableRBAC:      true,
		ImagePullPolicy: "IfNotPresent",
		SSHKeyPath:      "",
		DynamicPlugins:  []string{"e2e"},
		PluginEnvOverrides: map[string]map[string]string{
			"e2e": {
				"E2E_FOCUS":            `\[Conformance\]`,
				"E2E_SKIP":             "",
				"E2E_PARALLEL":         "false",
				"SONOBUOY_K8S_VERSION": k8sVersion,
			},
		},
		PluginTransforms: map[string][]func(*manifest.Manifest) error{
			"e2e": {configureDockerLibraryRegistry},
		},
		ShowDefaultPodSpec: false,
		KubeVersion:        k8sVersion,
	}
}

func gen() {
	// Generate does not require any client configuration
	sbc := &client.SonobuoyClient{}

	genConfig := getGenConfig()

	bytes, err := sbc.GenerateManifest(&genConfig)
	if err == nil {
		fmt.Printf("%s\n", bytes)
		return
	}
	errlog.LogError(errors.Wrap(err, "error attempting to generate sonobuoy manifest"))
	os.Exit(1)
}

func run() {
	restConfig, err := ctrlConfig.GetConfig()
	if err != nil {
		log.Fatalf("Error getting Kubernetes config: %q", err)
	}

	sbc, err := getSonobuoyClient(restConfig)
	if err != nil {
		errlog.LogError(errors.Wrap(err, "could not create sonobuoy client"))
		os.Exit(1)
	}

	runCfg := &client.RunConfig{
		GenConfig:  getGenConfig(),
		Wait:       0,
		WaitOutput: "",
	}

	if !skipPreflight {
		// TODO specify PreflightConfig fields
		pcfg := &client.PreflightConfig{
			Namespace: "sonobuoy",
		}

		if errs := sbc.PreflightChecks(pcfg); len(errs) > 0 {
			errlog.LogError(errors.New("Preflight checks failed"))
			for _, err := range errs {
				errlog.LogError(err)
			}
			os.Exit(1)
		}
	}

	if err := sbc.Run(runCfg); err != nil {
		errlog.LogError(errors.Wrap(err, "error attempting to run sonobuoy"))
		os.Exit(1)
	}
}

func main() {
	if len(os.Args) <= 1 {
		errlog.LogError(errors.New("no argument given, try 'gen' or 'run'"))
		os.Exit(1)
	}

	switch os.Args[1] {
	case "gen":
		gen()
	case "run":
		run()
	default:
		errlog.LogError(errors.New(fmt.Sprintf("argument '%s' is not supported", os.Args[1])))
		os.Exit(1)
	}
}
