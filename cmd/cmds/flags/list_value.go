package flags

import (
	"fmt"

	"github.com/urfave/cli"
)

// command
const (
	valueSeparator = " "
)

type listValue string

func (f *listValue) Set(value string) error {
	*f = listValue(value)
	return nil
}

func (f *listValue) String() string {
	return string(*f)
}

func (f *listValue) Get() ([]string, error) {
	if f == nil || f.IsEmpty() {
		return nil, nil
	}

	// Split the string by spaces, but respect escaped quotes
	var ret []string
	inEscapedDoubleQuotes := false
	inEscapedSingleQuotes := false
	currVal := ""
	for _, c := range f.String() {
		if string(c) == "\"" && !inEscapedSingleQuotes {
			// toggle if not in single quotes
			inEscapedDoubleQuotes = !inEscapedDoubleQuotes
		}
		if string(c) == "'" && !inEscapedDoubleQuotes {
			// toggle if not in single quotes
			inEscapedSingleQuotes = !inEscapedSingleQuotes
		}
		if string(c) == " " && !inEscapedDoubleQuotes && !inEscapedSingleQuotes {
			// found an item to add the the list of args
			ret = append(ret, currVal)
			currVal = ""
			continue // do not add the space to the entry
		}
		currVal += string(c)
	}
	// Check if input is malformed for the last entry
	if inEscapedDoubleQuotes || inEscapedSingleQuotes {
		return nil, fmt.Errorf("malformed listValue contains an unpaired escaped quote")
	}
	// Add the final field
	ret = append(ret, currVal)
	return ret, nil
}

func (f *listValue) IsEmpty() bool {
	if f == nil {
		return true
	}
	return f.String() == ""
}

func NewListValue() cli.Generic {
	return new(listValue)
}

func GetListValue(cliCtx *cli.Context, name string) *listValue {
	return cliCtx.Generic(name).(*listValue)
}
