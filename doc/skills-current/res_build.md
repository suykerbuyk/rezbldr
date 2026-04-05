Generate a targeted resume from vault data, matched to a specific job posting.
Optionally generates a cover letter.

Input: $ARGUMENTS — optional job file path and/or flags:
- Path to a job file in jobs/target/
- `--cover-letter` to also generate a cover letter
- `--max N` to limit experience files included (default: 5)

Parse $ARGUMENTS to extract the job file path and flags. If no path is
provided, the most recent file in jobs/target/ is used automatically.

## Step 1: Rank and select experience files

Call `rezbldr_rank` with:
- `job_file`: the job file path from $ARGUMENTS, or `"latest"`
- `top_n`: the `--max` value (default 5)

This returns `job` (title, company, file_path), `ranked[]` (experience files
sorted by relevance with scores), and `match_score`.

Also include any `highlight: true` file that scored in the top 10, even if
outside the top N cutoff.

Present the selection to the user:

```
Building resume for: {job.title} @ {job.company}

Selected experience files (by relevance):
1. {role} @ {company} ({start}-{end}) — score: {N}
2. ...

Proceed? (y/n/adjust)
```

If "adjust": ask which files to add or remove, update the list, continue.

## Step 2: Load source data

Read these files in parallel (multiple Read calls in one message):

1. **Job file**: Read `job.file_path` — parse frontmatter (title, company,
   company_slug, required_skills, preferred_skills, domain, seniority) and
   full body text.

2. **Profile data**:
   - `profile/contact.md` — parse frontmatter: name, email, phone, location,
     linkedin, github, tagline
   - `profile/summary-core.md` — read the full professional identity narrative
   - `profile/skills.md` — parse the skills table
   - `profile/publications.md` — read for optional inclusion

3. **Selected experience files**: Read each file from `ranked[].file`.
   Parse frontmatter and body text.

## Step 3: Synthesize the resume

Generate a complete, ATS-compatible resume following these rules precisely:

### Header
- Candidate name as h1
- Contact line: email | phone | location — end with `\` (hard line break)
- LinkedIn URL on its own line, end with `\` (hard line break)
- GitHub URL on its own line (no trailing backslash)
- IMPORTANT: Use markdown hard line breaks (`\` at end of line) to keep
  contact info and each link on separate lines. Do NOT join links with
  pipes on the contact line. Pandoc treats bare newlines as soft wraps,
  so without `\` everything collapses into one long unreadable line.

Example:
```
john@suykerbuyk.org | +1-303-578-2497 | Loveland, CO\
https://www.linkedin.com/in/john-suykerbuyk/\
https://github.com/suykerbuyk
```

### Professional Summary (3-4 sentences)
- Tailored specifically to THIS role at THIS company
- Open with years of experience and primary domain expertise
- Connect the candidate's career arc to what this role needs
- End with a differentiator or unique value proposition
- Draw from summary-core.md but rewrite for this job — do not copy verbatim

### Core Competencies
- Select 8-12 skills from skills.md most relevant to the job requirements
- Order by relevance to the job, not alphabetically
- Use exact skill names from skills.md (ATS keyword matching)
- Format as a single line with " | " separators

### Professional Experience
For each selected experience file, ordered by RELEVANCE TO THE JOB (not
chronologically):

```
### {Role}
**{Company}** | {Location} | {Start} – {End or "Present"}

- {Rewritten bullet with action verb and quantified impact}
- {3-5 bullets per role, fewer for older/less relevant roles}
```

Rules for experience bullets:
- Lead every bullet with a strong action verb (Architected, Designed, Led,
  Built, Deployed, Migrated, Scaled, Automated, Negotiated, etc.)
- Preserve exact metrics from source files — never inflate numbers
- If a metric is approximate, use conservative language ("~50 PiB", "approximately")
- Prioritize bullets that directly address the job's requirements
- Consolidate or omit bullets irrelevant to this specific role
- For roles > 10 years old: include ONLY if marked `highlight: true` AND
  they contain uniquely relevant skills. Use 1-2 bullets maximum.

### Publications & Industry Recognition (optional section)
- Include ONLY if publications.md has entries relevant to the job's domain
- Select the 2-3 most relevant entries
- Brief format: title, context, key metric or outcome

### ABSOLUTE RULES — DO NOT VIOLATE
1. Do NOT invent experience, metrics, companies, or skills not in source files
2. Every company name must appear in the experience files used
3. Every metric must trace back to a specific experience file
4. ATS compatible: no tables, no columns, no graphics, clean heading hierarchy
5. Target 600-800 words for the body (excluding frontmatter and header)
6. Clean Markdown only — h1 for name, h2 for sections, h3 for roles

## Step 4: Compose the output file

Wrap the resume in vault-tracking frontmatter:

```yaml
---
job_file: jobs/target/{job filename}
generated: "YYYY-MM-DDTHH:MM:SSZ"
model: claude-opus-4-6
status: draft
experience_files:
  - experience/{file1}
  - experience/{file2}
