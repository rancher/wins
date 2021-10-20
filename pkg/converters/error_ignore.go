package converters

import (
	"strconv"

	"github.com/buger/jsonparser"
	"golang.org/x/sys/windows/registry"
)

func GetStringFormJSON(jsonData []byte, key ...string) string {
	val, _ := jsonparser.GetUnsafeString(jsonData, key...)
	return val
}

func GetIntFromRegistryKey(k registry.Key, name string) int {
	val, _, _ := k.GetIntegerValue(name)
	return int(val)
}

func GetIntStringFormRegistryKey(k registry.Key, name string) string {
	return strconv.Itoa(GetIntFromRegistryKey(k, name))
}

func GetStringFromRegistryKey(k registry.Key, name string) string {
	val, _, _ := k.GetStringValue(name)
	return val
}
