Export a resume (and matching cover letter) to DOCX or PDF.

Input: $ARGUMENTS — optional resume file path and/or flags:
- Path to a resume in `resumes/generated/`
- `--pdf` to export as PDF instead of DOCX (default: DOCX)
- `--template PATH` to use a Word reference document for styling

Parse $ARGUMENTS to extract the file path and flags.

Call `rezbldr_export` with:
- `source`: the resume path from $ARGUMENTS, or omit for latest
- `format`: `"pdf"` if `--pdf` flag is present, otherwise `"docx"`
- `template`: the `--template` value if provided

Report the results: file paths, sizes, format, and any errors.
If a cover letter was auto-detected and exported, report that too.

If `rezbldr_export` reports errors (e.g., pandoc not found, LaTeX missing for
PDF), show the error and suggest remediation.
