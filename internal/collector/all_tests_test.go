// Copyright (c) 2026 Google LLC
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package collector

import (
	"reflect"
	"testing"
	"time"
)

func TestParseTestOutput(t *testing.T) {
	inputJSON := `
{"Time":"2023-10-27T10:00:00Z","Action":"run","Package":"example.com/my/pkg","Test":"TestMyFunction"}
{"Time":"2023-10-27T10:00:01Z","Action":"pass","Package":"example.com/my/pkg","Test":"TestMyFunction","Elapsed":1.23}
{"Time":"2023-10-27T10:00:02Z","Action":"run","Package":"example.com/my/pkg","Test":"TestAnotherFunction"}
{"Time":"2023-10-27T10:00:03Z","Action":"fail","Package":"example.com/my/pkg","Test":"TestAnotherFunction","Elapsed":2.34}
`
	expectedTime1, _ := time.Parse(time.RFC3339, "2023-10-27T10:00:01Z")
	expectedTime2, _ := time.Parse(time.RFC3339, "2023-10-27T10:00:03Z")
	elapsed1 := 1.23
	elapsed2 := 2.34

	expected := []TestEvent{
		{Time: expectedTime1, Action: "pass", Package: "example.com/my/pkg", Test: "TestMyFunction", Elapsed: &elapsed1},
		{Time: expectedTime2, Action: "fail", Package: "example.com/my/pkg", Test: "TestAnotherFunction", Elapsed: &elapsed2},
	}

	allEvents, err := parseTestOutput([]byte(inputJSON))
	if err != nil {
		t.Fatalf("parseTestOutput failed: %v", err)
	}

	got := filterAndEnrichTestEvents(allEvents, nil)

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("filterAndEnrichTestEvents() got = %+v, want %+v", got, expected)
	}
}

func TestSubtestHierarchyEnrichment(t *testing.T) {
	inputJSON := `
{"Time":"2023-10-27T10:00:00Z","Action":"run","Package":"example.com/my/pkg","Test":"TestTable"}
{"Time":"2023-10-27T10:00:00Z","Action":"run","Package":"example.com/my/pkg","Test":"TestTable/case_a"}
{"Time":"2023-10-27T10:00:01Z","Action":"pass","Package":"example.com/my/pkg","Test":"TestTable/case_a","Elapsed":0.10}
{"Time":"2023-10-27T10:00:01Z","Action":"run","Package":"example.com/my/pkg","Test":"TestTable/case_b"}
{"Time":"2023-10-27T10:00:01Z","Action":"output","Package":"example.com/my/pkg","Test":"TestTable/case_b","Output":"    table_test.go:20: boom\n"}
{"Time":"2023-10-27T10:00:02Z","Action":"fail","Package":"example.com/my/pkg","Test":"TestTable/case_b","Elapsed":0.20}
{"Time":"2023-10-27T10:00:02Z","Action":"fail","Package":"example.com/my/pkg","Test":"TestTable","Elapsed":0.30}
{"Time":"2023-10-27T10:00:03Z","Action":"pass","Package":"example.com/my/pkg","Test":"TestStandalone","Elapsed":0.05}
`
	allEvents, err := parseTestOutput([]byte(inputJSON))
	if err != nil {
		t.Fatalf("parseTestOutput failed: %v", err)
	}

	got := filterAndEnrichTestEvents(allEvents, nil)
	if len(got) != 4 {
		t.Fatalf("expected 4 test events, got %d", len(got))
	}

	byName := make(map[string]TestEvent)
	for _, ev := range got {
		byName[ev.Test] = ev
	}

	parent := byName["TestTable"]
	if parent.IsParent != 1 || parent.IsSubtest != 0 || parent.ParentTest != nil {
		t.Errorf("unexpected TestTable hierarchy: %+v", parent)
	}

	subA := byName["TestTable/case_a"]
	if subA.IsParent != 0 || subA.IsSubtest != 1 || subA.ParentTest == nil || *subA.ParentTest != "TestTable" {
		t.Errorf("unexpected TestTable/case_a hierarchy: %+v", subA)
	}

	subB := byName["TestTable/case_b"]
	if subB.IsParent != 0 || subB.IsSubtest != 1 || subB.ParentTest == nil || *subB.ParentTest != "TestTable" {
		t.Errorf("unexpected TestTable/case_b hierarchy: %+v", subB)
	}
	if subB.Output == nil || *subB.Output != "    table_test.go:20: boom\n" {
		t.Errorf("expected captured output on TestTable/case_b, got %v", subB.Output)
	}

	standalone := byName["TestStandalone"]
	if standalone.IsParent != 0 || standalone.IsSubtest != 0 || standalone.ParentTest != nil {
		t.Errorf("unexpected TestStandalone hierarchy: %+v", standalone)
	}
}
