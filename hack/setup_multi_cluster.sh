#!/bin/bash


# create parent cluster with 1 master and 2 work node
./setup_vm.sh -c 4 -m 8 -d 40 -v jammy -p parent -s 0
./setup_vm.sh -c 4 -m 8 -d 40 -v jammy -p parent -s 1
./setup_vm.sh -c 4 -m 8 -d 40 -v jammy -p parent -s 2

# create child-1 cluster with 1 master and 2 work node
./setup_vm.sh -c 4 -m 8 -d 40 -v jammy -p child-1 -s 0
./setup_vm.sh -c 4 -m 8 -d 40 -v jammy -p child-1 -s 1
./setup_vm.sh -c 4 -m 8 -d 40 -v jammy -p child-1 -s 2

# create child-2 cluster with 1 master and 2 work node
./setup_vm.sh -c 4 -m 8 -d 40 -v jammy -p child-2 -s 0
./setup_vm.sh -c 4 -m 8 -d 40 -v jammy -p child-2 -s 1
./setup_vm.sh -c 4 -m 8 -d 40 -v jammy -p child-2 -s 2

# create 3 cluster
multipass exec parent-node-0  -- bash -c "cd /home/ubuntu/hack && ./setup_k8s.sh -f on ;  kubectl apply -f ./deployment/kube-flannel.yml"
multipass exec child-1-node-0 -- bash -c "cd /home/ubuntu/hack && ./setup_k8s.sh -f on ;  kubectl apply -f ./deployment/kube-flannel.yml"
multipass exec child-2-node-0 -- bash -c "cd /home/ubuntu/hack && ./setup_k8s.sh -f on ;  kubectl apply -f ./deployment/kube-flannel.yml"

multipass exec parent-node-1  -- bash -c "cd /home/ubuntu/hack && ./setup_k8s.sh -f on -m false "
multipass exec child-1-node-1 -- bash -c "cd /home/ubuntu/hack && ./setup_k8s.sh -f on -m false "
multipass exec child-2-node-1 -- bash -c "cd /home/ubuntu/hack && ./setup_k8s.sh -f on -m false "

multipass exec parent-node-2  -- bash -c "cd /home/ubuntu/hack && ./setup_k8s.sh -f on -m false "
multipass exec child-1-node-2 -- bash -c "cd /home/ubuntu/hack && ./setup_k8s.sh -f on -m false "
multipass exec child-2-node-2 -- bash -c "cd /home/ubuntu/hack && ./setup_k8s.sh -f on -m false "