word_count: {actual word count of body below frontmatter}
version: 1
---
```

For the timestamp, use today's date with a reasonable time. For the model
field, use "claude-opus-4-6" (the model powering this session).

## Step 5: Resolve output path and write

Call `rezbldr_resolve` with:
- `type`: `"resume"`
- `action`: `"generate"`
- `slug`: the company_slug from the job frontmatter
- `date`: today's date (YYYY-MM-DD)

Use the returned `path` as the output filename. If `exists` is true, append
`-v{N}` before `_resume.md` where N is the next available version number.

Present the full resume to the user for review. Then ask:
"Write this resume to `{filepath}`? (y/n/edit)"

- "y": Write the file. Report the path.
- "edit": Ask what to change. Make edits. Present again.
- "n": Discard. Suggest adjustments and offer to regenerate.

Ensure `resumes/generated/` exists — create with `mkdir -p` via Bash if not.

## Step 6: Validate

Call `rezbldr_validate` with:
- `resume_path`: the path of the written resume file

Check the response:
- `word_count_ok` is false → warn about word count ({word_count} words)
- `heading_errors` is non-empty → fix heading hierarchy issues
- `unknown_skills` is non-empty → verify skills exist in skills.md
- `unknown_companies` is non-empty → verify companies match experience files
- `contact_match` is false → check contact info against profile/contact.md
- `warnings` → report any additional warnings

If there are fixable issues, fix them and re-validate.

## Step 7: Cover letter (if --cover-letter)

If the user requested a cover letter, generate one:

### Cover Letter Rules
1. **Strong hook** — NOT "I am writing to apply for..." or "Dear Hiring
   Manager, I am excited to..." Instead, lead with a specific technical
   insight, a shared challenge, or a concrete accomplishment that connects
   to the company's mission.
2. **Body (2-3 paragraphs)**: Connect 2-3 specific experiences to the job's
   key requirements. Reference concrete projects and quantified outcomes
   from the experience files. Demonstrate knowledge of the company and the
   specific role using details from the job posting body.
3. **Close**: Confident call to action — not passive ("I hope to hear from
   you") but direct ("I'd welcome the opportunity to discuss how my
   experience with X maps to your Y challenge").
4. **Length**: 3-4 paragraphs, 300-400 words total.
5. **Tone**: Professional but not stiff. Match the company's culture signals.
6. Do NOT repeat the resume. The cover letter adds narrative context.
7. Do NOT fabricate claims not supported by source files.

### Cover letter file format

```yaml
---
job_file: jobs/target/{job filename}
resume_file: resumes/generated/{resume filename}
generated: "YYYY-MM-DDTHH:MM:SSZ"
model: claude-opus-4-6
status: draft
---
```

Then the letter content:

```
{Today's date, formatted as Month Day, Year}

{Company}

Dear {Hiring Team or specific name if known},

{cover letter body — 3-4 paragraphs}

Sincerely,
{Name}
{email} | {phone}
{linkedin}
```

Write to: `cover-letters/{name}_{YYYY-MM-DD}-{company_slug}_cover.md`

Present to the user for review before writing. Create the directory with
`mkdir -p cover-letters` via Bash if needed.

## Step 8: Suggest next steps

After writing, suggest:
- "Review the draft and set `status: ready` in frontmatter when satisfied"
- "Run `/res_export` to render as DOCX or PDF"
- "Run `/res_match` first if you want to see the full gap analysis"
