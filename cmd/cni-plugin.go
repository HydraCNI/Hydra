/*
   Copyright

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package main

import (
	"context"
	"flag"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/containerd/nri/pkg/api"
	"github.com/containerd/nri/pkg/stub"
	"github.com/hydra-cni/hydra/pkg/cni"
	"github.com/hydra-cni/hydra/pkg/plugin"
)

func main() {

	var (
		pluginName string
		pluginIdx  string
		events     string
		cniConf    string
		opts       []stub.Option
		err        error
	)

	logrus.StandardLogger()
	logrus.SetLevel(logrus.TraceLevel)
	logrus.SetReportCaller(true)
	logrus.SetFormatter(&logrus.TextFormatter{
		PadLevelText:  true,
		ForceColors:   true,
		DisableColors: false,
	})

	flag.StringVar(&pluginName, "name", "cni", "plugin name to register to NRI")
	flag.StringVar(&pluginIdx, "idx", "00", "plugin index to register to NRI")
	flag.StringVar(&events, "events", "runpodsandbox,stoppodsandbox,removepodsandbox", "comma-separated list of events to subscribe for")
	flag.StringVar(&cniConf, "cni-conf", "hydra", "cni_config name")
	flag.Parse()

	if cniConf == "" {
		panic("please specify a cni name through -cni-conf,this defined in the cni config file")
	}
	logrus.Infof(" the cni config name is %s\n", cniConf)

	if pluginName != "" {
		opts = append(opts, stub.WithPluginName(pluginName))
	}
	if pluginIdx != "" {
		opts = append(opts, stub.WithPluginIdx(pluginIdx))
	}

	cni.DefaultCNIPlugin = cni.CNIPlugin{Name: cniConf}

	p := &plugin.CNIPlugin{}

	if p.Mask, err = api.ParseEventMask(events); err != nil {
		logrus.Fatalf("failed to parse events: %v", err)
	}

	if p.Stub, err = stub.New(p, append(opts, stub.WithOnClose(p.OnClose))...); err != nil {
		logrus.Fatalf("failed to create plugin stub: %v", err)
	}

	logrus.Infof(">>>>>>>>>>>>>>>>>>>>>  CNI Plugin Started - Version Tag 0.0.1 <<<<<<<<<<<<<<<<<<<<<<<<<<")
	err = p.Stub.Run(context.Background())
	if err != nil {
		logrus.Errorf("CNIPlugin exited with error %v", err)
		os.Exit(1)
	}
}
