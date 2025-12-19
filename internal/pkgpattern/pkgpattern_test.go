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
	"testing"
)

func TestListPackages(t *testing.T) {
	testCases := []struct {
		specifier string
		min       int
	}{
		{"./...", 1},
		{"", 1},
		{"github.com/danicat/testquery/internal/query", 1},
	}

	for _, tc := range testCases {
		packages, err := ListPackages(tc.specifier)
		if err != nil {
			t.Fatalf("ListPackages(%q) failed: %v", tc.specifier, err)
		}

		if len(packages) < tc.min {
			t.Errorf("ListPackages(%q) returned %d packages, want at least %d", tc.specifier, len(packages), tc.min)
		}
	}
}
