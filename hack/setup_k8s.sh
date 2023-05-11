#!/bin/bash -e

set -x
#---------------------------- Parse Argument -----------------------------------
http_proxy='http://proxy-prc.intel.com:913'
proxy_flag='on'
init_for_master="true"

k8s_version='1.26.3'
pod_cidr='10.244.0.0/16'
service_cidr='10.96.0.0/16'
api_server_ip=

while getopts ":a:f:v:m:h" options; do
  case "$options" in
  "a")
    api_server_ip=$OPTARG
    echo ">>>>>>>>>>>>>> api server ip ${api_server_ip}"
    ;;
  "f")
    echo ">>>>>>>>>>>>>> proxy off"
    proxy_flag=$OPTARG
    ;;
  "v")
    k8s_version=$OPTARG
    echo ">>>>>>>>>>>>>> specific K8s version to ${k8s_version}"
    ;;
  "m")
    init_for_master=$OPTARG
    echo ">>>>>>>>>>>>>> init k8s master $OPTARG"
    ;;
  "h")
    echo '
      -f [off/on] proxy off
      -v specify the k8s version
      -h help
    '
    exit 1
    ;;
  ":")
    echo 'unknown options $OPTARG'
    exit 1
    ;;
  *)
    echo "unknown error while processing options"
    exit 1
    ;;
  esac
done

if [ $proxy_flag == 'off' ]; then
  echo "WARNING: Suggest use -f option to set proxy for apt & docker"
fi

#-------------------- Set Swapoff ----------------------------------------
sudo swapoff -a
sudo sed -i '/ swap / s/^\(.*\)$/#\1/g' /etc/fstab

#-------------------- Set proxy for APT ----------------------------------------
function set_proxy_for_apt() {
  cat <<EOF | sudo tee /etc/apt/apt.conf.d/proxy.conf
Acquire::http::Proxy "${http_proxy}";
Acquire::https::Proxy "${http_proxy}";
EOF
  echo ">>>>>>>>>>>>>>>>>>>>>>> Set Proxy for Apt <<<<<<<<<<<<<<<<<<<<<<<<<<<<<"
}

if [ $proxy_flag == 'on' ]; then
  set_proxy_for_apt
  export http_proxy='http://proxy-prc.intel.com:913'
  export https_proxy='http://proxy-prc.intel.com:913'

fi

#-------------------- Set proxy for Containerd -------------------------------------
function set_proxy_for_containerd() {
  echo ">>>>>>>>>>>>>>>>>>>>>>> Set Proxy for Containerd <<<<<<<<<<<<<<<<<<<<<<<"
  sudo mkdir -p /etc/systemd/system/containerd.service.d/
  cat <<EOF | sudo tee /etc/systemd/system/containerd.service.d/proxy.conf
[Service]
Environment="HTTP_PROXY=http://proxy-prc.intel.com:913/"
Environment="HTTPS_PROXY=http://proxy-prc.intel.com:913/"
Environment="NO_PROXY=localhost,127.0.0.1,10.239.0.0/16,10.244.0.0/16,10.96.0.0/16,10.0.0/24"
EOF
  sudo systemctl daemon-reload
  sudo systemctl restart containerd
  sleep 20s

}

#-------------------------------------------------------------------------------
#                          Pre Configure of K8s
#-------------------------------------------------------------------------------
#----------------- Let iptables see bridged traffic ----------------------------
function iptables_conf() {
  cat <<EOF | sudo tee /etc/modules-load.d/k8s.conf
overlay
br_netfilter
EOF

  sudo modprobe overlay
  sudo modprobe br_netfilter

  # sysctl params required by setup, params persist across reboots
  cat <<EOF | sudo tee /etc/sysctl.d/k8s.conf
net.bridge.bridge-nf-call-iptables  = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward                 = 1
EOF

  # Apply sysctl params without reboot
  sudo sysctl --system

  lsmod | grep br_netfilter
  lsmod | grep overlay
  sysctl net.bridge.bridge-nf-call-iptables net.bridge.bridge-nf-call-ip6tables net.ipv4.ip_forward
}

