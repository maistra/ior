// Copyright 2019 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package route

import (
	"fmt"
	"strings"

	"github.com/maistra/ior/pkg/util"
	v1 "github.com/openshift/api/route/v1"
	routev1 "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	mcp "istio.io/api/mcp/v1alpha1"
	networking "istio.io/api/networking/v1alpha3"
	"istio.io/istio/pkg/kube"
	"istio.io/istio/pkg/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	maistraPrefix          = "maistra.io/"
	istioNamespace         = "istio-system"
	ingressService         = "istio-ingressgateway"
	generatedByLabel       = maistraPrefix + "generated-by"
	generatedByValue       = "ior"
	originalHostAnnotation = maistraPrefix + "original-host"
	gatewayNameLabel       = maistraPrefix + "gateway-name"
	gatewayNamespaceLabel  = maistraPrefix + "gateway-namespace"
)

// GatewayInfo ...
type GatewayInfo struct {
	Metadata *mcp.Metadata
	Gateway  *networking.Gateway
}

type syncedRoute struct {
	route *v1.Route
	valid bool
}

// Route ...
type Route struct {
	client *routev1.RouteV1Client
	routes map[string]*syncedRoute
}

// New ...
func New() (*Route, error) {
	r := &Route{}

	err := r.initClient()
	if err != nil {
		return nil, err
	}

	err = r.initRoutes()
	if err != nil {
		return nil, err
	}

	return r, nil
}

func getHost(route v1.Route) string {
	if host := route.ObjectMeta.Annotations[originalHostAnnotation]; host != "" {
		return host
	}
	return route.Spec.Host
}

func (r *Route) initRoutes() error {
	routes, err := r.client.Routes(istioNamespace).List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", generatedByLabel, generatedByValue),
	})
	if err != nil {
		return err
	}

	r.routes = make(map[string]*syncedRoute, len(routes.Items))
	for _, route := range routes.Items {
		localRoute := route
		r.routes[getHost(localRoute)] = &syncedRoute{
			route: &localRoute,
		}
	}
	return nil
}

// DumpRoutes ...
func (r *Route) DumpRoutes() {
	out := fmt.Sprintf("%d item(ns)\n", len(r.routes))
	for host, route := range r.routes {
		out += fmt.Sprintf("%s: %s/%s\n", host, route.route.ObjectMeta.Namespace, route.route.ObjectMeta.Name)
	}

	log.Debugf("Current state: %s\n", out)
}

// Sync ...
func (r *Route) Sync(gatewaysInfo []GatewayInfo) error {
	for _, sRoute := range r.routes {
		sRoute.valid = false
	}

	for _, gatewayInfo := range gatewaysInfo {
		for _, server := range gatewayInfo.Gateway.Servers {
			for _, host := range server.GetHosts() {
				sRoute, ok := r.routes[host]
				_ = sRoute // FIXME
				if ok {
					r.editRoute(gatewayInfo.Metadata, host)
				} else {
					r.createRoute(gatewayInfo.Metadata, host, server.Tls != nil)
				}
			}
		}
	}

	for _, sRoute := range r.routes {
		if !sRoute.valid {
			r.deleteRoute(sRoute.route)
		}
	}

	return nil
}

func (r *Route) editRoute(metadata *mcp.Metadata, host string) {
	log.Debugf("Editing route for hostname %s\n", host)
	r.routes[host].valid = true
}

func (r *Route) deleteRoute(route *v1.Route) {
	var immediate int64
	host := getHost(*route)
	log.Debugf("Deleting route %s (hostname: %s)\n", route.ObjectMeta.Name, host)
	err := r.client.Routes(istioNamespace).Delete(route.ObjectMeta.Name, &metav1.DeleteOptions{GracePeriodSeconds: &immediate})
	delete(r.routes, getHost(*route))
	if err == nil {
		log.Infof("Deleted route %s/%s (hostname: %s)\n", route.ObjectMeta.Namespace, route.ObjectMeta.Name, host)
	} else {
		log.Errorf("Error deleting route %s: %s\n", route.ObjectMeta.Name, err)
	}
}

func (r *Route) createRoute(metadata *mcp.Metadata, originalHost string, tls bool) {
	var wildcard = v1.WildcardPolicyNone
	actualHost := originalHost

	log.Debugf("Creating route for hostname %s\n", originalHost)

	if originalHost == "*" {
		log.Infof("Gateway %s: Wildcard * is not supported at the moment. Letting OpenShift create one instead.\n", metadata.Name)
		actualHost = ""
	} else if strings.HasPrefix(originalHost, "*.") {
		// Wildcards are not enabled by default in OCP 3.x.
		// See https://docs.openshift.com/container-platform/3.11/install_config/router/default_haproxy_router.html#using-wildcard-routes
		wildcard = v1.WildcardPolicySubdomain
		actualHost = "wildcard." + strings.TrimPrefix(originalHost, "*.")
	}

	var tlsConfig *v1.TLSConfig
	targetPort := "http2"
	if tls {
		tlsConfig = &v1.TLSConfig{Termination: v1.TLSTerminationPassthrough}
		targetPort = "https"
	}

	namespace, gatewayName := util.ExtractNameNamespace(metadata.Name)

	// FIXME: Can we create the route in the same namespace as the Gateway pointing to a service in the istio-system namespace?
	nr, err := r.client.Routes(istioNamespace).Create(&v1.Route{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", gatewayName),
			Labels: map[string]string{
				generatedByLabel:      generatedByValue,
				gatewayNamespaceLabel: namespace,
				gatewayNameLabel:      gatewayName,
			},
			Annotations: map[string]string{
				originalHostAnnotation: originalHost,
			},
		},
		Spec: v1.RouteSpec{
			Host: actualHost,
			Port: &v1.RoutePort{
				TargetPort: intstr.IntOrString{
					Type:   intstr.String,
					StrVal: targetPort,
				},
			},
			To: v1.RouteTargetReference{
				Name: ingressService,
			},
			TLS:            tlsConfig,
			WildcardPolicy: wildcard,
		},
	})

	if err != nil {
		log.Errorf("Error creating a route for host %s: %s\n", originalHost, err)
	}

	if actualHost == "" {
		log.Infof("Generated hostname by OpenShift: %s\n", nr.Spec.Host)
	}

	log.Infof("Created route %s/%s for hostname %s\n", nr.ObjectMeta.Namespace, nr.ObjectMeta.Name, originalHost)

	r.routes[originalHost] = &syncedRoute{
		route: nr,
		valid: true,
	}
}

func (r *Route) initClient() error {
	config, err := kube.BuildClientConfig("", "")
	if err != nil {
		return err
	}

	client, err := routev1.NewForConfig(config)
	if err != nil {
		return err
	}

	r.client = client

	return nil
}
