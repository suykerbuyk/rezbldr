Parse a job posting into a structured vault file.

Input: $ARGUMENTS — a URL, file path, or the literal text "paste" to enter
job description text interactively.

## Step 1: Acquire the job posting content

Determine the input type from `$ARGUMENTS`:

- **URL** (starts with http:// or https://): Use the WebFetch tool to fetch
  the page content. Use this prompt: "Extract the complete job posting text
  from this page. Include the job title, company name, location, all
  responsibilities, all requirements (required and preferred), compensation
  if listed, and any company culture or values information. Return the full
  text, not a summary."
  If WebFetch fails (auth wall, 404, redirect loop), tell the user and ask
  them to paste the job description text directly.

- **File path** (ends in .md, .txt, .html, or contains /): Read the file
  using the Read tool. If the file does not exist, report the error and stop.

- **"paste" or empty**: Ask the user to paste the full job description text.
  Wait for their response before continuing.

- **Anything else**: Treat it as pasted job description text directly.

Store the raw content for use in later steps.

## Step 2: Analyze and extract structured data

Read the raw job posting content and extract these fields. Use your judgment
for each — do not guess or fabricate. If a field cannot be determined from
the posting, omit it from the frontmatter entirely.

| Field | Type | Notes |
|-------|------|-------|
| title | string | Exact job title from posting |
| company | string | Full company name |
| company_slug | string | Lowercase, hyphenated (e.g., "acme-corp") |
| location | string | As listed; include Remote/Hybrid/On-site if specified |
| type | string | Full-time, Part-time, Contract |
| mode | string | Remote, Hybrid, On-site |
| seniority | string | Entry, Mid, Senior, Staff, Principal, Director |
| domain | string | Primary domain (storage, infrastructure, backend, etc.) |
| source | string | URL if input was a URL, otherwise omit |
| parsed | string | Today's date in YYYY-MM-DD format |
| status | string | Always "targeting" for new parses |
| required_skills | string[] | Skills explicitly listed as required/must-have |
| preferred_skills | string[] | Skills listed as preferred/nice-to-have |
| culture_signals | string[] | Key cultural values or work style indicators |
| compensation.min | number | Low end of range (annual, in base currency) |
| compensation.max | number | High end of range |
| compensation.currency | string | USD, EUR, etc. |
| compensation.equity | boolean | Whether equity/stock is mentioned |
| tags | string[] | Lowercase hyphenated keywords for tag matching |

For `tags`: generate a comprehensive list of lowercase, hyphenated keywords
that capture the role's domain, technologies, and focus areas. These are used
for tag intersection matching against experience files. Include both specific
technologies (nvme, rdma, ceph) and broader domain terms (storage, hpc,
gpu-infrastructure). These tags are critical for the /res_match skill's scoring.

For `required_skills` and `preferred_skills`: use canonical skill names from
the posting. These will be matched against `profile/skills.md` and experience
file `skills` arrays.

## Step 3: Generate the filename

Construct the filename: `YYYY-MM-DD-{company_slug}-{title_slug}.md`

- Date: today's date
- company_slug: from step 2
- title_slug: lowercase the title, replace spaces with hyphens, remove
  special characters, truncate to 40 characters at a word boundary

Full path: `jobs/target/{filename}`

## Step 4: Compose the job file

Structure the file as:

```
---
{YAML frontmatter from step 2}
---

# {title}

## Company Overview

{2-3 sentence summary of the company and what they do, based on the posting}

## Core Responsibilities

{Organized list of responsibilities from the posting. Group into subsections
if the posting has natural groupings. Use the posting's own structure where
possible.}

## Required Qualifications

{Bulleted list of required qualifications, preserving the posting's detail}

## Preferred Qualifications

{Bulleted list of preferred/nice-to-have qualifications}

## Culture & Values

{Cultural signals, work style, team dynamics mentioned in the posting.
Omit this section if no culture information is available.}

## Benefits

{Benefits if mentioned, otherwise omit this section entirely.}
```

Preserve the substance of the original posting in the body sections. Do not
summarize away detail — the body is used as context for gap analysis and
resume synthesis. Reorganize for clarity but do not omit requirements or
responsibilities.

## Step 5: Confirm and write

Present to the user:
1. The proposed filename
2. The extracted frontmatter (formatted as YAML)
3. A brief summary: "{title} at {company}, {location}, {seniority}"
4. Count of required vs preferred skills extracted

Ask: "Write this job file? (y/n)"

If confirmed, write the file using the Write tool. Report the full path.

If the file already exists, warn the user and ask whether to overwrite or
choose a different name.

Ensure the `jobs/target/` directory exists before writing — if not, create it
with `mkdir -p jobs/target` via Bash.

## Step 6: Suggest next steps

After writing, suggest:
- "Run `/res_match jobs/target/{filename}` to see how your experience aligns"
- "Or run `/res_build jobs/target/{filename}` to generate a targeted resume directly"

## Error handling

- If WebFetch returns content that does not look like a job posting (no title,
  no requirements section), tell the user and ask them to paste the content.
- Never fabricate skills or requirements not present in the source text.
- If compensation is not listed, omit the compensation block entirely.
