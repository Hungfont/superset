# MemPalace Integration

## MemPalace-First Discovery Pattern

**Rule:** For any codebase exploration or discovery task, search MemPalace **first** before falling back to grep/glob/filesystem search.

```
MemPalace semantic search (/mempalace:search)
    ↓ miss, stale, or confidence < high
Grep/Glob (exact symbol/file search)
    ↓ still insufficient
Read files directly
```

## When MemPalace Excels

Semantic/conceptual questions where MemPalace outperforms grep:
- "What is the tech stack?" (mined from configs, not grep-friendly)
- "How does auth flow work?" (multi-file concept, hard to grep)
- "What architecture decisions were made?" (conversation history)
- "Where is the caching layer?" (concept, not literal string)
- "What was the reason for choosing X over Y?" (past discussions)

## When to Skip MemPalace

Go straight to grep/glob for:
- Literal symbol searches: "find all callers of `getUserPermissions`"
- File pattern matching: "find all *.test.tsx files"
- Current state that may not be mined yet (recent changes)
- When MemPalace was last mined >1 week ago and the area changed significantly

## Explore Agent Protocol

When the Explore agent is invoked for open-ended discovery:

1. **First action:** Run `/mempalace:search` with the user's question or discovery intent
2. **Evaluate results:** If results are recent (<7 days) and relevant, use them as the primary source
3. **Fill gaps:** Only grep/glob for what MemPalace missed
4. **Fall back:** If MemPalace returns empty or clearly stale results, proceed with full filesystem search

This saves context window and reduces redundant filesystem scans across sessions.

## Keeping MemPalace Fresh

| Trigger | Action |
|---------|--------|
| After major implementation | `/mempalace:mine <changed_dirs>` |
| Before context compaction | `/mempalace:mine docs/` to persist key docs |
| After architectural decisions | `/mempalace:mine docs/diagram/ docs/CODEMAPS/` |
| Weekly (if active development) | `/mempalace:mine docs/` to refresh stale entries |

## Palace Location

This project's MemPalace: `D:\superset\.mempalace` (configured via `MEMPALACE_PALACE_PATH` in `.claude/settings.json`).
