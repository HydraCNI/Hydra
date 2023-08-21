## Hydra

Hydra enables attaching multiple network interfaces to pods in Kubernetes leverage NRI.

### Make Sure Containerd Version >= v1.7.0 with NRI enabled.

```shell
CONTAINERD_VERSION=1.7.0
wget -c https://github.com/containerd/containerd/releases/download/v$CONTAINERD_VERSION/containerd-$CONTAINERD_VERSION-linux-amd64.tar.gz 
sudo tar Czxvf /usr/local containerd-$CONTAINERD_VERSION-linux-amd64.tar.gz 
sudo mkdir -p /etc/containerd/ 
containerd config default | sudo tee /etc/containerd/config.toml > /dev/null 2>&1
sudo sed -i 's/SystemdCgroup \= false/SystemdCgroup \= true/g' /etc/containerd/config.toml 
sudo sed -i 's/disable \= true/disable \= false/g' /etc/containerd/config.toml
```

### How to Deploy with another flannel

#### Prepare etcd for flannel subnetManger on master node
```shell
git clone https://github.com/HydraCNI/Hydra.git
cd Hydra/hack/deployment/flannel
./update_etcd_cert.sh
kubectl apply -f hydra-cni-flannel-etcd-cert.yaml
kubectl apply -f etcd-service.yaml
```

check etcd service status
```shell
 kubectl describe svc -n kube-system kube-etcd
```
#### Deploy hydra-flannel

```shell
kubectl apply -f hydra-cni-kube-flannel.yml
```

#### Deploy hydra-cni
```shell
cd Hydra/hack/deployment/ 
kubectl apply -f clusternet-hydra-cni-daemonset.yaml
```