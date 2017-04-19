/*
Copyright 2017 The Kubernetes Authors.

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

package controller

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"

	"github.com/golang/glog"
	"github.com/jcmoraisjr/haproxy-ingress/pkg/types"
	"github.com/jcmoraisjr/haproxy-ingress/pkg/version"
	"github.com/spf13/pflag"
	api "k8s.io/client-go/pkg/api/v1"
	"k8s.io/ingress/core/pkg/ingress"
	"k8s.io/ingress/core/pkg/ingress/controller"
	"k8s.io/ingress/core/pkg/ingress/defaults"
)

// HAProxyController has internal data of a HAProxyController instance
type HAProxyController struct {
	controller *controller.GenericController
	configMap  *api.ConfigMap
	command    string
	configFile string
	template   *template
	endpoints  map[string]map[types.Endpoint]int
}

// NewHAProxyController constructor
func NewHAProxyController() *HAProxyController {
	return &HAProxyController{
		command:    "/haproxy-wrapper",
		configFile: "/usr/local/etc/haproxy/haproxy.cfg",
		template:   newTemplate("haproxy.tmpl", "/usr/local/etc/haproxy/haproxy.tmpl"),
		endpoints:  make(map[string]map[types.Endpoint]int),
	}
}

// Info provides controller name and repository infos
func (haproxy *HAProxyController) Info() *ingress.BackendInfo {
	return &ingress.BackendInfo{
		Name:       "HAProxy",
		Release:    version.RELEASE,
		Build:      version.COMMIT,
		Repository: version.REPO,
	}
}

// Start starts the controller
func (haproxy *HAProxyController) Start() {
	haproxy.controller = controller.NewIngressController(haproxy)
	haproxy.controller.Start()
}

// Stop shutdown the controller process
func (haproxy *HAProxyController) Stop() error {
	err := haproxy.controller.Stop()
	return err
}

// Name provides the complete name of the controller
func (haproxy *HAProxyController) Name() string {
	return "HAProxy Ingress Controller"
}

// DefaultIngressClass returns the ingress class name
func (haproxy *HAProxyController) DefaultIngressClass() string {
	return "haproxy"
}

// Check health check implementation
func (haproxy *HAProxyController) Check(_ *http.Request) error {
	return nil
}

// SetListers give access to the store listers
func (haproxy *HAProxyController) SetListers(l ingress.StoreLister) {
}

// OverrideFlags allows controller to override command line parameter flags
func (haproxy *HAProxyController) OverrideFlags(*pflag.FlagSet) {
}

// SetConfig receives the ConfigMap the user has configured
func (haproxy *HAProxyController) SetConfig(configMap *api.ConfigMap) {
	haproxy.configMap = configMap
}

// BackendDefaults defines default values to the ingress core
func (haproxy *HAProxyController) BackendDefaults() defaults.Backend {
	return newHAProxyConfig(haproxy.configMap).Backend
}

func endpointsForBackend(cfg ingress.Configuration, name string) []ingress.Endpoint {
	for _, backend := range cfg.Backends {
		if backend.Name == name {
			return backend.Endpoints
		}
	}
	return nil
}

// OnUpdate regenerate the configuration file of the backend
func (haproxy *HAProxyController) OnUpdate(cfg ingress.Configuration) ([]byte, error) {
	for _, backend := range cfg.Backends {
		if _, ok := haproxy.endpoints[backend.Name]; !ok {
			haproxy.endpoints[backend.Name] = make(map[types.Endpoint]int)
		}

		// Set weight to 0 for all backends
		for ep, _ := range haproxy.endpoints[backend.Name] {
			haproxy.endpoints[backend.Name][ep] = 0
		}

		// Set back to 256 for those that are alive
		for _, ep := range backend.Endpoints {
			haproxy.endpoints[backend.Name][types.MapEndpoint(ep)] = 256
		}
	}

	glog.Infof("%+v\n", haproxy.endpoints)

	data, err := haproxy.template.execute(newControllerConfig(&cfg, haproxy.endpoints, haproxy.configMap))
	if err != nil {
		return nil, err
	}
	return data, nil
}

// Reload reload the backend if the configuration file has changed
func (haproxy *HAProxyController) Reload(data []byte) ([]byte, bool, error) {
	if !haproxy.configChanged(data) {
		return nil, false, nil
	}
	// TODO missing HAProxy validation before overwrite and try to reload
	err := ioutil.WriteFile(haproxy.configFile, data, 0644)
	if err != nil {
		return nil, false, err
	}
	out, err := haproxy.reloadHaproxy()
	if len(out) > 0 {
		glog.Infof("HAProxy output:\n%v", string(out))
	}
	return out, true, err
}

func (haproxy *HAProxyController) configChanged(data []byte) bool {
	if _, err := os.Stat(haproxy.configFile); os.IsNotExist(err) {
		return true
	}
	cfg, err := ioutil.ReadFile(haproxy.configFile)
	if err != nil {
		return false
	}
	return !bytes.Equal(cfg, data)
}

func (haproxy *HAProxyController) reloadHaproxy() ([]byte, error) {
	out, err := exec.Command(haproxy.command, haproxy.configFile).CombinedOutput()
	return out, err
}
