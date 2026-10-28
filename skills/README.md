# Skills

One skill directory. The folder name matches `name` in `SKILL.md`.

```text
sensecraft-cli/
├── SKILL.md              # only entry
├── references/           # product procedures, opened when needed
├── create-app/           # nested skill: local project → app
│   ├── SKILL.md
│   ├── references/
│   └── assets/
```

`SKILL.md` links straight to a file in `references/` or to `create-app/`. Do not add another folder between them.

Install by symlinking this directory into the agent skills folder. Do not copy it. Steps: [`../INSTALL.md`](../INSTALL.md) §9.
