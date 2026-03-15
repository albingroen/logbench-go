package logbench

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"reflect"
)

type typedEnvelope struct {
	Type  string `json:"_type"`
	Value any    `json:"_value,omitempty"`
}

func encodeValue(v any) any {
	if v == nil {
		return nil
	}

	// Check big types before json.Marshaler (they implement it)
	switch val := v.(type) {
	case *big.Int:
		return typedEnvelope{Type: "@go/big.Int", Value: val.String()}
	case *big.Float:
		return typedEnvelope{Type: "@go/big.Float", Value: val.Text('g', -1)}
	}

	if err, ok := v.(error); ok {
		return typedEnvelope{
			Type: "@go/error",
			Value: map[string]any{
				"message": err.Error(),
				"type":    reflect.TypeOf(v).String(),
			},
		}
	}
	if _, ok := v.(json.Marshaler); ok {
		return v
	}

	rv := reflect.ValueOf(v)
	return encodeReflect(rv)
}

func encodeReflect(rv reflect.Value) any {
	if !rv.IsValid() {
		return nil
	}

	// Dereference pointers
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Float32, reflect.Float64:
		f := rv.Float()
		if math.IsNaN(f) {
			return typedEnvelope{Type: "@go/NaN"}
		}
		if math.IsInf(f, 1) {
			return typedEnvelope{Type: "@go/Infinity", Value: 1}
		}
		if math.IsInf(f, -1) {
			return typedEnvelope{Type: "@go/Infinity", Value: -1}
		}
		return f

	case reflect.Complex64, reflect.Complex128:
		c := rv.Complex()
		return typedEnvelope{
			Type:  "@go/" + rv.Type().String(),
			Value: fmt.Sprintf("(%g+%gi)", real(c), imag(c)),
		}

	case reflect.Func:
		return typedEnvelope{Type: "@go/Function", Value: rv.Type().String()}

	case reflect.Chan:
		return typedEnvelope{
			Type:  "@go/chan",
			Value: rv.Type().String(),
		}

	case reflect.Slice:
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			// []byte → base64
			b := rv.Bytes()
			return typedEnvelope{
				Type:  "@go/bytes",
				Value: base64.StdEncoding.EncodeToString(b),
			}
		}
		return encodeSlice(rv)

	case reflect.Array:
		return encodeSequence(rv)

	case reflect.Map:
		return encodeMap(rv)

	case reflect.Struct:
		return encodeStruct(rv)

	default:
		if rv.CanInterface() {
			return rv.Interface()
		}
		return nil
	}
}

func encodeSlice(rv reflect.Value) any {
	if rv.IsNil() {
		return nil
	}
	return encodeSequence(rv)
}

func encodeSequence(rv reflect.Value) []any {
	result := make([]any, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		result[i] = encodeValue(rv.Index(i).Interface())
	}
	return result
}

func encodeMap(rv reflect.Value) any {
	if rv.IsNil() {
		return nil
	}
	result := make(map[string]any)
	iter := rv.MapRange()
	for iter.Next() {
		k := iter.Key()
		var key string
		if k.Kind() == reflect.String {
			key = k.String()
		} else {
			key = fmt.Sprintf("%v", k.Interface())
		}
		result[key] = encodeValue(iter.Value().Interface())
	}
	return result
}

func encodeStruct(rv reflect.Value) any {
	t := rv.Type()
	fields := make(map[string]any)
	fields["__name__"] = t.Name()

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		fields[f.Name] = encodeValue(rv.Field(i).Interface())
	}

	return typedEnvelope{
		Type:  "@go/Struct",
		Value: fields,
	}
}

func prepareContent(content []any) any {
	encoded := make([]any, len(content))
	for i, v := range content {
		encoded[i] = encodeValue(v)
	}
	if len(encoded) == 1 {
		return encoded[0]
	}
	return encoded
}
