# Skills

One skill directory. The folder name matches `name` in `SKILL.md`.

```text
sensecraft-cli/
├── SKILL.md              # only entry
├── references/           # product procedures, opened when needed
├── create-app/           # project discovery → reviewed, deployable CodeCraft Hub app
│   ├── SKILL.md
│   ├── references/
│   └── assets/
```

`create-app/SKILL.md` routes to its project-discovery, content-review, and manifest references, plus the shared app-fields, tool-block, command, and upload references. Read each applicable reference before authoring that information. The workflow resolves user decisions, verifies BOM catalog matches through the API, builds a local manifest, reviews it, then uploads and verifies a private app. Update sync uses the same manifest.

Any `*-legacy/` directory is an archived draft, not an entry point or current reference. The npm packaging script excludes those directories.

`SKILL.md` links straight to a file in `references/` or to `create-app/`. Do not add another folder between them.

Install by symlinking this directory into the agent skills folder. Do not copy it. Steps: [`../INSTALL.md`](../INSTALL.md) §9.
