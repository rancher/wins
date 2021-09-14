package converters

import (
	"strings"
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
		t.Errorf("error occurred, " + err.Error())
	}

	if got != want {
		t.Errorf("error, should be " + want + ", but got " + got)
	}
}

func TestToYaml(t *testing.T) {
	yaml := struct {
		A string
		B string `json:"-"`
		C string `json:"-"`
	}{
		A: "a",
		B: "b",
		C: "c",
	}

	want := "A: a"

	got, err := ToYaml(yaml)
	if err != nil {
		t.Errorf("error occurred, " + err.Error())
	}

	if strings.TrimSpace(got) != want {
		t.Errorf("error, should be " + want + ", but got " + got)
	}
}
