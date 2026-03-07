// Package helm_test validates the Operator OS Helm chart structure and templates.
// These tests verify chart files exist with correct structure, templates are valid YAML,
// and default values produce a deployable configuration. They do NOT require Helm CLI
// or a Kubernetes cluster — they validate chart files and structure directly.
package helm_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const chartDir = "operator-os"

// TestChartYAMLExists validates Chart.yaml is present and has required fields.
func TestChartYAMLExists(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(chartDir, "Chart.yaml"))
	require.NoError(t, err, "Chart.yaml must exist")

	var chart map[string]interface{}
	require.NoError(t, yaml.Unmarshal(data, &chart))

	assert.Equal(t, "v2", chart["apiVersion"], "apiVersion must be v2")
	assert.Equal(t, "operator-os", chart["name"])
	assert.NotEmpty(t, chart["version"], "version is required")
	assert.NotEmpty(t, chart["appVersion"], "appVersion is required")
	assert.NotEmpty(t, chart["description"])
	assert.Equal(t, "application", chart["type"])
}

// TestValuesYAMLExists validates values.yaml is present and parseable.
func TestValuesYAMLExists(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(chartDir, "values.yaml"))
	require.NoError(t, err, "values.yaml must exist")

	var values map[string]interface{}
	require.NoError(t, yaml.Unmarshal(data, &values))

	// Validate top-level keys
	expectedKeys := []string{
		"image", "gateway", "worker", "service", "ingress",
		"postgresql", "nats", "redis", "config", "secrets",
		"resourceQuota", "metrics", "serviceAccount",
	}
	for _, key := range expectedKeys {
		assert.Contains(t, values, key, "values.yaml must contain key: %s", key)
	}
}

// TestValuesGatewayDefaults validates gateway default values.
func TestValuesGatewayDefaults(t *testing.T) {
	values := loadValues(t)

	gw, ok := values["gateway"].(map[string]interface{})
	require.True(t, ok)

	assert.EqualValues(t, 2, gw["replicaCount"])

	resources := gw["resources"].(map[string]interface{})
	assert.NotNil(t, resources["requests"])
	assert.NotNil(t, resources["limits"])

	as := gw["autoscaling"].(map[string]interface{})
	assert.True(t, as["enabled"].(bool))
	assert.EqualValues(t, 2, as["minReplicas"])
	assert.EqualValues(t, 10, as["maxReplicas"])

	pdb := gw["podDisruptionBudget"].(map[string]interface{})
	assert.True(t, pdb["enabled"].(bool))
	assert.EqualValues(t, 1, pdb["minAvailable"])
}

// TestValuesWorkerDefaults validates worker default values.
func TestValuesWorkerDefaults(t *testing.T) {
	values := loadValues(t)

	w, ok := values["worker"].(map[string]interface{})
	require.True(t, ok)

	assert.True(t, w["enabled"].(bool))
	assert.EqualValues(t, 2, w["replicaCount"])
	assert.EqualValues(t, 4, w["concurrency"])
	assert.Equal(t, "5m", w["processTimeout"])
	assert.EqualValues(t, 3, w["maxRetries"])

	as := w["autoscaling"].(map[string]interface{})
	assert.True(t, as["enabled"].(bool))
	assert.EqualValues(t, 20, as["maxReplicas"])

	pdb := w["podDisruptionBudget"].(map[string]interface{})
	assert.True(t, pdb["enabled"].(bool))
}

// TestValuesServiceDefaults validates service default values.
func TestValuesServiceDefaults(t *testing.T) {
	values := loadValues(t)

	svc, ok := values["service"].(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, "ClusterIP", svc["type"])
	assert.EqualValues(t, 18790, svc["port"])
}

// TestValuesIngressDisabledByDefault validates ingress is disabled by default.
func TestValuesIngressDisabledByDefault(t *testing.T) {
	values := loadValues(t)

	ing, ok := values["ingress"].(map[string]interface{})
	require.True(t, ok)
	assert.False(t, ing["enabled"].(bool))
}

// TestValuesExternalServicesDisabledByDefault validates PostgreSQL, NATS, Redis are disabled.
func TestValuesExternalServicesDisabledByDefault(t *testing.T) {
	values := loadValues(t)

	for _, svc := range []string{"postgresql", "nats", "redis"} {
		m, ok := values[svc].(map[string]interface{})
		require.True(t, ok, "%s must exist", svc)
		assert.False(t, m["enabled"].(bool), "%s must be disabled by default", svc)
	}
}

// TestValuesResourceQuotaDisabledByDefault validates resource quota is disabled.
func TestValuesResourceQuotaDisabledByDefault(t *testing.T) {
	values := loadValues(t)

	rq, ok := values["resourceQuota"].(map[string]interface{})
	require.True(t, ok)
	assert.False(t, rq["enabled"].(bool))
}

