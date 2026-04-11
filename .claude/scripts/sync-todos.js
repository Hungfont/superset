#!/usr/bin/env node
/**
 * .claude/scripts/sync-todos.js
 *
 * PostToolUse hook — fires after every TodoWrite call.
 * Reads the todo list from stdin and syncs it into
 * .claude/memory/implementation-log.md under the detected REQ-ID section.
 *
 * Stdin payload (JSON from Claude Code):
 * {
 *   tool_name:     "TodoWrite",
 *   tool_input:    { todos: [{ content, status, activeForm }] },
 *   tool_response: "Todos have been modified successfully..."
 * }
 */

'use strict';

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

const REPO_ROOT = path.resolve(__dirname, '../..');
const LOG_PATH  = path.resolve(__dirname, '../memory/implementation-log.md');

// Matches requirement IDs like AUTH-001, RBAC-002, etc.
const REQ_ID_RE = /\b([A-Z]{2,}-\d+)\b/;

// ── helpers ────────────────────────────────────────────────────────────────

function readStdin() {
  return new Promise((resolve) => {
    let buf = '';
    process.stdin.setEncoding('utf8');
    process.stdin.on('data', (chunk) => (buf += chunk));
    process.stdin.on('end', () => resolve(buf));
    process.stdin.resume();
  });
}

/**
 * Detect the current requirement ID.
 * Priority:
 *   1. Any REQ-ID found in the todo content strings
 *   2. REQ-ID found in the most recent git commit subject
 *   3. null → caller uses 'CURRENT-SESSION'
 */
function detectReqId(todos) {
  for (const todo of todos) {
    const m = todo.content.match(REQ_ID_RE);
    if (m) return m[1];
  }

  try {
    const subject = execSync('git log -1 --pretty=%s', {
      cwd: REPO_ROOT,
      timeout: 2000,
      stdio: ['ignore', 'pipe', 'ignore'],
    }).toString().trim();
    const m = subject.match(REQ_ID_RE);
    if (m) return m[1];
  } catch {
    // git unavailable or no commits — ignore
  }

  return null;
}

/** Convert todos array → markdown task lines */
function todosToMarkdown(todos) {
  return todos
    .map((t) => {
      const tick = t.status === 'completed' ? 'x' : ' ';
      return `- [${tick}] ${t.content}`;
    })
    .join('\n');
}

/**
 * Build or update the log file.
 *
 * Section format:
 *   ## [AUTH-002] Todos — 2026-04-11
 *
 *   - [x] task one
 *   - [ ] task two
 *
 * If a section with the same label already exists it is replaced in-place;
 * otherwise a new section is appended after a '---' separator.
 */
function syncLog(label, todos) {
  if (!fs.existsSync(LOG_PATH)) return;

  const today = new Date().toISOString().slice(0, 10);
  const header = `## [${label}] Todos — ${today}`;
  const body = todosToMarkdown(todos);
  const newSection = `${header}\n\n${body}\n`;

  // Escape special regex chars in label (e.g. 'CURRENT-SESSION')
  const escaped = label.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');

  // Match the existing section for this label through to the next section/separator/EOF
  const BLOCK_RE = new RegExp(
    `## \\[${escaped}\\] Todos[^\n]*\n[\\s\\S]*?(?=\\n---\\n|\\n## |$)`
  );

  let log = fs.readFileSync(LOG_PATH, 'utf8');

  if (BLOCK_RE.test(log)) {
    log = log.replace(BLOCK_RE, newSection);
  } else {
    log = log.trimEnd() + '\n\n---\n\n' + newSection + '\n';
  }

  fs.writeFileSync(LOG_PATH, log, 'utf8');
}

// ── main ───────────────────────────────────────────────────────────────────

async function main() {
  const raw = await readStdin();

  let payload;
  try {
    payload = JSON.parse(raw);
  } catch {
    return; // not JSON — ignore
  }

  if (payload.tool_name !== 'TodoWrite') return;

  const todos = payload.tool_input?.todos;
  if (!Array.isArray(todos) || todos.length === 0) return;

  const label = detectReqId(todos) ?? 'CURRENT-SESSION';
  syncLog(label, todos);
}

// Never let the hook crash the session
main().catch(() => {});
