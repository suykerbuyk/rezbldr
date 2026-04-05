Gap analysis between a job posting and your experience vault, with
interactive coaching to address skill gaps.

Input: $ARGUMENTS — path to a job file in jobs/target/. If omitted, the most
recent file in jobs/target/ is used automatically.

## Step 1: Get ranked data

Call `rezbldr_rank` with:
- `job_file`: $ARGUMENTS if provided, otherwise `"latest"`
- `top_n`: 8

This returns:
- `job` — title, company, required_count, preferred_count, file_path
- `ranked[]` — experience files sorted by relevance, each with file, role,
  company, score, matched_required, matched_preferred
- `match_score` — overall (0-100), required_coverage, preferred_coverage,
  required_hits, required_misses, preferred_hits, preferred_misses

Tell the user which job file was selected (from `job.file_path`).

## Step 2: Load context for semantic analysis

Read these files in parallel (multiple Read calls in one message):

1. The top 8 experience files listed in `ranked[].file` — read full content
   for semantic matching
2. `profile/skills.md` — parse the skills table for proficiency data
3. `profile/summary-core.md` — for domain context

## Step 3: Semantic analysis

With the ranked experience files, job posting, and skills inventory loaded,
perform a thorough skill-by-skill analysis.

Use `match_score.required_hits` and `match_score.required_misses` to know
which skills were matched or missed by tag intersection. Now add semantic
judgment on top of the mechanical scoring:

For **each required skill** in the job posting:

- **Strong match**: The skill is in `required_hits` AND appears in skills.md
  at Advanced/Expert level AND at least one experience file has concrete
  evidence (specific projects, metrics, technologies used). Record: skill
  name, proficiency level, years of experience, which experience file(s)
  provide evidence, confidence level.

- **Partial match**: The skill is in `required_hits` but at a lower
  proficiency, OR the skill is in `required_misses` but adjacent/transferable
  skills exist in the experience files. Record: what evidence exists, what
  gap remains, which files are relevant.

- **Gap**: The skill is in `required_misses` AND no evidence in skills.md AND
  no adjacent evidence in experience files. Record: why this skill matters
  for the role, and craft a specific coaching question to help the user
  recall undocumented experience.

Repeat for **preferred skills** (using `preferred_hits` / `preferred_misses`).
Flag them as preferred, not required.

Also evaluate qualitatively:
- **Domain alignment**: How well does the candidate's career arc match?
- **Seniority fit**: Does career progression support the target level?
- **Culture fit**: Do the culture signals align with demonstrated work style?
- **Unique advantages**: What does this candidate bring that the posting
  doesn't explicitly ask for but would clearly add value?

## Step 4: Present results

Format the output clearly:

```
## Gap Analysis: {title} @ {company}

**Overall Score: {match_score.overall}/100**

{2-3 sentence narrative assessment — honest, direct, actionable}

### Strong Matches
For each:
- [check] **{Skill}** ({proficiency}, {years} yrs) — {evidence file(s)}

### Partial Matches
For each:
- [~] **{Skill}** — {what evidence exists}
  - Gap: {what's missing or needs strengthening}

### Gaps
For each:
- [x] **{Skill}** ({required|preferred})
  - Why it matters: {context for this role}
  - Coaching question: "{targeted question}"

### Top Experience Files for This Role
1. {filename} — {role} @ {company} (score: {N}) — {why relevant}
2. ...

### Recommendations
- Whether to proceed with application
- Top 1-3 actions to strengthen the application
- Which experience to emphasize in resume
```

## Step 5: Interactive coaching loop

After presenting results, if there are any gaps, offer:

"Would you like to walk through the gaps? I'll ask targeted questions to
surface experience you may not have documented yet."

If the user agrees, for each gap (required skills first, then preferred):

1. **Context**: Explain briefly why this skill matters for the role.

2. **Ask**: Pose the coaching question. Make it specific — reference the
   user's existing roles to jog memory. Example: "During your time at
   Seagate building reference architectures, did you ever handle procurement
   or BOM tracking for the hardware you specified?"

3. **Listen**: Wait for the user's response.

4. **If they recall experience**:
   - Propose a specific vault update: which experience file to edit, what
     bullet to add under Key Contributions, what tag or skill to add to
     the frontmatter arrays.
   - Ask the user to confirm the edit.
   - If confirmed, use the Edit tool to make the change. Only ADD to
     existing arrays — never remove existing tags, skills, or bullets.
   - If the skill is not in `profile/skills.md`, suggest adding it with
     an appropriate proficiency level and ask for confirmation.

5. **If no relevant experience**: Acknowledge honestly and move on.
   Note it as a genuine gap in the final summary.

After coaching, if any files were updated, call `rezbldr_score_diff` with:
- `job_file`: the job file path from Step 1

Show the improvement:
"Score improved from {old_score}/100 to {new_score}/100 (+{delta}) after
coaching updates. Improved skills: {improved_skills}"

## Error handling

- If `rezbldr_rank` returns an error (e.g., no job files found), tell the
  user: "No job files found. Run `/res_parse` first to create one."
- If an experience file has malformed or missing frontmatter, still read its
  body text for semantic matching. Warn but do not skip.
- Never fabricate experience or skills. If a gap is real, say so directly.
  Honest assessment is more valuable than an inflated score.
