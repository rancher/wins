package outputs

import (
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/rancher/wins/pkg/converters"
)

func Json(w io.Writer, obj interface{}) error {
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

	objJson, err := converters.ToJson(obj)
	if err != nil {
		return err
	}
	return fprint(w, objJson)
}

func fprint(w io.Writer, obj interface{}) (err error) {
	_, err = fmt.Fprint(w, obj)
	if err != nil {
		err = errors.Wrapf(err, "failed to output result")
	}
	return
}
