# KubePodTap

This directory contains the Helm chart and deployment resources for the KubePodTop project.

## Directory Structure

- `helm/`         Helm chart for deploying KubeSeek and its components

## Usage

### Deploy with Helm

1. Edit `helm/values.yaml` to customize your deployment.
2. Install or upgrade with Helm:
   ```sh
   helm install kubeseek ./helm -n <namespace> --create-namespace
   # or
   helm upgrade --install kubeseek ./helm -n <namespace>
   ```

## License

This project is licensed under the Apache 2.0 License.
