package outputs

import (
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/rancher/wins/pkg/converters"
)

func JSON(w io.Writer, obj interface{}) error {
	if obj == nil {
		return nil
	}

	bytes, ok := obj.([]byte)
	if ok {
		if len(bytes) == 0 {
			return nil
		}
		return fprint(w, converters.UnsafeBytesToString(bytes))
	}

	json, err := converters.ToJSON(obj)
	if err != nil {
		return err
	}
	return fprint(w, json)
}

func fprint(w io.Writer, obj interface{}) (err error) {
	_, err = fmt.Fprint(w, obj)
	if err != nil {
		err = errors.Wrap(err, "failed to output result")
	}
	return
}
