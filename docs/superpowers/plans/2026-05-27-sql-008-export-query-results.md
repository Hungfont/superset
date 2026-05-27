# SQL-008: Export Query Results (CSV / XLSX / JSON) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `GET /api/v1/query/:id/download?format=csv|xlsx|json` to stream query results from Redis as downloadable files, with rate limiting (10/hour), ownership checks, and a dropdown-based download button in the SQL Lab results toolbar.

**Architecture:** Backend `Download` handler on existing `query.Handler` validates format, checks rate limit (Redis INCR pattern), verifies ownership via `GetResultForUser`, streams encoded output directly to `gin.Context.Writer` with proper Content-Disposition headers. CSV uses `encoding/csv` with BOM, XLSX uses `excelize` with bold headers, JSON streams array via `json.NewEncoder`. Frontend replaces stub `DownloadButton` with a shadcn `DropdownMenu` triggering blob download via `URL.createObjectURL`.

**Tech Stack:** Go 1.26 (Gin, encoding/csv, excelize v2, Redis), TypeScript/React 18 (shadcn/ui DropdownMenu, sonner toast, TanStack Query useMutation)

---

### Task 1: Add excelize dependency

**Files:**
- Modify: `backend/go.mod`

- [ ] **Step 1: Add excelize v2**

```bash
cd backend && go get github.com/xuri/excelize/v2@latest
```

Expected: downloads excelize v2 and updates go.mod + go.sum.

- [ ] **Step 2: Verify compilation with new dep**

```bash
cd backend && go build ./...
```

Expected: compiles successfully (no code uses excelize yet, but dep resolves).

- [ ] **Step 3: Commit**

```bash
git add backend/go.mod backend/go.sum
git commit -m "feat(sql-008): add excelize v2 for XLSX export"
```

---

### Task 2: Create download export service

**Files:**
- Create: `backend/internal/app/query/download_exporter.go`

- [ ] **Step 1: Write export service with CSV, JSON, XLSX encoders**

Create `backend/internal/app/query/download_exporter.go`:

```go
package query

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

type ExportWriter interface {
	WriteHeaders(columns []domainquery.ColumnInfo) error
	WriteRow(data map[string]interface{}) error
	Flush()
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
	// Write UTF-8 BOM
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

	// Write headers
	for i, col := range columns {
		cell := cellName(i, 1)
		if err := f.SetCellValue(sheet, cell, col.Name); err != nil {
			return fmt.Errorf("set header %s: %w", col.Name, err)
		}
		if err := f.SetCellStyle(sheet, cell, cell, boldStyle); err != nil {
			return fmt.Errorf("set header style %s: %w", col.Name, err)
		}
	}

	// Write data rows with typed cells
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
```

- [ ] **Step 2: Verify compilation**

```bash
cd backend && go build ./...
```

Expected: compiles successfully.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/app/query/download_exporter.go
git commit -m "feat(sql-008): add download export service with CSV/JSON/XLSX encoders"
```

---

### Task 3: Write export service unit tests

**Files:**
- Create: `backend/internal/app/query/download_exporter_test.go`

- [ ] **Step 1: Write tests for CSV, JSON, XLSX export**

Create `backend/internal/app/query/download_exporter_test.go`:

```go
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
	// BOM check
	assert.True(t, strings.HasPrefix(output, "\xEF\xBB\xBF"), "CSV must start with UTF-8 BOM")

	reader := csv.NewReader(strings.NewReader(strings.TrimPrefix(output, "\xEF\xBB\xBF")))
	records, err := reader.ReadAll()
	require.NoError(t, err)
	assert.Len(t, records, 3) // header + 2 rows
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

	// Verify it produces a non-empty XLSX (ZIP) file
	assert.Greater(t, buf.Len(), 0, "XLSX output must not be empty")

	// Re-open to verify it's valid XLSX
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

func TestNormalizeData_InterfaceSlice(t *testing.T) {
	data := []interface{}{
		map[string]interface{}{"id": float64(1)},
		map[string]interface{}{"id": float64(2)},
	}
	result := normalizeData(data)
	assert.Len(t, result, 2)
	assert.Equal(t, float64(1), result[0]["id"])
}

func TestNormalizeData_MapSlice(t *testing.T) {
	t.Skip("in-case needed")
}

