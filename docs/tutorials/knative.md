# Knative + urunc: Deploying Serverless Unikernels

This guide walks you through deploying [Knative Serving](https://knative.dev/)
using [`urunc`](https://github.com/nubificus/urunc), a unikernel
container runtime. You’ll build Knative from a custom branch and use
[`ko`](https://github.com/ko-build/ko) for seamless image building and
deployment.

## Prerequisites

-   A running Kubernetes cluster
-   A Docker-compatible registry (e.g. Harbor, Docker Hub)
-   Ubuntu 20.04 or newer
-   Basic `git`, `curl`, `kubectl`, and `docker` installed
    

## Environment Setup

Install Docker, Go >= 1.21, and `ko`:

### Install Docker
```console
$ sudo apt-get update
$ sudo apt-get install -y ca-certificates curl
$ sudo install -m 0755 -d /etc/apt/keyrings
$ curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo tee /etc/apt/keyrings/docker.asc > /dev/null
$ sudo chmod a+r /etc/apt/keyrings/docker.asc echo  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu \ $(. /etc/os-release && echo "${UBUNTU_CODENAME:-$VERSION_CODENAME}") stable" | \
$ sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
$ sudo apt-get update && sudo apt-get install -y docker-ce docker-ce-cli containerd.io 
```

### Install Go 1.21  
```console
$ sudo mkdir /usr/local/go1.21
$ wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
$ sudo tar -zxvf go1.21.5.linux-amd64.tar.gz -C /usr/local/go1.21/
$ rm go1.21.5.linux-amd64.tar.gz
```

### Verify Go installation (Should be 1.21.5)

```console
$ export GOROOT=/usr/local/go1.21/go 
$ export PATH=$GOROOT/bin:$PATH  
$ export GOPATH=$HOME/go 
$ go version
go version go1.21.5 linux/amd64
```

### Install ko VERSION=0.15.1
```console
$ export OS=Linux
$ export ARCH=x86_64
$ curl -sSfL "https://github.com/ko-build/ko/releases/download/v${VERSION}/ko_${VERSION}_${OS}_${ARCH}.tar.gz" -o ko.tar.gz
$ sudo tar -zxvf ko.tar.gz -C /usr/local/bin` 
```

----------

## Clone and Build Knative with the queue-proxy patch

### Set your container registry  

> Note: You should be able to use dockerhub for this. e.g. `<yourdockerhubid>/knative
```console
$ export KO_DOCKER_REPO='harbor.nbfc.io/nubificus/knative-install-urunc'  
```

### Clone urunc-enabled Knative Serving 
```console
$ git clone https://github.com/nubificus/serving -b feat_urunc 
$ cd serving/
$ ko resolve -Rf ./config/core/ > knative-custom.yaml
```

### Apply knative's manifests to the local k8s
```console
$ kubectl apply -f knative-custom.yaml`
```

Alternatively, you could use our latest build:
```console
$ kubectl apply -f https://s3.nbfc.io/knative/knative-v1.17.0-urunc-5220308.yaml
```

----------

## Setup Networking (Kourier)


### Install kourier, patch ingress and domain configs

```console
$ kubectl apply -f https://github.com/knative/net-kourier/releases/latest/download/kourier.yaml 
$ kubectl patch configmap/config-network -n knative-serving --type merge -p \ 
  '{"data":{"ingress.class":"kourier.ingress.networking.knative.dev"}}' kubectl patch configmap/config-domain -n knative-serving --type merge -p \ 
  '{"data":{"127.0.0.1.nip.io":""}}'
```

----------

## Enable RuntimeClass and urunc Support


### Install `urunc`

You can follow the documentation to install `urunc` from: [Installing](https://urunc.io/tutorials/How-to-urunc-on-k8s/)

### Enable runtimeClass for services, nodeSelector and affinity

```bash
$ kubectl patch configmap/config-features --namespace knative-serving --type merge --patch '{"data":{
  "kubernetes.podspec-affinity":"enabled",
  "kubernetes.podspec-runtimeclassname":"enabled",
  "kubernetes.podspec-nodeselector":"enabled"
}}'
```

----------

## Deploy a Sample urunc Service
```bash
$ kubectl get ksvc -A -o wide
```
Should be empty. Get an example manifest and apply it:

```console
$ wget https://raw.githubusercontent.com/nubificus/openinfradayshu-demos/main/serverless-sandboxes/service-container-hello.yaml
$ kubectl apply -f service-container-hello.yaml 
```

### Check Knative Service 
 
```console
kubectl get ksvc -A -o wide 
```

### Test the service (replace IP with actual ingress IP) 

```bash
curl -v -H "Host: hellocontainer.default.127.0.0.1.nip.io" http://<INGRESS_IP>` 
```

Now, let's create a `urunc`-compatible function. Create a file (e.g. `urunc-function.yaml`) with the following contents:

```yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: http-c-urunc
  namespace: default
spec:
  template:
    spec:
      runtimeClassName: "urunc"
      containers:
        - image: harbor.nbfc.io/nubificus/knative/http-c:qemu-urunc
          imagePullPolicy: IfNotPresent
          ports:
          - containerPort: 8080
            protocol: TCP
          resources:
            requests:
              cpu: 10m
```

and apply it:

```console
$ kubectl apply -f urunc-function.yaml
```

You should be able to see this being created:

```console
$ kubectl get ksvc -o wide
NAME             URL                                              LATESTCREATED          LATESTREADY            READY   REASON
http-c-urunc     http://http-c-urunc.default.127.0.0.1.nip.io     http-c-urunc-00001     http-c-urunc-00001     True
```

and once it's on a `Ready` state, you could issue a request:
> Note: 10.244.9.220 is the IP of the `kourier-internal` svc. You can check your own from:
> `kubectl get svc -n kourier-system |grep kourier-internal`

```console
$ curl -v -H "Host: http-c-urunc.default.127.0.0.1.nip.io" http://10.244.9.220:80
*   Trying 10.244.9.220:80...
* Connected to 10.244.9.220 (10.244.9.220) port 80 (#0)
> GET / HTTP/1.1
> Host: http-c-urunc.default.127.0.0.1.nip.io
> User-Agent: curl/7.81.0
> Accept: */*
>
* Mark bundle as not supporting multiuse
< HTTP/1.1 200 OK
< content-length: 14
< content-type: text/html; charset=UTF-8
< date: Tue, 08 Apr 2025 15:47:45 GMT
< x-envoy-upstream-service-time: 774
< server: envoy
<
Hello, World!
* Connection #0 to host 10.244.9.220 left intact
```

## Wrapping Up

You're now running unikernel-based workloads via Knative and `urunc`! With this
setup, you can push the boundaries of lightweight, secure, and high-performance
serverless deployments — all within a Kubernetes-native environment.
