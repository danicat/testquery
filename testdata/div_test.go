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

package testdata

import (
	"errors"
	"testing"
)

func TestDivide(t *testing.T) {
	expected := 10
	res, err := divide(1, 1)
	if err != nil {
		t.Fatalf("expected no error, but got %s", err)
	}

	if res != expected {
		t.Fatalf("expected result %d, but got %d", expected, res)
	}
}

func TestDivideByZero(t *testing.T) {
	_, err := divide(1, 0)
	if !errors.Is(err, ErrDivideByZero) {
		t.Fatalf("expected %s, but got %s", ErrDivideByZero, err)
	}
}