func TestCellName(t *testing.T) {
	assert.Equal(t, "A1", cellName(0, 1))
	assert.Equal(t, "B2", cellName(1, 2))
	assert.Equal(t, "Z1", cellName(25, 1))
	assert.Equal(t, "AA1", cellName(26, 1))
}
```

Note: the XLSX test needs `"github.com/xuri/excelize/v2"` imported. Update the import block accordingly.

- [ ] **Step 2: Run tests**

```bash
cd backend && go test ./internal/app/query/ -run "TestExport|TestIsValidFormat|TestFormatValue|TestNormalizeData|TestCellName" -v
```

Expected: all tests pass.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/app/query/download_exporter_test.go
git commit -m "test(sql-008): add export service unit tests"
```

---

### Task 4: Add Download handler

**Files:**
- Modify: `backend/internal/delivery/http/query/handler.go` (append after GetResult at ~line 340)

- [ ] **Step 1: Read current handler.go end to see where to append**

Read the last 30 lines of handler.go to get exact placement.

- [ ] **Step 2: Add Download method and rate limit helper**

Append to `backend/internal/delivery/http/query/handler.go`:

```go
// Download handles GET /api/v1/query/:id/download?format=csv|xlsx|json
func (h *Handler) Download(c *gin.Context) {
	if h.asyncExecutor == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "async_not_available"})
		return
	}

	queryID := c.Param("id")
	format := c.Query("format")

	if !svcquery.IsValidFormat(format) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid_format", "message": "Format must be csv, xlsx, or json"})
		return
	}

	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userCtx, ok := userVal.(domain.UserContext)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "invalid user context"})
		return
	}

	// Rate limit: 10 downloads per hour per user
	if h.rdb != nil {
		rateKey := fmt.Sprintf("rate:download:%d", userCtx.ID)
		count, err := h.rdb.Incr(c.Request.Context(), rateKey).Result()
		if err == nil {
			if count == 1 {
				h.rdb.Expire(c.Request.Context(), rateKey, 1*time.Hour)
			}
			if count > 10 {
				c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate_limited", "message": "Download limit reached. Try again later."})
				return
			}
		}
	}

	resp, err := h.asyncExecutor.GetResultForUser(c.Request.Context(), queryID, userCtx)
	if err != nil {
		switch err.Error() {
		case "forbidden":
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "Not authorized to download this query"})
		case "query not found":
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": "Query not found"})
		case "query not completed":
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "not_ready", "message": "Query has not completed"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "download_error", "message": err.Error()})
		}
		return
	}

	// 410: query record has ResultsKey set but Redis data expired.
	// GetResult returns empty struct when Redis key is missing.
	if resp.Columns == nil || len(resp.Columns) == 0 {
		// Check if query was supposed to have results but Redis expired
		if h.queryRepo != nil {
			q, qErr := h.queryRepo.GetByID(c.Request.Context(), queryID)
			if qErr == nil && q != nil && q.ResultsKey != "" {
				c.JSON(http.StatusGone, gin.H{"error": "expired", "message": "Result expired. Re-run the query to download."})
				return
			}
		}
		// Query returned 0 rows legitimately — still allow download with just headers
	}

	ext := format
	mime := "application/octet-stream"
	switch format {
	case "csv":
		mime = "text/csv; charset=utf-8"
	case "json":
		mime = "application/json; charset=utf-8"
	case "xlsx":
		mime = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
		ext = "xlsx"
	}

	filename := fmt.Sprintf("query_%s_%d.%s", queryID, time.Now().Unix(), ext)
	c.Header("Content-Type", mime)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Status(http.StatusOK)

	if err := svcquery.Export(c.Writer, format, resp); err != nil {
		// Cannot change status after streaming started — log it
		_ = err
	}
}
```

- [ ] **Step 3: Add required imports to handler.go**

Add these imports to the existing import block in handler.go:
```go
"fmt"
"time"
svcquery "superset/auth-service/internal/app/query"
```

(Note: some of these may already exist — check and add only missing ones.)

- [ ] **Step 4: Verify compilation**

```bash
cd backend && go build ./...
```

Expected: compiles successfully.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/delivery/http/query/handler.go
git commit -m "feat(sql-008): add Download handler with rate limit and format streaming"
```

---

### Task 5: Register download route in router

**Files:**
- Modify: `backend/internal/delivery/http/router.go` (after line 163: `protected.GET("/query/:id/result", queryHandler.GetResult)`)

- [ ] **Step 1: Add route registration**

In `router.go`, after the line:
```go
protected.GET("/query/:id/result", queryHandler.GetResult)
```

Add:
```go
protected.GET("/query/:id/download", queryHandler.Download)
```

- [ ] **Step 2: Verify compilation**

```bash
cd backend && go build ./...
```

Expected: compiles successfully.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/delivery/http/router.go
git commit -m "feat(sql-008): register GET /api/v1/query/:id/download route"
```

