# RBAC Controller

[![Go Report Card](https://goreportcard.com/badge/github.com/GGh41th/rbac-controller)](https://goreportcard.com/report/github.com/GGh41th/rbac-controller)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

A Kubernetes controller that simplifies RBAC management by providing a declarative way to manage RoleBindings and ClusterRoleBindings across multiple namespaces.

## Overview

RBAC Controller extends Kubernetes with a custom resource `RBACRule` that allows you to:

- Define RoleBindings and ClusterRoleBindings for multiple subjects in a single manifest
- Automatically propagate bindings across multiple namespaces using label selectors or match expressions
- Manage RBAC for Users, Groups, and ServiceAccounts with a unified API
- Reduce boilerplate when managing RBAC across many namespaces

## Features

- 🎯 **Multi-namespace binding**: Apply RoleBindings across multiple namespaces 
- 🔐 **Unified RBAC management**: Manage both RoleBindings and ClusterRoleBindings in one resource
- 🏷️ **Label-based selection**: Use namespace label selectors for dynamic binding management


## Getting Started

### Prerequisites

- A running Kubernetes cluster (you can use kubeadm,minikube,kind etc.. )
- Go 1.25+ (for development)

### Installation

Install the controller using kubectl:

```bash
kubectl apply -f https://raw.githubusercontent.com/GGh41th/rbac-controller/main/dist/install.yaml
```

Or build and deploy manually:

```bash
# Install CRDs
make docker-build registry/user/repository:tag

# Deploy controller
make docker-deploy IMG=registry/user/repository:tag
```

### Quick Start

Create an `RBACRule` to bind a ServiceAccount to a Role across multiple namespaces:

```yaml
apiVersion: rbac-controller.ggh41th.io/v1alpha1
kind: RBACRule
metadata:
  name: developer-access
spec:
  bindings:
  - name: dev-binding
    subjects:
    - kind: ServiceAccount
      name: developer-sa
      namespaceSelector:
        matchLabels:
          environment: development
    roleBindings:
    - role: developer-role
      namespaceSelector:
        matchLabels:
          environment: development
```

Apply the resource:

```bash
kubectl apply -f rbacrule.yaml
```

For more examples, see the [examples](./examples) directory.

## Usage

The `RBACRule` resource supports the following subject types:

- `User` - Kubernetes user
- `Group` - Kubernetes group  
- `ServiceAccount` - Kubernetes ServiceAccount

### Namespace Selection

You can select namespaces in three ways:

1. **Explicit list**: `namespaces: [ns1, ns2, ns3]`
2. **Label selector**: `namespaceSelector: {matchLabels: {env: prod}}`
3. **Match expression**: `namespaceMatchExpression: "metadata.name in ['ns1', 'ns2']"`

### Examples

#### RoleBinding across multiple namespaces

```yaml
apiVersion: rbac-controller.ggh41th.io/v1alpha1
kind: RBACRule
metadata:
  name: multi-ns-binding
spec:
  bindings:
  - name: my-binding
    subjects:
    - kind: User
      name: john@example.com
      namespaces: [team-a, team-b, team-c]
    roleBindings:
    - role: edit
      namespaces: [team-a, team-b, team-c]
```

#### ClusterRoleBinding for ServiceAccount

```yaml
apiVersion: rbac-controller.ggh41th.io/v1alpha1
kind: RBACRule
metadata:
  name: cluster-binding
spec:
  bindings:
  - name: cluster-admin-binding
    subjects:
    - kind: ServiceAccount
      name: admin-sa
      namespaces: [kube-system]
    clusterRoleBindings:
    - clusterRole: cluster-admin
```

See [examples](./examples) for more usage patterns.

## Development

### Building from Source

```bash
# Build the binary
make build

# Run tests
make test

# Build and push Docker image
make docker-build docker-push IMG=<your-registry>/rbac-controller:tag
```

### Running Locally

```bash
# Install CRDs
make install

# Run controller locally against your kubeconfig cluster
make run
```

### Testing

```bash
# Run unit tests
make test

# Run e2e tests
make test-e2e
```

## Uninstallation

To remove the controller and CRDs:

```bash
# Delete sample resources
kubectl delete -k config/samples/

# Undeploy controller
make undeploy

# Remove CRDs
make uninstall
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/feature-xyz`)
3. Open a Pull Request with that feature branch.

## License

Copyright 2025 Ghaith Gtari.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
