package route

import (
	"fmt"

	"github.com/maistra/ior/pkg/util"
	v1 "github.com/openshift/api/route/v1"
	routev1 "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	mcp "istio.io/api/mcp/v1alpha1"
	networking "istio.io/api/networking/v1alpha3"
	"istio.io/istio/pkg/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	istioNamespace   = "istio-system"
	ingressService   = "istio-ingressgateway"
	generatedByLabel = "generated-by"
	generatedByValue = "ior"
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

func (r *Route) initRoutes() error {
	routes, err := r.client.Routes(istioNamespace).List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", generatedByLabel, generatedByValue),
	})
	if err != nil {
		return err
	}

	r.routes = make(map[string]*syncedRoute, len(routes.Items))
	for _, route := range routes.Items {
		r.routes[route.Spec.Host] = &syncedRoute{
			route: &route,
			valid: true,
		}
	}
	return nil
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
	r.routes[host].valid = true
}

func (r *Route) deleteRoute(route *v1.Route) {
	var immediate int64
	err := r.client.Routes(istioNamespace).Delete(route.ObjectMeta.Name, &metav1.DeleteOptions{GracePeriodSeconds: &immediate})
	delete(r.routes, route.Spec.Host)
	if err != nil {
		fmt.Printf("Error deleting the route %s: %s\n", route.ObjectMeta.Name, err)
	}
}

func (r *Route) createRoute(metadata *mcp.Metadata, host string, tls bool) {
	namespace, gatewayName := util.ExtractNameNamespace(metadata.Name)
	if host == "*" {
		fmt.Printf("Gateway %s: Wildcard * is not supported at the moment. Letting OpenShift create one instead.\n", metadata.Name)
		host = ""
	}

	var tlsConfig *v1.TLSConfig
	if tls {
		tlsConfig = &v1.TLSConfig{Termination: v1.TLSTerminationPassthrough}
	}

	// FIXME: Can we create the route in the same namespace as the Gateway pointing to a service in the istio-system namespace?
	nr, err := r.client.Routes(istioNamespace).Create(&v1.Route{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", gatewayName),
			Labels: map[string]string{
				generatedByLabel:    generatedByValue,
				"gateway-namespace": namespace,
				"gateway-name":      gatewayName,
			},
		},
		Spec: v1.RouteSpec{
			Host: host,
			To: v1.RouteTargetReference{
				Name: ingressService,
			},
			TLS: tlsConfig,
		},
	})

	if err != nil {
		fmt.Printf("Error creating a route for host %s: %s\n", host, err)
	}

	if host == "" {
		fmt.Printf("Generated hostname by OpenShift: %s\n", nr.Spec.Host)
	}

	r.routes[host] = &syncedRoute{
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
