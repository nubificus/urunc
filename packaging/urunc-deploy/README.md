# urunc-deploy

## k3s quickstart

To create a k3s cluster:

```bash
POD_CIDR="10.240.32.0/19"
SERVICE_CIDR="10.240.0.0/19"
curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC='--flannel-backend=none' sh -s - --disable-network-policy --disable "servicelb" --disable "metrics-server" --cluster-cidr $POD_CIDR --service-cidr $SERVICE_CIDR

 sudo addgroup k3s-admin
 sudo adduser $USER k3s-admin
 sudo usermod -a -G k3s-admin $USER
 sudo chgrp k3s-admin /etc/rancher/k3s/k3s.yaml
 sudo chmod g+r /etc/rancher/k3s/k3s.yaml
 su $USER

kubectl create -f https://raw.githubusercontent.com/projectcalico/calico/v3.29.1/manifests/tigera-operator.yaml
wget https://raw.githubusercontent.com/projectcalico/calico/v3.29.1/manifests/custom-resources.yaml
sed -i.bak "s|192\.168\.0\.0/16|${POD_CIDR}|g" custom-resources.yaml
kubectl apply -f custom-resources.yaml
```

To install in a k3s cluster:

```bash
git clone https://github.com/nubificus/urunc.git
cd urunc
git checkout feat_urunc-deploy
cd packaging/urunc-deploy
kubectl apply -f urunc-rbac/base/urunc-rbac.yaml 
kubectl apply -k urunc-deploy/overlays/k3s
kubectl apply -f urunc-deploy/base/urunc-deploy.yaml



kubectl apply -f urunc-deploy/base/urunc-deploy.yaml && kubectl apply -k urunc-deploy/overlays/k3s && kubectl apply -f urunc-deploy/base/urunc-deploy.yaml
kubectl delete -f urunc-deploy/base/urunc-deploy.yaml && kubectl delete -k urunc-deploy/overlays/k3s && kubectl delete -f urunc-deploy/base/urunc-deploy.yaml
```

```bash
docker build --push -t gntouts/urunc-deploy:0.0.13 .
```