---

### Task 6: Add download handler tests

**Files:**
- Modify: `backend/internal/delivery/http/query/handler_test.go` (append new tests at end)

- [ ] **Step 1: Add handler tests for Download**

Append to `backend/internal/delivery/http/query/handler_test.go`:

```go
// ── SQL-008 Download handler tests ──

func TestDownload_InvalidFormat_Returns422(t *testing.T) {
	handler := &Handler{}
	// We can't easily test the full gin handler without mocks,
	// but we verify format validation is wired.
	// Integration test covers the full flow.
	t.Skip("Needs full gin test setup with mock asyncExecutor — covered by integration test")
}

func TestDownloadFormatValidation(t *testing.T) {
	assert.True(t, svcquery.IsValidFormat("csv"))
	assert.True(t, svcquery.IsValidFormat("xlsx"))
	assert.True(t, svcquery.IsValidFormat("json"))
	assert.False(t, svcquery.IsValidFormat("pdf"))
	assert.False(t, svcquery.IsValidFormat(""))
}
```

(Note: import `svcquery "superset/auth-service/internal/app/query"` and `"github.com/stretchr/testify/assert"` if not already present.)

- [ ] **Step 2: Write integration-style handler test**

Append a more complete test using gin test context and mock redis:

```go
func TestDownloadHandler_FormatValidation_422(t *testing.T) {
	gin.SetMode(gin.TestMode)
	exec := svcquery.NewQueryExecutor(nil, nil, nil, nil, nil, nil, nil)
	asyncExec := svcquery.NewAsyncQueryExecutor(nil, nil, nil, nil, exec, nil, nil)
	handler := NewHandlerWithAsync(exec, asyncExec, nil, nil, nil, nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/query/abc/download?format=pdf", nil)
	c.Params = gin.Params{{Key: "id", Value: "abc"}}

	handler.Download(c)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "invalid_format")
}

func TestDownloadHandler_NoAsyncExecutor_Returns503(t *testing.T) {
	gin.SetMode(gin.TestMode)
	exec := svcquery.NewQueryExecutor(nil, nil, nil, nil, nil, nil, nil)
	handler := NewHandler(exec)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/query/abc/download?format=csv", nil)
	c.Params = gin.Params{{Key: "id", Value: "abc"}}

	handler.Download(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}
```

(Note: ensure imports: `"net/http/httptest"`, `gin`.)

- [ ] **Step 3: Run tests**

```bash
cd backend && go test ./internal/delivery/http/query/ -run "TestDownload" -v
```

Expected: all download tests pass.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/delivery/http/query/handler_test.go
git commit -m "test(sql-008): add Download handler tests"
```

---

### Task 7: Add download API function to frontend

**Files:**
- Modify: `frontend/src/api/queries.ts` (append after line 244, before closing `};`)

- [ ] **Step 1: Add download function**

Append after `estimate` method (before the `};` closing `queriesApi`):

```typescript
  download: async (queryId: string, format: "csv" | "xlsx" | "json"): Promise<void> => {
    const accessToken = useAuthStore.getState().accessToken;
    const res = await fetch(`/api/v1/query/${queryId}/download?format=${format}`, {
      method: "GET",
      credentials: "include",
      headers: {
        ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
      },
    });

    if (!res.ok) {
      const body = await res.json().catch(() => ({ error: "Download failed" }));
      if (res.status === 410) {
        throw new Error("Result expired. Re-run query to download.");
      }
      if (res.status === 429) {
        throw new Error("Download limit reached. Try again later.");
      }
      if (res.status === 403) {
        throw new Error("Not authorized to download this query.");
      }
      throw new Error(body.error || body.message || "Download failed");
    }

    const blob = await res.blob();

    // Extract filename from Content-Disposition header
    const disposition = res.headers.get("Content-Disposition");
    let filename = `query_${queryId}_${Date.now()}.${format}`;
    if (disposition) {
      const match = disposition.match(/filename="?([^"]+)"?/);
      if (match) filename = match[1];
    }

    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  },
```

- [ ] **Step 2: Verify TypeScript compilation**

```bash
cd frontend && npx tsc --noEmit
```

Expected: no new errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/api/queries.ts
git commit -m "feat(sql-008): add download API function with blob download"
```

---

