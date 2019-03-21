package cmd

import (
	"bytes"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/linkerd/linkerd2/controller/gen/config"
)

func TestRender(t *testing.T) {
	defaultOptions := testInstallOptions()
	defaultValues, defaultConfig, err := defaultOptions.validateAndBuild()
	if err != nil {
		t.Fatalf("Unexpected error validating options: %v", err)
	}
	defaultValues.UUID = "deaab91a-f4ab-448a-b7d1-c832a2fa0a60"

	// A configuration that shows that all config setting strings are honored
	// by `render()`.
	metaOptions := testInstallOptions()
	metaConfig := metaOptions.configs(nil)
	metaConfig.Global.LinkerdNamespace = "Namespace"
	metaValues := &installValues{
		Namespace:                "Namespace",
		ControllerImage:          "ControllerImage",
		WebImage:                 "WebImage",
		PrometheusImage:          "PrometheusImage",
		GrafanaImage:             "GrafanaImage",
		ControllerReplicas:       1,
		ImagePullPolicy:          "ImagePullPolicy",
		UUID:                     "UUID",
		CliVersion:               "CliVersion",
		ControllerLogLevel:       "ControllerLogLevel",
		PrometheusLogLevel:       "PrometheusLogLevel",
		ControllerComponentLabel: "ControllerComponentLabel",
		CreatedByAnnotation:      "CreatedByAnnotation",
		ProxyContainerName:       "ProxyContainerName",
		ProxyAutoInjectEnabled:   true,
		ProxyInjectAnnotation:    "ProxyInjectAnnotation",
		ProxyInjectDisabled:      "ProxyInjectDisabled",
		ControllerUID:            2103,
		EnableH2Upgrade:          true,
		NoInitContainer:          false,
		GlobalConfig:             "GlobalConfig",
		ProxyConfig:              "ProxyConfig",
		Identity:                 defaultValues.Identity,
	}

	haOptions := testInstallOptions()
	haOptions.highAvailability = true
	haValues, haConfig, _ := haOptions.validateAndBuild()
	haValues.UUID = defaultValues.UUID

	haWithOverridesOptions := testInstallOptions()
	haWithOverridesOptions.highAvailability = true
	haWithOverridesOptions.controllerReplicas = 2
	haWithOverridesOptions.proxyCPURequest = "400m"
	haWithOverridesOptions.proxyMemoryRequest = "300Mi"
	haWithOverridesValues, haWithOverridesConfig, _ := haWithOverridesOptions.validateAndBuild()
	haWithOverridesValues.UUID = defaultValues.UUID

	noInitContainerOptions := testInstallOptions()
	noInitContainerOptions.noInitContainer = true
	noInitContainerValues, noInitContainerConfig, _ := noInitContainerOptions.validateAndBuild()
	noInitContainerValues.UUID = defaultValues.UUID

	noInitContainerWithProxyAutoInjectOptions := testInstallOptions()
	noInitContainerWithProxyAutoInjectOptions.noInitContainer = true
	noInitContainerWithProxyAutoInjectOptions.proxyAutoInject = true
	noInitContainerWithProxyAutoInjectValues, noInitContainerWithProxyAutoInjectConfig, _ := noInitContainerWithProxyAutoInjectOptions.validateAndBuild()
	noInitContainerWithProxyAutoInjectValues.UUID = defaultValues.UUID

	testCases := []struct {
		values         *installValues
		configs        *config.All
		goldenFileName string
	}{
		{defaultValues, defaultConfig, "install_default.golden"},
		{metaValues, metaConfig, "install_output.golden"},
		{haValues, haConfig, "install_ha_output.golden"},
		{haWithOverridesValues, haWithOverridesConfig, "install_ha_with_overrides_output.golden"},
		{noInitContainerValues, noInitContainerConfig, "install_no_init_container.golden"},
		{noInitContainerWithProxyAutoInjectValues, noInitContainerWithProxyAutoInjectConfig, "install_no_init_container_auto_inject.golden"},
	}

	for i, tc := range testCases {
		tc := tc // pin
		t.Run(fmt.Sprintf("%d: %s", i, tc.goldenFileName), func(t *testing.T) {
			controlPlaneNamespace = tc.configs.GetGlobal().GetLinkerdNamespace()

			var buf bytes.Buffer
			if err := render(tc.values, &buf, tc.configs); err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			diffTestdata(t, tc.goldenFileName, buf.String())
		})
	}
}

func testInstallOptions() *installOptions {
	o := newInstallOptionsWithDefaults()
	o.ignoreCluster = true
	o.identityOptions.crtPEMFile = filepath.Join("testdata", "crt.pem")
	o.identityOptions.keyPEMFile = filepath.Join("testdata", "key.pem")
	o.identityOptions.trustPEMFile = filepath.Join("testdata", "trust-anchors.pem")
	return o
}

func TestValidate(t *testing.T) {
	t.Run("Accepts the default options as valid", func(t *testing.T) {
		if err := testInstallOptions().validate(); err != nil {
			t.Fatalf("Unexpected error: %s", err)
		}
	})

	t.Run("Rejects invalid controller log level", func(t *testing.T) {
		options := testInstallOptions()
		options.controllerLogLevel = "super"
		expected := "--controller-log-level must be one of: panic, fatal, error, warn, info, debug"

		err := options.validate()
		if err == nil {
			t.Fatal("Expected error, got nothing")
		}
		if err.Error() != expected {
			t.Fatalf("Expected error string\"%s\", got \"%s\"", expected, err)
		}
	})

	t.Run("Properly validates proxy log level", func(t *testing.T) {
		testCases := []struct {
			input string
			valid bool
		}{
			{"", false},
			{"info", true},
			{"somemodule", true},
			{"bad%name", false},
			{"linkerd2_proxy=debug", true},
			{"linkerd2%proxy=debug", false},
			{"linkerd2_proxy=foobar", false},
			{"linker2d_proxy,std::option", true},
			{"warn,linkerd2_proxy=info", true},
			{"warn,linkerd2_proxy=foobar", false},
		}

		options := testInstallOptions()
		for _, tc := range testCases {
			options.proxyLogLevel = tc.input
			err := options.validate()
			if tc.valid && err != nil {
				t.Fatalf("Error not expected: %s", err)
			}
			if !tc.valid && err == nil {
				t.Fatalf("Expected error string \"%s is not a valid proxy log level\", got nothing", tc.input)
			}
			expectedErr := "\"%s\" is not a valid proxy log level - for allowed syntax check https://docs.rs/env_logger/0.6.0/env_logger/#enabling-logging"
			if !tc.valid && err.Error() != fmt.Sprintf(expectedErr, tc.input) {
				t.Fatalf("Expected error string \""+expectedErr+"\"", tc.input, err)
			}
		}
	})
}
