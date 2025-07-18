package operator

import (
	"fmt"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/dynamic"
	"os"
	"path/filepath"
)

// Config represents the operator configuration
type Config struct {
	KubeConfig   string `yaml:"kubeConfig"`
	Namespace    string `yaml:"namespace"`
	LogLevel     string `yaml:"logLevel"`
	ProbeImage   string `yaml:"probeImage"`
	VisorImage   string `yaml:"visorImage"`
	ProbeEnabled bool   `yaml:"probeEnabled"`
	VisorEnabled bool   `yaml:"visorEnabled"`
}

// LoadConfig loads the configuration from a file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Default configuration
	config := &Config{
		Namespace:    "kpt-system",
		LogLevel:     "info",
		ProbeImage:   "kpt-probe:latest",
		VisorImage:   "kpt-visor:latest",
		ProbeEnabled: true,
		VisorEnabled: true,
	}

	// If configuration file exists, use its values
	if len(data) > 0 {
		// Here, you can parse the YAML file and override default values
		fmt.Println("Using configuration file:", path)
	}

	return config, nil
}

// GetKubernetesConfig returns a kubernetes rest.Config
func GetKubernetesConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}

	// First try to use in-cluster configuration
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	// If outside the cluster, try using kubeconfig
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("unable to get user home directory: %v", err)
	}

	kubeconfigPath := filepath.Join(homeDir, ".kube", "config")
	return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
}

// GetKubernetesClient returns a kubernetes clientset
func GetKubernetesClient() (*kubernetes.Clientset, error) {
	config, err := GetKubernetesConfig("")
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

// GetKubernetesClientWithConfig returns a kubernetes clientset with the given kubeconfig
func GetKubernetesClientWithConfig(kubeconfig string) (*kubernetes.Clientset, error) {
	config, err := GetKubernetesConfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

// GetDynamicClient returns a dynamic client
func GetDynamicClient(kubeconfig string) (dynamic.Interface, error) {
	config, err := GetKubernetesConfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	return dynamic.NewForConfig(config)
}
