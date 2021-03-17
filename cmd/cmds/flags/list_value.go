package flags

import (
	"strings"

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

func (f *listValue) Get() []string {
	if f == nil || f.IsEmpty() {
		return nil
	}
	ret := strings.Split(f.String(), valueSeparator)
	return ret
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
