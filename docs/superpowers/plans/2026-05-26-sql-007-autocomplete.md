# SQL-007: SQL Autocomplete Hints — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Register a Monaco Editor autocomplete provider in SQL Lab that returns ranked SQL suggestions (keywords, schemas, tables, columns, functions) via `POST /api/v1/sqllab/autocomplete`.

**Architecture:** Backend handler reads schema from existing Redis-cached `DatabaseService` methods (ListSchemas/ListTables/ListColumns), merges with static keyword/function lists, fuzzy-filters by prefix or levenshtein ≤2, context-boosts tables after FROM/JOIN, and returns top 20. Frontend registers a Monaco `CompletionItemProvider("sql")` that calls the endpoint with current word + prefix + tab context.


**Tech Stack:** Go 1.25 (Gin, GORM), TypeScript/React 18 (Monaco Editor `@monaco-editor/react` v4.7, Zustand store)

---

### Task 1: Add Autocomplete types to backend domain

**Files:**
- Modify: `backend/internal/domain/query/sqllab_types.go` (append after line 122)

- [ ] **Step 1: Append types to sqllab_types.go**

Append after the `GetSchemaResponse` struct (end of file):

```go
// ── Autocomplete (SQL-007) ──

// AutocompleteRequest is the body for POST /api/v1/sqllab/autocomplete.
type AutocompleteRequest struct {
	Word   string `json:"word" binding:"required"`
	Prefix string `json:"prefix"`
	DbID   uint   `json:"db_id"`
	Schema string `json:"schema"`
}

// AutocompleteSuggestion is a single suggestion item.
type AutocompleteSuggestion struct {
	Text   string `json:"text"`
	Type   string `json:"type"`   // "keyword"|"schema"|"table"|"column"|"function"
	Score  int    `json:"score"`
	Detail string `json:"detail"`
}

// AutocompleteResponse is the API response for autocomplete.
type AutocompleteResponse struct {
	Suggestions []AutocompleteSuggestion `json:"suggestions"`
	CacheMiss   bool                     `json:"cache_miss"`
}
```

- [ ] **Step 2: Verify compilation**

```bash
cd backend && go build ./...
```

Expected: compiles successfully.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/domain/query/sqllab_types.go
git commit -m "feat(sql-007): add AutocompleteRequest, AutocompleteSuggestion, AutocompleteResponse types"
```

---

### Task 2: Add autocomplete handler logic

**Files:**
- Modify: `backend/internal/delivery/http/sqllab/handler.go` (append new method at end of file)
- Create: `backend/internal/delivery/http/sqllab/autocomplete.go`

- [ ] **Step 1: Create autocomplete.go with static data, levenshtein, and handler method**

Create `backend/internal/delivery/http/sqllab/autocomplete.go`:

```go
package sqllab

import (
	"fmt"
	"net/http"
	"strings"
	"unicode"

	domdb "superset/auth-service/internal/domain/db"
	domainquery "superset/auth-service/internal/domain/query"

	"github.com/gin-gonic/gin"
)

var sqlKeywords = []string{
	"SELECT", "FROM", "WHERE", "JOIN", "LEFT", "RIGHT", "INNER", "OUTER", "CROSS",
	"ON", "AND", "OR", "NOT", "IN", "EXISTS", "BETWEEN", "LIKE", "IS", "NULL", "AS",
	"GROUP", "BY", "ORDER", "ASC", "DESC", "HAVING", "UNION", "ALL", "INSERT", "INTO",
	"VALUES", "UPDATE", "SET", "DELETE", "CREATE", "ALTER", "TABLE", "DROP", "INDEX",
	"VIEW", "TRIGGER", "PROCEDURE", "FUNCTION", "SCHEMA", "DATABASE", "DISTINCT",
	"COUNT", "LIMIT", "OFFSET", "FETCH", "CASE", "WHEN", "THEN", "ELSE", "END",
	"CAST", "COALESCE", "NULLIF", "PRIMARY", "KEY", "FOREIGN", "REFERENCES",
	"CONSTRAINT", "UNIQUE", "CHECK", "DEFAULT", "CASCADE", "TRUNCATE", "BEGIN",
	"COMMIT", "ROLLBACK", "WITH", "RECURSIVE", "OVER", "PARTITION", "WINDOW", "ROWS",
	"RANGE", "INTERSECT", "EXCEPT", "NATURAL", "USING", "LATERAL", "IF", "RETURNING",
	"EXPLAIN", "ANALYZE", "GRANT", "REVOKE", "TEMPORARY", "TEMP", "MATERIALIZED",
	"REPLACE", "TOP", "FIRST", "NEXT", "ONLY", "PERCENT", "CUBE", "ROLLUP",
	"GROUPING", "SETS",
}

