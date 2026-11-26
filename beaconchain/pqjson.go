package beaconchain

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

var (
	jsonMarshalerType = reflect.TypeFor[json.Marshaler]()
	byteType          = reflect.TypeFor[byte]()
)

func toJSONValue(v reflect.Value) (any, error) {
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil, nil
		}

		if v.Type().Implements(jsonMarshalerType) {
			return marshalWith(v)
		}

		return toJSONValue(v.Elem())
	}

	if v.Type().Implements(jsonMarshalerType) {
		return marshalWith(v)
	}

	switch v.Kind() {
	case reflect.Struct:
		obj := make(orderedObject, 0, v.NumField())

		for i := range v.NumField() {
			name := strings.Split(v.Type().Field(i).Tag.Get("json"), ",")[0]
			if name == "" || name == "-" {
				return nil, fmt.Errorf("field %s.%s has no json name", v.Type(), v.Type().Field(i).Name)
			}

			value, err := toJSONValue(v.Field(i))
			if err != nil {
				return nil, err
			}

			obj = append(obj, objectField{name, value})
		}

		return obj, nil
	case reflect.Slice, reflect.Array:
		if v.Type().Elem() == byteType {
			data := make([]byte, v.Len())
			reflect.Copy(reflect.ValueOf(data), v)

			return "0x" + hex.EncodeToString(data), nil
		}

		list := make([]any, v.Len())

		for i := range v.Len() {
			value, err := toJSONValue(v.Index(i))
			if err != nil {
				return nil, err
			}

			list[i] = value
		}

		return list, nil
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10), nil
	case reflect.Bool:
		return v.Bool(), nil
	default:
		return nil, fmt.Errorf("unsupported json type %s", v.Type())
	}
}

func marshalWith(v reflect.Value) (any, error) {
	marshaler, ok := v.Interface().(json.Marshaler)
	if !ok {
		return nil, fmt.Errorf("%s is not a json marshaler", v.Type())
	}

	data, err := marshaler.MarshalJSON()
	if err != nil {
		return nil, err
	}

	return json.RawMessage(data), nil
}

type objectField struct {
	name  string
	value any
}

type orderedObject []objectField

func (o orderedObject) MarshalJSON() ([]byte, error) {
	var sb strings.Builder

	sb.WriteByte('{')

	for i, field := range o {
		if i > 0 {
			sb.WriteByte(',')
		}

		name, err := json.Marshal(field.name)
		if err != nil {
			return nil, err
		}

		value, err := json.Marshal(field.value)
		if err != nil {
			return nil, err
		}

		sb.Write(name)
		sb.WriteByte(':')
		sb.Write(value)
	}

	sb.WriteByte('}')

	return []byte(sb.String()), nil
}

func marshalPQJSON(value any) ([]byte, error) {
	jsonValue, err := toJSONValue(reflect.ValueOf(value))
	if err != nil {
		return nil, err
	}

	return json.Marshal(jsonValue)
}
