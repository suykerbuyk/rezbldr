Generate a targeted resume from vault data, matched to a specific job posting.
Optionally generates a cover letter.

Input: $ARGUMENTS — optional job file path and/or flags:
- Path to a job file in jobs/target/
- `--cover-letter` to also generate a cover letter
- `--max N` to limit experience files included (default: 5)

Parse $ARGUMENTS to extract the job file path and flags. If no path is
provided, resolve to the most recent file in jobs/target/ (same logic as
/res_match step 1).

## Step 1: Load all source data

Read these files in parallel (multiple Read calls in one message):

1. **Job file**: Read and parse frontmatter (title, company, company_slug,
   required_skills, preferred_skills, domain, seniority) and full body text.

2. **Profile data**:
   - `profile/contact.md` — parse frontmatter: name, email, phone, location,
     linkedin, github, tagline
   - `profile/summary-core.md` — read the full professional identity narrative
   - `profile/skills.md` — parse the skills table
   - `profile/publications.md` — read for optional inclusion

3. **Experience files**: Use Glob to list all `experience/*.md`, then Read
   each one. Parse frontmatter and body text.

## Step 2: Select experience files

Run tag intersection scoring to rank experience files by relevance:

1. Build job keyword set from: required_skills + preferred_skills + tags
   (lowercased). If frontmatter fields are missing, extract from body text.
2. Build each experience file's keyword set from: tags + skills (lowercased).
3. Score: required match = 2.0, preferred = 1.0, tag = 0.5 per match.
4. Boost `highlight: true` by 10%. Penalize end date > 10 years ago by 30%.
5. Exclude `visibility: hidden` files.
6. Select top N files (default 5, or --max value).

Also include any `highlight: true` file that scored in the top 10, even if
outside the top N cutoff.

Present the selection to the user:

```
Building resume for: {title} @ {company}

Selected experience files (by relevance):
1. {role} @ {company} ({start}–{end}) — score: {N}
2. ...

Proceed? (y/n/adjust)
```

If "adjust": ask which files to add or remove, update the list, continue.

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

## Step 5: Validate

Before presenting to the user, check:

1. Professional Summary section exists and is 3-4 sentences
2. Word count is 600-800 (warn if outside, but proceed)
3. Every company name in the resume exists in the selected experience files
4. No skill is claimed that does not appear in skills.md or an experience file
5. Contact information matches profile/contact.md exactly
6. Heading hierarchy: exactly one h1, h2 for sections, h3 for roles

Report any validation warnings.

## Step 6: Write the resume file

**Naming convention**: Read `profile/contact.md` to get the candidate's name.
Lowercase and replace spaces with underscores (e.g., "John Suykerbuyk" →
"john_suykerbuyk").

Filename: `resumes/generated/{name}_{YYYY-MM-DD}-{company_slug}_resume.md`

Example: `resumes/generated/john_suykerbuyk_2026-04-03-reddit_resume.md`

If the file already exists, append `-v{N}` before `_resume.md` where N is
the next available version number.

Present the full resume to the user for review. Then ask:
"Write this resume to `{filepath}`? (y/n/edit)"

- "y": Write the file. Report the path.
- "edit": Ask what to change. Make edits. Present again.
- "n": Discard. Suggest adjustments and offer to regenerate.

Ensure `resumes/generated/` exists — create with `mkdir -p` via Bash if not.

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
resume_file: resumes/generated/{name}_{date}-{company_slug}_resume.md
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

Example: `cover-letters/john_suykerbuyk_2026-04-03-reddit_cover.md`

Present to the user for review before writing. Create the directory with
`mkdir -p cover-letters` via Bash if needed.

## Step 8: Suggest next steps

After writing, suggest:
- "Review the draft and set `status: ready` in frontmatter when satisfied"
- "Run `/res_export resumes/generated/{filename}` to render as DOCX or PDF"
- "Run `/res_match` first if you want to see the full gap analysis"
