# rezbldr Implementation Plan

## Project Goal

Build a Go MCP server that offloads all deterministic, mechanical operations
from the ResumeCTL Claude Code skills into local, testable code. The frontier
model concentrates exclusively on creative synthesis (resume writing, cover
letters) and interactive judgment (coaching loops).

**Measurable target:** Reduce frontier token consumption per job application
cycle from ~118K to ~44K tokens (~63% reduction in frontier spend).

## Architecture

```
rezbldr (single Go binary)
├── cmd/rezbldr/main.go          — MCP server entrypoint
├── internal/
│   ├── vault/                   — Vault data access layer
│   │   ├── vault.go             — Vault struct, path resolution, loading
│   │   ├── frontmatter.go       — YAML frontmatter parse/generate/strip
│   │   ├── experience.go        — Experience file type + loading
│   │   ├── job.go               — Job file type + loading
│   │   ├── resume.go            — Generated resume type
│   │   ├── cover.go             — Cover letter type
│   │   ├── contact.go           — Contact profile type
│   │   ├── skills.go            — Skills table parser
│   │   └── training.go          — Training file type
│   ├── scoring/                 — Tag intersection scoring engine
│   │   ├── scorer.go            — Core scoring algorithm
│   │   ├── scorer_test.go       — Golden-file tests
│   │   └── cache.go             — Score caching per job file
│   ├── export/                  — Pandoc export pipeline
│   │   ├── export.go            — Frontmatter strip + pandoc invocation
│   │   └── export_test.go
│   ├── gitops/                  — Git operations
│   │   ├── gitops.go            — Stage, commit, remote discovery, push
│   │   └── gitops.go_test.go
│   ├── resolve/                 — File path resolution
│   │   ├── resolve.go           — Naming conventions, most-recent lookup
│   │   └── resolve_test.go
│   └── validate/                — Resume validation rules
│       ├── validate.go          — Word count, headings, skill/company checks
│       └── validate_test.go
├── doc/                         — Project documentation
│   ├── implementation-plan.md   — This file
│   ├── workflow-optimization.md — Analysis from retrospective
│   ├── vault-schema.md          — YAML schema reference
│   └── skills-current/          — Current skill definitions (requirements ref)
├── testdata/                    — Golden test fixtures
│   ├── experience/              — Sample experience files
│   ├── jobs/                    — Sample job files
│   └── expected/                — Expected scoring/validation outputs
├── go.mod
├── go.sum
├── Makefile
└── CLAUDE.md
```

## Configuration

The vault path is discovered, not hardcoded:

1. `REZBLDR_VAULT` environment variable (highest priority)
2. `--vault` CLI flag
3. Auto-detect: walk up from CWD looking for `profile/contact.md`
4. Fallback: `~/obsidian/ResumeCTL`

Store as a single `Config` struct passed through the tool handlers.

---

## Phase 1: Foundation — Vault Access Layer

**Goal:** Parse every vault file type into Go structs. This underpins
every tool.

### Task 1.1: YAML Frontmatter Engine

File: `internal/vault/frontmatter.go`

```go
// Parse extracts YAML frontmatter from markdown, returns frontmatter bytes
// and body bytes separately.
func Parse(content []byte) (frontmatter []byte, body []byte, err error)

// Strip removes YAML frontmatter, returning clean markdown body only.
func Strip(content []byte) []byte

// Generate creates a YAML frontmatter block from a struct.
func Generate(v any) ([]byte, error)
```

- Detect `---` delimiters (must be first line for opening)
- Handle edge cases: no frontmatter, empty frontmatter, `---` in body
- Use `gopkg.in/yaml.v3` for parsing

**Tests:** Files with/without frontmatter, malformed delimiters, empty files.

### Task 1.2: Type Definitions

One file per vault type in `internal/vault/`. Each type:
- Go struct with `yaml` tags matching the schema in `doc/vault-schema.md`
- `Load(path string) (*Type, error)` function
- `LoadAll(dir string) ([]*Type, error)` for collection types

Types to implement:

| File | Struct | Key fields |
|------|--------|------------|
| `experience.go` | `Experience` | Role, Company, CompanySlug, Start, End, Tags, Skills, Highlight, Visibility |
| `job.go` | `Job` | Title, Company, CompanySlug, RequiredSkills, PreferredSkills, Tags, Domain, Seniority, Compensation |
| `contact.go` | `Contact` | Name, Email, Phone, Location, LinkedIn, GitHub |
| `skills.go` | `SkillEntry` | Skill, Proficiency, LastUsed, Years, Category — parsed from markdown table, not YAML |
| `resume.go` | `Resume` | JobFile, Generated, ExperienceFiles, WordCount, Version |
| `cover.go` | `CoverLetter` | JobFile, ResumeFile, Generated |
| `training.go` | `Training` | Skill, Category, Priority, Status, SurfacedBy, RelatedSkills |

**Special case — `skills.go`:** The skills inventory is a markdown table,
not YAML frontmatter. Write a table parser:

```go
func ParseSkillsTable(content []byte) ([]SkillEntry, error)
```

Parse the `| Skill | Proficiency | ... |` table format. Handle the
separator row (`|---|---|...|`). Return structured entries.

### Task 1.3: Vault Root Object

File: `internal/vault/vault.go`

```go
type Vault struct {
    Root        string
    Contact     *Contact
    Skills      []SkillEntry
    Experiences []*Experience
    Jobs        []*Job
}

// Open loads the vault from a root directory path.
// Reads contact, skills, and indexes experience/job directories.
func Open(root string) (*Vault, error)

// LoadJob loads a specific job file by path (absolute or relative to vault).
func (v *Vault) LoadJob(path string) (*Job, error)

// LatestJob returns the most recent job file in jobs/target/.
func (v *Vault) LatestJob() (*Job, error)

// LatestResume returns the most recent generated resume.
func (v *Vault) LatestResume() (string, error)
```

**Tests:** Load the testdata vault fixtures, verify all types parse
correctly. Error cases: missing files, malformed YAML, empty directories.

**Acceptance criteria:** `vault.Open(path)` loads all data, every field
accessible as typed Go values.

---

## Phase 2: Scoring Engine

**Goal:** Implement the tag intersection algorithm once, correctly,
with golden-file tests.

### Task 2.1: Core Scorer

File: `internal/scoring/scorer.go`

```go
type ScoredExperience struct {
    File             string   `json:"file"`
    Role             string   `json:"role"`
    Company          string   `json:"company"`
    Score            float64  `json:"score"`
    NormalizedScore  int      `json:"normalized_score"`  // 0-100
    MatchedRequired  []string `json:"matched_required"`
    MatchedPreferred []string `json:"matched_preferred"`
    MatchedTags      []string `json:"matched_tags"`
    Boosted          bool     `json:"boosted"`   // highlight: true
    Penalized        bool     `json:"penalized"` // >10yr old
}

// Rank scores all experience files against a job posting.
// Returns results sorted by score descending.
func Rank(job *vault.Job, experiences []*vault.Experience) []ScoredExperience
```

Algorithm (from the current skill prompts):
1. Build job keyword set: `required_skills` + `preferred_skills` + `tags`
   (all lowercased)
2. Build each experience keyword set: `tags` + `skills` (all lowercased)
3. Score per match: required = 2.0, preferred = 1.0, tag = 0.5
4. Modifiers: `highlight: true` → +10%, end date > 10 years ago → -30%
5. `visibility: hidden` → exclude entirely
6. Normalize to 0-100 scale (relative to max possible score)
7. Sort descending

### Task 2.2: Overall Match Score

```go
type MatchScore struct {
    Overall          int              `json:"overall"`           // 0-100
    RequiredCoverage float64          `json:"required_coverage"` // 0.0-1.0
    PreferredCoverage float64         `json:"preferred_coverage"`
    DomainMatch      float64          `json:"domain_match"`      // 0, 0.5, 1.0
    SeniorityMatch   float64          `json:"seniority_match"`   // 0, 0.5, 1.0
    RequiredHits     []string         `json:"required_hits"`
    RequiredMisses   []string         `json:"required_misses"`
    PreferredHits    []string         `json:"preferred_hits"`
    PreferredMisses  []string         `json:"preferred_misses"`
}

// Score computes the overall match score.
// Formula: required(60%) + preferred(20%) + domain(10%) + seniority(10%)
func Score(job *vault.Job, v *vault.Vault, ranked []ScoredExperience) MatchScore
```

Domain and seniority matching are heuristic — use simple keyword overlap
for domain, and a seniority level hierarchy for seniority fit.

