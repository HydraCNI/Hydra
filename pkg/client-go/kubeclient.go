package kubeclient

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"path/filepath"
	"time"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var clientSet *kubernetes.Clientset
var informerFactory informers.SharedInformerFactory

func KubeInitializer() {
	config, err := rest.InClusterConfig()
	if err != nil {
		// fallback to kubeconfig
		kubeconfig := filepath.Join("./conf", "kubeconf.yaml")
		if val := os.Getenv("KUBECONFIG"); len(val) != 0 {
			kubeconfig = val
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			fmt.Printf("The kubeconfig cannot be loaded: %v\n", err)
			os.Exit(1)
		}
	}

	clientSet, err = kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("clinetset init failed")
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
