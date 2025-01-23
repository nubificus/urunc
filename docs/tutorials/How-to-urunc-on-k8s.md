# How to use urunc with k8s

This guide assumes you have a working Kubernetes cluster and have [installed urunc](../installation.md) on one or more nodes.

## Add urunc as a RuntimeClass

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

## Create a test deployment

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
      - image: harbor.nbfc.io/nubificus/urunc/nginx-hvt-rumprun-block:latest
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
