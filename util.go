package liveconfig

import (
	"fmt"
	"reflect"
	"strings"
	"unicode/utf8"
)

// trimFirstRune
func trimFirstRune(s string) string {
	_, i := utf8.DecodeRuneInString(s)
	return s[i:]
}

// read struct field's tags json and etcd
func getStructTags(structField reflect.StructField) (string, string) {
	jsonTag := structField.Tag.Get("json")
	etcdTag := structField.Tag.Get("etcd")

	tags := strings.Split(jsonTag, ",")
	if len(tags) > 1 {
		jsonTag = tags[0]
		etcdTag = tags[1]
	}
	return jsonTag, etcdTag
}

func ConvertToMap(value interface{}, keyString, delimiter string) (map[string]interface{}, error) {
	if len(strings.Split(keyString, delimiter)) > 5 {
		return nil, fmt.Errorf("%s", "Too many levels.")
	}

	var currKey string
	result := make(map[string]interface{})

	idx := strings.Index(keyString, delimiter)
	if idx == -1 {
		// Does not contain delim
		// Base case
		result[keyString] = value
	} else {
		currKey = keyString[:idx]
		result[currKey], _ = ConvertToMap(value, keyString[idx+1:], delimiter)
	}

	return result, nil
}
