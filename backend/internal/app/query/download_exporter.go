package query

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	domainquery "superset/auth-service/internal/domain/query"

	"github.com/xuri/excelize/v2"
)

const (
	FormatCSV  = "csv"
	FormatJSON = "json"
	FormatXLSX = "xlsx"
)

var supportedFormats = map[string]bool{
	FormatCSV:  true,
	FormatJSON: true,
	FormatXLSX: true,
}

func IsValidFormat(format string) bool {
	return supportedFormats[format]
}

// Export streams query results to w in the requested format.
func Export(w io.Writer, format string, resp *domainquery.ExecuteResponse) error {
	columns := resp.Columns
	rows := normalizeData(resp.Data)

	switch format {
	case FormatCSV:
		return exportCSV(w, columns, rows)
	case FormatJSON:
		return exportJSON(w, columns, rows)
	case FormatXLSX:
		return exportXLSX(w, columns, rows)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// normalizeData converts the Data field to []map[string]interface{}.
func normalizeData(data interface{}) []map[string]interface{} {
	switch d := data.(type) {
	case []map[string]interface{}:
		return d
	case []interface{}:
		result := make([]map[string]interface{}, 0, len(d))
		for _, item := range d {
			if m, ok := item.(map[string]interface{}); ok {
				result = append(result, m)
			}
		}
		return result
	default:
		return nil
	}
}

func exportCSV(w io.Writer, columns []domainquery.ColumnInfo, rows []map[string]interface{}) error {
	if _, err := w.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return fmt.Errorf("write BOM: %w", err)
	}

	writer := csv.NewWriter(w)
	defer writer.Flush()

	headers := make([]string, len(columns))
	for i, col := range columns {
		headers[i] = col.Name
	}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("write CSV headers: %w", err)
	}

	for _, row := range rows {
		record := make([]string, len(columns))
		for i, col := range columns {
			record[i] = formatValue(row[col.Name])
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("write CSV row: %w", err)
		}
	}
	return nil
}

func exportJSON(w io.Writer, columns []domainquery.ColumnInfo, rows []map[string]interface{}) error {
	encoder := json.NewEncoder(w)
	if _, err := w.Write([]byte("[")); err != nil {
		return err
	}
	for i, row := range rows {
		if i > 0 {
			if _, err := w.Write([]byte(",")); err != nil {
				return err
			}
		}
		if err := encoder.Encode(row); err != nil {
			return fmt.Errorf("write JSON row: %w", err)
		}
	}
	if _, err := w.Write([]byte("]")); err != nil {
		return err
	}
	return nil
}

func exportXLSX(w io.Writer, columns []domainquery.ColumnInfo, rows []map[string]interface{}) error {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Sheet1"

	boldStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
	})
	if err != nil {
		return fmt.Errorf("create bold style: %w", err)
	}

	for i, col := range columns {
		cell := cellName(i, 1)
		if err := f.SetCellValue(sheet, cell, col.Name); err != nil {
			return fmt.Errorf("set header %s: %w", col.Name, err)
		}
		if err := f.SetCellStyle(sheet, cell, cell, boldStyle); err != nil {
			return fmt.Errorf("set header style %s: %w", col.Name, err)
		}
	}

	for ri, row := range rows {
		for ci, col := range columns {
			cell := cellName(ci, ri+2)
			val := row[col.Name]
			if err := f.SetCellValue(sheet, cell, typedValue(val, col.Type)); err != nil {
				return fmt.Errorf("set cell %s: %w", cell, err)
			}
		}
	}

	if err := f.Write(w); err != nil {
		return fmt.Errorf("write XLSX: %w", err)
	}
	return nil
}

// typedValue converts a value to its typed Go representation based on column type.
func typedValue(val interface{}, colType string) interface{} {
	if val == nil {
		return ""
	}
	switch colType {
	case "int4", "int8", "int2", "integer", "bigint", "smallint", "numeric", "decimal", "float4", "float8", "real", "double precision":
		s := formatValue(val)
		if n, err := strconv.ParseFloat(s, 64); err == nil {
			return n
		}
		return s
	case "bool", "boolean":
		s := formatValue(val)
		if b, err := strconv.ParseBool(s); err == nil {
			return b
		}
		return s
	default:
		return formatValue(val)
	}
}

func formatValue(val interface{}) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// cellName converts 0-indexed col, 1-indexed row to Excel cell reference (A1, B2, etc.).
func cellName(col, row int) string {
	name := ""
	for c := col; c >= 0; c = c/26 - 1 {
		name = string(rune('A'+c%26)) + name
	}
	return name + strconv.Itoa(row)
}
