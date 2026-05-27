# SQL-007: SQL Autocomplete Hints — Design

**Date:** 2026-05-26  
**Status:** design  
**Depends on:** DBC-007 (schema cache in Redis), SQL-001 (tab provides DB/schema)

## Overview

Monaco Editor autocomplete provider in SQL Lab that returns ranked suggestions: SQL keywords, schemas, tables, columns, and DB functions. Fuzzy-matched by levenshtein ≤2 or prefix. Context-aware scoring (FROM/JOIN boosts tables).

## Architecture

```
Monaco Editor provideCompletionItems
  → POST /api/v1/sqllab/autocomplete {word, prefix, db_id?, schema?}
    → Backend Handler.Autocomplete
      → Redis schema cache (dbSvc)
      → Merge candidates: keywords, functions, schemas, tables, columns
      → Filter (prefix OR levenshtein ≤2)
      → Context boost (FROM/JOIN → tables+100)
      → Sort score DESC, top 20
    → Response: {suggestions: [...], cache_miss: bool}
  → Map suggestions → Monaco CompletionItem[] (kind + detail)
```

## Backend

### Route

`POST /api/v1/sqllab/autocomplete` — registered in `router.go` alongside existing sqllab routes.

### Types (sqllab_types.go)

```go
type AutocompleteRequest struct {
    Word   string `json:"word" binding:"required"`
    Prefix string `json:"prefix"`
    DbID   uint   `json:"db_id"`
    Schema string `json:"schema"`
}

type AutocompleteSuggestion struct {
    Text   string `json:"text"`
    Type   string `json:"type"`   // "keyword"|"schema"|"table"|"column"|"function"
    Score  int    `json:"score"`
    Detail string `json:"detail"` // e.g., "keyword", "table in public"
}

type AutocompleteResponse struct {
    Suggestions []AutocompleteSuggestion `json:"suggestions"`
    CacheMiss   bool                     `json:"cache_miss"`
}
```

### Handler (sqllab/handler.go)

New method `Handler.Autocomplete(c *gin.Context)`:

1. Bind `AutocompleteRequest`. Empty word → 400.
2. If `db_id == 0` or `schema == ""`: return keywords + functions only, `cache_miss: false`.
3. Read schemas, tables, columns from `dbSvc` (ListSchemas, ListTables, ListColumns — same cache-backed methods used by schema browser).
   - Redis miss for any of these → return keywords + functions only, `cache_miss: true`.
4. Build candidate pool with base scores:
   - SQL keywords: 300
   - Schema names: 250
   - Table names: 200
   - Column names: 100
   - DB functions: 150
5. Filter: each candidate's text must prefix-match `word` (case-insensitive) OR have levenshtein distance ≤ 2 from `word`.
6. Context boost: if `prefix` contains FROM or JOIN before cursor (simple substring check, no full AST parse), table candidates get +100.
7. Sort by score DESC, trim to top 20.
8. Return `AutocompleteResponse`.

### Static data

- **SQL keywords** (~100): SELECT, FROM, WHERE, JOIN, LEFT, RIGHT, INNER, OUTER, CROSS, ON, AND, OR, NOT, IN, EXISTS, BETWEEN, LIKE, IS, NULL, AS, GROUP, BY, ORDER, ASC, DESC, HAVING, UNION, ALL, INSERT, INTO, VALUES, UPDATE, SET, DELETE, CREATE, ALTER, TABLE, DROP, INDEX, VIEW, TRIGGER, PROCEDURE, FUNCTION, SCHEMA, DATABASE, DISTINCT, COUNT, LIMIT, OFFSET, FETCH, CASE, WHEN, THEN, ELSE, END, CAST, COALESCE, NULLIF, PRIMARY, KEY, FOREIGN, REFERENCES, CONSTRAINT, UNIQUE, CHECK, DEFAULT, CASCADE, TRUNCATE, BEGIN, COMMIT, ROLLBACK, WITH, RECURSIVE, OVER, PARTITION, WINDOW, ROWS, RANGE, INTERSECT, EXCEPT, NATURAL, USING, LATERAL, IF, RETURNING, EXPLAIN, ANALYZE, GRANT, REVOKE, TEMPORARY, TEMP, MATERIALIZED, REPLACE, TOP, FIRST, NEXT, ONLY, PERCENT, CUBE, ROLLUP, GROUPING, SETS
- **DB functions** (~30): COUNT, SUM, AVG, MIN, MAX, COALESCE, NULLIF, CAST, CONVERT, CONCAT, SUBSTRING, SUBSTR, UPPER, LOWER, TRIM, LTRIM, RTRIM, LENGTH, CHAR_LENGTH, REPLACE, NOW, CURRENT_DATE, CURRENT_TIME, CURRENT_TIMESTAMP, DATE_TRUNC, EXTRACT, DATEADD, DATEDIFF, ABS, ROUND, CEIL, CEILING, FLOOR, MOD, RAND, RANDOM, ROW_NUMBER, RANK, DENSE_RANK, LAG, LEAD, FIRST_VALUE, LAST_VALUE

