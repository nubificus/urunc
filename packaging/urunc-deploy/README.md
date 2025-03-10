# `urunc-deploy`

[`urunc-deploy`](.) provides a Dockerfile, which contains all of the binaries
and artifacts required to run `urunc`, as well as reference DaemonSets, which can
be utilized to install `urunc` runtime  on a running Kubernetes cluster.

## Install `urunc` using `urunc-deploy`

### k3s

To install in a k3s cluster, first we need to create the RBAC:

```bash
git clone https://github.com/nubificus/urunc.git
cd urunc
kubectl apply -f packaging/urunc-deploy/urunc-rbac/base/urunc-rbac.yaml
```

Then, we create the `urunc-deploy` Daemonset, followed by the k3s customization:

```bash
kubectl apply -f packaging/urunc-deploy/urunc-deploy/base/urunc-deploy.yaml
kubectl apply -k packaging/urunc-deploy/urunc-deploy/overlays/k3s
```

Finally, we need to create the appropriate k8s runtime class:

```bash
kubectl apply -f packaging/urunc-deploy/runtimeclasses/runtimeclass.yaml
```

To uninstall:

```bash
kubectl delete -f packaging/urunc-deploy/urunc-deploy/base/urunc-deploy.yaml
kubectl apply -k packaging/urunc-deploy/urunc-cleanup/overlays/k3s
kubectl apply -f packaging/urunc-deploy/urunc-cleanup/base/urunc-cleanup.yaml
kubectl delete -f packaging/urunc-deploy/urunc-cleanup/base/urunc-cleanup.yaml
kubectl delete -f packaging/urunc-deploy/urunc-rbac/base/urunc-rbac.yaml
kubectl delete -f packaging/urunc-deploy/runtimeclasses/runtimeclass.yaml
```

### k8s with containerd 1.7.x

To install in a k8s cluster, first we need to create the RBAC:

```bash
git clone https://github.com/nubificus/urunc.git
cd urunc
kubectl apply -f packaging/urunc-deploy/urunc-rbac/base/urunc-rbac.yaml
```

Then, we create the `urunc-deploy` Daemonset:

```bash
kubectl apply -f packaging/urunc-deploy/urunc-deploy/base/urunc-deploy.yaml
```

Finally, we need to create the appropriate k8s runtime class:

```bash
kubectl apply -f packaging/urunc-deploy/runtimeclasses/runtimeclass.yaml
```

To uninstall:

```bash
kubectl delete -f packaging/urunc-deploy/urunc-deploy/base/urunc-deploy.yaml
kubectl apply -f packaging/urunc-deploy/urunc-cleanup/base/urunc-cleanup.yaml
kubectl delete -f packaging/urunc-deploy/urunc-cleanup/base/urunc-cleanup.yaml
kubectl delete -f packaging/urunc-deploy/urunc-rbac/base/urunc-rbac.yaml
kubectl delete -f packaging/urunc-deploy/runtimeclasses/runtimeclass.yaml
```

### k8s with containerd 2.0.x

To install in a k8s cluster, first we need to create the RBAC:

```bash
git clone https://github.com/nubificus/urunc.git
cd urunc
kubectl apply -f packaging/urunc-deploy/urunc-rbac/base/urunc-rbac.yaml
```

Then, we create the `urunc-deploy` Daemonset:

```bash
kubectl apply -f packaging/urunc-deploy/urunc-deploy/base/urunc-deploy.yaml
```

Finally, we need to create the appropriate k8s runtime class:

```bash
kubectl apply -f packaging/urunc-deploy/runtimeclasses/runtimeclass.yaml
```

To uninstall:

```bash
kubectl delete -f packaging/urunc-deploy/urunc-deploy/base/urunc-deploy.yaml
kubectl apply -f packaging/urunc-deploy/urunc-cleanup/base/urunc-cleanup.yaml
kubectl delete -f packaging/urunc-deploy/urunc-cleanup/base/urunc-cleanup.yaml
kubectl delete -f packaging/urunc-deploy/urunc-rbac/base/urunc-rbac.yaml
kubectl delete -f packaging/urunc-deploy/runtimeclasses/runtimeclass.yaml
```
