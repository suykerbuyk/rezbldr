Gap analysis between a job posting and your experience vault, with
interactive coaching to address skill gaps.

Input: $ARGUMENTS — path to a job file in jobs/target/. If omitted, the most
recent file in jobs/target/ is used automatically.

## Step 1: Resolve the job file

If `$ARGUMENTS` is provided and is a valid path, use it. If it is just a
filename (no directory prefix), prepend `jobs/target/`.

If `$ARGUMENTS` is empty or omitted:
1. Use Glob with pattern `*.md` in path `jobs/target/` to list all job files.
2. If no files exist, tell the user: "No job files found. Run `/res_parse`
   first to create one." Stop here.
3. Sort by filename (date-prefixed, so lexicographic = chronological).
4. Use the most recent file. Tell the user which file was selected.

Read the job file using the Read tool. Parse its frontmatter to extract:
- title, company
- required_skills, preferred_skills
- tags, domain, seniority
- The full body text (responsibilities, qualifications)

If `required_skills` is missing from frontmatter, extract skill requirements
directly from the body text (look for "Required Qualifications" or similar
sections and pull out technology names and skill terms).

## Step 2: Load vault data

Read these files in parallel (use multiple Read tool calls in one message):

1. **Experience files**: Use Glob with pattern `*.md` in path `experience/`
   to list them, then Read each file. Parse frontmatter fields: tags, skills,
   domain, highlight, start, end, role, company. Also read the body text
   for semantic matching.

2. **Skills inventory**: Read `profile/skills.md` — parse the markdown table
   into skill entries (Skill, Proficiency, Last Used, Years, Category).

3. **Professional summary**: Read `profile/summary-core.md` for domain context.

## Step 3: Tag intersection scoring (Stage 1)

For each experience file, compute a relevance score:

1. Build the job's keyword set from: required_skills + preferred_skills +
   tags (all lowercased). If these frontmatter fields are absent, extract
   keywords from the body text.

2. Build each experience file's keyword set from: tags + skills (all
   lowercased), plus significant technology terms from the body text.

3. Score each experience file:
   - Required skill match: weight 2.0 per match
   - Preferred skill match: weight 1.0 per match
   - Tag match: weight 0.5 per match
   - Normalize to 0-100 scale

4. Apply modifiers:
   - `highlight: true` → boost score by 10%
   - End date more than 10 years ago → reduce score by 30%
   - `visibility: hidden` → exclude entirely

5. Rank all experience files by score. Keep the top 8 for deep analysis.

## Step 4: Semantic analysis (Stage 2)

With the ranked experience files, job posting, and skills inventory loaded,
perform a thorough skill-by-skill analysis.

For **each required skill** in the job posting:

- **Strong match**: The skill appears in skills.md at Advanced/Expert level
  AND at least one experience file has concrete evidence (specific projects,
  metrics, technologies used). Record: skill name, proficiency level, years
  of experience, which experience file(s) provide evidence, confidence level.

- **Partial match**: The skill appears in skills.md at a lower level, OR
  appears in experience files without strong direct evidence, OR the candidate
  has closely adjacent/transferable skills. Record: what evidence exists,
  what gap remains, which files are relevant.

- **Gap**: No evidence in skills.md AND no evidence in experience files.
  Record: why this skill matters for the role, and craft a specific coaching
  question to help the user recall undocumented experience.

Repeat for **preferred skills** (flag them as preferred, not required).

Also evaluate qualitatively:
- **Domain alignment**: How well does the candidate's career arc match?
- **Seniority fit**: Does career progression support the target level?
- **Culture fit**: Do the culture signals align with demonstrated work style?
- **Unique advantages**: What does this candidate bring that the posting
  doesn't explicitly ask for but would clearly add value?

## Step 5: Compute overall score

Calculate a score from 0-100:

| Component | Weight | Calculation |
|-----------|--------|-------------|
| Required skills | 60% | (strong * 1.0 + partial * 0.5) / total_required |
| Preferred skills | 20% | (strong * 1.0 + partial * 0.5) / total_preferred |
| Domain alignment | 10% | Qualitative: 0, 0.5, or 1.0 |
| Seniority fit | 10% | Qualitative: 0, 0.5, or 1.0 |

## Step 6: Present results

Format the output clearly:

```
## Gap Analysis: {title} @ {company}

**Overall Score: {score}/100**

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

## Step 7: Interactive coaching loop

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

After coaching, if any files were updated, re-run the scoring (recalculate
the overall score) and show the improvement:
"Score improved from {old}/100 to {new}/100 after coaching updates."

## Error handling

- If an experience file has malformed or missing frontmatter, still read its
  body text for semantic matching. Warn but do not skip.
- If the job file has no parseable skills (neither frontmatter nor body),
  warn the user that the posting may need manual enrichment via /res_parse.
- Never fabricate experience or skills. If a gap is real, say so directly.
  Honest assessment is more valuable than an inflated score.
