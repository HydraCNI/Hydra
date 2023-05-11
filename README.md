## Hydra

### How to Deploy with another flannel

#### Prepare etcd for flannel subnetManger on master node
```shell
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
kubectl apply -f hydra-cni-daemonset.yaml
```