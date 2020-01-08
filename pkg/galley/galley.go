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
	"fmt"
	"net/url"
	"os"
	"time"

	"google.golang.org/grpc"
	mcpapi "istio.io/api/mcp/v1alpha1"

	"istio.io/pkg/log"

	"github.com/maistra/ior/pkg/bootstrap"
	"github.com/maistra/ior/pkg/route"
	networking "istio.io/api/networking/v1alpha3"
	_ "istio.io/istio/galley/pkg/metadata" // Import the resource package to pull in all proto types.
	"istio.io/istio/pkg/mcp/creds"
	"istio.io/istio/pkg/mcp/sink"
	"istio.io/istio/pkg/mcp/testing/monitoring"
)

const (
	requiredCertCheckFreq = 1 * time.Second
)

// Galley is responsible to interact with Galley server
type Galley struct {
	args *bootstrap.Args
}

// New returns a Galley instance or an error
func New(args *bootstrap.Args) (*Galley, error) {
	if err := args.Validate(); err != nil {
		return nil, fmt.Errorf("error validating arguments: %v", err)
	}

	log.Infof("Started IOR with\n%v", args)

	return &Galley{args: args}, nil
}

// Run connects to Galley and runs the main loop
func (g *Galley) Run() error {

	url, err := url.Parse(g.args.McpAddr)

	if err != nil {
		return fmt.Errorf("invalid MCP URL %s %v", g.args.McpAddr, err)
	}

	securityOption := grpc.WithInsecure()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	switch url.Scheme {
	case "mcp":
	case "mcps":
		if g.args.CredentialOptions == nil {
			return fmt.Errorf("no credentials specified with secure MCP scheme")
		}

		requiredFiles := []string{g.args.CredentialOptions.CertificateFile, g.args.CredentialOptions.KeyFile, g.args.CredentialOptions.CACertificateFile}
		log.Infof("Secure MCP configured. Waiting for required certificate files to become available: %v", requiredFiles)
		for len(requiredFiles) > 0 {
			if _, err = os.Stat(requiredFiles[0]); os.IsNotExist(err) {
				log.Infof("%v not found. Checking again in %v", requiredFiles[0], requiredCertCheckFreq)
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(requiredCertCheckFreq):
					// retry
				}
				continue
			}

			log.Infof("%v found", requiredFiles[0])
			requiredFiles = requiredFiles[1:]
		}

		watcher, er := creds.WatchFiles(ctx.Done(), g.args.CredentialOptions)
		if er != nil {
			return er
		}
		credentials := creds.CreateForClient(url.Hostname(), watcher)
		securityOption = grpc.WithTransportCredentials(credentials)

	default:
		return fmt.Errorf("unknown MCP URL %s", g.args.McpAddr)
	}

	conn, err := grpc.DialContext(ctx, url.Host, securityOption)
	if err != nil {
		return err
	}

	r, err := route.New(g.args)
	if err != nil {
		return fmt.Errorf("Error creating a route object: %v", err)
	}
	r.DumpRoutes()
	u := &update{Route: r}

	supportedTypes := []string{"istio/networking/v1alpha3/gateways"}
	options := &sink.Options{
		Updater:           u,
		CollectionOptions: sink.CollectionOptionsFromSlice(supportedTypes),
		Reporter:          monitoring.NewInMemoryStatsContext(),
	}

	cl := mcpapi.NewResourceSourceClient(conn)
	c := sink.NewClient(cl, options)
	c.Run(ctx)

	return nil
}

type update struct {
	*route.Route
}

func (u *update) Apply(change *sink.Change) error {
	log.Infof("Got info from MCP - %d object(s)\n", len(change.Objects))
	gatewaysInfo := []route.GatewayInfo{}

	for i, obj := range change.Objects {
		log.Debugf("Object %d: Metadata = %v ", i+1, obj.Metadata)
		gateway, ok := obj.Body.(*networking.Gateway)
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
