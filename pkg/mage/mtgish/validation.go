package mtgish

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
)

const maxJSONDepth = 128

func pointerPath(path, key string) string {
	return path + "/" + strings.ReplaceAll(strings.ReplaceAll(key, "~", "~0"), "/", "~1")
}

func indexPath(path string, index int) string {
	return pointerPath(path, strconv.Itoa(index))
}

func jsonError(path string, err error, offset int64) error {
	if syntax, ok := errors.AsType[*json.SyntaxError](err); ok {
		offset = syntax.Offset
	}
	return diagnostic(path, InvalidInput, "", "", fmt.Sprintf("%v (byte offset %d)", err, offset))
}

func validateJSON(data []byte, path string) error {
	return validateJSONDepth(data, path, 0)
}

func validateJSONDepth(data []byte, path string, ancestorDepth int) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := walkJSON(decoder, path, ancestorDepth); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			err = fmt.Errorf("expected EOF after one JSON value")
		}
		return jsonError(path, err, decoder.InputOffset())
	}
	return nil
}

func walkJSON(decoder *json.Decoder, path string, depth int) error {
	token, err := decoder.Token()
	if err != nil {
		return jsonError(path, err, decoder.InputOffset())
	}
	delim, container := token.(json.Delim)
	if !container {
		return nil
	}
	depth++
	if depth > maxJSONDepth {
		return diagnostic(path, ResourceLimit, "", "", "JSON container depth exceeds 128")
	}
	switch delim {
	case '{':
		seen := make(map[string]bool)
		for decoder.More() {
			key, err := decoder.Token()
			if err != nil {
				return jsonError(path, err, decoder.InputOffset())
			}
			name, ok := key.(string)
			if !ok {
				return diagnostic(path, InvalidInput, "", "", "expected an object key")
			}
			childPath := pointerPath(path, name)
			if seen[name] {
				return diagnostic(childPath, InvalidInput, "", "", "duplicate object key")
			}
			seen[name] = true
			if err := walkJSON(decoder, childPath, depth); err != nil {
				return err
			}
		}
	case '[':
		for i := 0; decoder.More(); i++ {
			if err := walkJSON(decoder, indexPath(path, i), depth); err != nil {
				return err
			}
		}
	default:
		return diagnostic(path, InvalidInput, "", "", "unexpected container delimiter")
	}
	if _, err := decoder.Token(); err != nil {
		return jsonError(path, err, decoder.InputOffset())
	}
	return nil
}

func isNull(data []byte) bool {
	return bytes.Equal(bytes.TrimSpace(data), []byte("null"))
}

func decodeValue(data []byte, path string, value any) error {
	if len(data) == 0 || isNull(data) {
		return diagnostic(path, InvalidInput, "", "", "required value is missing or null")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		var field *json.UnmarshalTypeError
		if errors.As(err, &field) && field.Field != "" {
			for name := range strings.SplitSeq(field.Field, ".") {
				path = pointerPath(path, name)
			}
		}
		return jsonError(path, err, decoder.InputOffset())
	}
	return nil
}

func objectFields(data []byte, path string) (map[string]json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if err := decodeValue(data, path, &fields); err != nil {
		return nil, err
	}
	return fields, nil
}

func checkFields(fields map[string]json.RawMessage, path string, allowed ...string) error {
	var unknown []string
	for key := range fields {
		if !slices.Contains(allowed, key) {
			unknown = append(unknown, key)
		}
	}
	if len(unknown) != 0 {
		slices.Sort(unknown)
		return diagnostic(pointerPath(path, unknown[0]), InvalidInput, "", "", "unknown field")
	}
	return nil
}

func decodeStringList(data []byte, path string) ([]string, error) {
	var items []json.RawMessage
	if err := decodeValue(data, path, &items); err != nil {
		return nil, err
	}
	result := make([]string, len(items))
	for i, raw := range items {
		if err := decodeValue(raw, indexPath(path, i), &result[i]); err != nil {
			return nil, err
		}
		if result[i] == "" {
			return nil, diagnostic(indexPath(path, i), InvalidInput, "", "", "expected a nonempty string")
		}
	}
	return result, nil
}
