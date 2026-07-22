package converters

import (
	"encoding/json"

	"github.com/pkg/errors"
)

func ToJSON(obj interface{}) (string, error) {
	ret, err := json.Marshal(obj)
	if err != nil {
		return "", errors.Wrapf(err, "could not convert %T obj to JSON", obj)
	}

	return UnsafeBytesToString(ret), nil
}
