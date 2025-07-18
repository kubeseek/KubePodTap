package operator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
)

// Operator represents the KPT operator
type Operator struct {
	clientset        *kubernetes.Clientset
	extensionsClient *apiextensionsclientset.Clientset
	namespace        string
}

// NewOperator creates a new KPT operator
func NewOperator(clientset *kubernetes.Clientset, namespace string) *Operator {
	// Create apiextensions client
	config, err := GetKubernetesConfig("")
	if err != nil {
		log.Fatalf("Unable to get Kubernetes configuration: %v", err)
	}

	extensionsClient, err := apiextensionsclientset.NewForConfig(config)
	if err != nil {
		log.Fatalf("Unable to create apiextensions client: %v", err)
	}

	return &Operator{
		clientset:        clientset,
		extensionsClient: extensionsClient,
		namespace:        namespace,
	}
}

// EnsureCRDsExist ensures that all required CRDs exist
func (o *Operator) EnsureCRDsExist() error {
	// Find all YAML files in the CRD directory
	crdDir := "/etc/kpt-operator/crds"
	files, err := os.ReadDir(crdDir)
	if err != nil {
		return fmt.Errorf("Unable to read CRD directory %s: %v", crdDir, err)
	}

	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".yaml" {
			continue
		}

		// Read CRD file
		crdPath := filepath.Join(crdDir, file.Name())
		data, err := os.ReadFile(crdPath)
		if err != nil {
			return fmt.Errorf("Unable to read CRD file %s: %v", crdPath, err)
		}

		// Parse YAML
		var crd apiextensionsv1.CustomResourceDefinition
		if err := yaml.Unmarshal(data, &crd); err != nil {
			return fmt.Errorf("Unable to parse CRD file %s: %v", crdPath, err)
		}

		// Check if CRD already exists
		_, err = o.extensionsClient.ApiextensionsV1().CustomResourceDefinitions().Get(context.TODO(), crd.Name, metav1.GetOptions{})
		if err == nil {
			fmt.Printf("CRD %s already exists\n", crd.Name)
			continue
		}

		// If CRD doesn't exist, create it
		if apierrors.IsNotFound(err) {
			_, err = o.extensionsClient.ApiextensionsV1().CustomResourceDefinitions().Create(context.TODO(), &crd, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("Unable to create CRD %s: %v", crd.Name, err)
			}
			fmt.Printf("Created CRD %s\n", crd.Name)
		} else {
			return fmt.Errorf("Error checking CRD %s: %v", crd.Name, err)
		}
	}

	return nil
}

