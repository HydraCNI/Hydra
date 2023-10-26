package main

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	"github.com/hydra-cni/hydra/pkg/cni"
	endpointcontroller "github.com/hydra-cni/hydra/pkg/controller/endpoint"
	endpointslicecontroller "github.com/hydra-cni/hydra/pkg/controller/endpointslice"
	"github.com/hydra-cni/hydra/pkg/kubeclient"
)

var (
	kubeconfig string
	cniConf    string
)

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", filepath.Join(os.Getenv("HOME"), ".kube", "config"), "absolute path to the kubeconfig file")
	flag.StringVar(&cniConf, "cni-conf", "hydra", "cni_config name")
}

func main() {
	flag.Parse()

	cni.DefaultCNIPlugin = cni.CNIPlugin{Name: cniConf}
	kubeclient.CreateDedicatedIPAnnotationKey()

	config, err := rest.InClusterConfig()
	if err != nil {
		// fallback to kube config
		if val := os.Getenv("KUBECONFIG"); len(val) != 0 {
			kubeconfig = val
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			logrus.Fatalf("The kubeconfig cannot be loaded: %v\n", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatal(err)
	}

	factory := informers.NewSharedInformerFactory(clientset, time.Minute)

	stop := make(chan struct{})
	defer close(stop)
	factory.Start(stop)

	ctx := context.Background()
	ep := endpointcontroller.NewEndpointController(
		factory.Core().V1().Pods(),
		factory.Core().V1().Services(),
		factory.Core().V1().Endpoints(),
		clientset,
		1*time.Second,
	)
	eps := endpointslicecontroller.NewController(ctx,
		factory.Core().V1().Pods(),
		factory.Core().V1().Services(),
		factory.Core().V1().Nodes(),
		factory.Discovery().V1().EndpointSlices(),
		100,
		clientset,
		1*time.Second)

	factory.Start(wait.NeverStop)

	go ep.Run(ctx, 1)
	go eps.Run(ctx, 1)

	select {}
}
