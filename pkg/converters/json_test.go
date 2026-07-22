package converters

import (
	"testing"
)

func TestToJson(t *testing.T) {
	json := struct {
		A string
		B string `json:"bAlias"`
		C string `json:"-"`
	}{
		A: "a",
		B: "b",
		C: "c",
	}

	want := `{"A":"a","bAlias":"b"}`

	got, err := ToJSON(json)
	if err != nil {
		t.Errorf("error occurred, %v", err)
	}

	if got != want {
		t.Errorf("error, should be %s, but got %s", want, got)
	}
}
