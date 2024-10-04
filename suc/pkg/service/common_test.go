package service

import "testing"

func Test_UnorderedSlicesEqual(t *testing.T) {
	type test struct {
		name        string
		sliceA      []string
		sliceB      []string
		shouldMatch bool
	}

	tests := []test{
		{
			name: "Slices match",
			sliceA: []string{
				"One", "Two",
			},
			sliceB: []string{
				"One", "Two",
			},
			shouldMatch: true,
		},
		{
			name: "Slices match out of order",
			sliceA: []string{
				"Two", "One",
			},
			sliceB: []string{
				"One", "Two",
			},
			shouldMatch: true,
		},
		{
			name: "Slices do not match",
			sliceA: []string{
				"Two", "Three",
			},
			sliceB: []string{
				"One", "Two",
			},
			shouldMatch: false,
		},
		{
			name: "Slices do not match due to length difference",
			sliceA: []string{
				"Two",
			},
			sliceB: []string{
				"One", "Two",
			},
			shouldMatch: false,
		},
	}

	for _, tst := range tests {
		t.Run(tst.name, func(t *testing.T) {
			match := UnorderedSlicesEqual(tst.sliceA, tst.sliceB)
			if match != tst.shouldMatch {
				t.Errorf("Expected %t when determining if slices match, got %t. SliceA %v, SliceB %v", tst.shouldMatch, match, tst.sliceA, tst.sliceB)
			}
		})
	}

}
