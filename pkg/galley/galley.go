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

package galley

import (
	"context"

	"google.golang.org/grpc"
	mcpapi "istio.io/api/mcp/v1alpha1"

	"istio.io/istio/pkg/log"

	"github.com/maistra/ior/pkg/route"
	networking "istio.io/api/networking/v1alpha3"
	mcpclient "istio.io/istio/pkg/mcp/client"
)

// ConnectToGalley ...
func ConnectToGalley(galleyAddr string) {
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, galleyAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Unable to dial MCP Server %q: %v", galleyAddr, err)
		return
	}

	r, err := route.New()
	if err != nil {
		log.Fatalf("Error creating a route object: %v", err)
		return
	}
	r.DumpRoutes()
	u := &update{Route: r}

	client := mcpapi.NewAggregatedMeshConfigServiceClient(conn)

	supportedTypes := []string{"type.googleapis.com/istio.networking.v1alpha3.Gateway"}

	mcpClient := mcpclient.New(client, supportedTypes, u, "ior", map[string]string{}, mcpclient.NewStatsContext("ior"))

	mcpClient.Run(ctx)
}

type update struct {
	*route.Route
}

func (u *update) Apply(change *mcpclient.Change) error {
	log.Infof("Got info from MCP - %d object(s)\n", len(change.Objects))
	gatewaysInfo := []route.GatewayInfo{}

	for i, obj := range change.Objects {
		log.Debugf("Object %d: Metadata = %v ", i+1, obj.Metadata)
		gateway, ok := obj.Resource.(*networking.Gateway)
		if ok {
			log.Debugf("Object %d: Gateway = %v\n", i+1, gateway)
			gatewaysInfo = append(gatewaysInfo, route.GatewayInfo{
				Metadata: obj.Metadata,
				Gateway:  gateway,
			})
		} else {
			log.Errorf("Error decoding gateway for object %d", i+1)
		}
	}

	ret := u.Sync(gatewaysInfo)
	u.DumpRoutes()
	return ret
}
