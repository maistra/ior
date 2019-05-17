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

package version

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	buildVersion     = "unknown"
	buildGitRevision = "unknown"
	buildStatus      = "unknown"
	buildTag         = "unknown"

	// Info exports the build version information.
	Info BuildInfo
)

// BuildInfo describes version information about the binary build.
type BuildInfo struct {
	Version     string
	GitRevision string
	BuildStatus string
	GitTag      string
}

func (b BuildInfo) String() string {
	return fmt.Sprintf("%#v", b)
}

// GetVersionCmd returns command line flags for `version' subcommand
func GetVersionCmd() *cobra.Command {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Prints version number and exits",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("%s\n", Info)
		},
	}

	return versionCmd
}

func init() {
	Info = BuildInfo{
		Version:     buildVersion,
		GitRevision: buildGitRevision,
		BuildStatus: buildStatus,
		GitTag:      buildTag,
	}
}