// TestValuesMetricsEnabled validates metrics are enabled by default.
func TestValuesMetricsEnabled(t *testing.T) {
	values := loadValues(t)

	m, ok := values["metrics"].(map[string]interface{})
	require.True(t, ok)
	assert.True(t, m["enabled"].(bool))

	sm := m["serviceMonitor"].(map[string]interface{})
	assert.False(t, sm["enabled"].(bool), "serviceMonitor should be opt-in")
}

// TestValuesConfigDefaults validates config default values.
func TestValuesConfigDefaults(t *testing.T) {
	values := loadValues(t)

	cfg, ok := values["config"].(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, "json", cfg["logFormat"])
	assert.Equal(t, "info", cfg["logLevel"])
	assert.Equal(t, "0.0.0.0", cfg["healthHost"])
	assert.EqualValues(t, 18790, cfg["healthPort"])
}

// TestTemplateFilesExist validates all expected template files exist.
func TestTemplateFilesExist(t *testing.T) {
	expectedFiles := []string{
		"_helpers.tpl",
		"configmap.yaml",
		"secret.yaml",
		"serviceaccount.yaml",
		"gateway-deployment.yaml",
		"worker-deployment.yaml",
		"service.yaml",
		"ingress.yaml",
		"gateway-hpa.yaml",
		"worker-hpa.yaml",
		"gateway-pdb.yaml",
		"worker-pdb.yaml",
		"resourcequota.yaml",
		"servicemonitor.yaml",
	}

	for _, f := range expectedFiles {
		path := filepath.Join(chartDir, "templates", f)
		_, err := os.Stat(path)
		assert.NoError(t, err, "template file must exist: %s", f)
	}
}

// TestTemplateTestConnectionExists validates test pod template exists.
func TestTemplateTestConnectionExists(t *testing.T) {
	path := filepath.Join(chartDir, "templates", "tests", "test-connection.yaml")
	_, err := os.Stat(path)
	assert.NoError(t, err, "test-connection.yaml must exist")
}

// TestTemplatesAreValidYAML validates all non-helper templates contain valid YAML fragments.
// Note: These contain Go template directives, so we just check they're not empty and have
// apiVersion fields (indicating valid K8s manifest structure).
func TestTemplatesAreValidYAML(t *testing.T) {
	templates := []string{
		"configmap.yaml",
		"secret.yaml",
		"serviceaccount.yaml",
		"gateway-deployment.yaml",
		"worker-deployment.yaml",
		"service.yaml",
		"ingress.yaml",
		"gateway-hpa.yaml",
		"worker-hpa.yaml",
		"gateway-pdb.yaml",
		"worker-pdb.yaml",
		"resourcequota.yaml",
		"servicemonitor.yaml",
	}

	for _, f := range templates {
		data, err := os.ReadFile(filepath.Join(chartDir, "templates", f))
		require.NoError(t, err, "failed to read %s", f)

		content := string(data)
		assert.NotEmpty(t, content, "%s must not be empty", f)

		// All K8s manifests should have an apiVersion (even conditional ones)
		assert.Contains(t, content, "apiVersion:", "%s must contain apiVersion", f)

		// Check for common K8s fields
		assert.Contains(t, content, "kind:", "%s must contain kind", f)
		assert.Contains(t, content, "metadata:", "%s must contain metadata", f)
	}
}

// TestHelpersContainAllDefinitions validates _helpers.tpl has required template definitions.
func TestHelpersContainAllDefinitions(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(chartDir, "templates", "_helpers.tpl"))
	require.NoError(t, err)

	content := string(data)

	expectedDefs := []string{
		"operator-os.name",
		"operator-os.fullname",
		"operator-os.chart",
		"operator-os.labels",
		"operator-os.selectorLabels",
		"operator-os.gateway.selectorLabels",
		"operator-os.worker.selectorLabels",
		"operator-os.serviceAccountName",
	}

	for _, def := range expectedDefs {
		assert.Contains(t, content, def, "_helpers.tpl must define %s", def)
	}
}

// TestGatewayDeploymentHasProbes validates gateway deployment has liveness and readiness probes.
func TestGatewayDeploymentHasProbes(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(chartDir, "templates", "gateway-deployment.yaml"))
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "livenessProbe:")
	assert.Contains(t, content, "readinessProbe:")
	assert.Contains(t, content, "/health")
	assert.Contains(t, content, "/ready")
}

// TestWorkerDeploymentHasProbes validates worker deployment has liveness and readiness probes.
func TestWorkerDeploymentHasProbes(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(chartDir, "templates", "worker-deployment.yaml"))
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "livenessProbe:")
	assert.Contains(t, content, "readinessProbe:")
	assert.Contains(t, content, "/health")
	assert.Contains(t, content, "/ready")
}

