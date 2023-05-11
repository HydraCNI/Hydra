#!/bin/bash

export key=$(sudo cat /etc/kubernetes/pki/etcd/server.key)
export cert=$(cat /etc/kubernetes/pki/etcd/server.crt)
export ca=$(cat /etc/kubernetes/pki/etcd/ca.crt)

yq -i '.data."server.key" =  strenv(key)' /home/ubuntu/hack/deployment/flannel-etcd-cert.yaml
yq -i '.data."server.crt"=  strenv(cert)'  /home/ubuntu/hack/deployment/flannel-etcd-cert.yaml
yq -i '.data."ca.crt" =  strenv(ca)' /home/ubuntu/hack/deployment/flannel-etcd-cert.yaml
