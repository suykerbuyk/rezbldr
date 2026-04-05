Extract skill gaps from a job posting and create or update training files
in the `training/` directory.

Input: $ARGUMENTS — path to a job file. If omitted, the most recent file
in `jobs/target/` is used automatically.

## Step 1: Get ranked data and identify gaps

Call `rezbldr_rank` with:
- `job_file`: $ARGUMENTS if provided, otherwise `"latest"`
- `top_n`: 99 (we need the full picture, not just top files)

This returns `match_score` with:
- `required_misses` — required skills not matched by any experience file
- `preferred_misses` — preferred skills not matched
- `required_hits` — required skills that were matched
- `preferred_hits` — preferred skills that were matched

Tell the user which job file was selected (from `job.file_path`).

## Step 2: Classify gaps

Read `profile/skills.md` in parallel with any existing training files
(Glob `training/*.md`, read each one's frontmatter for `skill` and
`surfaced_by` fields).

For each skill in `required_misses` and `preferred_misses`:

1. **Check skills.md**: Does the skill appear? At what proficiency level?
2. **Classify**:
   - **Partial match**: Skill is in skills.md at a lower level, OR
     adjacent/transferable skill exists. Flag for training.
   - **Gap**: No evidence in skills.md or experience files. Flag for training.

Skip any skill that appears in `required_hits` or `preferred_hits` — the
scoring engine already confirmed these are covered.

For each gap/partial, note:
- Whether this is a **required** or **preferred** skill
- What adjacent/transferable skills exist (from skills.md)
- A brief note on why this skill matters for the role

## Step 3: Create or update training files

For each identified gap or partial match:

### Slugification rules for filename:
a. Lowercase the skill name.
b. Replace spaces, `/`, and `&` with hyphens.
c. Remove all characters except `a-z`, `0-9`, and `-`.
d. Collapse consecutive hyphens into one; trim leading/trailing hyphens.
e. Truncate to 40 characters (break at a hyphen boundary if possible).

Related tools that are commonly learned together (e.g., Terraform and Ansible)
may be combined into a single training file. Use judgment — combine when:
- They serve the same function (both are IaC tools)
- They appear together in most postings
- Learning one without the other provides little value

### If `training/{slug}.md` already exists:

1. Read the existing file.
2. Check if this job is already in the `surfaced_by` array (match on job
   title + company). If so, skip — already captured.
3. Add the new job to the `surfaced_by` array in frontmatter.
4. Recalculate priority using these rules:
   - **high**: Appeared as `required` in any job posting
   - **medium**: Appeared as `preferred` in 2+ job postings
   - **low**: Appeared as `preferred` in only 1 job posting
5. Update the `updated` date.
6. Add a row to the "Jobs Requiring This Skill" table at the bottom.
7. Do NOT modify the Learning Path section — the user may have customized
   it or checked off items. Preserve all existing content outside frontmatter
   and the jobs table.
8. Use the Edit tool to make targeted changes only.

### If `training/{slug}.md` does not exist:

Create a new file with this structure:

```markdown
---
skill: "{Skill Name}"
category: "{Category from skills.md or best guess}"
priority: {high|medium|low}
status: not-started
surfaced_by:
  - job: "{Job Title}"
    company: "{Company}"
    requirement: {required|preferred}
    date: "{YYYY-MM-DD}"
related_skills:
  - "{Adjacent skill 1}"
  - "{Adjacent skill 2}"
created: "{YYYY-MM-DD}"
updated: "{YYYY-MM-DD}"
---

## Why This Matters

{2-3 sentences explaining market relevance. Reference the specific job
posting(s). Explain how this skill connects to or extends existing strengths.}

## Current State

{Honest assessment of where the candidate stands. List adjacent skills with
proficiency levels from skills.md. Note what transfers and what doesn't.}

## Learning Path

### Tier 1: Foundations ({estimated time})
- [ ] {Concrete action items — prefer hands-on tasks over passive reading}
- [ ] {Tailor to existing home lab: 3-node XCP-NG cluster, 160 TiB ZFS,
      100GbE networking, Ceph experience}

### Tier 2: Applied / Domain-Specific ({estimated time})
- [ ] {Items connecting the new skill to existing expertise}
- [ ] {e.g., for K8s training: deploy Rook-Ceph to leverage existing Ceph
      knowledge}

### Tier 3: Production-Ready ({estimated time})
- [ ] {Items that would make this a "strong match" in future gap analyses}
- [ ] {Certification or portfolio project if applicable}

## Resources

- {Official documentation URL}
- {Best book or course for practitioners}
- {Any community/lab resources relevant to the skill}

## Jobs Requiring This Skill

| Job | Company | Requirement | Score Impact | Date |
|-----|---------|------------|-------------|------|
| {title} | {company} | {required/preferred} | {estimated pts} | {date} |
```

Guidelines for learning paths:
- Prefer hands-on lab work over passive reading
- Reference the candidate's home lab setup for practical exercises
- Build on existing skills — e.g., if learning Ansible, compare to SaltStack
  concepts the candidate already knows
- Tier 2 should connect the new skill to storage/infrastructure domain
  expertise specifically
- Tier 3 should target what would satisfy the "strong match" bar in a
  future `/res_match` run
- Keep timelines realistic for a working professional (part-time study)
- Include certification paths where they add credentialing value

## Step 4: Present summary

After processing all gaps, display:

```
## Training Items: {Job Title} @ {Company}

| Skill | Priority | Status | Jobs | File |
|-------|----------|--------|------|------|
| ...   | high     | not-started | 1 | training/{slug}.md |

### New training files created:
- training/{slug}.md — {brief description}

### Existing training files updated:
- training/{slug}.md — added {company} reference, priority unchanged

### ROI Assessment
{Which 1-2 training items would unlock the most job opportunities based on:
- Frequency across job postings in the surfaced_by arrays
- Alignment with career-objectives.md variants
- Time-to-competency given existing adjacent skills}
```

## Step 5: Suggest next steps

- "Run `/res_match` to re-score after completing training items"
- If the user just ran `/res_match`, remind them the training files are
  linked to that analysis
- Suggest which Tier 1 item to start with based on lowest effort / highest
  impact

## Error handling

- If `training/` directory doesn't exist, create it.
- If `rezbldr_rank` returns an error (no job files), tell the user to run
  `/res_parse` first.
- Never fabricate gaps. If the candidate has strong evidence for a skill,
  it is not a gap — even if it's not in skills.md.
- When in doubt about whether something is a gap or partial match, classify
  it as partial match and note the ambiguity.
