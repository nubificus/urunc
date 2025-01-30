# urunc-deploy

TODO:

- k3s with containerd<2: DONE
- k3s with containerd>2: not sure if possible ATM
- k8s with containerd<2: PENDING
- k8s with containerd>2: PENDING
- k8s with CRI-0: WIP
- k0s: WIP

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

kubectl apply -f urunc-rbac/base/urunc-rbac.yaml && kubectl apply -f urunc-deploy/base/urunc-deploy.yaml && kubectl apply -k urunc-deploy/overlays/k3s && echo "OK"
kubectl delete -f urunc-deploy/base/urunc-deploy.yaml && kubectl delete -k urunc-deploy/overlays/k3s && kubectl delete -f urunc-deploy/base/urunc-deploy.yaml
```

Test the successful installation:

```bash
kubectl apply -f examples/nginx-urunc.yaml
```

```bash
docker build --push -t gntouts/urunc-deploy:0.1.1 .
docker build --push -t harbor.nbfc.io/nubificus/urunc/urunc-deploy:0.4.0-rc1 .
```

## k8s - crio quickstart

To create a k8s cluster with crio:

```bash
# Run as root
apt-get update
apt-get install -y software-properties-common curl

curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.28/deb/Release.key |
    gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg
echo "deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.28/deb/ /" |
    tee /etc/apt/sources.list.d/kubernetes.list

curl -fsSL https://pkgs.k8s.io/addons:/cri-o:/prerelease:/main/deb/Release.key |
    gpg --dearmor -o /etc/apt/keyrings/cri-o-apt-keyring.gpg
echo "deb [signed-by=/etc/apt/keyrings/cri-o-apt-keyring.gpg] https://pkgs.k8s.io/addons:/cri-o:/prerelease:/main/deb/ /" |
    tee /etc/apt/sources.list.d/cri-o.list

apt-get update
apt-get install -y cri-o kubelet kubeadm kubectl
systemctl start crio.service
```

```bash
sudo sed -i '/swap.img/s/^/#/' /etc/fstab
sudo swapoff -a
sudo rm -fr /swap.img
cat <<EOF | sudo tee /etc/modules-load.d/k8s.conf
overlay
br_netfilter
EOF

sudo modprobe overlay
sudo modprobe br_netfilter

cat <<EOF | sudo tee /etc/sysctl.d/k8s.conf
net.bridge.bridge-nf-call-iptables = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward = 1
EOF

sudo sysctl --system
```

```bash
NETWORK_CIDR=10.80.0.0/16
sudo kubeadm init --pod-network-cidr=$NETWORK_CIDR
```

```bash
mkdir -p $HOME/.kube
sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config

wget -q https://raw.githubusercontent.com/projectcalico/calico/v3.26.0/manifests/tigera-operator.yaml
wget -q https://raw.githubusercontent.com/projectcalico/calico/v3.26.0/manifests/custom-resources.yaml
sed -i 's/cidr: 192.168.0.0\/16/cidr: 10.80.0.0\/16/' custom-resources.yaml
kubectl create -f tigera-operator.yaml
kubectl create -f custom-resources.yaml

rm -f tigera-operator.yaml
rm -f custom-resources.yaml

kubectl taint nodes --all node-role.kubernetes.io/control-plane-
```