var sqlFunctions = []string{
	"COUNT", "SUM", "AVG", "MIN", "MAX", "COALESCE", "NULLIF", "CAST", "CONVERT",
	"CONCAT", "SUBSTRING", "SUBSTR", "UPPER", "LOWER", "TRIM", "LTRIM", "RTRIM",
	"LENGTH", "CHAR_LENGTH", "REPLACE", "NOW", "CURRENT_DATE", "CURRENT_TIME",
	"CURRENT_TIMESTAMP", "DATE_TRUNC", "EXTRACT", "DATEADD", "DATEDIFF", "ABS",
	"ROUND", "CEIL", "CEILING", "FLOOR", "MOD", "RAND", "RANDOM", "ROW_NUMBER",
	"RANK", "DENSE_RANK", "LAG", "LEAD", "FIRST_VALUE", "LAST_VALUE",
}

// levenshtein computes the edit distance between a and b.
func levenshtein(a, b string) int {
	ar, br := []rune(a), []rune(b)
	n, m := len(ar), len(br)
	if n == 0 {
		return m
	}
	if m == 0 {
		return n
	}

	prev := make([]int, m+1)
	cur := make([]int, m+1)
	for j := 0; j <= m; j++ {
		prev[j] = j
	}

	for i := 1; i <= n; i++ {
		cur[0] = i
		for j := 1; j <= m; j++ {
			cost := 1
			if ar[i-1] == br[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[m]
}

// matchesWord checks if candidate matches the given word via prefix or levenshtein≤2.
func matchesWord(candidate, word string) bool {
	if word == "" {
		return false
	}
	cl := strings.ToLower(candidate)
	wl := strings.ToLower(word)
	if strings.HasPrefix(cl, wl) {
		return true
	}
	return levenshtein(cl, wl) <= 2
}

// hasFromJoinContext checks if the prefix SQL text contains FROM or JOIN near the end.
func hasFromJoinContext(prefix string) bool {
	upper := strings.ToUpper(prefix)
	// Check last 50 chars to avoid false positives deep in the SQL
	tail := upper
	if len(tail) > 50 {
		tail = tail[len(tail)-50:]
	}
	return strings.Contains(tail, "FROM") || strings.Contains(tail, "JOIN")
}

func (h *Handler) Autocomplete(c *gin.Context) {
	_, ok := getUserContext(c)
	if !ok {
		return
	}

	var req domainquery.AutocompleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return
	}

	word := strings.TrimSpace(req.Word)
	if word == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "word is required"})
		return
	}

	suggestions := make([]domainquery.AutocompleteSuggestion, 0, 200)

	// Always add keywords and functions
	for _, kw := range sqlKeywords {
		if matchesWord(kw, word) {
			suggestions = append(suggestions, domainquery.AutocompleteSuggestion{
				Text: kw, Type: "keyword", Score: 300, Detail: "keyword",
			})
		}
	}
	for _, fn := range sqlFunctions {
		if matchesWord(fn, word) {
			suggestions = append(suggestions, domainquery.AutocompleteSuggestion{
				Text: fn, Type: "function", Score: 150, Detail: "function",
			})
		}
	}

	cacheMiss := false

	// Schema-dependent suggestions: only when db_id and schema are set
	if req.DbID > 0 && req.Schema != "" {
		ctx := c.Request.Context()

		// Read schemas from cache
		schemas, err := h.dbSvc.ListSchemas(ctx, 0, req.DbID, false, "")
		if err != nil {
			// Cache miss or db unreachable — keywords+functions only
			cacheMiss = true
		} else {
			for _, s := range schemas {
				if matchesWord(s, word) {
					suggestions = append(suggestions, domainquery.AutocompleteSuggestion{
						Text: s, Type: "schema", Score: 250, Detail: "schema",
					})
				}
			}

			// Read tables from cache
			tableReq := domdb.ListDatabaseTablesRequest{Schema: req.Schema, Page: 1, PageSize: 500}
			tablesResp, err := h.dbSvc.ListTables(ctx, 0, req.DbID, tableReq, false, "")
			if err != nil {
				cacheMiss = true
			} else {
				inFromJoin := hasFromJoinContext(req.Prefix)
				for _, t := range tablesResp.Items {
					if matchesWord(t.Name, word) {
						score := 200
						if inFromJoin {
							score += 100
						}
						suggestions = append(suggestions, domainquery.AutocompleteSuggestion{
							Text: t.Name, Type: "table", Score: score, Detail: fmt.Sprintf("table in %s", req.Schema),
						})
					}
				}

				// Read columns from cache (limit to first 500 tables to avoid explosion)
				tableLimit := min(len(tablesResp.Items), 20)
				for _, t := range tablesResp.Items[:tableLimit] {
					colReq := domdb.ListDatabaseColumnsRequest{Schema: req.Schema, Table: t.Name}
					columns, err := h.dbSvc.ListColumns(ctx, 0, req.DbID, colReq, false, "")
					if err != nil {
						break
					}
					for _, col := range columns {
						if matchesWord(col.Name, word) {
							suggestions = append(suggestions, domainquery.AutocompleteSuggestion{
								Text: col.Name, Type: "column", Score: 100, Detail: fmt.Sprintf("column in %s.%s", req.Schema, t.Name),
							})
						}
					}
				}
			}
		}
	}

	// Sort by score descending, stable
	sortByScore(suggestions)

	// Top 20
	if len(suggestions) > 20 {
		suggestions = suggestions[:20]
	}

	c.JSON(http.StatusOK, domainquery.AutocompleteResponse{
		Suggestions: suggestions,
		CacheMiss:   cacheMiss,
	})
}

