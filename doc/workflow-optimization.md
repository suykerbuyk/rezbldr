# ResumeCTL Workflow Optimization Analysis

**Date:** 2026-04-04
**Scope:** Retrospective analysis of 21 commits, 50 sessions, 6 Claude Code
skills across a 3-day sprint producing 15+ targeted applications.

---

## Part 1: Intelligence Classification

Every step across all 6 skills classified by what level of compute
actually does the work.

### Legend

- **F** = Frontier LLM required (creative synthesis, judgment, conversation)
- **M** = Mid-tier LLM sufficient (extraction, summarization — Haiku/Sonnet)
- **C** = Pure code (deterministic logic, shell commands, math)

### `/res_parse` — Parse Job Posting

| Step | Description | Level | Notes |
|------|-------------|-------|-------|
| 1a | Determine input type (URL/file/paste) | C | String pattern match |
| 1b | Fetch URL content | C | WebFetch call |
| 2 | Extract structured fields from posting | M | Structured extraction, not creative |
| 3 | Generate filename | C | Date + slug formatting |
| 4 | Compose file with frontmatter + sections | M | Template fill with light rewriting |
| 5 | Confirm and write | C | File I/O |

**Verdict:** 0% Frontier. Haiku-tier extraction + local code for file ops.

### `/res_match` — Gap Analysis + Coaching

| Step | Description | Level | Notes |
|------|-------------|-------|-------|
| 1 | Resolve job file (glob + sort) | C | Most-recent-file lookup |
| 2 | Load vault data (read all files) | C | File I/O + frontmatter parse |
| 3 | Tag intersection scoring | C | Weighted set intersection math |
| 4 | Semantic skill-by-skill analysis | F | Requires understanding nuance |
| 5 | Compute overall score | C | Weighted formula |
| 6 | Present results | C | Template formatting |
| 7 | Interactive coaching loop | **F** | **Core value** — surfaces undocumented experience |

**Verdict:** Steps 1-3, 5-6 are pure code. Steps 4 and 7 require Frontier.
The coaching loop is the single highest-value use of the frontier model in
the entire pipeline.

### `/res_build` — Generate Resume + Cover Letter

| Step | Description | Level | Notes |
|------|-------------|-------|-------|
| 1 | Load source data | C | File I/O |
| 2 | Tag intersection scoring + ranking | C | **Duplicated** from /res_match |
| 3 | Present selection for approval | C | Template formatting |
| 4 | Synthesize professional summary | **F** | Creative, job-specific writing |
| 5 | Select and order core competencies | F | Judgment call on relevance |
| 6 | Rewrite experience bullets | **F** | **Core value** — tailored narrative |
| 7 | Cover letter | **F** | **Core value** — hooks, narrative, voice |
| 8 | Compose frontmatter | C | Template fill |
| 9 | Validate (word count, headings, skills) | C | Rules-based checking |
| 10 | Write file | C | File I/O |

**Verdict:** Steps 4-7 are genuine Frontier work. Everything else is code.

### `/res_export` — Export to DOCX/PDF

| Step | Description | Level | Notes |
|------|-------------|-------|-------|
| 1 | Resolve resume file | C | Glob + sort |
| 2 | Strip frontmatter | C | YAML delimiter detection |
| 3 | Verify heading hierarchy | C | Regex check |
| 4 | Write temp file | C | File I/O |
| 5 | Check pandoc availability | C | `which pandoc` |
| 6 | Determine output path | C | String formatting |
| 7 | Run pandoc | C | Shell command |
| 8 | Find matching cover letter | C | Glob pattern |
| 9 | Export cover letter | C | Same as 2-7 |
| 10 | Clean up temp files | C | Shell command |

**Verdict: 0% LLM of any kind.** This is a shell script masquerading as a
frontier model prompt. ~10K tokens per invocation, entirely wasted.

### `/res_train` — Training File Generation

| Step | Description | Level | Notes |
|------|-------------|-------|-------|
| 1 | Resolve job file | C | Glob + sort |
| 2 | Load vault data | C | File I/O |
| 3 | Identify gaps (skills cross-ref) | C | Set difference against skills.md |
| 4 | Generate learning paths | M | Templated but needs some creativity |
| 5 | Update existing training files | C | Frontmatter merge + table append |
| 6 | Present summary | C | Template formatting |

