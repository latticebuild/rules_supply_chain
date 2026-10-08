package jsondata

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"
)

func Read(path string, target any) error { return ReadMode(path, target, false) }
func ReadMode(path string, target any, denyUnknown bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	normalized, err := typedJSON(data, reflect.TypeOf(target).Elem(), denyUnknown)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return json.Unmarshal(normalized, target)
}

func typedJSON(data []byte, kind reflect.Type, denyUnknown bool) ([]byte, error) {
	if kind == reflect.TypeFor[json.RawMessage]() {
		return data, validJSONStrings(data)
	}
	if kind.Kind() == reflect.Pointer {
		if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
			return data, nil
		}
		return typedJSON(data, kind.Elem(), denyUnknown)
	}
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return nil, fmt.Errorf("null is not a %s", kind)
	}
	switch kind.Kind() {
	case reflect.Struct:

		// Serde's derived structs also accept positional sequences. Preserve defaults
		// for omitted trailing fields and validate every present element.
		if trimmed := bytes.TrimSpace(data); len(trimmed) > 0 && trimmed[0] == '[' {
			var elements []json.RawMessage
			if err := json.Unmarshal(data, &elements); err != nil {
				return nil, err
			}
			if len(elements) > kind.NumField() {
				return nil, errors.New("too many struct elements")
			}
			values := map[string]json.RawMessage{}
			for i := 0; i < kind.NumField(); i++ {
				field := kind.Field(i)
				key := strings.Split(field.Tag.Get("json"), ",")[0]
				if key == "" {
					key = field.Name
				}
				if i >= len(elements) {
					if field.Type.Kind() != reflect.Pointer && field.Tag.Get("optional") != "true" {
						return nil, fmt.Errorf("missing field %s", key)
					}
					continue
				}
				normalized, err := typedJSON(elements[i], field.Type, denyUnknown)
				if err != nil {
					return nil, err
				}
				values[key] = normalized
			}
			return json.Marshal(values)
		}
		decoder := json.NewDecoder(bytes.NewReader(data))
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		if token != json.Delim('{') {
			return nil, errors.New("expected an object")
		}
		fields := map[string]reflect.StructField{}
		for i := 0; i < kind.NumField(); i++ {
			field := kind.Field(i)
			key := strings.Split(field.Tag.Get("json"), ",")[0]
			if key == "" {
				key = field.Name
			}
			fields[key] = field
		}
		values := map[string]json.RawMessage{}
		for decoder.More() {
			start := decoder.InputOffset()
			token, err := decoder.Token()
			if err == nil {
				err = validJSONStrings(data[start:decoder.InputOffset()])
			}
			if err != nil {
				return nil, err
			}
			key, ok := token.(string)
			if !ok {
				return nil, errors.New("expected field name")
			}
			var raw json.RawMessage
			if err := decoder.Decode(&raw); err != nil {
				return nil, err
			}
			field, known := fields[key]
			if !known {
				if denyUnknown {
					return nil, fmt.Errorf("unknown field %s", key)
				}
				continue
			}
			if _, seen := values[key]; seen {
				return nil, fmt.Errorf("duplicate field %s", key)
			}
			normalized, err := typedJSON(raw, field.Type, denyUnknown)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", key, err)
			}
			values[key] = normalized
		}
		if _, err := decoder.Token(); err != nil {
			return nil, err
		}
		if _, err := decoder.Token(); err != io.EOF {
			return nil, errors.New("trailing JSON")
		}
		for key, field := range fields {
			if _, seen := values[key]; !seen && field.Type.Kind() != reflect.Pointer && field.Tag.Get("optional") != "true" {
				return nil, fmt.Errorf("missing field %s", key)
			}
		}
		return json.Marshal(values)
	case reflect.Slice, reflect.Array:
		var values []json.RawMessage
		if err := json.Unmarshal(data, &values); err != nil {
			return nil, err
		}
		for i, value := range values {
			normalized, err := typedJSON(value, kind.Elem(), denyUnknown)
			if err != nil {
				return nil, err
			}
			values[i] = normalized
		}
		return json.Marshal(values)
	case reflect.Map:
		decoder := json.NewDecoder(bytes.NewReader(data))
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		if token != json.Delim('{') {
			return nil, errors.New("expected an object")
		}
		values := map[string]json.RawMessage{}
		for decoder.More() {
			start := decoder.InputOffset()
			token, err := decoder.Token()
			if err != nil {
				return nil, err
			}
			if err := validJSONStrings(data[start:decoder.InputOffset()]); err != nil {
				return nil, err
			}
			key, ok := token.(string)
			if !ok {
				return nil, errors.New("expected field name")
			}
			var raw json.RawMessage
			if err := decoder.Decode(&raw); err != nil {
				return nil, err
			}
			normalized, err := typedJSON(raw, kind.Elem(), denyUnknown)
			if err != nil {
				return nil, err
			}
			values[key] = normalized
		}
		if _, err := decoder.Token(); err != nil {
			return nil, err
		}
		if _, err := decoder.Token(); err != io.EOF {
			return nil, errors.New("trailing JSON")
		}
		return json.Marshal(values)
	default:
		if kind.Kind() == reflect.String {
			if err := validJSONStrings(data); err != nil {
				return nil, err
			}
		}
		if err := json.Unmarshal(data, reflect.New(kind).Interface()); err != nil {
			return nil, err
		}
		return data, nil
	}
}

func validJSONStringEscapes(data []byte) error {
	quoted := false
	unit := func(at int) (uint64, bool) {
		if at+6 > len(data) || data[at] != '\\' || data[at+1] != 'u' {
			return 0, false
		}
		n, e := strconv.ParseUint(string(data[at+2:at+6]), 16, 16)
		return n, e == nil
	}
	for i := 0; i < len(data); i++ {
		if data[i] == '"' {
			quoted = !quoted
			continue
		}
		if !quoted || data[i] != '\\' {
			continue
		}
		n, ok := unit(i)
		if !ok {
			i++
			continue
		}
		if n >= 0xD800 && n <= 0xDBFF {
			next, ok := unit(i + 6)
			if !ok || next < 0xDC00 || next > 0xDFFF {
				return errors.New("unpaired JSON surrogate")
			}
			i += 6
		} else if n >= 0xDC00 && n <= 0xDFFF {
			return errors.New("unpaired JSON surrogate")
		}
		i += 5
	}
	return nil
}

func validJSONStrings(data []byte) error {
	if !utf8.Valid(data) {
		return errors.New("invalid UTF-8")
	}
	return validJSONStringEscapes(data)
}
