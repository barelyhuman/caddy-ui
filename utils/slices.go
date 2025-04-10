package utils

import (
	"fmt"
	"reflect"
	"strings"
)

func Join[T string | int | int32 | int64](sli []T, sep string) string {
	var joined strings.Builder
	for in, part := range sli {
		v := reflect.ValueOf(part)
		switch v.Kind() {
		case reflect.String:
			{
				joined.WriteString(v.String())
				break
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			{
				joined.WriteString(fmt.Sprintf("%v", v.Int()))
				break
			}
		}

		if len(sli)-1 != in {
			joined.WriteString(",")
		}
	}
	return joined.String()
}
