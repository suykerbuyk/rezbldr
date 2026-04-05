Export a resume (and optional cover letter) to DOCX or PDF via pandoc.

Input: $ARGUMENTS — optional resume file path and/or flags:
- Path to a resume in resumes/generated/
- `--pdf` to export as PDF instead of DOCX (default: DOCX)
- `--template PATH` to use a Word reference document for styling
- `--output PATH` to override the output file path

Parse $ARGUMENTS to extract the file path and flags.

## Step 1: Resolve the resume file

If a path is provided, use it. If it is just a filename (no directory),
prepend `resumes/generated/`.

If no path is provided:
1. Use Glob with pattern `*.md` in path `resumes/generated/`.
2. If no files exist: "No generated resumes found. Run `/res_build`
   first." Stop here.
3. Sort by filename (date-prefixed) and use the most recent.
4. Tell the user which file was selected.

Read the resume file. Verify it has frontmatter with `job_file` and
`generated` fields to confirm it is a generated resume.

## Step 2: Prepare markdown for pandoc

Pandoc needs clean markdown without our vault-tracking frontmatter.

1. **Strip YAML frontmatter**: Remove the `---` delimited block at the top.
   Pandoc has its own metadata handling; our frontmatter is for vault tracking.

2. **Verify heading hierarchy**:
   - Exactly one `# Name` (h1) at the top
   - `## Sections` (h2) for Professional Summary, Core Competencies, etc.
   - `### Roles` (h3) for individual experience entries
   - Fix any deviations.

3. **Normalize contact lines**: Ensure the contact info directly below h1
   is on one or two lines with pipe separators.

4. **Write to a temporary file**:
   ```
   mktemp /tmp/resume-export-XXXXXX.md
   ```
   Run via Bash to create the temp file, then write the cleaned content
   using the Write tool.

## Step 3: Check pandoc availability

Run `which pandoc` via Bash.

If not found:
- "pandoc is required for export but was not found."
- Suggest: `sudo pacman -S pandoc` (Arch) or `sudo apt install pandoc`
- Stop here.

For PDF export (`--pdf` flag), also check for a LaTeX engine:
- Run `which xelatex` via Bash.
- If not found: "PDF export requires XeLaTeX. Install `texlive-xetex` or
  use DOCX export instead."
- If available, proceed.

## Step 4: Determine output path

If `--output` was specified, use that path.

Otherwise:
- Base name: same as the input filename, without `.md`
- Extension: `.docx` (default) or `.pdf` (if --pdf)
- Directory: same as the input file (`resumes/generated/`)

**Naming convention**: All exported files (resume and cover letter) must
follow the pattern `{name}_{date}-{company_slug}_{type}.{ext}` where:
- `{name}` is the candidate's name from `profile/contact.md`, lowercased
  with spaces replaced by underscores (e.g., `john_suykerbuyk`)
- `{date}-{company_slug}` is the date and company slug from the source
  markdown filename
- `{type}` is `resume` or `cover`
- `{ext}` is `md`, `docx`, or `pdf`

If the input markdown file already follows this convention, derive the base
name from it. If it uses the old convention (e.g., `2026-04-03-reddit.md`),
construct the new name by reading `profile/contact.md` for the candidate name.

## Step 5: Run pandoc

### For DOCX:
```bash
pandoc {temp_file} -o {output_path} \
  --from markdown \
  --to docx \
  -V geometry:margin=0.75in
```

If `--template` was specified, add: `--reference-doc={template_path}`

### For PDF:
```bash
pandoc {temp_file} -o {output_path} \
  --from markdown \
  --to pdf \
  --pdf-engine=xelatex \
  -V geometry:margin=0.75in \
  -V fontsize=11pt \
  -V mainfont="Liberation Sans"
```

Run the command via Bash with a 60-second timeout. Capture stdout and stderr.

## Step 6: Handle results

**On success:**
1. Clean up temp file: `rm {temp_file}` via Bash.
2. Verify output exists and report size: `ls -lh {output_path}` via Bash.
3. Report: "Exported to: {output_path} ({size})"

**On failure:**
1. Show the full pandoc error output.
2. Diagnose common issues:
   - "Resource not found" → missing LaTeX package
   - "Could not find reference doc" → template path is wrong
   - Unicode errors → suggest xelatex engine or a different font
3. Do NOT clean up the temp file on failure (user may want to inspect it).
4. If PDF failed, suggest trying DOCX instead.

## Step 7: Cover letter export

Check for a matching cover letter:
- Use Glob to search `cover-letters/` for a file matching the same date
  and company slug as the resume. Files follow the naming convention
  `{name}_{date}-{company_slug}_cover.md`.

If found, ask: "A matching cover letter exists at `{path}`. Export it too?"

If confirmed:
1. Read the cover letter file.
2. Strip frontmatter (same as step 2).
3. Write to temp file.
4. Run pandoc with same format and template settings.
5. Report the output path.

## Step 8: Clean up

Ensure all temp files are removed after successful exports.

Report final summary:
```
Export complete:
  Resume:       {path} ({size})
  Cover letter: {path} ({size})  ← if applicable
```