### Levenshtein dependency

Use `github.com/texttheater/golang-levenshtein/levenshtein` — no transitive dependencies.

## Frontend

### API (api/sqllab.ts)

```typescript
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

### Monaco Provider (SQLLabPage.tsx)

Registered in `handleEditorMount`:

```typescript
monaco.languages.registerCompletionItemProvider("sql", {
  provideCompletionItems: async (model, position) => {
    const word = model.getWordUntilPosition(position);
    const prefix = model.getValueInRange({
      startLineNumber: 1, startColumn: 1,
      endLineNumber: position.lineNumber, endColumn: position.column,
    });

    // Skip if word too short (< 1 char) to avoid noise
    if (!word.word || word.word.length < 1) return { suggestions: [] };

    const res = await fetchAutocomplete({
      word: word.word,
      prefix,
      db_id: activeTab?.databaseId,
      schema: activeTab?.schema,
    });

    return {
      suggestions: res.suggestions.map(s => ({
        label: s.text,
        kind: kindMap[s.type],  // keyword→Keyword, schema→Module, table→Class, column→Field, function→Function
        detail: s.detail,
        sortText: String(99999 - s.score).padStart(5, "0"),
        insertText: s.text,
      })),
    };
  },
});
```

Provider registered once on mount via `monaco.languages.registerCompletionItemProvider("sql", ...)`. Monaco natively debounces completion triggers — no external debounce needed.

### Type → CompletionItemKind mapping

| type | CompletionItemKind |
|------|-------------------|
| keyword | `monaco.languages.CompletionItemKind.Keyword` |
| schema | `monaco.languages.CompletionItemKind.Module` |
| table | `monaco.languages.CompletionItemKind.Class` |
| column | `monaco.languages.CompletionItemKind.Field` |
| function | `monaco.languages.CompletionItemKind.Function` |

### cache_miss Alert

Local state in SQLLabPage: `const [autocompleteCacheMiss, setAutocompleteCacheMiss] = useState(false)`.

- Pass `onCacheMiss` callback to the provider (via closure). Provider calls it with `true`/`false` based on response.
- When `true`: render `<Alert variant="default" className="mt-2">Schema loading — full autocomplete will be available shortly.</Alert>` below the editor.
- Alert auto-hides when next response comes back with `cache_miss: false`.
- Alert is dismissible via `AlertDialog` / close button.

### No TanStack Query

Autocomplete requests are ephemeral (every keystroke produces different word+prefix). TanStack Query caching/invalidation adds overhead with no benefit. Plain `fetch` inside the provider is the right call.

## Error Handling

| Scenario | Backend | Frontend |
|----------|---------|----------|
| Empty word | 400 `{error: "invalid_request", message: "..."}` | Provider returns empty suggestions |
| Schema not cached | 200 `{cache_miss: true}` + keywords only | Show cache_miss alert |
| DB introspection fails | 500 | Provider returns empty suggestions (graceful degradation) |
| Network error | — | Provider catches, returns empty, no toast |

## Acceptance Criteria

- <50ms p99 response time (Redis local, no DB introspection)
- FROM/JOIN context → tables scored higher than other item types
- Cache miss → keywords + functions only + cache_miss:true
- Top 20 results returned
- Monaco shows suggestions with correct icons per type
- Alert shown on cache miss, auto-hides on recovery

## Files Changed

| File | Change |
|------|--------|
| `backend/internal/domain/query/sqllab_types.go` | Add AutocompleteRequest, AutocompleteSuggestion, AutocompleteResponse |
| `backend/internal/delivery/http/sqllab/handler.go` | Add Autocomplete method |
| `backend/internal/delivery/http/router.go` | Add route |
| `frontend/src/api/sqllab.ts` | Add types + fetchAutocomplete |
| `frontend/src/pages/sqllab/SQLLabPage.tsx` | Register completion provider, cache_miss alert |