// TestDeploymentsHaveSecurityContext validates both deployments run as non-root.
func TestDeploymentsHaveSecurityContext(t *testing.T) {
	for _, f := range []string{"gateway-deployment.yaml", "worker-deployment.yaml"} {
		data, err := os.ReadFile(filepath.Join(chartDir, "templates", f))
		require.NoError(t, err, "reading %s", f)

		content := string(data)
		assert.Contains(t, content, "runAsNonRoot: true", "%s must run as non-root", f)
		assert.Contains(t, content, "runAsUser: 1000", "%s must set runAsUser", f)
	}
}

// TestDeploymentsHaveConfigAndSecretRefs validates deployments reference ConfigMap and Secret.
func TestDeploymentsHaveConfigAndSecretRefs(t *testing.T) {
	for _, f := range []string{"gateway-deployment.yaml", "worker-deployment.yaml"} {
		data, err := os.ReadFile(filepath.Join(chartDir, "templates", f))
		require.NoError(t, err, "reading %s", f)

		content := string(data)
		assert.Contains(t, content, "configMapRef:")
		assert.Contains(t, content, "secretRef:")
	}
}

// TestDeploymentsHaveChecksumAnnotations validates pods restart on config/secret changes.
func TestDeploymentsHaveChecksumAnnotations(t *testing.T) {
	for _, f := range []string{"gateway-deployment.yaml", "worker-deployment.yaml"} {
		data, err := os.ReadFile(filepath.Join(chartDir, "templates", f))
		require.NoError(t, err, "reading %s", f)

		content := string(data)
		assert.Contains(t, content, "checksum/config:", "%s must have config checksum", f)
		assert.Contains(t, content, "checksum/secret:", "%s must have secret checksum", f)
	}
}

// TestHPAScalingBehavior validates HPA has scale-up and scale-down behavior configured.
func TestHPAScalingBehavior(t *testing.T) {
	for _, f := range []string{"gateway-hpa.yaml", "worker-hpa.yaml"} {
		data, err := os.ReadFile(filepath.Join(chartDir, "templates", f))
		require.NoError(t, err, "reading %s", f)

		content := string(data)
		assert.Contains(t, content, "behavior:")
		assert.Contains(t, content, "scaleUp:")
		assert.Contains(t, content, "scaleDown:")
		assert.Contains(t, content, "stabilizationWindowSeconds:")
	}
}

// TestWorkerConditionalRendering validates worker templates are conditional on worker.enabled.
func TestWorkerConditionalRendering(t *testing.T) {
	for _, f := range []string{"worker-deployment.yaml", "worker-hpa.yaml", "worker-pdb.yaml"} {
		data, err := os.ReadFile(filepath.Join(chartDir, "templates", f))
		require.NoError(t, err, "reading %s", f)

		content := string(data)
		assert.Contains(t, content, ".Values.worker.enabled", "%s must be conditional on worker.enabled", f)
	}
}

