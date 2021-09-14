package flags

import (
	"fmt"
	"testing"
)

func TestNormal(t *testing.T) {
	val := ListValue("RANCHER=hello WINS=world")
	valList, err := val.Get()
	if err != nil {
		t.Fatal(err)
	}
	// Check outputted values
	expectedList := []string{"RANCHER=hello", "WINS=world"}
	err = fmt.Errorf("Failed to parse list value: expected %v, got %v", expectedList, valList)
	for i, expected := range expectedList {
		if valList[i] != expected {
			t.Fatal(err)
		}
	}
}

func TestEscapedDoubleQuotes(t *testing.T) {
	val := ListValue("NICK=bye LUTHER=\"hello\" RANCHER=\"hello world\" WINS=world")
	valList, err := val.Get()
	if err != nil {
		t.Fatal(err)
	}
	// Check outputted values
	expectedList := []string{"NICK=bye", "LUTHER=\"hello\"", "RANCHER=\"hello world\"", "WINS=world"}
	err = fmt.Errorf("Failed to parse list value: expected %v, got %v", expectedList, valList)
	for i, expected := range expectedList {
		if valList[i] != expected {
			t.Fatal(err)
		}
	}
}

func TestEscapedSingleQuotes(t *testing.T) {
	val := ListValue("NICK=bye LUTHER='hello' RANCHER='hello world' WINS=world")
	valList, err := val.Get()
	if err != nil {
		t.Fatal(err)
	}
	// Check outputted values
	expectedList := []string{"NICK=bye", "LUTHER='hello'", "RANCHER='hello world'", "WINS=world"}
	err = fmt.Errorf("Failed to parse list value: expected %v, got %v", expectedList, valList)
	for i, expected := range expectedList {
		if valList[i] != expected {
			t.Fatal(err)
		}
	}
}

func TestEscapedSingleAndDoubleQuotes(t *testing.T) {
	val := ListValue("NICK=bye LUTHER=\"hello\" RANCHER='hello world' WINS=world")
	valList, err := val.Get()
	if err != nil {
		t.Fatal(err)
	}
	// Check outputted values
	expectedList := []string{"NICK=bye", "LUTHER=\"hello\"", "RANCHER='hello world'", "WINS=world"}
	err = fmt.Errorf("Failed to parse list value: expected %v, got %v", expectedList, valList)
	for i, expected := range expectedList {
		if valList[i] != expected {
			t.Fatal(err)
		}
	}
}

func TestEscapedQuotesInQuotes(t *testing.T) {
	val := ListValue("NICK=bye LUTHER=\"'hello'\" RANCHER='\"hello world\"' WINS=world")
	valList, err := val.Get()
	if err != nil {
		t.Fatal(err)
	}
	// Check outputted values
	expectedList := []string{"NICK=bye", "LUTHER=\"'hello'\"", "RANCHER='\"hello world\"'", "WINS=world"}
	err = fmt.Errorf("Failed to parse list value: expected %v, got %v", expectedList, valList)
	for i, expected := range expectedList {
		if valList[i] != expected {
			t.Fatal(err)
		}
	}
}
