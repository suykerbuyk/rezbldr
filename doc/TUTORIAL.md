<!-- Copyright (c) 2026 John Suykerbuyk and SykeTech LTD -->
<!-- SPDX-License-Identifier: MIT OR Apache-2.0 -->

# Getting Started with rezbldr

This tutorial walks you through installing rezbldr, connecting it to Claude
Code, and using it to build tailored resumes from the command line. You do not
need to know Go, MCP, or Obsidian to follow along.

## Prerequisites

You need four tools installed before you begin.

**Go 1.22+** -- The Go compiler builds rezbldr from source. Download it from
<https://go.dev/dl/> or install via your system package manager. Verify with
`go version`.

**pandoc** -- Converts markdown resumes into DOCX and PDF files. Install from
<https://pandoc.org/installing.html> or your package manager (e.g.,
`sudo apt install pandoc`, `brew install pandoc`). Verify with
`pandoc --version`.

**git** -- Tracks changes to your resume vault and pushes to remote
repositories. You almost certainly have this already. Verify with
`git --version`.

**Claude Code CLI** -- The AI assistant that orchestrates the resume pipeline.
rezbldr runs as a tool server that Claude Code calls during resume builds.
Install from <https://docs.anthropic.com/en/docs/claude-code>.

## Install rezbldr

### From source

```
git clone https://github.com/suykerbuyk/rezbldr.git
cd rezbldr
make build
make install
```

`make build` compiles the binary with version metadata. `make install` copies
it to your `$GOPATH/bin`.

### From Go directly

```
go install github.com/suykerbuyk/rezbldr/cmd/rezbldr@latest
```

This downloads, builds, and places the binary in your `$GOPATH/bin` in one
step.

### Verify the installation

```
rezbldr version
```

You should see output like:

```
rezbldr v0.5.0 (commit: fa3651c, built: 2026-04-05)
```

If the command is not found, ensure `$GOPATH/bin` is in your `$PATH`.

## Register with Claude Code

Run:

```
rezbldr install
```

This writes an MCP server entry into `~/.claude/settings.local.json` so that
Claude Code knows how to launch rezbldr. The resulting JSON looks like this:

```json
{
  "mcpServers": {
    "rezbldr": {
      "command": "/home/you/go/bin/rezbldr",
      "args": ["serve"],
      "env": {}
    }
  }
}
```

rezbldr auto-detects the vault at the default location
(`~/obsidian/RezBldrVault`). If your vault is elsewhere, pass the path
explicitly:

```
rezbldr install --vault /path/to/your/vault
```

This adds `--vault /path/to/your/vault` to the `args` array in the settings
file so that every MCP session uses the correct vault path.

## Verify your setup

Run:

```
rezbldr check
```

Example output when everything is working:

```
[✓] go: go1.22.4
[✓] pandoc: pandoc 3.1.11
[✓] git: git version 2.45.0
[✓] vault: /home/you/obsidian/RezBldrVault
[✓] vault-structure: found profile, jobs/target, resumes
[✓] contact: profile/contact.md
[✓] claude-settings: rezbldr registered in settings.local.json
```

If something fails:

| Check | Fix |
|-------|-----|
| `go` | Install Go 1.22+ from <https://go.dev/dl/> |
| `pandoc` | Install pandoc from <https://pandoc.org/installing.html> |
| `git` | Install git from your package manager |
| `vault` | Create the vault directory or pass `--vault` to point to it |
| `vault-structure` | Create the missing subdirectories (see next section) |
| `contact` | Create `profile/contact.md` in your vault |
| `claude-settings` | Run `rezbldr install` |

## Set up a vault

The vault is an Obsidian-compatible directory of markdown files. Each file
uses YAML frontmatter for structured metadata. The minimum directory structure
is:

