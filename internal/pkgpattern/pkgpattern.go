// Copyright 2025 Google LLC
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

package pkgpattern

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

// ListPackages returns a list of packages matching the given specifier.
func ListPackages(specifier string) ([]string, error) {
	args := []string{"list", "-json"}
	if specifier != "" {
		args = append(args, specifier)
	}

	cmd := exec.Command("go", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to list packages: %w, stderr: %s", err, stderr.String())
	}

	var packages []string
	decoder := json.NewDecoder(&stdout)
	for decoder.More() {
		var pkg struct {
			Dir string
		}
		if err := decoder.Decode(&pkg); err != nil {
			return nil, fmt.Errorf("failed to decode package json: %w", err)
		}
		packages = append(packages, pkg.Dir)
	}

	return packages, nil
}
