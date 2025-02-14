# Setup K8S cluster with containerd v2

```bash
#!/env/bash

sudo apt-get update
sudo swapoff -a
sudo sed -i '/swap/s/^/#/' /etc/fstab
echo 'overlay' | sudo tee /etc/modules-load.d/kubernetes.conf
echo 'br_netfilter' | sudo tee -a /etc/modules-load.d/kubernetes.conf
sudo modprobe overlay
sudo modprobe br_netfilter
sudo tee -a /etc/sysctl.conf << 'EOL'
net.ipv4.ip_forward = 1
net.bridge.bridge-nf-call-iptables  = 1
net.bridge.bridge-nf-call-ip6tables = 1
EOL
sudo sysctl -p

wget https://github.com/opencontainers/runc/releases/download/v1.2.4/runc.amd64
chmod +x runc.amd64
sudo mv runc.amd64 /usr/local/sbin/runc

wget https://github.com/containernetworking/plugins/releases/download/v1.6.2/cni-plugins-linux-amd64-v1.6.2.tgz
sudo mkdir -p /opt/cni/bin
sudo tar Cxzvf /opt/cni/bin cni-plugins-linux-amd64-v1.6.2.tgz
rm cni-plugins-linux-amd64-v1.6.2.tgz

# Containerd 1.7.25
wget https://github.com/containerd/containerd/releases/download/v1.7.25/containerd-1.7.25-linux-amd64.tar.gz
sudo tar Cxzvf /usr/local containerd-1.7.25-linux-amd64.tar.gz
rm containerd-1.7.25-linux-amd64.tar.gz
sudo mkdir -p /etc/containerd
containerd config default | sed 's/SystemdCgroup = false/SystemdCgroup = true/' | sudo tee /etc/containerd/config.toml
sudo sed -i '/pause:3.8/s/3.8/3.10/' /etc/containerd/config.toml
sudo systemctl restart containerd

wget https://raw.githubusercontent.com/containerd/containerd/main/containerd.service
sudo mkdir -p /usr/local/lib/systemd/system
sudo mv containerd.service /usr/local/lib/systemd/system/containerd.service

sudo systemctl daemon-reload
sudo systemctl enable --now containerd

sudo apt install gnupg2 -y
curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.32/deb/Release.key | sudo gpg --dearmor -o /etc/apt/trusted.gpg.d/k8s.gpg
echo "deb https://pkgs.k8s.io/core:/stable:/v1.32/deb/ /" | sudo tee /etc/apt/sources.list.d/kurbenetes.list
sudo apt update
sudo apt install kubelet kubeadm kubectl -y
sudo apt-mark hold kubelet kubeadm kubectl
sudo systemctl enable --now kubelet

sudo kubeadm init --pod-network-cidr=10.244.0.0/16 | tee kubeadm-init.log

mkdir -p $HOME/.kube
sudo rm -rf $HOME/.kube/config
sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config

#Flannel
kubectl apply -f https://github.com/flannel-io/flannel/releases/latest/download/kube-flannel.yml
kubectl delete -f https://github.com/flannel-io/flannel/releases/latest/download/kube-flannel.yml

# Calico
wget -q https://raw.githubusercontent.com/projectcalico/calico/v3.29.1/manifests/tigera-operator.yaml
wget -q https://raw.githubusercontent.com/projectcalico/calico/v3.29.1/manifests/custom-resources.yaml
sed -i 's/cidr: 192.168.0.0\/16/cidr: 10.244.0.0\/16/' custom-resources.yaml
kubectl create -f tigera-operator.yaml
kubectl create -f custom-resources.yaml
rm -f tigera-operator.yaml custom-resources.yaml

kubectl taint nodes --all node-role.kubernetes.io/control-plane-
```
