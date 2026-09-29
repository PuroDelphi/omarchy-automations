package core

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

func field(data map[string]any, path string) (any, bool) {
	var current any = data
	for _, part := range strings.Split(path, ".") {
		switch value := current.(type) {
		case map[string]any:
			var ok bool
			current, ok = value[part]
			if !ok {
				return nil, false
			}
		case []any:
			index, err := strconv.Atoi(part)
			if err != nil || index < 0 || index >= len(value) || strconv.Itoa(index) != part {
				return nil, false
			}
			current = value[index]
		default:
			return nil, false
		}
	}
	return current, true
}
func number(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	}
	return 0, false
}
func matches(f Flow, ev Event) bool {
	if !f.Enabled || f.Source != ev.Source {
		return false
	}
	for _, c := range f.Conditions {
		if !conditionMatches(c, ev) {
			return false
		}
	}
	return true
}

// conditionMatches is shared by dispatch and simulation explanations.
func conditionMatches(c Condition, ev Event) bool {
	v, ok := field(map[string]any{"data": ev.Data, "type": ev.Type, "source": ev.Source}, c.Field)
	if !ok {
		return false
	}
	switch c.Op {
	case "eq":
		return reflect.DeepEqual(v, c.Value)
	case "ne":
		return !reflect.DeepEqual(v, c.Value)
	case "gt", "lt":
		a, aok := number(v)
		b, bok := number(c.Value)
		return aok && bok && ((c.Op == "gt" && a > b) || (c.Op == "lt" && a < b))
	case "contains":
		a, aok := v.(string)
		b, bok := c.Value.(string)
		return aok && bok && strings.Contains(a, b)
	}
	return false
}

// Render performs bounded literal substitution, never evaluation. Encoding for
// HTTP/notifications is applied later by the action-specific implementation.
func render(template string, ev Event) (string, error) {
	if len(template) > 4096 {
		return "", fmt.Errorf("template too long")
	}
	data := map[string]any{"data": ev.Data, "type": ev.Type, "source": ev.Source, "id": ev.ID}
	var out strings.Builder
	for n := 0; strings.Contains(template, "{{"); n++ {
		if n >= 32 {
			return "", fmt.Errorf("too many substitutions")
		}
		start := strings.Index(template, "{{")
		end := strings.Index(template[start+2:], "}}")
		if end < 0 {
			return "", fmt.Errorf("unclosed template field")
		}
		end += start + 2
		out.WriteString(template[:start])
		key := strings.TrimSpace(template[start+2 : end])
		v, ok := field(data, key)
		if !ok {
			return "", fmt.Errorf("missing field %s", key)
		}
		switch value := v.(type) {
		case string:
			out.WriteString(value)
		case bool, float64, int:
			out.WriteString(fmt.Sprint(value))
		default:
			return "", fmt.Errorf("field must be scalar")
		}
		template = template[end+2:]
		if out.Len() > 16384 {
			return "", fmt.Errorf("rendered content too long")
		}
	}
	out.WriteString(template)
	if out.Len() > 16384 {
		return "", fmt.Errorf("rendered content too long")
	}
	return out.String(), nil
}

// Interpolate string values after parsing JSON, then serialize. Remote text
// cannot inject keys or break out of a JSON string.
func renderJSON(template string, ev Event) ([]byte, error) {
	if template == "" {
		template = "{}"
	}
	var v any
	if err := json.Unmarshal([]byte(template), &v); err != nil {
		return nil, fmt.Errorf("HTTP body template must be valid JSON")
	}
	var walk func(any, int) (any, error)
	walk = func(v any, depth int) (any, error) {
		if depth > 16 {
			return nil, fmt.Errorf("JSON template too deep")
		}
		switch value := v.(type) {
		case string:
			if strings.HasPrefix(value, "{{") && strings.HasSuffix(value, "}}") && strings.Count(value, "{{") == 1 {
				key := strings.TrimSpace(value[2 : len(value)-2])
				v, ok := field(map[string]any{"data": ev.Data, "type": ev.Type, "source": ev.Source, "id": ev.ID}, key)
				if ok {
					switch v.(type) {
					case string, bool, float64, int, nil:
						return v, nil
					}
				}
			}
			return render(value, ev)
		case map[string]any:
			for k, item := range value {
				r, err := walk(item, depth+1)
				if err != nil {
					return nil, err
				}
				value[k] = r
			}
		case []any:
			for i, item := range value {
				r, err := walk(item, depth+1)
				if err != nil {
					return nil, err
				}
				value[i] = r
			}
		}
		return v, nil
	}
	value, err := walk(v, 0)
	if err != nil {
		return nil, err
	}
	return json.Marshal(value)
}
