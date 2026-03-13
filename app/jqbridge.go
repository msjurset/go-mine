package app

import (
	"fmt"
	"math"

	"github.com/msjurset/golars"
)

// dataFrameToJSON converts a golars DataFrame into a slice of maps suitable
// for gojq processing. Each row becomes a map[string]interface{} keyed by
// column name.
func dataFrameToJSON(df *golars.DataFrame) []interface{} {
	rows, cols := df.Shape()
	schema := df.Schema()

	fields := make([]golars.Field, cols)
	series := make([]*golars.Series, cols)
	for c := 0; c < cols; c++ {
		fields[c] = schema.Field(c)
		series[c], _ = df.Column(fields[c].Name)
	}

	result := make([]interface{}, rows)
	for r := 0; r < rows; r++ {
		row := make(map[string]interface{}, cols)
		for c := 0; c < cols; c++ {
			if series[c].IsNull(r) {
				row[fields[c].Name] = nil
				continue
			}
			row[fields[c].Name] = extractValue(series[c], r, fields[c].Dtype)
		}
		result[r] = row
	}
	return result
}

func extractValue(s *golars.Series, i int, dt golars.DataType) interface{} {
	switch dt {
	case golars.Int8, golars.Int16, golars.Int32, golars.Int64,
		golars.UInt8, golars.UInt16, golars.UInt32, golars.UInt64:
		v, _ := s.GetInt64(i)
		return int(v) // gojq uses int for integer arithmetic
	case golars.Float32, golars.Float64:
		v, _ := s.GetFloat64(i)
		return v
	case golars.Boolean:
		v, _ := s.GetBool(i)
		return v
	case golars.String:
		v, _ := s.GetString(i)
		return v
	default:
		v, _ := s.GetString(i)
		return v
	}
}

// jsonToDataFrame reconstructs a DataFrame from gojq results when the output
// is a flat array of maps with consistent string keys. Returns an error if
// the results are not tabular.
func jsonToDataFrame(results []interface{}) (*golars.DataFrame, error) {
	if len(results) == 0 {
		return nil, fmt.Errorf("empty result set")
	}

	// Check that all results are maps
	maps := make([]map[string]interface{}, len(results))
	for i, r := range results {
		m, ok := r.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("result %d is not an object", i)
		}
		maps[i] = m
	}

	// Collect column names from first row, preserving order
	firstRow := maps[0]
	colNames := make([]string, 0, len(firstRow))
	for k := range firstRow {
		colNames = append(colNames, k)
	}
	// Sort for deterministic output
	sortStrings(colNames)

	// Infer types from first non-nil value in each column
	colTypes := make(map[string]golars.DataType, len(colNames))
	for _, name := range colNames {
		colTypes[name] = inferColumnType(maps, name)
	}

	// Build Series for each column
	seriesList := make([]*golars.Series, len(colNames))
	for ci, name := range colNames {
		dt := colTypes[name]
		n := len(maps)
		switch dt {
		case golars.Int64:
			data := make([]int64, n)
			valid := make([]bool, n)
			for i, m := range maps {
				v := toInt64OrNil(m[name])
				if v != nil {
					data[i] = v.(int64)
					valid[i] = true
				}
			}
			seriesList[ci] = golars.NewInt64SeriesWithValidity(name, data, valid)
		case golars.Float64:
			data := make([]float64, n)
			valid := make([]bool, n)
			for i, m := range maps {
				v := toFloat64OrNil(m[name])
				if v != nil {
					data[i] = v.(float64)
					valid[i] = true
				}
			}
			seriesList[ci] = golars.NewFloat64SeriesWithValidity(name, data, valid)
		case golars.Boolean:
			data := make([]bool, n)
			valid := make([]bool, n)
			for i, m := range maps {
				v := toBoolOrNil(m[name])
				if v != nil {
					data[i] = v.(bool)
					valid[i] = true
				}
			}
			seriesList[ci] = golars.NewBooleanSeriesWithValidity(name, data, valid)
		default: // String
			data := make([]string, n)
			valid := make([]bool, n)
			for i, m := range maps {
				v := toStringOrNil(m[name])
				if v != nil {
					data[i] = v.(string)
					valid[i] = true
				}
			}
			seriesList[ci] = golars.NewStringSeriesWithValidity(name, data, valid)
		}
	}

	df, err := golars.NewDataFrame(seriesList...)
	if err != nil {
		return nil, fmt.Errorf("build dataframe: %w", err)
	}
	return df, nil
}

func inferColumnType(maps []map[string]interface{}, name string) golars.DataType {
	for _, m := range maps {
		v := m[name]
		if v == nil {
			continue
		}
		switch v.(type) {
		case int, int64, float64:
			// Check if all non-nil values are whole numbers
			allInt := true
			for _, m2 := range maps {
				v2 := m2[name]
				if v2 == nil {
					continue
				}
				switch n := v2.(type) {
				case float64:
					if n != math.Trunc(n) || math.IsInf(n, 0) || math.IsNaN(n) {
						allInt = false
					}
				case int, int64:
					// integer types are fine
				default:
					return golars.String
				}
			}
			if allInt {
				return golars.Int64
			}
			return golars.Float64
		case bool:
			return golars.Boolean
		case string:
			return golars.String
		default:
			return golars.String
		}
	}
	return golars.String
}

func toInt64OrNil(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	switch n := v.(type) {
	case int:
		return int64(n)
	case int64:
		return n
	case float64:
		return int64(n)
	default:
		return nil
	}
}

func toFloat64OrNil(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	switch n := v.(type) {
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case float64:
		return n
	default:
		return nil
	}
}

func toBoolOrNil(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	b, ok := v.(bool)
	if !ok {
		return nil
	}
	return b
}

func toStringOrNil(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Sprintf("%v", v)
	}
	return s
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