```
vault/
  profile/
    contact.md          -- your name, email, phone, location, links
    skills.md           -- skill inventory as a markdown table
  jobs/
    target/             -- job postings as markdown with YAML frontmatter
  resumes/
    generated/          -- where rezbldr writes output resumes
  experience/           -- one file per role in your career history
  cover-letters/        -- generated cover letters
  training/             -- skill gap training plans
```

Here is a minimal example of a job posting file at
`jobs/target/2026-04-05-acme-senior-engineer.md`:

```yaml
---
title: Senior Software Engineer
company: Acme Corp
company_slug: acme
location: Remote
domain: backend
status: targeting
required_skills:
  - Go
  - PostgreSQL
preferred_skills:
  - Kubernetes
  - Terraform
tags:
  - backend
  - distributed-systems
---

## Core Responsibilities

- Design and build backend services
- ...
```

For complete YAML schemas covering all file types (experience, resume, cover
letter, contact, skills, training), see
[doc/vault-schema.md](vault-schema.md).

## The workflow

When you build a resume, Claude Code and rezbldr work together. You trigger
the process; the AI handles creative writing while rezbldr handles
computation. Here is what happens step by step:

1. **Add a job posting.** Save a markdown file with YAML frontmatter in
   `jobs/target/`. Include the required skills, preferred skills, and tags.

2. **Trigger a build.** In Claude Code, run the build command (e.g.,
   `/res_build`). Claude takes over from here.

3. **Resolve the job file.** Claude calls `rezbldr_resolve` to find the job
   posting in your vault.

4. **Rank your experience.** Claude calls `rezbldr_rank` to score every
   experience file against the job posting using tag-intersection scoring.
   Required skill matches count double, preferred skills count single, and
   tag overlaps add a half point each. Recent and highlighted experiences
   get a boost.

5. **Select and write.** Claude picks the top-scoring experiences and writes
   a tailored resume, choosing which bullets to emphasize and how to frame
   your background for the specific role.

6. **Validate.** Claude calls `rezbldr_validate` to check the generated
   resume: word count (600-800 range), heading hierarchy, skills against
   your inventory, company names against your experience records, and
   contact information.

7. **Iterate if needed.** If validation finds issues, Claude fixes them.
   During coaching loops, `rezbldr_score_diff` shows how vault edits
   improve your match score.

8. **Export.** Claude calls `rezbldr_export` to produce DOCX and PDF files
   via pandoc. A matching cover letter is automatically exported if one
   exists.

9. **Commit.** Claude calls `rezbldr_wrap` to stage the new files, commit
   them to git, and push to all configured remotes.

You do not call these tools manually. Claude Code orchestrates the entire
pipeline. Your job is to provide the job posting, answer coaching questions
about your experience, and approve the final result.

## Troubleshooting

**"vault not found"** -- rezbldr cannot locate your vault directory. Run
`rezbldr check` to see which path it is looking for. Either create the
directory at the default location (`~/obsidian/RezBldrVault`) or specify a
custom path:

```
rezbldr install --vault /your/actual/vault/path
```

**"pandoc not found"** -- Install pandoc. On Debian/Ubuntu:
`sudo apt install pandoc`. On macOS: `brew install pandoc`. On Arch:
`sudo pacman -S pandoc`. Then run `rezbldr check` to confirm.

**Tools not appearing in Claude Code** -- Run `rezbldr install` to register
the MCP server, then restart Claude Code. Check that the registration
exists:

```
rezbldr check
```

Look for the `claude-settings` line. If it shows `warn`, the registration
did not take effect.

**MCP connection errors** -- Verify that rezbldr can start without errors:

```
echo '{}' | rezbldr serve 2>/dev/null
```

If this produces an error, check that your vault path is valid and that
`profile/contact.md` exists in it. rezbldr validates the vault at startup.

## Next steps

- [README.md](../README.md) -- Tool reference and project overview
- [ARCHITECTURE.md](ARCHITECTURE.md) -- Package structure, data flow
  diagrams, and design decisions
- [vault-schema.md](vault-schema.md) -- Complete YAML frontmatter schemas
  for every vault file type
