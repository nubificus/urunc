# Setup a single node k3s cluster in Ubuntu 22.04

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

POD_CIDR="10.240.32.0/19"
kubectl create -f https://raw.githubusercontent.com/projectcalico/calico/v3.29.1/manifests/tigera-operator.yaml
wget https://raw.githubusercontent.com/projectcalico/calico/v3.29.1/manifests/custom-resources.yaml
sed -i.bak "s|192\.168\.0\.0/16|${POD_CIDR}|g" custom-resources.yaml
kubectl apply -f custom-resources.yaml

kubectl taint nodes --all node-role.kubernetes.io/control-plane-
```