func sortByScore(s []domainquery.AutocompleteSuggestion) {
	for i := 0; i < len(s); i++ {
		best := i
		for j := i + 1; j < len(s); j++ {
			if s[j].Score > s[best].Score {
				best = j
			}
		}
		s[i], s[best] = s[best], s[i]
	}
}
```

Note: The `min` function is already used elsewhere in the handler file (the `autocomplete.go` file needs it imported or we rely on Go 1.25 builtin). Go 1.21+ has builtin `min`. Also remove `unicode` import if unused — check after save.

Let's fix the import: remove `unicode` since we don't need it:

Actually, let me rewrite without the `unicode` import:

```go
package sqllab

import (
	"fmt"
	"net/http"
	"strings"

	domdb "superset/auth-service/internal/domain/db"
	domainquery "superset/auth-service/internal/domain/query"

	"github.com/gin-gonic/gin"
)
```

(The `unicode` was erroneously included — removed.)

- [ ] **Step 2: Verify compilation**

```bash
cd backend && go build ./...
```

Expected: compiles successfully.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/delivery/http/sqllab/autocomplete.go
git commit -m "feat(sql-007): add Autocomplete handler with fuzzy matching and context boosting"
```

---

### Task 3: Add autocomplete route

**Files:**
- Modify: `backend/internal/delivery/http/router.go:189` (after ClearSchema line)

