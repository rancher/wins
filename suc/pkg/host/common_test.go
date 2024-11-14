package host

import "testing"

func TestParseWinsVersion(t *testing.T) {
	type test struct {
		name            string
		winsOutput      string
		expectedVersion string
		errExpected     bool
	}

	tests := []test{
		{
			name:            "Released version",
			winsOutput:      "rancher-wins version v0.4.20",
			expectedVersion: "v0.4.20",
			errExpected:     false,
		},
		{
			name:            "RC version",
			winsOutput:      "rancher-wins version v0.4.20-rc.1",
			expectedVersion: "v0.4.20-rc.1",
			errExpected:     false,
		},
		{
			name:            "Dirty Commit",
			winsOutput:      "rancher-wins version 06685df-dirty",
			expectedVersion: "",
			errExpected:     true,
		},
		{
			name:            "Unreleased Clean Commit",
			winsOutput:      "rancher-wins version 06685df",
			expectedVersion: "06685df",
			errExpected:     false,
		},
		{
			name:            "Empty output",
			winsOutput:      "",
			expectedVersion: "",
			errExpected:     true,
		},
		{
			name:            "unepxected format output",
			winsOutput:      "rancher-wins version",
			expectedVersion: "",
			errExpected:     true,
		},
	}

	for _, tst := range tests {
		t.Run(tst.name, func(t *testing.T) {
			version, err := parseWinsVersion(tst.winsOutput)
			if err != nil && !tst.errExpected {
				t.Fatalf("encountered unexpected errror, wins output: '%s', returned version: '%s': %v", tst.winsOutput, version, err)
			}
			if version != tst.expectedVersion {
				t.Fatalf("encountered unexpected version, wins output: '%s', returned version: '%s', expected version: '%s'", tst.winsOutput, version, tst.expectedVersion)
			}
		})
	}
}
