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

package bootstrap

import (
	"bytes"
	"fmt"
	"os"
)

// Args provides configuration parameters for IOR
type Args struct {
	McpAddr   string
	Namespace string
}

// DefaultArgs returns a new set of arguments, initialized to the defaults
func DefaultArgs() *Args {
	return &Args{
		McpAddr: "istio-galley:9901",
	}
}

// Validate checks if the arguments are ok
func (args *Args) Validate() error {
	if len(args.McpAddr) == 0 {
		return fmt.Errorf("MCP address cannot be empty")
	}

	if len(args.Namespace) == 0 {
		args.Namespace, _ = os.LookupEnv("POD_NAMESPACE")
		if len(args.Namespace) == 0 {
			return fmt.Errorf("control plane namespace cannot be empty")
		}
	}

	return nil
}

// String produces a stringified version of the arguments for debugging.
func (args *Args) String() string {
	buf := &bytes.Buffer{}

	fmt.Fprintln(buf, "McpAddr: ", args.McpAddr)
	fmt.Fprintln(buf, "Namespace: ", args.Namespace)

	return buf.String()
}
