package kubeclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/containerd/nri/pkg/api"
	"github.com/containernetworking/cni/pkg/types"
	cnitypes "github.com/containernetworking/cni/pkg/types/040"
	"github.com/hydra-cni/hydra/pkg/cni"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	CNF_SELECTER  = "cross-cluster.clusternet.io.cnf=wireguard"
	CNF_NAMESPACE = "CNF_NAMESPACE"
)

var (
	clientSet       *kubernetes.Clientset
	informerFactory informers.SharedInformerFactory

	ParallelIpKey string
	cnfNamespace  string
)

func init() {

	cnfNamespace = os.Getenv(CNF_NAMESPACE)
}

func KubeInitializer() {

	ParallelIpKey = fmt.Sprintf("cross-cluster.clusternet.io.%v", cni.DefaultCNIPlugin.Name)

	config, err := rest.InClusterConfig()
	if err != nil {
		// fallback to kubeconfig
		kubeconfig := filepath.Join("./conf", "kubeconf.yaml")
		if val := os.Getenv("KUBECONFIG"); len(val) != 0 {
			kubeconfig = val
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			logrus.Fatalf("The kubeconfig cannot be loaded: %v\n", err)
			os.Exit(1)
		}
	}

	clientSet, err = kubernetes.NewForConfig(config)
	if err != nil {
		logrus.Fatalf("clinetset init failed: %v", err)
		os.Exit(1)
	}

	informerFactory = informers.NewSharedInformerFactory(clientSet, time.Second*30)

}

func UpdatePodAnnotation(ctx context.Context, pod *v1.Pod) error {

	_, err := clientSet.CoreV1().Pods(pod.Namespace).UpdateStatus(ctx, pod, metav1.UpdateOptions{})
	if err != nil {
		logrus.Errorf("update pod annotation failed: %v", err)
	}

	return err
}

func GetCNFPodDedicatedCNIIP() (net.IP, error) {

	listOptions := metav1.ListOptions{
		LabelSelector: CNF_SELECTER,
	}
	pods, err := clientSet.CoreV1().Pods(cnfNamespace).List(context.TODO(), listOptions)
	if err != nil {
		return nil, err
	}
	if len(pods.Items) > 0 {
		pod := pods.Items[0]
		ip, err := GetDedicatedCNIIP(&pod)
		if len(ip) > 0 && err == nil {
			return ip, nil
		}
	}
	return nil, err
}

func GetDedicatedCNIIP(pod *v1.Pod) (ip net.IP, err error) {
	IP := cnitypes.Result{}
	if val, ok := pod.Annotations[ParallelIpKey]; ok {
		err := json.Unmarshal([]byte(val), &IP)
		if err != nil {
			return nil, err
		}
	}
	if len(IP.IPs) > 0 {
		ip = IP.IPs[0].Address.IP
		if len(ip) > 0 {
			return ip, nil
		}
	}
	return nil, errors.New("there is no dedicated ip")
}

func UpdatePodAnnotationIP(pod *api.PodSandbox, res types.Result, err error) error {
	IPDetail, enable := res.(*cnitypes.Result)
	if !enable {
		return errors.New("ip enable failed")
	}

	ipStr, _ := json.Marshal(IPDetail)
	pod.Annotations[ParallelIpKey] = string(ipStr)
	newPod := &v1.Pod{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:        pod.Name,
			Namespace:   pod.Namespace,
			Annotations: pod.Annotations,
			Labels:      pod.Labels,
		},
	}
	err = UpdatePodAnnotation(context.TODO(), newPod)
	if err != nil {
		logrus.Errorf("upate pod annotation failed: %v", err)
	}
	return err
}
