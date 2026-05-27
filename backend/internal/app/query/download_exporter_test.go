package query

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"

	domainquery "superset/auth-service/internal/domain/query"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func TestExportCSV_WritesBOMAndHeaders(t *testing.T) {
	resp := &domainquery.ExecuteResponse{
		Columns: []domainquery.ColumnInfo{
			{Name: "id", Type: "int4"},
			{Name: "name", Type: "text"},
		},
		Data: []map[string]interface{}{
			{"id": float64(1), "name": "Alice"},
			{"id": float64(2), "name": "Bob"},
		},
	}

	var buf bytes.Buffer
	err := Export(&buf, FormatCSV, resp)
	require.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.HasPrefix(output, "\xEF\xBB\xBF"), "CSV must start with UTF-8 BOM")

	reader := csv.NewReader(strings.NewReader(strings.TrimPrefix(output, "\xEF\xBB\xBF")))
	records, err := reader.ReadAll()
	require.NoError(t, err)
	assert.Len(t, records, 3)
	assert.Equal(t, []string{"id", "name"}, records[0])
	assert.Equal(t, []string{"1", "Alice"}, records[1])
	assert.Equal(t, []string{"2", "Bob"}, records[2])
}

func TestExportCSV_NilValues(t *testing.T) {
	resp := &domainquery.ExecuteResponse{
		Columns: []domainquery.ColumnInfo{
			{Name: "id", Type: "int4"},
			{Name: "extra", Type: "text"},
		},
		Data: []map[string]interface{}{
			{"id": float64(1), "extra": nil},
		},
	}

	var buf bytes.Buffer
	err := Export(&buf, FormatCSV, resp)
	require.NoError(t, err)

	output := buf.String()
	reader := csv.NewReader(strings.NewReader(strings.TrimPrefix(output, "\xEF\xBB\xBF")))
	records, err := reader.ReadAll()
	require.NoError(t, err)
	assert.Equal(t, []string{"1", ""}, records[1])
}

func TestExportCSV_EmptyResult(t *testing.T) {
	resp := &domainquery.ExecuteResponse{
		Columns: []domainquery.ColumnInfo{
			{Name: "id", Type: "int4"},
			{Name: "name", Type: "text"},
		},
		Data: []map[string]interface{}{},
	}

	var buf bytes.Buffer
	err := Export(&buf, FormatCSV, resp)
	require.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.HasPrefix(output, "\xEF\xBB\xBF"))
}

func TestExportJSON_StreamsArray(t *testing.T) {
	resp := &domainquery.ExecuteResponse{
		Columns: []domainquery.ColumnInfo{
			{Name: "id", Type: "int4"},
			{Name: "name", Type: "text"},
		},
		Data: []map[string]interface{}{
			{"id": float64(1), "name": "Alice"},
			{"id": float64(2), "name": "Bob"},
		},
	}

	var buf bytes.Buffer
	err := Export(&buf, FormatJSON, resp)
	require.NoError(t, err)

	var results []map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &results)
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "Alice", results[0]["name"])
	assert.Equal(t, "Bob", results[1]["name"])
}

func TestExportJSON_EmptyResult(t *testing.T) {
	resp := &domainquery.ExecuteResponse{
		Columns: []domainquery.ColumnInfo{
			{Name: "id", Type: "int4"},
		},
		Data: []map[string]interface{}{},
	}

	var buf bytes.Buffer
	err := Export(&buf, FormatJSON, resp)
	require.NoError(t, err)

	var results []map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &results)
	require.NoError(t, err)
	assert.Len(t, results, 0)
}

func TestExportXLSX_CreatesFile(t *testing.T) {
	resp := &domainquery.ExecuteResponse{
		Columns: []domainquery.ColumnInfo{
			{Name: "id", Type: "int4"},
			{Name: "name", Type: "text"},
			{Name: "score", Type: "float8"},
		},
		Data: []map[string]interface{}{
			{"id": float64(1), "name": "Alice", "score": float64(95.5)},
		},
	}

	var buf bytes.Buffer
	err := Export(&buf, FormatXLSX, resp)
	require.NoError(t, err)

	assert.Greater(t, buf.Len(), 0, "XLSX output must not be empty")

	reader := bytes.NewReader(buf.Bytes())
	f, err := excelize.OpenReader(reader)
	require.NoError(t, err)
	defer f.Close()

	cell, err := f.GetCellValue("Sheet1", "A1")
	require.NoError(t, err)
	assert.Equal(t, "id", cell)
}

func TestExport_UnsupportedFormat(t *testing.T) {
	resp := &domainquery.ExecuteResponse{
		Columns: []domainquery.ColumnInfo{{Name: "id"}},
		Data:    []map[string]interface{}{},
	}

	var buf bytes.Buffer
	err := Export(&buf, "pdf", resp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported format")
}

func TestIsValidFormat(t *testing.T) {
	assert.True(t, IsValidFormat("csv"))
	assert.True(t, IsValidFormat("json"))
	assert.True(t, IsValidFormat("xlsx"))
	assert.False(t, IsValidFormat("pdf"))
	assert.False(t, IsValidFormat(""))
}

func TestFormatValue(t *testing.T) {
	assert.Equal(t, "", formatValue(nil))
	assert.Equal(t, "hello", formatValue("hello"))
	assert.Equal(t, "42", formatValue(float64(42)))
	assert.Equal(t, "true", formatValue(true))
}

func TestNormalizeData_MapSlice(t *testing.T) {
	data := []map[string]interface{}{
		{"id": float64(1)},
		{"id": float64(2)},
	}
	result := normalizeData(data)
	assert.Len(t, result, 2)
	assert.Equal(t, float64(1), result[0]["id"])
}

func TestNormalizeData_InterfaceSlice(t *testing.T) {
	data := []interface{}{
		map[string]interface{}{"id": float64(1)},
		map[string]interface{}{"id": float64(2)},
	}
	result := normalizeData(data)
	assert.Len(t, result, 2)
	assert.Equal(t, float64(1), result[0]["id"])
}

func TestCellName(t *testing.T) {
	assert.Equal(t, "A1", cellName(0, 1))
	assert.Equal(t, "B2", cellName(1, 2))
	assert.Equal(t, "Z1", cellName(25, 1))
	assert.Equal(t, "AA1", cellName(26, 1))
}