**Verdict:** Mostly code. Learning path generation is the only LLM step,
and Haiku/Sonnet handles it fine.

### `/wrap` — Commit Vault Changes

| Step | Description | Level | Notes |
|------|-------------|-------|-------|
| 1 | Update resume.md | M | Summary writing |
| 2 | Append iterations.md | M | Narrative writing |
| 3 | Update commit.msg | M | Structured description |
| 4 | Stage files | C | `git add` |
| 5 | Commit | C | `git commit -F` |
| 6 | Discover remotes | C | `git remote` |
| 7 | Push to remotes | C | `git push` |

**Verdict:** Steps 1-3 need a mid-tier model. Steps 4-7 are shell commands.

---

## Summary: Token Waste by Skill

| Skill | Total Steps | Code Steps | LLM Steps | Frontier Steps | Waste |
|-------|-------------|------------|-----------|----------------|-------|
| /res_parse | 5 | 3 | 2 (M) | 0 | **100%** frontier waste |
| /res_match | 7 | 5 | 0 | 2 | 71% code, well-targeted LLM |
| /res_build | 10 | 6 | 0 | 4 | 60% code, well-targeted LLM |
| /res_export | 10 | 10 | 0 | 0 | **100%** total waste |
| /res_train | 6 | 5 | 1 (M) | 0 | **100%** frontier waste |
| /wrap | 7 | 4 | 3 (M) | 0 | **100%** frontier waste |

### Duplicated Logic

The tag intersection scoring algorithm appears in both `/res_match` (Step 3)
and `/res_build` (Step 2) with identical weights (required=2.0, preferred=1.0,
tag=0.5) and modifiers (highlight +10%, age -30%). This is ~30 lines of
prompt repeated twice, consuming tokens in both invocations for the same math.

### The "Most Recent File" Pattern

Four of six skills (res_match, res_build, res_train, res_export) begin with
identical "resolve the file" logic: glob a directory, sort by name, pick the
newest. This is a one-line shell command repeated four times across prompts.

---

## Part 2: Local Code Offload Architecture

### Proposed: `resumectl-mcp` — A lightweight MCP server

