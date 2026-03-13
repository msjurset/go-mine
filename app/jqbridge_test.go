package app

import (
	"testing"

	"github.com/msjurset/golars"
)

func TestDataFrameToJSON(t *testing.T) {
	df, err := golars.NewDataFrame(
		golars.NewStringSeries("name", []string{"Alice", "Bob"}),
		golars.NewInt64Series("age", []int64{30, 25}),
		golars.NewFloat64Series("score", []float64{95.5, 87.3}),
		golars.NewBooleanSeries("active", []bool{true, false}),
	)
	if err != nil {
		t.Fatal(err)
	}

	result := dataFrameToJSON(df)

	if len(result) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(result))
	}

	row0 := result[0].(map[string]interface{})
	if row0["name"] != "Alice" {
		t.Errorf("expected Alice, got %v", row0["name"])
	}
	if row0["age"] != int(30) {
		t.Errorf("expected 30, got %v (%T)", row0["age"], row0["age"])
	}
	if row0["score"] != 95.5 {
		t.Errorf("expected 95.5, got %v", row0["score"])
	}
	if row0["active"] != true {
		t.Errorf("expected true, got %v", row0["active"])
	}
}

func TestDataFrameToJSONWithNulls(t *testing.T) {
	df, err := golars.NewDataFrame(
		golars.NewStringSeriesWithValidity("name", []string{"Alice", ""}, []bool{true, false}),
		golars.NewInt64SeriesWithValidity("age", []int64{30, 0}, []bool{true, false}),
	)
	if err != nil {
		t.Fatal(err)
	}

	result := dataFrameToJSON(df)
	row1 := result[1].(map[string]interface{})

	if row1["name"] != nil {
		t.Errorf("expected nil for null name, got %v", row1["name"])
	}
	if row1["age"] != nil {
		t.Errorf("expected nil for null age, got %v", row1["age"])
	}
}

func TestJSONToDataFrame(t *testing.T) {
	input := []interface{}{
		map[string]interface{}{"name": "Alice", "age": float64(30), "active": true},
		map[string]interface{}{"name": "Bob", "age": float64(25), "active": false},
	}

	df, err := jsonToDataFrame(input)
	if err != nil {
		t.Fatal(err)
	}

	rows, cols := df.Shape()
	if rows != 2 || cols != 3 {
		t.Fatalf("expected 2x3, got %dx%d", rows, cols)
	}

	// age should be Int64 since values are whole numbers
	ageSeries, _ := df.Column("age")
	if ageSeries.DataType() != golars.Int64 {
		t.Errorf("expected age to be Int64, got %v", ageSeries.DataType())
	}
	v, _ := ageSeries.GetInt64(0)
	if v != 30 {
		t.Errorf("expected 30, got %d", v)
	}
}

func TestJSONToDataFrameFloat(t *testing.T) {
	input := []interface{}{
		map[string]interface{}{"val": float64(1.5)},
		map[string]interface{}{"val": float64(2.7)},
	}

	df, err := jsonToDataFrame(input)
	if err != nil {
		t.Fatal(err)
	}

	valSeries, _ := df.Column("val")
	if valSeries.DataType() != golars.Float64 {
		t.Errorf("expected Float64, got %v", valSeries.DataType())
	}
}

func TestJSONToDataFrameWithNulls(t *testing.T) {
	input := []interface{}{
		map[string]interface{}{"name": "Alice", "age": float64(30)},
		map[string]interface{}{"name": nil, "age": nil},
	}

	df, err := jsonToDataFrame(input)
	if err != nil {
		t.Fatal(err)
	}

	nameSeries, _ := df.Column("name")
	if !nameSeries.IsNull(1) {
		t.Error("expected row 1 name to be null")
	}

	ageSeries, _ := df.Column("age")
	if !ageSeries.IsNull(1) {
		t.Error("expected row 1 age to be null")
	}
}

func TestJSONToDataFrameNonTabular(t *testing.T) {
	input := []interface{}{"hello", "world"}

	_, err := jsonToDataFrame(input)
	if err == nil {
		t.Error("expected error for non-tabular data")
	}
}

func TestJSONToDataFrameEmpty(t *testing.T) {
	_, err := jsonToDataFrame(nil)
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestRoundTrip(t *testing.T) {
	original, err := golars.NewDataFrame(
		golars.NewStringSeries("city", []string{"NYC", "LA", "Chicago"}),
		golars.NewInt64Series("pop", []int64{8000000, 4000000, 2700000}),
		golars.NewFloat64Series("lat", []float64{40.7, 34.0, 41.8}),
	)
	if err != nil {
		t.Fatal(err)
	}

	jsonData := dataFrameToJSON(original)
	rebuilt, err := jsonToDataFrame(jsonData)
	if err != nil {
		t.Fatal(err)
	}

	origRows, origCols := original.Shape()
	newRows, newCols := rebuilt.Shape()
	if origRows != newRows || origCols != newCols {
		t.Fatalf("shape mismatch: %dx%d vs %dx%d", origRows, origCols, newRows, newCols)
	}
}
