package shared

import "reflect"

func Merge[T any](base T, ov T) T {
	rvBase := reflect.ValueOf(&base).Elem() // addressable
	rvOv := reflect.ValueOf(ov)

	for i := 0; i < rvBase.NumField(); i++ {
		fOv := rvOv.Field(i)
		if !fOv.IsZero() { // ← built-in zero test
			rvBase.Field(i).Set(fOv)
		}
	}
	return base
}

// FieldsDefined reports whether every name in fields is present on the
// (possibly-pointer) struct v. It returns the first missing field name
// (empty string means all present).
func FieldsDefined(v any, fields []string) (ok bool, missing string) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		panic("FieldsDefined: value must be a struct or *struct")
	}

	rt := rv.Type() // a reflect.Type is a bit faster for look-ups

	for _, name := range fields {
		if _, found := rt.FieldByName(name); !found {
			return false, name
		}
	}
	return true, ""
}
