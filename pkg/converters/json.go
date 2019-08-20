package converters

import (
	"encoding/json"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
)

func ToJson(obj interface{}) (string, error) {
	ret, err := json.Marshal(obj)
	if err != nil {
		return "", errors.Wrapf(err, "could not convert %T obj to JSON", obj)
	}

	return UnsafeBytesToString(ret), nil
}

func ToYaml(obj interface{}) (string, error) {
	ret, err := yaml.Marshal(obj)
	if err != nil {
		return "", errors.Wrapf(err, "could not convert %T obj to YAML", obj)
	}

	return UnsafeBytesToString(ret), nil
}