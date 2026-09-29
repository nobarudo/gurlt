package jsonpath

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Query parses the JSON string and evaluates the given path expression.
func Query(jsonStr string, path string) (any, error) {
	if strings.TrimSpace(jsonStr) == "" {
		return nil, fmt.Errorf("empty JSON content")
	}

	var root any
	if err := json.Unmarshal([]byte(jsonStr), &root); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	tokens, err := tokenize(path)
	if err != nil {
		return nil, err
	}

	return evaluate(root, tokens)
}

// QueryPretty evaluates the path and returns a pretty-printed JSON string (or clean primitive).
func QueryPretty(jsonStr string, path string) (string, error) {
	val, err := Query(jsonStr, path)
	if err != nil {
		return "", err
	}

	switch v := val.(type) {
	case string:
		return v, nil
	case nil:
		return "null", nil
	default:
		bytes, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return fmt.Sprintf("%v", v), nil
		}
		return string(bytes), nil
	}
}

func tokenize(path string) ([]string, error) {
	path = strings.TrimSpace(path)
	if path == "" || path == "." {
		return nil, nil
	}
	if strings.HasPrefix(path, ".") {
		path = path[1:]
	}

	var tokens []string
	var current strings.Builder
	inBracket := false
	inQuote := byte(0)

	for i := 0; i < len(path); i++ {
		ch := path[i]

		if inQuote != 0 {
			if ch == inQuote {
				inQuote = 0
			} else {
				current.WriteByte(ch)
			}
			continue
		}

		if ch == '\'' || ch == '"' {
			inQuote = ch
			continue
		}

		if ch == '[' {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			inBracket = true
			continue
		}

		if ch == ']' {
			inBracket = false
			token := current.String()
			if token == "" {
				token = "*"
			}
			tokens = append(tokens, token)
			current.Reset()
			if i+1 < len(path) && path[i+1] == '.' {
				i++ // skip following dot
			}
			continue
		}

		if ch == '.' && !inBracket {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			continue
		}

		current.WriteByte(ch)
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens, nil
}

func evaluate(current any, tokens []string) (any, error) {
	if len(tokens) == 0 {
		return current, nil
	}

	token := tokens[0]
	remaining := tokens[1:]

	if current == nil {
		return nil, fmt.Errorf("cannot access %q on null", token)
	}

	// 1. Wildcard [*] or [] across an array
	if token == "*" {
		slice, ok := current.([]any)
		if !ok {
			return nil, fmt.Errorf("wildcard '*' requires an array, got %T", current)
		}
		if len(remaining) == 0 {
			return slice, nil
		}
		var results []any
		for _, item := range slice {
			res, err := evaluate(item, remaining)
			if err == nil && res != nil {
				results = append(results, res)
			}
		}
		return results, nil
	}

	// 2. Length/size helpers
	if token == "length" || token == "size" || token == "count" {
		if m, ok := current.(map[string]any); ok {
			if val, exists := m[token]; exists {
				return evaluate(val, remaining)
			}
			if len(remaining) == 0 {
				return len(m), nil
			}
		}
		if s, ok := current.([]any); ok {
			if len(remaining) == 0 {
				return len(s), nil
			}
		}
	}

	// 3. Object property lookup
	if m, ok := current.(map[string]any); ok {
		val, exists := m[token]
		if !exists {
			return nil, fmt.Errorf("key %q not found in object", token)
		}
		return evaluate(val, remaining)
	}

	// 4. Array index lookup
	if s, ok := current.([]any); ok {
		idx, err := strconv.Atoi(token)
		if err != nil {
			return nil, fmt.Errorf("cannot access property %q on array (expected integer index)", token)
		}
		if idx < 0 {
			idx = len(s) + idx
		}
		if idx < 0 || idx >= len(s) {
			return nil, fmt.Errorf("index %d out of bounds (array length %d)", idx, len(s))
		}
		return evaluate(s[idx], remaining)
	}

	return nil, fmt.Errorf("cannot access property %q on %T", token, current)
}