Written in Go (matching John's strongest systems language), this server
replaces all deterministic operations with local tools. The frontier model
only fires for creative synthesis and interactive coaching.

### Tool Definitions

#### `vault_rank` — Experience File Scoring
```
Input:  job_file path (or "latest")
Output: JSON array of {file, role, company, score, matched_required[],
        matched_preferred[], matched_tags[]} sorted by score descending
```
Implements the tag intersection algorithm once, correctly, testably.
Eliminates the duplicated scoring logic from /res_match and /res_build.
Pre-loads and caches frontmatter from experience files.

#### `vault_resolve` — File Path Resolution
```
Input:  type ("job"|"resume"|"cover"), slug (optional), date (optional)
Output: {path, exists, alternatives[]}
```
Handles the "most recent file in directory" pattern, naming convention
generation, version numbering, and directory creation. One tool replaces
the file resolution preamble from 4 different skills.

#### `vault_validate` — Resume Validation
```
Input:  resume_path
Output: {word_count, heading_errors[], unknown_skills[], missing_companies[],
         contact_match: bool, warnings[]}
```
Checks all /res_build validation rules: word count 600-800, heading
hierarchy (h1/h2/h3), skills against skills.md, companies against
experience files, contact info against contact.md. Returns structured
results so the LLM can fix issues without re-reading source files.

#### `vault_export` — Pandoc Export Pipeline
```
Input:  source_path, format ("docx"|"pdf"), template (optional)
Output: {resume_path, resume_size, cover_path, cover_size, errors[]}
```
**Replaces /res_export entirely.** Strips frontmatter, writes temp file,
runs pandoc, auto-discovers matching cover letter, exports both, cleans up.
Zero LLM involvement. Could be invoked directly by the user or by other
tools.

#### `vault_frontmatter` — YAML Frontmatter Parser
```
Input:  file_path, action ("parse"|"generate"|"strip")
Output: JSON object of frontmatter fields, or cleaned markdown body
```
The LLM currently spends tokens on "parse frontmatter" instructions in
every skill. This extracts it once as structured JSON.

#### `vault_wrap` — Git Operations
```
Input:  commit_message, files[] (explicit paths)
Output: {committed: bool, hash, push_results: [{remote, success, error}]}
```
Stages specified files, commits with message, discovers and pushes to all
remotes. Replaces the git portion of /wrap. The LLM still writes the
commit message and narrative — this tool just executes.

#### `vault_score_diff` — Before/After Scoring
```
Input:  job_file, changes[] (file edits made during coaching)
Output: {old_score, new_score, delta, improved_skills[]}
```
Used during the coaching loop to show score improvement after vault edits,
without the LLM re-computing the math.

### What the Skills Become

After offloading to `resumectl-mcp`:

#### `/res_parse` (Simplified)
1. Determine input type → fetch/read content
2. **LLM (Haiku subagent):** Extract structured fields from raw text
3. `vault_resolve` → generate filename
4. Write file

**Token reduction:** ~60% (Haiku + no file resolution logic in prompt)

#### `/res_match` (Streamlined)
1. `vault_resolve` → find job file
2. `vault_rank` → pre-computed scores + matched skills
3. **LLM (Frontier):** Semantic analysis on top-8 files only
4. **LLM (Frontier):** Interactive coaching loop
5. `vault_score_diff` → show improvement after edits

**Token reduction:** ~40% (no scoring logic in prompt, skip bottom files)

#### `/res_build` (Streamlined)
1. `vault_resolve` → find job file
2. `vault_rank` → pre-ranked experience files
3. **LLM (Frontier):** Synthesize resume + cover letter
4. `vault_validate` → check output
5. `vault_resolve` → generate output path
6. Write files

**Token reduction:** ~35% (no scoring, no validation logic in prompt)

#### `/res_export` → **DELETED**
Replaced entirely by `vault_export`. User calls it directly:
```
vault_export resumes/generated/john_suykerbuyk_2026-04-04-micron_resume.md
```
Or /res_build calls it automatically after writing.

**Token reduction:** 100%

#### `/res_train` (Simplified)
1. `vault_resolve` → find job file
2. `vault_rank` → identify gaps (skills not in matched sets)
3. **LLM (Haiku subagent):** Generate learning paths for gaps
4. Write/update training files

**Token reduction:** ~50%

#### `/wrap` (Simplified)
1. **LLM (mid-tier):** Write iteration narrative + commit message
2. `vault_wrap` → stage, commit, push

**Token reduction:** ~40%

### Estimated Token Savings Per Application Cycle

| Skill | Current | Optimized | Savings |
|-------|---------|-----------|---------|
| /res_parse | ~15K | ~6K (Haiku) | 60% |
| /res_match | ~40K | ~24K | 40% |
| /res_build | ~30K | ~20K | 33% |
| /res_export | ~10K | 0 | **100%** |
| /res_train | ~8K | ~4K (Haiku) | 50% |
| /wrap | ~15K | ~9K | 40% |
| **Total** | **~118K** | **~63K** | **~47%** |

Frontier tokens specifically drop from ~118K to ~44K (the res_match and
res_build creative work). The remaining ~19K shifts to Haiku, which is
~25x cheaper per token.

### Implementation Priority

1. **`vault_export`** — Immediate, highest ROI. Zero LLM involvement,
   replaces an entire skill, simple to implement and test.
2. **`vault_rank`** — High ROI. Eliminates duplicated logic, reduces
   context loading in the two most token-heavy skills.
3. **`vault_resolve`** — Medium ROI. Small per-invocation savings but
   used in every skill.
4. **`vault_validate`** — Medium ROI. Catches errors without LLM
   re-reading source files.
5. **`vault_wrap`** — Lower ROI but removes git operation prompting.
6. **`vault_frontmatter`** — Nice-to-have. Reduces prompt complexity.
7. **`vault_score_diff`** — Nice-to-have. Coaching loop quality-of-life.

### Tech Stack Recommendation

- **Language:** Go — John's strongest systems language, excellent for
  MCP servers (fast startup, single binary, good YAML/JSON libraries)
- **MCP framework:** `github.com/mark3labs/mcp-go` or similar
- **YAML parsing:** `gopkg.in/yaml.v3` for frontmatter
- **Testing:** Standard Go testing + golden files for scoring algorithm
- **Distribution:** Single binary, no runtime dependencies

---

## Part 3: Workflow Diagrams

### Current Workflow — Full Pipeline

```mermaid
flowchart TD
    subgraph INPUT["Job Discovery"]
        URL([URL / Paste / File])
    end

    subgraph PARSE["/res_parse — 0% Frontier"]
        P1[Determine input type]
        P2[Fetch/Read content]
        P3["Extract structured data<br/><i>LLM: Haiku-tier</i>"]
        P4[Generate filename]
        P5[Write job file]
        P1 --> P2 --> P3 --> P4 --> P5
    end

    subgraph MATCH["/res_match — Frontier for coaching only"]
        M1[Resolve job file]
        M2[Load vault data]
        M3["Tag intersection scoring<br/><i>Pure math</i>"]
        M4["Semantic skill analysis<br/><i>LLM: Frontier</i>"]
        M5[Compute overall score]
        M6[Present results]
        M7{"Gaps exist?"}
        M8["Interactive coaching loop<br/><i>LLM: Frontier — CORE VALUE</i>"]
        M9[Edit vault files]
        M10[Re-score]
        M1 --> M2 --> M3 --> M4 --> M5 --> M6 --> M7
        M7 -->|Yes| M8 --> M9 --> M10
        M7 -->|No| MBUILD
    end

    subgraph TRAIN["/res_train — 0% Frontier"]
        T1[Identify gaps from scoring]
        T2["Generate learning paths<br/><i>LLM: Haiku-tier</i>"]
        T3[Create/update training files]
        T1 --> T2 --> T3
    end

    subgraph BUILD["/res_build — Frontier for synthesis only"]
        MBUILD[Load source data]
        B1["Rank experience files<br/><i>Pure math — DUPLICATED</i>"]
        B2[Present selection]
        B3{"User approves?"}
        B4["Synthesize resume<br/><i>LLM: Frontier — CORE VALUE</i>"]
        B5["Write cover letter<br/><i>LLM: Frontier — CORE VALUE</i>"]
        B6["Validate output<br/><i>Rules-based</i>"]
        B7[Write files]
        MBUILD --> B1 --> B2 --> B3
        B3 -->|Yes| B4 --> B5 --> B6 --> B7
        B3 -->|Adjust| B2
    end

    subgraph EXPORT["/res_export — 0% LLM"]
        E1[Resolve resume file]
        E2[Strip frontmatter]
        E3[Write temp file]
        E4[Run pandoc]
        E5[Find matching cover letter]
        E6[Export cover letter]
        E7[Clean up]
        E1 --> E2 --> E3 --> E4 --> E5 --> E6 --> E7
    end

    subgraph WRAP["/wrap — Mid-tier LLM"]
        W1["Write iteration narrative<br/><i>LLM: Sonnet-tier</i>"]
        W2["Write commit message<br/><i>LLM: Sonnet-tier</i>"]
        W3[Stage files]
        W4[Commit]
        W5[Push to remotes]
        W1 --> W2 --> W3 --> W4 --> W5
    end

    URL --> PARSE
    P5 --> MATCH
    M10 --> TRAIN
    M10 --> BUILD
    M7 -->|No gaps| BUILD
    B7 --> EXPORT
    E7 --> WRAP

    classDef frontier fill:#ff6b6b,stroke:#c0392b,color:#fff
    classDef midtier fill:#feca57,stroke:#f39c12,color:#333
    classDef code fill:#48dbfb,stroke:#0abde3,color:#333
    classDef corevalue fill:#ff4757,stroke:#c0392b,color:#fff,stroke-width:3px

    class M4,B4,B5 frontier
    class M8 corevalue
    class P3,T2,W1,W2 midtier
    class P1,P2,P4,P5,M1,M2,M3,M5,M6,M7,M9,M10,T1,T3,MBUILD,B1,B2,B3,B6,B7,E1,E2,E3,E4,E5,E6,E7,W3,W4,W5 code
```

### Proposed Optimized Workflow — With `resumectl-mcp`

```mermaid
flowchart TD
    subgraph INPUT["Job Discovery"]
        URL([URL / Paste / File])
    end

    subgraph PARSE["res_parse — Haiku subagent"]
        P1["vault_resolve → filename<br/><i>MCP tool</i>"]
        P2[Fetch/Read content]
        P3["Extract structured data<br/><i>Haiku subagent</i>"]
        P4[Write job file]
        P1 --> P2 --> P3 --> P4
    end

    subgraph MATCH["res_match — Frontier for coaching"]
        MR["vault_rank → scored files<br/><i>MCP tool</i>"]
        M4["Semantic analysis on top-8<br/><i>Frontier</i>"]
        M6[Present results]
        M7{"Gaps?"}
        M8["Coaching loop<br/><i>Frontier — CORE VALUE</i>"]
        M9[Edit vault files]
        MSD["vault_score_diff<br/><i>MCP tool</i>"]
        MR --> M4 --> M6 --> M7
        M7 -->|Yes| M8 --> M9 --> MSD
        M7 -->|No| BR
    end

    subgraph BUILD["res_build — Frontier for synthesis"]
        BR["vault_rank → top files<br/><i>MCP tool (cached)</i>"]
        B2[Present selection]
        B3{"Approved?"}
        B4["Synthesize resume + cover<br/><i>Frontier — CORE VALUE</i>"]
        BV["vault_validate<br/><i>MCP tool</i>"]
        BRE["vault_resolve → paths<br/><i>MCP tool</i>"]
        B7[Write files]
        BR --> B2 --> B3
        B3 -->|Yes| B4 --> BV --> BRE --> B7
        B3 -->|Adjust| B2
    end

    subgraph EXPORT["vault_export — Zero LLM"]
        EX["vault_export<br/><i>MCP tool — replaces entire skill</i>"]
    end

    subgraph WRAP["wrap — Simplified"]
        W1["Write narrative + commit msg<br/><i>Sonnet subagent</i>"]
        WR["vault_wrap<br/><i>MCP tool</i>"]
        W1 --> WR
    end

    URL --> PARSE
    P4 --> MATCH
    MSD --> BUILD
    M7 -->|No gaps| BUILD
    B7 --> EXPORT
    EX --> WRAP

    classDef frontier fill:#ff6b6b,stroke:#c0392b,color:#fff
    classDef corevalue fill:#ff4757,stroke:#c0392b,color:#fff,stroke-width:3px
    classDef haiku fill:#a29bfe,stroke:#6c5ce7,color:#fff
    classDef mcp fill:#55efc4,stroke:#00b894,color:#333
    classDef sonnet fill:#feca57,stroke:#f39c12,color:#333

    class M4 frontier
    class M8,B4 corevalue
    class P3 haiku
    class W1 sonnet
    class MR,MSD,BR,BV,BRE,EX,WR,P1 mcp
```

### State Diagram — Single Job Application Lifecycle

```mermaid
stateDiagram-v2
    [*] --> JobDiscovered: URL or paste

    JobDiscovered --> Parsed: /res_parse
    note right of Parsed
        Job file in jobs/target/
        Frontmatter: skills, tags, comp
    end note

    Parsed --> Scored: vault_rank
    note right of Scored
        Experience files ranked
        Required/preferred/tag matches
    end note

    Scored --> Analyzed: Frontier semantic analysis
    Analyzed --> GapCheck

    state GapCheck <<choice>>
    GapCheck --> Coaching: Gaps exist
    GapCheck --> ReadyToBuild: No gaps

    Coaching --> VaultEnriched: User recalls experience
    Coaching --> GapAcknowledged: Genuine gap
    VaultEnriched --> ReScored: vault_score_diff
    GapAcknowledged --> ReScored
    ReScored --> GapCheck: Next gap

    ReScored --> ReadyToBuild: All gaps addressed

    ReadyToBuild --> ResumeGenerated: Frontier synthesis
    note right of ResumeGenerated
        Resume + cover letter
        Validated by vault_validate
    end note

    ResumeGenerated --> ReviewCheck

    state ReviewCheck <<choice>>
    ReviewCheck --> Exported: Approved
    ReviewCheck --> ResumeGenerated: Edit requested

    Exported --> Committed: /wrap
    note right of Exported
        DOCX + PDF via vault_export
        Zero LLM tokens
    end note

    Committed --> [*]

    state Coaching {
        [*] --> AskQuestion
        AskQuestion --> ListenResponse
        ListenResponse --> ProposeEdit: Experience recalled
        ListenResponse --> AcknowledgeGap: No experience
        ProposeEdit --> ConfirmEdit
        ConfirmEdit --> ApplyEdit: Confirmed
        ConfirmEdit --> AskQuestion: Rejected
        ApplyEdit --> [*]
        AcknowledgeGap --> [*]
    }
```

### Execution Model — What Runs Where

```mermaid
flowchart LR
    subgraph LOCAL["Local Machine — resumectl-mcp"]
        direction TB
        VR[vault_resolve]
        VK[vault_rank]
        VV[vault_validate]
        VE[vault_export]
        VW[vault_wrap]
        VF[vault_frontmatter]
        VS[vault_score_diff]
    end

    subgraph HAIKU["Haiku Subagent — Cheap extraction"]
        direction TB
        RP[res_parse extraction]
        RT[res_train learning paths]
    end

    subgraph SONNET["Sonnet Subagent — Mid-tier writing"]
        direction TB
        WN[wrap narrative]
        WC[wrap commit message]
    end

    subgraph OPUS["Frontier — Creative + Interactive"]
        direction TB
        SA[Semantic skill analysis]
        CL[Coaching loop]
        RS[Resume synthesis]
        CW[Cover letter writing]
    end

    LOCAL ---|"Scoring data"| OPUS
    LOCAL ---|"Validation results"| OPUS
    HAIKU ---|"Extracted fields"| LOCAL
    SONNET ---|"Narrative text"| LOCAL
    OPUS ---|"Generated content"| LOCAL

    classDef local fill:#55efc4,stroke:#00b894,color:#333
    classDef haiku fill:#a29bfe,stroke:#6c5ce7,color:#fff
    classDef sonnet fill:#feca57,stroke:#f39c12,color:#333
    classDef opus fill:#ff6b6b,stroke:#c0392b,color:#fff

    class VR,VK,VV,VE,VW,VF,VS local
    class RP,RT haiku
    class WN,WC sonnet
    class SA,CL,RS,CW opus
```

---

## Part 4: Observations and Recommendations

### What's Already Working Well

1. **The coaching loop is genuinely valuable.** Sessions show score jumps
   of 10-20 points (62→79 for Sony DevOps, 86→91 for FarmGPU) from
   surfacing undocumented experience. This is the correct use of a frontier
   model — interactive judgment that requires understanding context.

2. **The vault enrichment pattern compounds.** Early sessions enrich
   experience files, and later sessions benefit ("No vault enrichment
   needed — reused context from iteration 18"). The vault is getting
   richer over time.

3. **Naming conventions stabilized by day 2.** The `{name}_{date}-{slug}`
   pattern locked in after one correction cycle. No regressions since.

4. **Honest gap assessment.** Sessions correctly rejected poor fits
   (PNC at 52, RIVA at 44) and honestly acknowledged genuine gaps
   (Kubernetes, Terraform, Weka). This is working as designed.

### What Should Change

1. **Kill /res_export immediately.** It is a 153-line prompt that
   instructs a frontier model to run `pandoc`. This is the single most
   wasteful construct in the pipeline. Replace with a 50-line Go function.

2. **Unify the scoring algorithm.** Tag intersection scoring is
   duplicated verbatim between /res_match and /res_build. Extract it
   once, test it, and serve results via MCP. This also enables caching —
   if /res_match already scored for a job, /res_build shouldn't re-score.

3. **Stop loading all 15 experience files into context for every skill.**
   After vault_rank pre-scores, the LLM only needs to read the top 5-8
   files plus the gap-specific files. This alone saves ~15K tokens per
   invocation in /res_match and /res_build.

4. **Shift /res_parse to Haiku.** Structured extraction from job postings
   is well within Haiku's capabilities. The current prompts don't require
   creative judgment — they require careful reading and field extraction.

5. **Consider auto-chaining.** The most common flow is parse → match →
   build → export → wrap. Each skill currently requires manual invocation.
   A single `/res_pipeline <url>` command could orchestrate the full chain
   with human checkpoints only at coaching (interactive) and resume review
   (approval gate).

### The Mermaid-as-Specification Experiment

The state diagram above is deliberately precise enough to be machine-
readable. The key insight: each state transition maps to either an MCP
tool call (deterministic) or an LLM invocation (creative), with the
transition type explicitly labeled. A code generator given this diagram
plus the tool signatures could produce:

- The MCP server scaffold (Go struct definitions, handler stubs)
- The simplified skill prompts (only the LLM-required states)
- Integration tests (state transition coverage)
- A CLI orchestrator (the pipeline command)

This is the "well-defined Mermaid diagram → optimized workflow" hypothesis.
The diagram encodes not just *what* happens but *where* each operation
should execute and *why* — which is the information a code generator needs
to make correct architectural decisions rather than just syntactic ones.
