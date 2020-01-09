// Copyright 2020 Red Hat, Inc.
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

	networking "istio.io/api/networking/v1alpha3"
	"istio.io/istio/pkg/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

// IngressGateway is a utility to find Ingress Gateways services and pods
type IngressGateway struct {
	cs *kubernetes.Clientset
}

// NewIG creates a new IngressGateway
func NewIG() (*IngressGateway, error) {
	cs, err := kube.CreateClientset("", "")
	if err != nil {
		return nil, fmt.Errorf("error creating kubernetes client: %v", err)
	}

	return &IngressGateway{cs: cs}, nil
}

// FindService tries to find a service that matches with the given gateway selector, in the given namespaces
// Returns the namespace and service name that is a match, or an error
func (ig *IngressGateway) FindService(namespaces []string, gateway *networking.Gateway) (string, string, error) {
	gwSelector := labels.SelectorFromSet(gateway.Selector)

	for _, ns := range namespaces {
		// Get the list of pods that match the gateway selector
		podList, err := ig.cs.CoreV1().Pods(ns).List(metav1.ListOptions{LabelSelector: gwSelector.String()})
		if err != nil { // FIXME: check for NotFound
			return "", "", fmt.Errorf("could not get the list of pods: %v", err)
		}

		// Get the list of services in this namespace
		svcList, err := ig.cs.CoreV1().Services(ns).List(metav1.ListOptions{})
		if err != nil { // FIXME: check for NotFound
			return "", "", fmt.Errorf("could not get the list of services: %v", err)
		}

		// Look for a service whose selector matches the pod labels
		for _, pod := range podList.Items {
			podLabels := labels.Set(pod.ObjectMeta.Labels)

			for _, svc := range svcList.Items {
				svcSelector := labels.SelectorFromSet(svc.Spec.Selector)
				if svcSelector.Matches(podLabels) {
					return ns, svc.Name, nil
				}
			}
		}
	}

	return "", "", fmt.Errorf("could not find a service that matches the selector %s in namespaces %v", gwSelector.String(), namespaces)
}