- [ ] **Step 1: Add route**

In `router.go`, after `sqlLab.DELETE("/tabs/:id/schema", sqllabHandler.ClearSchema)` (line 188), add:

```go
sqlLab.POST("/autocomplete", sqllabHandler.Autocomplete)
```

Full context — the `sqlLab` group block ends like:

```go
				sqlLab := protected.Group("/sqllab")
				{
					sqlLab.POST("/tabs", sqllabHandler.CreateTab)
					sqlLab.GET("/tabs", sqllabHandler.ListTabs)
					sqlLab.GET("/tabs/:id", sqllabHandler.GetTab)
					sqlLab.PUT("/tabs/:id", sqllabHandler.UpdateTab)
					sqlLab.PUT("/tabs/:id/close", sqllabHandler.CloseTab)
					sqlLab.DELETE("/tabs", sqllabHandler.CloseAllTabs)
					sqlLab.DELETE("/tabs/:id", sqllabHandler.HardDeleteTab)
					sqlLab.POST("/saved-queries", sqllabHandler.CreateSavedQuery)
					sqlLab.GET("/saved-queries", sqllabHandler.ListSavedQueries)
					sqlLab.PUT("/saved-queries/:id", sqllabHandler.UpdateSavedQuery)
					sqlLab.DELETE("/saved-queries/:id", sqllabHandler.DeleteSavedQuery)
					sqlLab.POST("/saved-queries/:id/fork", sqllabHandler.ForkSavedQuery)
					sqlLab.GET("/tabs/:id/schema", sqllabHandler.GetSchema)
					sqlLab.POST("/tabs/:id/schema", sqllabHandler.ExpandTable)
					sqlLab.DELETE("/tabs/:id/schema/:table", sqllabHandler.CollapseTable)
					sqlLab.DELETE("/tabs/:id/schema", sqllabHandler.ClearSchema)
					sqlLab.POST("/autocomplete", sqllabHandler.Autocomplete)
				}
```

- [ ] **Step 2: Verify compilation**

```bash
cd backend && go build ./...
```

Expected: compiles successfully.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/delivery/http/router.go
git commit -m "feat(sql-007): register POST /api/v1/sqllab/autocomplete route"
```

---

### Task 4: Backend autocomplete tests

**Files:**
- Modify: `backend/internal/delivery/http/sqllab/handler_test.go` (append tests at end of file)

- [ ] **Step 1: Add autocomplete test cases**

Append to `handler_test.go` after the last test (line 768):

```go
// ── Autocomplete tests ──

