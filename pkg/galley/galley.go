package galley

import (
	"context"
	"errors"
	"fmt"
	"log"

	"google.golang.org/grpc"
	mcpapi "istio.io/api/mcp/v1alpha1"

	"github.com/maistra/ior/pkg/route"
	"github.com/maistra/ior/pkg/util"
	networking "istio.io/api/networking/v1alpha3"
	mcpclient "istio.io/istio/pkg/mcp/client"
)

// ConnectToGalley ...
func ConnectToGalley() {
	ctx := context.Background()
	addr := "istio-galley.istio-system:9901"
	conn, err := grpc.DialContext(ctx, addr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Unable to dial MCP Server %q: %v", addr, err)
	}

	r, err := route.New()
	if err != nil {
		log.Fatalf("Error creating a route object: %v", err)
	}
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
	gatewaysInfo := []route.GatewayInfo{}

	for i, obj := range change.Objects {
		namespace, name := util.ExtractNameNamespace(obj.Metadata.Name)
		fmt.Printf("Object %d: Namespace = %s, Name = %s\n", i+1, namespace, name)
		fmt.Printf("Metadata = %v\n", obj.Metadata)
		gateway, ok := obj.Resource.(*networking.Gateway)
		if !ok {
			return errors.New("Error decoding gateway")
		}
		fmt.Printf("Gateway = %v\n", gateway)
		gatewaysInfo = append(gatewaysInfo, route.GatewayInfo{
			Metadata: obj.Metadata,
			Gateway:  gateway,
		})
	}

	return u.Sync(gatewaysInfo)
}
