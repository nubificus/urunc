# `urunc-deploy`

[`urunc-deploy`](.) provides a Dockerfile, which contains all of the binaries
and artifacts required to run `urunc`, as well as reference DaemonSets, which can
be utilized to install `urunc` runtime  on a running Kubernetes cluster.

## Install `urunc` using `urunc-deploy`

### k3s

To install in a k3s cluster, first we need to create the RBAC:

```bash
git clone https://github.com/nubificus/urunc.git
kubectl apply -f urunc/packaging/urunc-deploy/urunc-rbac/base/urunc-rbac.yaml
```

Then, we create the `urunc-deploy` Daemonset, followed by the k3s customization:

```bash
kubectl apply -k urunc/packaging/urunc-deploy/urunc-deploy/overlays/k3s
kubectl apply -f urunc/packaging/urunc-deploy/urunc-deploy/base/urunc-deploy.yaml
```