func TestAutocomplete_KeywordsOnly_NoDbID(t *testing.T) {
	repo := &mockSQLLabRepo{tabs: map[uint]*domainquery.TabState{}}
	router := newSQLLabRouter(repo)

	body := `{"word":"sel"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/sqllab/autocomplete", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"cache_miss":false`)) {
		t.Fatalf("expected cache_miss:false, got %s", w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"SELECT"`)) {
		t.Fatalf("expected SELECT keyword suggestion, got %s", w.Body.String())
	}
}

func TestAutocomplete_EmptyWord_Returns400(t *testing.T) {
	repo := &mockSQLLabRepo{tabs: map[uint]*domainquery.TabState{}}
	router := newSQLLabRouter(repo)

	body := `{"word":""}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/sqllab/autocomplete", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAutocomplete_Top20_Ensured(t *testing.T) {
	repo := &mockSQLLabRepo{tabs: map[uint]*domainquery.TabState{}}
	router := newSQLLabRouter(repo)

	// Empty word "s" matches many keywords
	body := `{"word":"s"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/sqllab/autocomplete", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Count suggestions in response — should be ≤20
	var resp domainquery.AutocompleteResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Suggestions) > 20 {
		t.Fatalf("expected at most 20 suggestions, got %d", len(resp.Suggestions))
	}
}

func TestAutocomplete_KeywordsOnly_WhenMissingBody(t *testing.T) {
	repo := &mockSQLLabRepo{tabs: map[uint]*domainquery.TabState{}}
	router := newSQLLabRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/sqllab/autocomplete", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAutocomplete_KeywordsReturned_WhenNoDb(t *testing.T) {
	repo := &mockSQLLabRepo{tabs: map[uint]*domainquery.TabState{}}
	router := newSQLLabRouter(repo)

	body := `{"word":"ins","db_id":0,"schema":""}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/sqllab/autocomplete", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	// "ins" should prefix-match INSERT, JOIN (via levenshtein ≤2 or prefix)
	if !bytes.Contains(w.Body.Bytes(), []byte(`"INSERT"`)) {
		t.Fatalf("expected INSERT suggestion, got %s", w.Body.String())
	}
}

func TestAutocomplete_Unauthorized_Returns401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(&mockSQLLabRepo{tabs: map[uint]*domainquery.TabState{}}, &mockDatabaseRepo{}, nil)
	r := gin.New()
	// No user middleware — unauthenticated
	r.POST("/api/v1/sqllab/autocomplete", h.Autocomplete)

	body := `{"word":"sel"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/sqllab/autocomplete", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}
```

- [ ] **Step 2: Add route to test router**

In `newSQLLabRouter` (line 229), add the autocomplete route after existing routes:

```go
	sqllab.POST("/autocomplete", h.Autocomplete)
```

Add `encoding/json` to the test file imports if not already present.

- [ ] **Step 3: Run tests**

```bash
cd backend && go test ./internal/delivery/http/sqllab/... -v -run "TestAutocomplete"
```

Expected: all 6 autocomplete tests PASS.

- [ ] **Step 4: Run all existing tests to check for regressions**

```bash
cd backend && go test ./internal/delivery/http/sqllab/... -v
```

Expected: all existing tests still PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/delivery/http/sqllab/handler_test.go
git commit -m "test(sql-007): add autocomplete handler unit tests"
```

---

### Task 5: Add frontend API types and fetch function

**Files:**
- Modify: `frontend/src/api/sqllab.ts` (append after line 251)

- [ ] **Step 1: Append autocomplete types and function**

After the last function `clearSchemaState` (line 251), add:

```typescript
// ── Autocomplete (SQL-007) ──

export interface AutocompleteRequest {
  word: string;
  prefix: string;
  db_id?: number;
  schema?: string;
}

export interface AutocompleteSuggestion {
  text: string;
  type: "keyword" | "schema" | "table" | "column" | "function";
  score: number;
  detail: string;
}

export interface AutocompleteResponse {
  suggestions: AutocompleteSuggestion[];
  cache_miss: boolean;
}

export async function fetchAutocomplete(data: AutocompleteRequest): Promise<AutocompleteResponse> {
  return request<AutocompleteResponse>("/api/v1/sqllab/autocomplete", {
    method: "POST",
    headers: getAuthHeaders(),
    body: JSON.stringify(data),
  });
}
```

- [ ] **Step 2: TypeScript check**

```bash
cd frontend && npx tsc --noEmit --pretty 2>&1 | Select-Object -First 30
```

Expected: no new type errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/api/sqllab.ts
git commit -m "feat(sql-007): add fetchAutocomplete API function and types"
```

---

### Task 6: Register Monaco autocomplete provider and cache_miss alert

**Files:**
- Modify: `frontend/src/pages/sqllab/SQLLabPage.tsx`

- [ ] **Step 1: Add import for fetchAutocomplete**

After the existing sqllab API import on line 70:
```typescript
import { fetchTabs, createTab as createTabApi, closeTab, closeAllTabs } from "@/api/sqllab";
```

Change to:
```typescript
import { fetchTabs, createTab as createTabApi, closeTab, closeAllTabs, fetchAutocomplete } from "@/api/sqllab";
```

- [ ] **Step 2: Add autocompleteCacheMiss state**

After `const [resultsTabValue, setResultsTabValue] = useState("results");` (search for it near the component state declarations), add:

```typescript
  const [autocompleteCacheMiss, setAutocompleteCacheMiss] = useState(false);
```

If `resultsTabValue` doesn't exist at top level, find the component state area (nearer to line 115-140). Add the state after the existing `useState` declarations in the component body.

Actually — find the right spot. The `resultsTabValue` state is likely near line 682 in the JSX. Let's place the state before the `handleEditorMount` callback (around line 637). Add it right before `handleEditorMount`:

```typescript
  const [autocompleteCacheMiss, setAutocompleteCacheMiss] = useState(false);
```

- [ ] **Step 3: Add Monaco completion provider inside handleEditorMount**

Modify `handleEditorMount` (currently lines 637-650) to register the completion provider after setting up the editor action:

```typescript
  const handleEditorMount: OnMount = (editor, monaco) => {
    editorRef.current = editor;
    editor.addAction({
      id: "run-query",
      label: "Run Query",
      keybindings: [2048 | 3],
      run: () => {
        const selection = editor.getSelection();
        const selectedText = selection ? editor.getModel()?.getValueInRange(selection)?.trim() : null;
        const sql = selectedText || activeTab?.sql || "";
        handleRun(sql);
      },
    });

    // Register SQL autocomplete provider
    const provider = monaco.languages.registerCompletionItemProvider("sql", {
      provideCompletionItems: async (model, position) => {
        const word = model.getWordUntilPosition(position);
        if (!word || !word.word || word.word.length < 1) {
          return { suggestions: [] };
        }

        const prefix = model.getValueInRange({
          startLineNumber: 1,
          startColumn: 1,
          endLineNumber: position.lineNumber,
          endColumn: position.column,
        });

        try {
          const tab = tabs.find(t => t.id === activeTabId);
          const res = await fetchAutocomplete({
            word: word.word,
            prefix,
            db_id: tab?.databaseId ?? undefined,
            schema: tab?.schema ?? undefined,
          });

          setAutocompleteCacheMiss(res.cache_miss);

          const kindMap: Record<string, number> = {
            keyword: monaco.languages.CompletionItemKind.Keyword,
            schema: monaco.languages.CompletionItemKind.Module,
            table: monaco.languages.CompletionItemKind.Class,
            column: monaco.languages.CompletionItemKind.Field,
            function: monaco.languages.CompletionItemKind.Function,
          };

          return {
            suggestions: res.suggestions.map(s => ({
              label: s.text,
              kind: kindMap[s.type] ?? monaco.languages.CompletionItemKind.Text,
              detail: s.detail,
              sortText: String(99999 - s.score).padStart(5, "0"),
              insertText: s.text,
            })),
          };
        } catch {
          return { suggestions: [] };
        }
      },
    });

    // Store disposable for cleanup (optional — provider lives for editor lifetime)
    editorRef.current = editor;
  };
```

Wait — there's a problem. `editorRef.current` is already set earlier in the function. And the `onMount` callback signature changed from `(editor) =>` to `(editor, monaco) =>`. Let me check the `@monaco-editor/react` API.

The `OnMount` type is `export type OnMount = (editor: editor.IStandaloneCodeEditor, monaco: Monaco) => void;` — so we need to destructure `monaco` as the second parameter.

Update the type and function signature:

```typescript
  const handleEditorMount: OnMount = (editor, monaco) => {
    editorRef.current = editor;

    // ... existing run-query action (unchanged) ...

    // Register SQL autocomplete provider
    monaco.languages.registerCompletionItemProvider("sql", {
      provideCompletionItems: async (model, position) => {
        const word = model.getWordUntilPosition(position);
        if (!word || !word.word || word.word.length < 1) {
          return { suggestions: [] };
        }

        const prefix = model.getValueInRange({
          startLineNumber: 1,
          startColumn: 1,
          endLineNumber: position.lineNumber,
          endColumn: position.column,
        });

        try {
          const tab = tabs.find(t => t.id === activeTabId);
          const res = await fetchAutocomplete({
            word: word.word,
            prefix,
            db_id: tab?.databaseId ?? undefined,
            schema: tab?.schema ?? undefined,
          });

          setAutocompleteCacheMiss(res.cache_miss);

          const kindMap: Record<string, number> = {
            keyword: monaco.languages.CompletionItemKind.Keyword,
            schema: monaco.languages.CompletionItemKind.Module,
            table: monaco.languages.CompletionItemKind.Class,
            column: monaco.languages.CompletionItemKind.Field,
            function: monaco.languages.CompletionItemKind.Function,
          };

          return {
            suggestions: res.suggestions.map(s => ({
              label: s.text,
              kind: kindMap[s.type] ?? monaco.languages.CompletionItemKind.Text,
              detail: s.detail,
              sortText: String(99999 - s.score).padStart(5, "0"),
              insertText: s.text,
            })),
          };
        } catch {
          return { suggestions: [] };
        }
      },
    });
  };
```

Note: The existing `editorRef.current = editor;` line at the top of handleEditorMount must be kept. The `monaco` parameter is new — the `OnMount` type should already accept 2 params, so this is backward-compatible.

- [ ] **Step 4: Add cache_miss Alert below the editor**

After the Editor div closing `</div>` (after line 1124, after `</div>` that closes `<div className="border rounded-md overflow-hidden flex-1">`), add:

```typescript
                    {autocompleteCacheMiss && (
                      <Alert variant="default" className="mt-2">
                        <AlertDescription>
                          Schema loading — full autocomplete will be available shortly.
                        </AlertDescription>
                      </Alert>
                    )}
```

- [ ] **Step 5: TypeScript check**

```bash
cd frontend && npx tsc --noEmit --pretty 2>&1 | Select-Object -First 30
```

Expected: no new type errors. The `monaco` parameter in `OnMount` should be compatible.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/pages/sqllab/SQLLabPage.tsx
git commit -m "feat(sql-007): register Monaco autocomplete provider with cache_miss alert"
```

---

### Task 7: End-to-end verification

- [ ] **Step 1: Start backend and test with curl**

```bash
curl -s -X POST http://localhost:8080/api/v1/sqllab/autocomplete \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <valid-token>" \
  -d '{"word":"sel"}'
```

Expected: 200 with `{"suggestions":[{"text":"SELECT","type":"keyword","score":300,...}], "cache_miss":false}`. Verify "SELECT" is the first suggestion.

- [ ] **Step 2: Test with db_id+no-schema → keywords+functions only**

```bash
curl -s -X POST http://localhost:8080/api/v1/sqllab/autocomplete \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <valid-token>" \
  -d '{"word":"sel","db_id":0,"schema":""}'
```

Expected: 200, `cache_miss:false`, only keywords and functions in suggestions.

- [ ] **Step 3: Open frontend, type "SEL" in SQL editor, verify autocomplete popup shows SELECT as top suggestion**
```

- [ ] **Step 4: Commit if config/auth changes needed**

No changes expected — only verification.

---

### Files Summary

| File | Action |
|------|--------|
| `backend/internal/domain/query/sqllab_types.go` | Append 3 types |
| `backend/internal/delivery/http/sqllab/autocomplete.go` | **Create** — handler, keywords, functions, levenshtein, matching, sorting |
| `backend/internal/delivery/http/router.go` | Add 1 route |
| `backend/internal/delivery/http/sqllab/handler_test.go` | Append 6 tests + add route to test router |
| `frontend/src/api/sqllab.ts` | Append types + fetchAutocomplete |
| `frontend/src/pages/sqllab/SQLLabPage.tsx` | Modify imports, add state, register provider, add alert |
