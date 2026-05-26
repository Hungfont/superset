package sqllab

import (
	"fmt"
	"net/http"
	"strings"

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

		schemas, err := h.dbSvc.ListSchemas(ctx, 0, req.DbID, false, "")
		if err != nil {
			cacheMiss = true
		} else {
			for _, s := range schemas {
				if matchesWord(s, word) {
					suggestions = append(suggestions, domainquery.AutocompleteSuggestion{
						Text: s, Type: "schema", Score: 250, Detail: "schema",
					})
				}
			}

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

	// Sort by score descending
	sortByScore(suggestions)

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
