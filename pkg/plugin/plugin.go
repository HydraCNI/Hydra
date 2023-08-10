package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	types "github.com/containernetworking/cni/pkg/types/040"
	kubeclient "github.com/hydra-cni/hydra/pkg/client-go"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/containerd/nri/pkg/api"
	"github.com/containerd/nri/pkg/stub"
	"github.com/hydra-cni/hydra/pkg/cni"
)

type config struct {
	LogFile       string   `json:"logFile"`
	Events        []string `json:"events"`
	AddAnnotation string   `json:"addAnnotation"`
	SetAnnotation string   `json:"setAnnotation"`
	AddEnv        string   `json:"addEnv"`
	SetEnv        string   `json:"setEnv"`
}
type CNIPlugin struct {
	Stub stub.Stub
	Mask stub.EventMask
}

var (
	cfg config
	_   = stub.ConfigureInterface(&CNIPlugin{})
)

//var log = logrus.New()

func (p *CNIPlugin) Configure(config, runtime, version string) (stub.EventMask, error) {
	logrus.Infof("got configuration data: %q from runtime %s %s", config, runtime, version)
	if config == "" {
		return p.Mask, nil
	}

	oldCfg := cfg
	err := yaml.Unmarshal([]byte(config), &cfg)
	if err != nil {
		return 0, fmt.Errorf("failed to parse provided configuration: %w", err)
	}

	p.Mask, err = api.ParseEventMask(cfg.Events...)
	if err != nil {
		return 0, fmt.Errorf("failed to parse events in configuration: %w", err)
	}

	if cfg.LogFile != oldCfg.LogFile {
		f, err := os.OpenFile(cfg.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			logrus.Errorf("failed to open log file %q: %v", cfg.LogFile, err)
			return 0, fmt.Errorf("failed to open log file %q: %w", cfg.LogFile, err)
		}
		logrus.SetOutput(f)
	}

	return p.Mask, nil
}

func (p *CNIPlugin) Synchronize(pods []*api.PodSandbox, containers []*api.Container) ([]*api.ContainerUpdate, error) {
	//dump("Synchronize", "pods", pods, "containers", containers)
	return nil, nil
}

func (p *CNIPlugin) Shutdown() {
	//dump("Shutdown")
}

func (p *CNIPlugin) RunPodSandbox(pod *api.PodSandbox) error {
	logrus.Debugf("[RunPodSandbox]: the pod is %s--%s", pod.Namespace, pod.Name)
	nsPath, err := GetNSPathFromPod(pod)
	if err != nil {
		return err
	}
	logrus.Infof("the namespace path is: %s ", nsPath)
	res, err := cni.DefaultCNIPlugin.AddNetworkInterface(nsPath)

	IPdetail, enalbe := res.(*types.Result)
	if !enalbe {
		return err
	}

	ipstr, _ := json.Marshal(IPdetail)
	pod.Annotations["hydra.clusternet.io"] = string(ipstr)
	newpod := &v1.Pod{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:        pod.Name,
			Namespace:   pod.Namespace,
			Annotations: pod.Annotations,
			Labels:      pod.Labels,
		},
	}
	err = kubeclient.UpdatePodAnnotation(context.TODO(), newpod)
	return err
}

func (p *CNIPlugin) StopPodSandbox(pod *api.PodSandbox) error {
	//dump("StopPodSandbox", "pod", pod)
	logrus.Debugf("[StopPodSandbox]: the pod is %s--%s", pod.Namespace, pod.Name)
	nsPath, err := GetNSPathFromPod(pod)
	if err != nil {
		logrus.Errorf(" get ns path failed: %s \n ", err)
		logrus.Infof("pod: %v", pod)
		return err
	}
	logrus.Infof("the namespace path is: %s ", nsPath)
	_, err = cni.DefaultCNIPlugin.DelNetworkInterface(nsPath)
	return err
}

func (p *CNIPlugin) RemovePodSandbox(pod *api.PodSandbox) error {
	logrus.Debugf("[RemovePodSandbox]: the pod is %s--%s", pod.Namespace, pod.Name)
	return nil
}

func (p *CNIPlugin) OnClose() {
	os.Exit(0)
}

func GetNSPathFromPod(pod *api.PodSandbox) (nsPath string, err error) {
	for _, ns := range pod.Linux.Namespaces {
		if ns.Type == "network" {
			nsPath = ns.Path
			break
		}
	}
	if nsPath == "" {
		return "", errors.New("ns not exist")
	}
	return
}