// MonitorCustomResources monitors custom resources
func (o *Operator) MonitorCustomResources(ctx context.Context) {
	fmt.Println("Starting to monitor custom resources...")
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	// For recording processed resource versions
	processedVersions := make(map[string]string)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Get kptmonitor CR
			dynClient, err := GetDynamicClient("")
			if err != nil {
				fmt.Printf("Failed to create dynamic client: %v\n", err)
				continue
			}

			gvr := schema.GroupVersionResource{
				Group:    "kpt.kubeseek.com",
				Version:  "v1",
				Resource: "kptmonitors",
			}

			crList, err := dynClient.Resource(gvr).Namespace(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
			if err != nil {
				fmt.Printf("Failed to get kptmonitor CR: %v\n", err)
				continue
			}

			fmt.Printf("Found kptmonitor CR: %d\n", len(crList.Items))

			// Process each CR
			for _, cr := range crList.Items {
				crName := cr.GetName()
				crNamespace := cr.GetNamespace()
				resourceVersion := cr.GetResourceVersion()

				// Check if this version has already been processed
				key := fmt.Sprintf("%s/%s", crNamespace, crName)
				if lastVersion, exists := processedVersions[key]; exists && lastVersion == resourceVersion {
					// If already processed this version, skip
					fmt.Printf("Skipping already processed CR version: %s/%s (version: %s)\n", crNamespace, crName, resourceVersion)
					continue
				}

				// Check CR current status, skip if already in monitoring state
				var currentStatus string
				if statusObj, hasStatus := cr.Object["status"].(map[string]interface{}); hasStatus {
					if status, hasStatusField := statusObj["status"].(string); hasStatusField {
						currentStatus = status
						if currentStatus == "Monitoring" {
							fmt.Printf("Skipping CR already in Monitoring state: %s/%s\n", crNamespace, crName)
							continue
						}
					}
				}

				// Record the version being processed
				processedVersions[key] = resourceVersion

				fmt.Printf("Processing kptmonitor CR: %s/%s (version: %s)\n", crNamespace, crName, resourceVersion)
				fmt.Printf("  Spec: %v\n", cr.Object["spec"])
				
				// Check current CR status and record
				if statusObj, hasStatus := cr.Object["status"].(map[string]interface{}); hasStatus {
					if status, hasStatusField := statusObj["status"].(string); hasStatusField {
						fmt.Printf("  Current status: %s\n", status)
					} else {
						fmt.Printf("  Current status: Not set\n")
					}
				} else {
					fmt.Printf("  Current status: Not set\n")
				}

				// Extract monitoring information from CR
				spec, ok := cr.Object["spec"].(map[string]interface{})
				if !ok {
					fmt.Printf("Failed to parse CR spec: %s/%s\n", crNamespace, crName)
					continue
				}

				namespace, _ := spec["namespace"].(string)
				targetPods, _ := spec["targetPods"].(string)
				tapDuration, _ := spec["tapDuration"].(string)

				// Create monitoring data
				monitorData := map[string]string{
					"namespace":   namespace,
					"targetPods":  targetPods,
					"tapDuration": tapDuration,
				}

				// Serialize to JSON
				jsonData, err := json.Marshal(monitorData)
				if err != nil {
					fmt.Printf("Failed to serialize monitoring data: %v\n", err)
					continue
				}

				// Find all kpt-probe pods
				pods, err := o.clientset.CoreV1().Pods(metav1.NamespaceAll).List(ctx, metav1.ListOptions{
					LabelSelector: "app=kpt-probe",
				})
				if err != nil {
					fmt.Printf("Failed to get probe pods: %v\n", err)
					continue
				}

				fmt.Printf("Found %d probe pods\n", len(pods.Items))

				successCount := 0
				// Send monitoring info to each probe pod
				for _, pod := range pods.Items {
					podName := pod.Name
					podIP := pod.Status.PodIP
					
					// Use Pod IP to build URL directly, avoid DNS resolution issues
					probeURL := fmt.Sprintf("http://%s:8081/api/monitor", podIP)
					fmt.Printf("  Sending monitoring data to probe: %s (IP: %s, URL: %s)\n", podName, podIP, probeURL)
					fmt.Printf("  Data content: %s\n", string(jsonData))

					// Send HTTP POST request
					resp, err := http.Post(probeURL, "application/json", bytes.NewBuffer(jsonData))
					if err != nil {
						fmt.Printf("  Send failed: %v\n", err)
						continue
					}

					// Ensure response body is closed
					if resp.Body != nil {
						resp.Body.Close()
					}

					// Check response status
					if resp.StatusCode == http.StatusOK {
						successCount++
						fmt.Printf("  Send successful, status code: %d\n", resp.StatusCode)
					} else {
						fmt.Printf("  Send failed, status code: %d\n", resp.StatusCode)
					}
				}

				// Update CR status
				status := make(map[string]interface{})

				if successCount > 0 {
					status["status"] = "Monitoring"
					status["message"] = fmt.Sprintf("Processed and sent to %d/%d probes", successCount, len(pods.Items))
				} else if len(pods.Items) == 0 {
					status["status"] = "Failed"
					status["message"] = "No probes found"
				} else {
					status["status"] = "Failed"
					status["message"] = "Failed to send to any probe"
				}

				// Set last update time
				status["lastUpdated"] = metav1.Now().Format(time.RFC3339)

				// Add status object to CR
				crCopy := cr.DeepCopy()
				if crCopy.Object["status"] == nil {
					crCopy.Object["status"] = make(map[string]interface{})
					// If this is a new CR (first time setting status), set initial state to Created
					if len(status) == 0 {
						status["status"] = "Created"
						status["message"] = "CR has been created"
					}
				}

				crStatus := crCopy.Object["status"].(map[string]interface{})
				for k, v := range status {
					crStatus[k] = v
				}

				// Update CR status
				_, err = dynClient.Resource(gvr).Namespace(crNamespace).UpdateStatus(ctx, crCopy, metav1.UpdateOptions{})
				if err != nil {
					fmt.Printf("Failed to update CR status: %v\n", err)
				} else {
					statusMsg := ""
					if message, ok := status["message"].(string); ok {
						statusMsg = message
					}
					fmt.Printf("Updated CR status: %s (%s)\n", status["status"], statusMsg)
				}
			}
		}
	}
}

// ReportStatus reports the status of the operator
func (o *Operator) ReportStatus() {
	// This function is called periodically to record the status of CRs
	fmt.Println("Checking custom resource status...")

	// Use RESTClient to directly query CR
	// TODO: Need to implement dynamic client usage
	// Temporarily comment out unused variables
	_, err := GetDynamicClient("")
	if err != nil {
		fmt.Printf("Failed to create dynamic client: %v\n", err)
		return
	}

	// Ensure CRDs exist
	crdList, err := o.extensionsClient.ApiextensionsV1().CustomResourceDefinitions().List(context.TODO(), metav1.ListOptions{
		LabelSelector: "app=kpt",
	})
	if err != nil {
		fmt.Printf("Failed to get CRD list: %v\n", err)
		return
	}

	for _, crd := range crdList.Items {
		fmt.Printf("Found CRD: %s\n", crd.Name)
	}

	// Query manager pods
	managerPods, err := o.clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{
		LabelSelector: "app=kpt-manager",
	})
	if err != nil {
		fmt.Printf("Failed to get manager pods: %v\n", err)
	} else {
		fmt.Printf("Found manager pods: %d\n", len(managerPods.Items))
		for _, pod := range managerPods.Items {
			fmt.Printf("  - %s/%s (%s)\n", pod.Namespace, pod.Name, pod.Status.Phase)
		}
	}

	// Query probe pods
	probePods, err := o.clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{
		LabelSelector: "app=kpt-probe",
	})
	if err != nil {
		fmt.Printf("Failed to get probe pods: %v\n", err)
	} else {
		fmt.Printf("Found probe pods: %d\n", len(probePods.Items))
		for _, pod := range probePods.Items {
			fmt.Printf("  - %s/%s (%s)\n", pod.Namespace, pod.Name, pod.Status.Phase)
		}
	}

	// Query visor pods
	visorPods, err := o.clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{
		LabelSelector: "app=kpt-visor",
	})
	if err != nil {
		fmt.Printf("Failed to get visor pods: %v\n", err)
	} else {
		fmt.Printf("Found visor pods: %d\n", len(visorPods.Items))
		for _, pod := range visorPods.Items {
			fmt.Printf("  - %s/%s (%s)\n", pod.Namespace, pod.Name, pod.Status.Phase)
		}
	}
}
