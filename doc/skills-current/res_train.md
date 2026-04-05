Extract skill gaps from a job posting and create or update training files
in the `training/` directory.

Input: $ARGUMENTS — path to a job file. If omitted, the most recent file
in `jobs/` is used automatically.

## Step 1: Resolve the job file

If `$ARGUMENTS` is provided and is a valid path, use it. If it is just a
filename (no directory prefix), prepend `jobs/target/`.

If `$ARGUMENTS` is empty or omitted:
1. Use Glob with pattern `*.md` in path `jobs/` to list all job files
   (search recursively including `jobs/target/`).
2. If no files exist, tell the user: "No job files found. Run `/res_parse`
   first to create one." Stop here.
3. Sort by filename (date-prefixed, so lexicographic = chronological).
4. Use the most recent file. Tell the user which file was selected.

Read the job file using the Read tool. Parse its frontmatter and body to
extract: title, company, required skills, preferred skills.

If `required_skills` is missing from frontmatter, extract skill requirements
directly from the body text (look for "Requirements", "Required Qualifications",
"Skills & Knowledge", or similar sections and pull out technology names and
skill terms).

## Step 2: Load vault data

Read these files in parallel (use multiple Read tool calls in one message):

1. **Skills inventory**: Read `profile/skills.md` — parse the markdown table
   into skill entries (Skill, Proficiency, Last Used, Years, Category).

2. **Experience files**: Use Glob with pattern `*.md` in path `experience/`
   to list them, then Read each file. Parse frontmatter fields: tags, skills.
   Also scan body text for technology mentions.

3. **Existing training files**: Use Glob with pattern `*.md` in path
   `training/` to list any existing training files. Read each one's
   frontmatter to get the `skill` field and `surfaced_by` array.

## Step 3: Identify gaps

For each required and preferred skill in the job posting:

1. **Check skills.md**: Does the skill appear? At what proficiency level?
2. **Check experience files**: Is there concrete evidence (specific projects,
   technologies used, metrics)?
3. **Classify**:
   - **Strong match**: Skill in skills.md at Advanced/Expert AND backed by
     experience evidence. Skip — no training needed.
   - **Partial match**: Skill in skills.md at lower level, OR adjacent skill
     exists but not the specific tool. Flag for training.
   - **Gap**: No evidence in skills.md or experience files. Flag for training.

For partial matches and gaps, also note:
- Whether this is a **required** or **preferred** skill
- What adjacent/transferable skills exist
- A brief note on why this skill matters for the role

## Step 4: Create or update training files

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

## Step 5: Present summary

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

## Step 6: Suggest next steps

- "Run `/res_match` to re-score after completing training items"
- If the user just ran `/res_match`, remind them the training files are
  linked to that analysis
- Suggest which Tier 1 item to start with based on lowest effort / highest
  impact

## Error handling

- If `training/` directory doesn't exist, create it.
- If a job file has no parseable skills, warn the user and suggest
  `/res_parse` to enrich it.
- Never fabricate gaps. If the candidate has strong evidence for a skill,
  it is not a gap — even if it's not in skills.md.
- When in doubt about whether something is a gap or partial match, classify
  it as partial match and note the ambiguity.