// TestIngressConditionalRendering validates ingress is conditional.
func TestIngressConditionalRendering(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(chartDir, "templates", "ingress.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(data), ".Values.ingress.enabled")
}

// TestResourceQuotaConditionalRendering validates resource quota is conditional.
func TestResourceQuotaConditionalRendering(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(chartDir, "templates", "resourcequota.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(data), ".Values.resourceQuota.enabled")
}

// TestServiceMonitorConditionalRendering validates service monitor is conditional.
func TestServiceMonitorConditionalRendering(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(chartDir, "templates", "servicemonitor.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(data), ".Values.metrics.serviceMonitor.enabled")
}

// TestSecretContainsExpectedKeys validates secret template references expected env vars.
func TestSecretContainsExpectedKeys(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(chartDir, "templates", "secret.yaml"))
	require.NoError(t, err)

	content := string(data)
	expectedKeys := []string{
		"OPERATOR_ENCRYPTION_KEY",
		"OPERATOR_JWT_SECRET",
		"STRIPE_API_KEY",
		"STRIPE_WEBHOOK_SECRET",
		"OPERATOR_PG_DSN",
	}
	for _, key := range expectedKeys {
		assert.Contains(t, content, key, "secret must contain %s", key)
	}
}

// TestConfigMapContainsExpectedKeys validates configmap references expected env vars.
func TestConfigMapContainsExpectedKeys(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(chartDir, "templates", "configmap.yaml"))
	require.NoError(t, err)

	content := string(data)
	expectedKeys := []string{
		"OPERATOR_LOG_FORMAT",
		"OPERATOR_LOG_LEVEL",
		"OPERATOR_HEALTH_HOST",
		"OPERATOR_HEALTH_PORT",
	}
	for _, key := range expectedKeys {
		assert.Contains(t, content, key, "configmap must contain %s", key)
	}
}

// TestChartImageConfiguration validates image configuration structure.
func TestChartImageConfiguration(t *testing.T) {
	values := loadValues(t)

	img, ok := values["image"].(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, "docker.io/standardws/operator", img["repository"])
	assert.Equal(t, "latest", img["tag"])
	assert.Equal(t, "IfNotPresent", img["pullPolicy"])
}

// TestServiceAccountCreation validates service account is created by default.
func TestServiceAccountCreation(t *testing.T) {
	values := loadValues(t)

	sa, ok := values["serviceAccount"].(map[string]interface{})
	require.True(t, ok)
	assert.True(t, sa["create"].(bool))
}

// TestGatewayWorkerResourceSeparation validates workers have higher resource limits than gateway.
func TestGatewayWorkerResourceSeparation(t *testing.T) {
	values := loadValues(t)

	gwRes := values["gateway"].(map[string]interface{})["resources"].(map[string]interface{})
	wkRes := values["worker"].(map[string]interface{})["resources"].(map[string]interface{})

	gwLimits := gwRes["limits"].(map[string]interface{})
	wkLimits := wkRes["limits"].(map[string]interface{})

	// Workers should have more resources than gateways (they do LLM processing)
	assert.Equal(t, "512Mi", gwLimits["memory"])
	assert.Equal(t, "1Gi", wkLimits["memory"])
}

// TestWorkerHPAScalesFasterThanGateway validates worker HPA scales more aggressively.
func TestWorkerHPAScalesFasterThanGateway(t *testing.T) {
	values := loadValues(t)

	gwAS := values["gateway"].(map[string]interface{})["autoscaling"].(map[string]interface{})
	wkAS := values["worker"].(map[string]interface{})["autoscaling"].(map[string]interface{})

	// Workers should scale to more replicas (they're stateless + CPU-bound)
	gwMax, _ := gwAS["maxReplicas"].(int)
	wkMax, _ := wkAS["maxReplicas"].(int)
	assert.Greater(t, wkMax, gwMax, "worker maxReplicas should exceed gateway maxReplicas")

	// Workers should target lower CPU (scale out sooner)
	gwCPU, _ := gwAS["targetCPUUtilizationPercentage"].(int)
	wkCPU, _ := wkAS["targetCPUUtilizationPercentage"].(int)
	assert.Greater(t, gwCPU, wkCPU, "worker should scale at lower CPU threshold")
}

// TestNATSURLConditionalInConfigMap validates NATS URL only appears when enabled.
func TestNATSURLConditionalInConfigMap(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(chartDir, "templates", "configmap.yaml"))
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, ".Values.nats.enabled")
	assert.Contains(t, content, "OPERATOR_NATS_URL")
}

// TestRedisURLConditionalInConfigMap validates Redis URL only appears when enabled.
func TestRedisURLConditionalInConfigMap(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(chartDir, "templates", "configmap.yaml"))
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, ".Values.redis.enabled")
	assert.Contains(t, content, "OPERATOR_REDIS_URL")
}

// TestNoHardcodedSecrets validates no templates contain hardcoded secret values.
func TestNoHardcodedSecrets(t *testing.T) {
	sensitivePatterns := []string{"sk-", "sk_live_", "whsec_", "password123"}

	err := filepath.Walk(filepath.Join(chartDir, "templates"), func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(data)
		for _, pattern := range sensitivePatterns {
			assert.NotContains(t, content, pattern,
				"template %s must not contain hardcoded secret pattern: %s", info.Name(), pattern)
		}
		return nil
	})
	require.NoError(t, err)
}

// TestGatewayCommandIsCorrect validates gateway deployment uses correct command.
func TestGatewayCommandIsCorrect(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(chartDir, "templates", "gateway-deployment.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(data), ".Values.gateway.command")
}

// TestWorkerCommandIsCorrect validates worker deployment uses correct command.
func TestWorkerCommandIsCorrect(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(chartDir, "templates", "worker-deployment.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(data), ".Values.worker.command")
}

// TestAllTemplatesUseHelmLabels validates all templates reference standard Helm labels.
func TestAllTemplatesUseHelmLabels(t *testing.T) {
	templates := []string{
		"configmap.yaml", "secret.yaml", "serviceaccount.yaml",
		"gateway-deployment.yaml", "worker-deployment.yaml", "service.yaml",
	}
	for _, f := range templates {
		data, err := os.ReadFile(filepath.Join(chartDir, "templates", f))
		require.NoError(t, err, "reading %s", f)
		assert.True(t,
			strings.Contains(string(data), "operator-os.labels"),
			"%s must use standard Helm labels", f)
	}
}

// loadValues loads and parses values.yaml.
func loadValues(t *testing.T) map[string]interface{} {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(chartDir, "values.yaml"))
	require.NoError(t, err)

	var values map[string]interface{}
	require.NoError(t, yaml.Unmarshal(data, &values))
	return values
}
