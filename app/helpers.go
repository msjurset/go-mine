package app

import "github.com/msjurset/golars"

// schemaFields extracts all fields from a golars Schema.
func schemaFields(s *golars.Schema) []golars.Field {
	n := s.Len()
	fields := make([]golars.Field, n)
	for i := 0; i < n; i++ {
		fields[i] = s.Field(i)
	}
	return fields
}

// getColumn is a convenience wrapper that ignores the error from Column().
func getColumn(df *golars.DataFrame, name string) *golars.Series {
	col, _ := df.Column(name)
	return col
}