#---------------------- Install Container Runtime ------------------------------
# ----------------------  Containerd Install------------------------------------
function containerd_install() {

  RUNC_VERSION=v1.1.4
  if [ ! -e runc.amd64 ]; then
    wget -c  https://github.com/opencontainers/runc/releases/download/$RUNC_VERSION/runc.amd64
  fi
  sudo install -m 755 runc.amd64 /usr/local/sbin/runc

  CONTAINERD_VERSION=1.7.0
  if [ ! -e containerd-$CONTAINERD_VERSION-linux-amd64.tar.gz ]; then
    wget -c  https://github.com/containerd/containerd/releases/download/v$CONTAINERD_VERSION/containerd-$CONTAINERD_VERSION-linux-amd64.tar.gz
  fi
  sudo tar Czxvf /usr/local containerd-$CONTAINERD_VERSION-linux-amd64.tar.gz

  if [ ! -e containerd.service ]; then
    wget -c https://raw.githubusercontent.com/containerd/containerd/main/containerd.service
  fi

  sudo mv containerd.service /usr/lib/systemd/system/
  sudo systemctl daemon-reload
  sudo systemctl enable --now containerd
  #  sudo systemctl status containerd

  # set_proxy_for_containerd
  if [ $proxy_flag == "on" ]; then
    set_proxy_for_containerd
  fi

  sudo mkdir -p /etc/containerd/
  containerd config default | sudo tee /etc/containerd/config.toml > /dev/null 2>&1
  sudo sed -i 's/SystemdCgroup \= false/SystemdCgroup \= true/g' /etc/containerd/config.toml
  sudo sed -i 's/disable \= true/disable \= false/g' /etc/containerd/config.toml

  sudo systemctl restart containerd

  sleep 20s

  sudo chmod 666 /run/containerd/containerd.sock
}

function kube_install() {
  #sudo apt-get update
  sudo apt-get install -y apt-transport-https ca-certificates curl \
    net-tools ipvsadm

  if [ $proxy_flag == 'on' ]; then
    sudo curl -x ${http_proxy} \
      -fsSLo /usr/share/keyrings/kubernetes-archive-keyring.gpg \
      https://packages.cloud.google.com/apt/doc/apt-key.gpg

  else
    sudo curl -fsSLo /usr/share/keyrings/kubernetes-archive-keyring.gpg \
      https://packages.cloud.google.com/apt/doc/apt-key.gpg
  fi

  echo "deb [signed-by=/usr/share/keyrings/kubernetes-archive-keyring.gpg] \
  https://apt.kubernetes.io/ kubernetes-xenial main" |
    sudo tee /etc/apt/sources.list.d/kubernetes.list

  sudo apt-get update
  sudo apt-mark unhold kubelet kubeadm kubectl
  sudo apt-get install -y \
    kubelet=${k8s_version}-00 kubeadm=${k8s_version}-00 \
    kubectl=${k8s_version}-00
  sudo apt-mark hold kubelet kubeadm kubectl

  if [ ! -z $docker_user ]; then
    # temp solution to avoid docker pull from docker hub reach limit issue
    sudo cp /root/.docker/config.json /var/lib/kubelet/
  fi
}

# --------------------------- K8s Cluster Init ---------------------------------
function kube_init_master() {
  ipaddr=$(hostname -I | cut -d' ' -f1)

  echo "auto detected IP for api server:" $ipaddr

  if [ ! -z ${api_server_ip} ]; then
    echo "Use user specfied api server IP ${api_server_ip}"
    ipaddr=${api_server_ip}
  else
    echo "auto detected IP for api server:" $ipaddr
    echo "Using Server IP:" $ipaddr
    # read -p "K8s public IP for apiserver will set as [$ipaddr], if it is correct, just click <enter>, else
    # input the write public ip:" newIP

    if [ $newIP ]; then
      echo "specfic an IP "
      ipaddr=$newIP
    fi

  fi

  echo "---------------K8s API Ip is $ipaddr------------------------"
  echo "---------------K8s API Ip is $ipaddr------------------------"
  echo "---------------K8s API Ip is $ipaddr------------------------"

  sudo kubeadm init --kubernetes-version=${k8s_version} \
    --pod-network-cidr=${pod_cidr} --service-cidr=${service_cidr} \
    --apiserver-advertise-address=${ipaddr} \
    --cri-socket=unix:///run/containerd/containerd.sock

  mkdir -p $HOME/.kube
  sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
  sudo chown $(id -u):$(id -g) $HOME/.kube/config
  # let master also be a work node
  # kubectl taint nodes --all node-role.kubernetes.io/master-
  # kubectl taint nodes --all node-role.kubernetes.io/control-plane-

  # kubectl autocomplete
  cat <<EOF | sudo tee ~/.alias
source <(kubectl completion bash)
alias k="kubectl"
complete -o default -F __start_kubectl k
EOF
  echo 'source ~/.alias' >>~/.bashrc

  # helm install

  if [ $proxy_flag == "on" ]; then
    sudo snap set system proxy.https="${http_proxy}"
    sudo snap set system proxy.http="${http_proxy}"
  fi # set_
  sudo snap install helm --classic
  sudo snap install yq --classic
  sudo apt install -y jq

  #   sleep 30s
}

# apt non interactive
export DEBIAN_FRONTEND=noninteractive

# ---------------- Prepare the Network -----------------------------------------
iptables_conf
containerd_install
kube_install
if [ $init_for_master == "true" ]; then
  kube_init_master
  echo "WARNING: Suggest use -f option to set proxy for apt & docker"
fi