### Task 2.3: Score Caching

File: `internal/scoring/cache.go`

Cache scored results keyed by `(job_file_path, vault_mtime)` where
vault_mtime is the max mtime across all experience files. If any
experience file changes (vault enrichment during coaching), the cache
invalidates. Use in-memory map — no persistence needed since the MCP
server lives for one session.

### Task 2.4: Score Diff

```go
type ScoreDiff struct {
    OldScore       int      `json:"old_score"`
    NewScore       int      `json:"new_score"`
    Delta          int      `json:"delta"`
    ImprovedSkills []string `json:"improved_skills"`
}

// Diff computes the score change after vault edits.
// Reloads modified experience files and re-scores.
func Diff(job *vault.Job, v *vault.Vault, previousRanked []ScoredExperience) ScoreDiff
```

**Tests:** Golden files with known experience/job combinations and
expected scores. Test modifier behavior (highlight boost, age penalty).
Test cache invalidation.

---

## Phase 3: MCP Server + First Tools

**Goal:** Wire up the MCP transport and expose the first high-value tools.

### Task 3.1: MCP Server Skeleton

File: `cmd/rezbldr/main.go`

- Use `github.com/mark3labs/mcp-go` (or `github.com/modelcontextprotocol/go-sdk` if stable)
- stdio transport (standard for Claude Code MCP servers)
- Register tool handlers
- Accept `--vault` flag for vault path override
- Structured logging to stderr (never stdout — that's MCP transport)

### Task 3.2: `vault_rank` Tool

**Replaces:** Tag intersection scoring in `/res_match` and `/res_build`.

```
Tool: vault_rank
Input:
  job_file: string  — path to job file, or "latest"
  top_n: int        — number of results to return (default: 8)
Output:
  job: {title, company, required_count, preferred_count}
  ranked: [{file, role, company, score, matched_required, matched_preferred, matched_tags}]
  match_score: {overall, required_coverage, preferred_coverage, required_hits, required_misses, preferred_hits, preferred_misses}
```

Handler:
1. `vault.Open(root)`
2. Resolve job file (latest if "latest")
3. `scoring.Rank(job, experiences)`
4. `scoring.Score(job, vault, ranked)`
5. Return top_n results + match score

### Task 3.3: `vault_export` Tool

**Replaces:** The entire `/res_export` skill (153 lines of prompt → 0 LLM tokens).

```
Tool: vault_export
Input:
  source: string    — path to resume .md file, or "latest"
  format: string    — "docx" or "pdf" (default: "docx")
  template: string  — optional reference doc path
Output:
  resume: {path, size, format}
  cover: {path, size, format}  — if matching cover letter found
  errors: []string
```

Handler:
1. Resolve source (latest if omitted)
2. Strip frontmatter → write to temp file
3. Determine output path (same naming convention)
4. Check pandoc availability
5. Run pandoc with appropriate flags
6. Glob for matching cover letter (same date + company_slug)
7. If found, export cover letter too
8. Clean up temp files
9. Return results

### Task 3.4: `vault_resolve` Tool

**Replaces:** The "resolve file" preamble in 4 skills.

```
Tool: vault_resolve
Input:
  type: string      — "job" | "resume" | "cover" | "experience"
  slug: string      — optional company slug
  date: string      — optional date (YYYY-MM-DD)
  action: string    — "latest" | "generate" | "exists"
Output:
  path: string
  exists: bool
  alternatives: []string  — other matching files if ambiguous
```

Actions:
- `latest`: Find most recent file of the given type
- `generate`: Construct a filename from slug + date + candidate name
- `exists`: Check if a specific file exists, suggest alternatives if not

---

## Phase 4: Validation + Git Tools

### Task 4.1: `vault_validate` Tool

**Replaces:** The validation step in `/res_build`.

```
Tool: vault_validate
Input:
  resume_path: string
Output:
  word_count: int
  word_count_ok: bool          — 600-800 range
  heading_errors: []string     — h1/h2/h3 hierarchy issues
  unknown_skills: []string     — skills not in skills.md
  unknown_companies: []string  — companies not in experience files
  contact_match: bool          — contact info matches contact.md
  warnings: []string
```

File: `internal/validate/validate.go`

Implementation:
- Count words (split on whitespace, exclude frontmatter)
- Parse heading levels with regex (`^#{1,6}\s`)
- Check h1 count == 1, h2 for sections, h3 for roles
- Extract skill names from "Core Competencies" section, check against skills.md
- Extract company names from h3 headings, check against experience files
- Compare contact line against contact.md fields

### Task 4.2: `vault_wrap` Tool

**Replaces:** The git portion of `/wrap`.

```
Tool: vault_wrap
Input:
  commit_message: string
  files: []string             — explicit file paths to stage
Output:
  committed: bool
  hash: string
  push_results: [{remote: string, success: bool, error: string}]
```

File: `internal/gitops/gitops.go`

Implementation:
1. `git add` each file in the list (never `git add -A`)
2. `git commit -m <message>` (via `-F` with temp file for multiline)
3. `git remote` → discover all remotes
4. For each remote: `git push <remote> main`, capture result
5. Return aggregate result

### Task 4.3: `vault_frontmatter` Tool

**Replaces:** Repeated frontmatter handling across all skills.

```
Tool: vault_frontmatter
Input:
  file: string
  action: string  — "parse" | "strip" | "generate"
  data: object    — only for "generate" action
Output:
  frontmatter: object   — for "parse"
  body: string          — for "strip"
  content: string       — for "generate"
```

### Task 4.4: `vault_score_diff` Tool

**Used during:** Coaching loop in `/res_match`.

```
Tool: vault_score_diff
Input:
  job_file: string
  changed_files: []string  — experience files modified during coaching
Output:
  old_score: int
  new_score: int
  delta: int
  improved_skills: []string
```

Invalidates cache for changed files, re-scores, returns diff.

---

## Phase 5: Simplified Claude Code Skills

**Goal:** Rewrite the skill prompts to delegate deterministic work to
rezbldr MCP tools.

### Task 5.1: Register rezbldr as MCP server

Add to Claude Code's MCP configuration:
```json
{
  "mcpServers": {
    "rezbldr": {
      "command": "/home/johns/code/rezbldr/rezbldr",
      "args": ["--vault", "/home/johns/obsidian/ResumeCTL"]
    }
  }
}
```

### Task 5.2: Rewrite `/res_match`

Current: 186 lines, includes scoring algorithm description.
Target: ~80 lines, delegates scoring to `vault_rank`.

```markdown
Gap analysis between a job posting and your experience vault.

Input: $ARGUMENTS — path to job file, or omit for latest.

## Step 1: Get ranked data

Call `vault_rank` with the job file path (or "latest").
Read the match_score and ranked results.

## Step 2: Semantic analysis

For each required skill in match_score.required_misses:
[... same semantic analysis instructions, but without scoring math ...]

## Step 3: Coaching loop

[... same coaching instructions ...]
After each vault edit, call `vault_score_diff` to show improvement.
```

### Task 5.3: Rewrite `/res_build`

Current: 247 lines, includes scoring + validation.
Target: ~120 lines, delegates ranking and validation.

```markdown
## Step 1: Get ranked experience files

Call `vault_rank` with the job file. Use the top N files.

## Step 2: Synthesize resume

[... same synthesis instructions, unchanged ...]

## Step 3: Validate

Call `vault_validate` on the written file.
If warnings, fix them. If clean, proceed.

## Step 4: Export

Call `vault_export` to generate DOCX + PDF.
```

### Task 5.4: Rewrite `/res_export`

**Delete entirely.** Replace with a one-liner in CLAUDE.md:
"To export a resume, call the `vault_export` MCP tool directly."

Or keep as a thin passthrough skill:
```markdown
Call `vault_export` with source=$ARGUMENTS. Report the results.
```

### Task 5.5: Rewrite `/wrap`

Current: 60 lines mixing LLM narrative + git operations.
Target: ~30 lines. LLM writes narrative, `vault_wrap` handles git.

### Task 5.6: Consider `/res_parse` subagent

For a future phase: `/res_parse` could invoke a Haiku subagent for the
extraction step. This requires Claude Code's subagent model override
support, which may or may not be available. Park this as a stretch goal.

---

## Phase 6: Testing Strategy

### Unit Tests (per package)

| Package | Test Focus | Method |
|---------|-----------|--------|
| `vault/` | Frontmatter parsing, type loading, edge cases | Table-driven with fixtures |
| `scoring/` | Algorithm correctness, modifiers, normalization | Golden files |
| `export/` | Frontmatter stripping, path generation | Unit + integration (needs pandoc) |
| `gitops/` | Command construction, remote discovery | Mock exec |
| `resolve/` | Naming conventions, latest-file logic | Temp directory fixtures |
| `validate/` | Word count, headings, skill/company checks | Golden files |

### Golden File Tests for Scoring

Create `testdata/` with:
- 3-4 sample experience files (varied tags, dates, highlight flags)
- 2-3 sample job files (varied required/preferred skills)
- Expected scoring output as JSON

Run `go test ./internal/scoring/ -update` to regenerate golden files
when the algorithm intentionally changes.

### Integration Tests

- `vault_export`: Requires pandoc installed. Skip with
  `testing.Short()` if pandoc not found.
- `vault_wrap`: Use a temp git repo. Verify commit hash, file staging,
  remote push (mock remote with `git init --bare`).

### Target: 80% Coverage

All deterministic logic should be well-tested. The MCP handler layer
is thin glue — test the underlying packages thoroughly.

---

## Phase 7: Stretch Goals

### 7.1: Pipeline Orchestrator

A `/res_pipeline <url>` command that chains:
parse → rank → (pause for coaching if gaps) → build → export → wrap

This is an orchestration layer on top of the tools. Could be:
- Another Claude Code skill that calls tools in sequence
- A Go function in rezbldr that manages the flow
- A state machine driven by the Mermaid diagram

### 7.2: Vault Enrichment Suggestions

Extend `vault_rank` to return not just scores but specific enrichment
suggestions: "Experience file X mentions technology Y in body text but
doesn't have it in the tags array." This turns passive scoring into
active vault maintenance.

### 7.3: Application Tracker

Extend the job file schema with application state tracking:
`targeting → applied → screening → interviewing → offer → accepted/rejected`

Add a `vault_status` tool that shows pipeline state across all jobs.

### 7.4: Haiku Subagent for Parsing

If Claude Code supports model overrides for subagents, route `/res_parse`
extraction through Haiku. This is a ~60% cost reduction on parsing alone.

---

## Implementation Order + Dependencies

```
Phase 1 (Foundation)
  1.1 Frontmatter engine
  1.2 Type definitions ← depends on 1.1
  1.3 Vault root object ← depends on 1.2

Phase 2 (Scoring)
  2.1 Core scorer ← depends on 1.2
  2.2 Overall match score ← depends on 2.1
  2.3 Score caching ← depends on 2.1
  2.4 Score diff ← depends on 2.1, 2.3

Phase 3 (MCP + First Tools)
  3.1 MCP server skeleton ← independent
  3.2 vault_rank tool ← depends on 1.3, 2.1, 2.2, 3.1
  3.3 vault_export tool ← depends on 1.1 (frontmatter strip), 3.1
  3.4 vault_resolve tool ← depends on 1.3, 3.1

Phase 4 (Remaining Tools)
  4.1 vault_validate ← depends on 1.3, 3.1
  4.2 vault_wrap ← depends on 3.1
  4.3 vault_frontmatter ← depends on 1.1, 3.1
  4.4 vault_score_diff ← depends on 2.4, 3.1

Phase 5 (Skill Rewrites)
  5.1 Register MCP server ← depends on 3.1
  5.2-5.5 Rewrite skills ← depends on respective tools

Phase 6 (Testing)
  Concurrent with all phases — tests written alongside code.
```

**Critical path:** 1.1 → 1.2 → 1.3 → 2.1 → 3.1+3.2 → 5.2 (first
end-to-end improvement visible).

**Quick win path:** 3.1 → 3.3 (vault_export kills /res_export immediately,
no vault access layer needed — just frontmatter strip + pandoc).

---

## Dependencies

| Dependency | Version | Purpose |
|-----------|---------|---------|
| Go | 1.22+ | Language runtime |
| `gopkg.in/yaml.v3` | latest | YAML frontmatter parsing |
| `github.com/mark3labs/mcp-go` | latest | MCP server framework |
| pandoc | 3.x | External: DOCX/PDF export |
| xelatex | any | External: PDF export (optional) |
| git | 2.x | External: vault operations |

No other runtime dependencies. Single static binary.
