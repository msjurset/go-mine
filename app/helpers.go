package app

import (
	"strings"

	"github.com/msjurset/golars"
)

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

func isNumeric(dt golars.DataType) bool {
	switch dt {
	case golars.Int8, golars.Int16, golars.Int32, golars.Int64,
		golars.UInt8, golars.UInt16, golars.UInt32, golars.UInt64,
		golars.Float32, golars.Float64:
		return true
	}
	return false
}

func shortTypeName(dt golars.DataType) string {
	switch dt {
	case golars.Int8:
		return "i8"
	case golars.Int16:
		return "i16"
	case golars.Int32:
		return "i32"
	case golars.Int64:
		return "i64"
	case golars.UInt8:
		return "u8"
	case golars.UInt16:
		return "u16"
	case golars.UInt32:
		return "u32"
	case golars.UInt64:
		return "u64"
	case golars.Float32:
		return "f32"
	case golars.Float64:
		return "f64"
	case golars.Boolean:
		return "bool"
	case golars.String:
		return "str"
	case golars.Date:
		return "date"
	case golars.DateTime:
		return "datetime"
	case golars.Time:
		return "time"
	case golars.Duration:
		return "dur"
	default:
		return "?"
	}
}

func cleanFileName(name string) string {
	name = strings.TrimSuffix(name, ".csv")
	name = strings.TrimSuffix(name, ".parquet")
	name = strings.TrimSuffix(name, ".json")
	name = strings.TrimSuffix(name, ".tsv")
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, " ", "_")
	return name
}

func clampedScroll(scrollY, lineCount, visibleHeight int) (start, end int) {
	maxScroll := lineCount - visibleHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scrollY > maxScroll {
		scrollY = maxScroll
	}
	end = min(scrollY+visibleHeight, lineCount)
	return scrollY, end
}
