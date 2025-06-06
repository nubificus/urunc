# How to use urunc with k8s

This guide assumes you have a working Kubernetes cluster.

To use `urunc` in a k8s cluster there are 2 options:

- [Manual installation](#manual-installation)
- [Using urunc-deploy](#urunc-deploy)

## Manual Installation

### Install urunc

Before we start, we need to have working Kubernetes cluster with [urunc installed](../installation.md) on one or more nodes.

### Add urunc as a RuntimeClass

First, we need to add `urunc` as a runtime class for the k8s cluster:

```bash
cat << EOF | tee urunc-runtimeClass.yaml
kind: RuntimeClass
apiVersion: node.k8s.io/v1
metadata:
    name: urunc
handler: urunc
EOF

kubectl apply -f urunc-runtimeClass.yaml
```

To verify the runtimeClass was added:

```bash
kubectl get runtimeClass
```

### Create a test deployment

To properly test the newly added k8s runtime class, create a test deployment:

```bash
cat <<EOF | tee nginx-urunc.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    run: nginx-urunc
  name: nginx-urunc
spec:
  replicas: 1
  selector:
    matchLabels:
      run: nginx-urunc
  template:
    metadata:
      labels:
        run: nginx-urunc
    spec:
      runtimeClassName: urunc
      containers:
      - image: harbor.nbfc.io/nubificus/urunc/nginx-firecracker-unikraft-initrd:latest
        imagePullPolicy: Always
        name: nginx-urunc
        command: ["sleep"]
        args: ["infinity"]
        ports:
        - containerPort: 80
          protocol: TCP
        resources:
          requests:
            cpu: 10m
      restartPolicy: Always
---
apiVersion: v1
kind: Service
metadata:
  name: nginx-urunc
spec:
  ports:
  - port: 80
    protocol: TCP
    targetPort: 80
  selector:
    run: nginx-urunc
  sessionAffinity: None
  type: ClusterIP
EOF

kubectl apply -f nginx-urunc.yaml
```

Now, we should be able to see the created Pod:

```bash
kubectl get pods
```

## urunc-deploy

[`urunc-deploy`](https://github.com/urunc-dev/urunc/tree/main/deployment/urunc-deploy) provides a Dockerfile, which contains all of the binaries
and artifacts required to run `urunc`, as well as reference DaemonSets, which can
be utilized to install `urunc` runtime  on a running Kubernetes cluster.

### urunc-deploy in k3s

To install in a k3s cluster, first we need to create the RBAC:

```bash
git clone https://github.com/urunc-dev/urunc.git
cd urunc
kubectl apply -f deployment/urunc-deploy/urunc-rbac/urunc-rbac.yaml
```

Then, we create the `urunc-deploy` Daemonset, followed by the k3s customization:

```bash
kubectl apply -k deployment/urunc-deploy/urunc-deploy/overlays/k3s
```

Finally, we need to create the appropriate k8s runtime class:

```bash
kubectl apply -f deployment/urunc-deploy/runtimeclasses/runtimeclass.yaml
```

To uninstall:

```bash
kubectl delete -k deployment/urunc-deploy/urunc-deploy/overlays/k3s
kubectl apply -k deployment/urunc-deploy/urunc-cleanup/overlays/k3s
```

After the cleanup is completed and the `urunc-deploy` Pod is terminated:

```bash
kubectl delete -k deployment/urunc-deploy/urunc-cleanup/overlays/k3s
kubectl delete -f deployment/urunc-deploy/urunc-rbac/urunc-rbac.yaml
kubectl delete -f deployment/urunc-deploy/runtimeclasses/runtimeclass.yaml
```

### urunc-deploy in k8s with containerd

To install in a k8s cluster, first we need to create the RBAC:

```bash
git clone https://github.com/urunc-dev/urunc.git
cd urunc
kubectl apply -f deployment/urunc-deploy/urunc-rbac/urunc-rbac.yaml
```

Then, we create the `urunc-deploy` Daemonset:

```bash
kubectl apply -f deployment/urunc-deploy/urunc-deploy/base/urunc-deploy.yaml
```

Finally, we need to create the appropriate k8s runtime class:

```bash
kubectl apply -f deployment/urunc-deploy/runtimeclasses/runtimeclass.yaml
```

To uninstall:

```bash
kubectl delete -f deployment/urunc-deploy/urunc-deploy/base/urunc-deploy.yaml
kubectl apply -f deployment/urunc-deploy/urunc-cleanup/base/urunc-cleanup.yaml
```

After the cleanup is completed:

```bash
kubectl delete -f deployment/urunc-deploy/urunc-cleanup/base/urunc-cleanup.yaml
kubectl delete -f deployment/urunc-deploy/urunc-rbac/urunc-rbac.yaml
kubectl delete -f deployment/urunc-deploy/runtimeclasses/runtimeclass.yaml
```

Now, we can create new `urunc` deployments using the [instruction provided in manual installation](#create-a-test-deployment).

### How urunc-deploy works

`urunc-deploy` consists of several components and steps that install `urunc` along with the supported hypervisors,
configure `containerd` and Kubernetes (k8s) to use `urunc`, and provide a simple way to remove those components from the cluster.

During installation, the following steps take place:

- A RBAC role is created to allow `urunc-deploy` to run with privileged access.
- The `urunc-deploy` Pod is deployed with privileges on the host, and the `containerd` configuration is mounted inside the Pod.
- `urunc-deploy` performs the following tasks:
    * Copies `urunc` and hypervisor binaries to the host under `usr/local/bin`.
    * Creates a backup of the current `containerd` configuration file.
    * Edits the `containerd` configuration file to add `urunc` as a supported runtime.
    * Restarts `containerd`, if necessary.
    * Labels the Node with label `urunc.io/urunc-runtime=true`.
- Finally, `urunc` is added as a runtime class in k8s.

> Note: `urunc-deploy` will install a static version of QEMU along with the QEMU BIOS files. The QEMU BIOS files are placed
under the `/usr/share` directory. If the host system already had installed QEMU, then QEMU binary and the BIOS files in `/usr/share`, will get overwritten.

During cleanup, these changes are reverted:

- The `urunc` and hypervisor binaries are deleted.
- The `containerd` configuration file is restored to the pre-`urunc-deploy` state.
- The `urunc.io/urunc-runtime=true` label is removed from the Node.
- The RBAC role, the `urunc-deploy` Pod and the runtime class are removed.
