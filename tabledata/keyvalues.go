package tabledata

// Entry is one caller-ordered key/value pair. Empty values remain empty.
type Entry struct{ Key, Value string }

// KeyValues creates an owned two-column table without inventing a "no data" row.
// A slice preserves caller order; callers with a map choose a deterministic order.
func KeyValues(keyHeader, valueHeader string, entries []Entry) Data {
	d := Data{Headers: []string{keyHeader, valueHeader}}
	for _, entry := range entries {
		d.Rows = append(d.Rows, []string{entry.Key, entry.Value})
	}
	return d
}