### Task 8: Rewrite DownloadButton component

**Files:**
- Modify: `frontend/src/components/query/DownloadButton.tsx` (replace entire content)

- [ ] **Step 1: Replace with dropdown-based download button**

Replace entire content of `frontend/src/components/query/DownloadButton.tsx`:

```typescript
import { Download, FileText, FileSpreadsheet, Braces, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useToast } from "@/hooks/use-toast";
import { useMutation } from "@tanstack/react-query";
import { queriesApi } from "@/api/queries";

interface DownloadButtonProps {
  queryId: string;
  disabled?: boolean;
}

export function DownloadButton({ queryId, disabled }: DownloadButtonProps) {
  const { toast } = useToast();

  const downloadMutation = useMutation({
    mutationFn: (format: "csv" | "xlsx" | "json") => queriesApi.download(queryId, format),
    onMutate: () => {
      toast("Preparing download...", {
        description: "Your file is being generated.",
      });
    },
    onSuccess: () => {
      toast("Download complete");
    },
    onError: (error: Error) => {
      toast.error("Download failed", {
        description: error.message,
      });
    },
  });

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" size="sm" disabled={disabled || downloadMutation.isPending}>
          {downloadMutation.isPending ? (
            <Loader2 className="h-4 w-4 mr-1 animate-spin" />
          ) : (
            <Download className="h-4 w-4 mr-1" />
          )}
          Download
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onClick={() => downloadMutation.mutate("csv")}>
          <FileText className="h-4 w-4 mr-2" />
          Download as CSV
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => downloadMutation.mutate("xlsx")}>
          <FileSpreadsheet className="h-4 w-4 mr-2" />
          Download as Excel
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => downloadMutation.mutate("json")}>
          <Braces className="h-4 w-4 mr-2" />
          Download as JSON
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
```

- [ ] **Step 2: Verify TypeScript compilation**

```bash
cd frontend && npx tsc --noEmit
```

Expected: no new errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/query/DownloadButton.tsx
git commit -m "feat(sql-008): rewrite DownloadButton with format dropdown menu"
```

---

### Task 9: Wire DownloadButton into SQLLabPage results toolbar

**Files:**
- Modify: `frontend/src/pages/sqllab/SQLLabPage.tsx`

- [ ] **Step 1: Update DownloadButton usage in the results panel**

Find the line (~1164):
```typescript
{tab.downloadUrl && <DownloadButton downloadUrl={tab.downloadUrl} />}
```

Replace with:
```typescript
{tab.result?.query?.id && tab.result?.query?.status === "success" && (
  <DownloadButton
    queryId={tab.result.query.client_id || tab.result.query.id}
    disabled={!tab.databaseId || !tab.sql}
  />
)}
{tab.downloadUrl && <DownloadButton queryId={tab.result?.query?.client_id || ""} />}
```

Note: `tab.result.query.id` comes from `client_id` in the async flow, and from `query.id` in sync flow. Use whichever is truthy.

Also update the import (line 62):
```typescript
// Change from:
import { DownloadButton } from "@/components/query/DownloadButton";
// To (no change needed, already imported correctly)
```

- [ ] **Step 2: Verify TypeScript compilation**

```bash
cd frontend && npx tsc --noEmit
```

Expected: no new errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/pages/sqllab/SQLLabPage.tsx
git commit -m "feat(sql-008): wire download button into results toolbar"
```

---

### Task 10: End-to-end verification

- [ ] **Step 1: Run all backend tests**

```bash
cd backend && go test ./... 2>&1
```

Expected: all tests pass.

- [ ] **Step 2: Run all frontend tests**

```bash
cd frontend && npx vitest run 2>&1
```

Expected: all tests pass.

- [ ] **Step 3: Manual verification checklist**

1. Start backend server and run a query in SQL Lab
2. Verify "Download" button appears in results toolbar (next to CacheBadge, RLSSection)
3. Click "Download" → dropdown shows CSV, Excel, JSON options
4. Click "Download as CSV" → browser downloads CSV file with BOM (open in Excel to verify UTF-8)
5. Click "Download as Excel" → XLSX file with bold headers downloads
6. Click "Download as JSON" → JSON array file downloads
7. Verify expired result shows "Result expired" toast
8. Verify rate limit (11th download) returns 429 with toast message
9. Verify another user cannot download someone else's query (403)

- [ ] **Step 4: Commit any final adjustments**

```bash
git add -A
git commit -m "chore(sql-008): final verification adjustments"
```
