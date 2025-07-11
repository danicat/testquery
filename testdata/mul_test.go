package testdata

import "testing"

func TestMultiply(t *testing.T) {
	testCases := []struct {
		name   string
		left   int
		right  int
		expected int
	}{
		{"positive numbers", 2, 3, 6},
		{"negative numbers", -2, -3, 6},
		{"positive and negative", 2, -3, -6},
		{"zero input (left)", 0, 5, 0},
		{"zero input (right)", 5, 0, 0},
		{"zero input (both)", 0, 0, 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := multiply(tc.left, tc.right)
			if res != tc.expected {
				t.Errorf("expected %d, but got %d", tc.expected, res)
			}
		})
	}
}
