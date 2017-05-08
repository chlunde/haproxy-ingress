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

package types

import (
	api "k8s.io/client-go/pkg/api/v1"
	"k8s.io/ingress/core/pkg/ingress"
	"k8s.io/ingress/core/pkg/ingress/annotations/authtls"
	"k8s.io/ingress/core/pkg/ingress/annotations/rewrite"
	"k8s.io/ingress/core/pkg/ingress/defaults"
)

type (
	// ControllerConfig has ingress generated and some transformations
	// compatible with HAProxy
	ControllerConfig struct {
		Userlists           map[string]Userlist
		Backends            []*Backend
		DefaultServer       *HAProxyServer
		HTTPServers         []*HAProxyServer
		HTTPSServers        []*HAProxyServer
		TCPEndpoints        []ingress.L4Service
		UDPEndpoints        []ingress.L4Service
		PassthroughBackends []*ingress.SSLPassthroughBackend
		Cfg                 *HAProxyConfig
	}

	// Backend describes one or more remote server/s (endpoints) associated with a service
	Backend struct {
		// Name represents an unique api.Service name formatted as <namespace>-<name>-<port>
		Name string
		// This indicates if the communication protocol between the backend and the endpoint is HTTP or HTTPS
		// Allowing the use of HTTPS
		// The endpoint/s must provide a TLS connection.
		// The certificate used in the endpoint cannot be a self signed certificate
		// TODO: add annotation to allow the load of ca certificate
		Secure bool
		// Endpoints contains the list of endpoints currently running
		Endpoints EndpointMap

		// StickySession contains the StickyConfig object with stickness configuration

		SessionAffinity ingress.SessionAffinityConfig
	}

	Endpoint struct {
		Address string
		Port    string
	}

	EndpointState struct {
		ObjectRef      *api.ObjectReference
		Weight         int
		PreviousWeight int
	}

	EndpointMap map[Endpoint]EndpointState

	// HAProxyConfig has HAProxy specific configurations from ConfigMap
	HAProxyConfig struct {
		defaults.Backend `json:",squash"`
		Syslog           string `json:"syslog-endpoint"`
	}
	// Userlist list of users for basic authentication
	Userlist struct {
		ListName string
		Realm    string
		Users    []AuthUser
	}
	// AuthUser authorization info for basic authentication
	AuthUser struct {
		Username  string
		Password  string
		Encrypted bool
	}
	// HAProxyServer and HAProxyLocation build some missing pieces
	// from ingress.Server used by HAProxy
	HAProxyServer struct {
		IsDefaultServer bool               `json:"isDefaultServer"`
		Hostname        string             `json:"hostname"`
		SSLCertificate  string             `json:"sslCertificate"`
		SSLPemChecksum  string             `json:"sslPemChecksum"`
		RootLocation    *HAProxyLocation   `json:"defaultLocation"`
		Locations       []*HAProxyLocation `json:"locations,omitempty"`
		SSLRedirect     bool               `json:"sslRedirect"`
	}
	// HAProxyLocation has location data as a HAProxy friendly syntax
	HAProxyLocation struct {
		IsRootLocation  bool                  `json:"isDefaultLocation"`
		Path            string                `json:"path"`
		Backend         string                `json:"backend"`
		Redirect        rewrite.Redirect      `json:"redirect,omitempty"`
		Userlist        Userlist              `json:"userlist,omitempty"`
		CertificateAuth authtls.AuthSSLConfig `json:"certificateAuth,omitempty"`
		HAMatchPath     string                `json:"haMatchPath"`
		HAWhitelist     string                `json:"whitelist,omitempty"`
	}
)

func NewEndpoint(ep ingress.Endpoint) Endpoint {
	return Endpoint{ep.Address, ep.Port}
}
