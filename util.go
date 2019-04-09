package clconfig

import (
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
