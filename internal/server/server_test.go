package server

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

func TestTemplateRendering(t *testing.T) {
	// Load the template file
	tmplBuffer, err := ioutil.ReadFile("../../katana-template.yaml")
	require.NoError(t, err, "Failed to read template file")

	// Parse the template
	tmpl, err := template.New("katana").Parse(string(tmplBuffer))
	require.NoError(t, err, "Failed to parse template")

	testCases := []struct {
		name     string
		values   map[string]interface{}
		expected []map[string]interface{}
	}{
		{
			name: "Basic deployment",
			values: map[string]interface{}{
				"id":           "test-001",
				"namespace":    "my-namespace",
				"version":      "latest",
				"seed":         42,
				"chainId":      1,
				"blockTime":    5,
				"storage":      "1Gi",
				"storageClass": "standard",
				"resources": map[string]interface{}{
					"requests": map[string]string{
						"memory": "1Gi",
						"cpu":    "500m",
					},
					"limits": map[string]string{
						"memory": "2Gi",
						"cpu":    "1",
					},
				},
			},
			expected: []map[string]interface{}{
				{
					"apiVersion": "v1",
					"kind":       "PersistentVolumeClaim",
					"metadata": map[string]interface{}{
						"name": "katana-data-test-001",
					},
					"spec": map[string]interface{}{
						"accessModes": []interface{}{"ReadWriteOnce"},
						"resources": map[string]interface{}{
							"requests": map[string]interface{}{
								"storage": "1Gi",
							},
						},
						"storageClassName": "standard",
					},
				},
				{
					"apiVersion": "apps/v1",
					"kind":       "Deployment",
					"metadata": map[string]interface{}{
						"name":      "katana-test-001",
						"namespace": "my-namespace",
						"labels": map[string]interface{}{
							"app.kubernetes.io/name": "katana-test-001",
						},
					},
					"spec": map[string]interface{}{
						"replicas": 1,
						"strategy": map[string]interface{}{
							"type": "Recreate",
						},
						"selector": map[string]interface{}{
							"matchLabels": map[string]interface{}{
								"app.kubernetes.io/name": "katana-test-001",
							},
						},
						"template": map[string]interface{}{
							"metadata": map[string]interface{}{
								"labels": map[string]interface{}{
									"app.kubernetes.io/name": "katana-test-001",
								},
							},
							"spec": map[string]interface{}{
								"securityContext": map[string]interface{}{
									"fsGroup": 1000,
								},
								"containers": []interface{}{
									map[string]interface{}{
										"name":            "katana",
										"image":           "ghcr.io/dojoengine/dojo:latest",
										"imagePullPolicy": "IfNotPresent",
										"command":         []interface{}{"/bin/sh", "-c"},
										"args": []interface{}{
											"katana --seed=42 --chain-id=1 --block-time=5 2>&1 | tee -a /data/katana.log",
										},
										"ports": []interface{}{
											map[string]interface{}{
												"containerPort": 5050,
												"name":          "http",
												"protocol":      "TCP",
											},
										},
										"volumeMounts": []interface{}{
											map[string]interface{}{
												"name":      "katana-data",
												"mountPath": "/data",
											},
										},
										"resources": map[string]interface{}{
											"requests": map[string]interface{}{
												"memory": "1Gi",
												"cpu":    "500m",
											},
											"limits": map[string]interface{}{
												"memory": "2Gi",
												"cpu":    "1",
											},
										},
									},
								},
								"volumes": []interface{}{
									map[string]interface{}{
										"name": "katana-data",
										"persistentVolumeClaim": map[string]interface{}{
											"claimName": "katana-data-test-001",
										},
									},
								},
							},
						},
					},
				},
				{
					"apiVersion": "v1",
					"kind":       "Service",
					"metadata": map[string]interface{}{
						"name": "katana-test-001",
						"labels": map[string]interface{}{
							"app.kubernetes.io/name": "katana-test-001",
						},
					},
					"spec": map[string]interface{}{
						"type": "ClusterIP",
						"ports": []interface{}{
							map[string]interface{}{
								"name":       "http",
								"port":       int64(80),
								"protocol":   "TCP",
								"targetPort": int32(5050),
							},
						},
						"selector": map[string]interface{}{
							"app.kubernetes.io/name": "katana-test-001",
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Render the template
			var buf bytes.Buffer
			err := tmpl.Execute(&buf, tc.values)
			require.NoError(t, err, "Failed to render template")

			// Print the rendered template
			fmt.Println(buf.String())

			// Parse the rendered content
			var renderedMap map[string]interface{}
			err = yaml.Unmarshal(buf.Bytes(), &renderedMap)
			require.NoError(t, err, "Failed to parse rendered content")

			// Check if the rendered content matches the expected output
			assertDeepEquals(t, tc.expected, renderedMap)
		})
	}
}

// assertDeepEquals is a helper function to compare nested maps
func assertDeepEquals(t *testing.T, expected, actual interface{}) {
	t.Helper()
	assert.Equal(t, expected, actual, "The rendered output does not match the expected output")
}
