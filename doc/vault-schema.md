# Vault Schema Reference

Exact YAML frontmatter schemas from the rezbldr vault. These define the
Go struct types for `internal/vault/`.

## Experience File (`experience/*.md`)

```yaml
---
role: Storage Solutions Architect          # string
company: Seagate Technology                # string
company_slug: seagate                      # string, lowercase-hyphenated
start: "2017"                              # string (year only)
end: "2023"                                # string (year only, or "Present")
current: false                             # bool
location: Longmont, CO                     # string
employment_type: full-time                 # string: full-time | contract | consulting
tags:                                      # []string, lowercase-hyphenated
  - storage
  - ceph
  - zfs
skills:                                    # []string, canonical names from skills.md
  - Ceph
  - ZFS
  - SaltStack
domain: storage                            # string
highlight: true                            # bool (boost in scoring)
visibility: resume                         # string: resume | hidden
created: "2026-03-30"                      # string, YYYY-MM-DD
updated: "2026-04-02"                      # string, YYYY-MM-DD
---
```

Body: Markdown with ## Summary, ## Key Contributions (bullets), ## Technologies.

## Job File (`jobs/target/*.md`)

```yaml
---
title: Senior Technical Program Manager    # string
company: FarmGPU                           # string
company_slug: farmgpu                      # string, lowercase-hyphenated
location: Rancho Cordova, CA               # string
type: Full-time                            # string: Full-time | Part-time | Contract
mode: Hybrid                               # string: Remote | Hybrid | On-site
seniority: Mid-Senior level                # string: Entry | Mid | Senior | Staff | Principal
domain: storage                            # string
source: https://...                        # string (URL), optional
parsed: 2026-04-02                         # string, YYYY-MM-DD
status: targeting                          # string: targeting | applied | rejected | interviewing
required_skills:                           # []string, optional (some files omit)
  - Ceph
  - Linux
preferred_skills:                          # []string, optional
  - Go
  - Python
culture_signals:                           # []string, optional
  - collaborative
tags:                                      # []string, lowercase-hyphenated
  - storage
  - ceph
  - hpc
compensation:                              # object, optional
  min: 120000                              # int
  max: 165000                              # int
  currency: USD                            # string
  equity: false                            # bool
---
```

Body: Markdown with ## Company Overview, ## Core Responsibilities,
## Required Qualifications, ## Preferred Qualifications, optional
## Culture & Values, ## Benefits.

## Generated Resume (`resumes/generated/*.md`)

```yaml
---
job_file: jobs/target/reddit-staff-software-engineer-storage.md  # string, relative path
generated: "2026-04-03T14:30:00Z"                                # string, ISO 8601
model: claude-opus-4-6                                           # string
status: draft                                                    # string: draft | ready
experience_files:                                                # []string, relative paths
  - experience/2017-seagate-solutions-architect.md
word_count: 712                                                  # int
version: 1                                                       # int
---
```

Body: Markdown resume (h1 name, h2 sections, h3 roles).

## Cover Letter (`cover-letters/*.md`)

```yaml
---
job_file: jobs/target/reddit-staff-software-engineer-storage.md     # string
resume_file: resumes/generated/john_suykerbuyk_..._resume.md        # string
generated: "2026-04-03T14:35:00Z"                                   # string, ISO 8601
model: claude-opus-4-6                                              # string
status: draft                                                       # string
---
```

Body: Plain letter (date, company, salutation, paragraphs, closing).

## Contact (`profile/contact.md`)

```yaml
---
name: John Suykerbuyk                      # string
email: john@suykerbuyk.org                 # string
phone: "+1-303-578-2497"                   # string
location: Loveland, CO                     # string
linkedin: https://...                      # string (URL)
github: https://...                        # string (URL)
tagline: "..."                             # string
languages:                                 # []string
  - English (native)
international_teams: Russia, Ukraine, India  # string (comma-separated)
---
```

## Skills (`profile/skills.md`)

Frontmatter: only `updated: "YYYY-MM-DD"`.

Body: Markdown table with header:
```
| Skill | Proficiency | Last Used | Years | Category |
```

Proficiency values: Expert, Advanced, Intermediate, Proficient, Capable.

## Training File (`training/*.md`)

```yaml
---
skill: "Kubernetes"                        # string
category: "DevOps"                         # string
priority: high                             # string: high | medium | low
status: not-started                        # string: not-started | in-progress | complete
surfaced_by:                               # []object
  - job: "Senior Storage Platform Engineer"  # string
    company: "Sony/PlayStation"              # string
    requirement: required                    # string: required | preferred
    date: "2026-04-04"                       # string, YYYY-MM-DD
related_skills:                            # []string
  - "Docker"
  - "Rook"
created: "2026-04-04"                      # string, YYYY-MM-DD
updated: "2026-04-04"                      # string, YYYY-MM-DD
---
```

Body: ## Why This Matters, ## Current State, ## Learning Path (Tier 1/2/3
with checkboxes), ## Resources, ## Jobs Requiring This Skill (table).

## File Naming Conventions

| Type | Pattern | Example |
|------|---------|---------|
| Experience | `{start_year}-{company_slug}-{role_slug}.md` | `2017-seagate-solutions-architect.md` |
| Job | `{YYYY-MM-DD}-{company_slug}-{title_slug}.md` | `2026-04-03-nvidia-JR2012991.md` |
| Resume | `{name}_{YYYY-MM-DD}-{company_slug}_resume.{ext}` | `john_suykerbuyk_2026-04-03-reddit_resume.md` |
| Cover | `{name}_{YYYY-MM-DD}-{company_slug}_cover.{ext}` | `john_suykerbuyk_2026-04-03-reddit_cover.md` |
| Training | `{skill_slug}.md` | `kubernetes.md` |

Where `{name}` is the candidate name from contact.md, lowercased with
spaces replaced by underscores.
