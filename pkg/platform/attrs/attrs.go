package attrs

// ExtractString extracts a string value from a key-value attribute slice.
// The slice should be formatted as [key1, value1, key2, value2, ...].
// Returns empty string if the key is not found or the value is not a string.
func ExtractString(attrs []any, key string) string {
	for i := 0; i < len(attrs)-1; i += 2 {
		k, ok := attrs[i].(string)
		if !ok {
			continue
		}
		if k == key {
			if v, ok := attrs[i+1].(string); ok {
				return v
			}
		}
	}
	return ""
}
