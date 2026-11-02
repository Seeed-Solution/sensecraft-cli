# Project discovery before app authoring

The discovery result must explain what the project actually does and how a new user can deploy and use it. Collect enough evidence for the app's fields, tool configs, BOM, and copy before presenting decisions. Scale inspection to the project; exhaustive source review is unnecessary, but reading only its title and package metadata is insufficient.

## Inspect the actual project

Start with the conversation's established choices. Locate files with `rg --files`, then read the relevant files themselves. Skip dependency caches and generated trees unless they contain the deployment artifact. Use targeted source searches to check implementation claims; do not treat matching filenames as evidence of functionality.

The agent handles CLI inspection and auth setup through the [parent auth workflow](../../SKILL.md#auth). If authentication is missing, start the interactive flow yourself and continue independent file inspection while the user completes browser authorization; do not give them command-line setup tasks. `auth info` shows site/org/Key ID, not an account name. Use the completed wizard's account identity when available and leave unknown identity metadata unset.

| Question to resolve | Useful evidence | Affects |
|---|---|---|
| What problem does it solve, for whom, with what output? | Conversation, README, user docs, demo/screenshots, implemented entry points | Name, excerpt, scene, introduction |
| Which features exist now, and which are plans/examples? | Runtime code, routes/screens, tests, documented limitations | Claims and scope of copy |
| What must a user have or connect? | Hardware docs, wiring diagrams, board configs, model/device compatibility | BOM, prerequisites, target tool |
| What runs on each component? | Firmware configs, package scripts, Dockerfile/compose, Node-RED JSON, Blockly project reference | End-to-end deployment plan |
| Which exact deployable inputs exist? | Build output/release files, artifact metadata, target board/architecture and version | Firmware/model/container/flow config |
| How does a first-time user configure, start, check success, and stop/reset it? | Runtime setup UI, configuration schema, code and user docs | Ordered instructions and expected results |
| What is required outside the page? | Accounts/services, network, peripherals, browser/USB prerequisites | Feasibility and user decisions |
| What may be distributed? | LICENSE and artifact-specific notices, config examples, image origins | License, assets, clean artifact selection |
| Does the project fit an open activity or competition? | Verified purpose/hardware/features, existing user preferences, live activity list and API-provided rules/guides | Activity selection or none, eligibility gaps, required deliverables and permission choices |
| What is the existing remote identity and local history? | Manifest, git state, auth output, prior sync/review notes | Create vs update, environment, duplicate prevention |

Inspect configuration **names and mechanisms**, not secret values. A `.env.example` may describe user inputs; a populated `.env`, credential store, or private key is not upload material. Review distributable archives and binaries too, since a build can embed credentials.

For each prospective asset/artifact, record its relative path, role, existence, byte size, SHA-256, target/architecture when relevant, and intended block/field. Do not upload a whole repo just because the request says “upload my project”: select the deployable artifacts the supported tools actually consume.

## Separate facts, proposals, and unresolved choices

Maintain concise notes in `.sensecraft/<manifest-stem>.review.md`:

| Item | Evidence | Status / next action |
|---|---|---|
| Project capability or field candidate | File/section, implementation location, or explicit user decision | Verified, proposed, contradicted, or unknown |
| Deployment action | Supported tool + actual artifact/config source | Ready, planned upload, user input at deployment, or blocked |
| BOM item | Project requirement and actual catalog query/selected result | API-confirmed, custom, ambiguous, or unverified |
| Activity decision | Activity API row and guide/rules, dates, project evidence and user preferences | Selected with reason, no fit, uncertain eligibility/choice, or lookup unavailable; record join outcome after upload |

“Verified” means the evidence supports that specific statement. The presence of a camera library does not prove object counting. A Dockerfile does not prove a Fleet app ID or supported platform integration. A board family name does not establish the exact SKU, `uniform_type`, or flash address.

User input required at **deployment time** can be a valid supported path (for example a tool's documented prompt for a personal token). Explain that prompt without distributing the token. An authoring-time missing artifact or unknown device target remains a gap.

When docs and code disagree, inspect enough to identify the conflict. Use established user intent and implementation evidence to propose a resolution. Ask if the conflict changes the app promise, supported device, deployment path, permission, or distributed artifact; do not silently turn uncertain claims into polished facts.

## Ask after discovery has made the choice concrete

Read the field and tool-block references before proposing options. After discovery, run `app activity list` and read the guides/rules for plausible activities per [app-fields.md](../../references/app-fields.md#activity-selection-during-app-authoring); the public activity list does not need an Access Key. Use the gathered project facts to rank the candidates, then ask the user to choose an activity or none. After the agent establishes auth, resolve scene IDs and BOM catalog candidates through the real CLI reads. Record the queries, selected results, source links, and lookup time; offline/sandbox data is not live confirmation. Then batch remaining decisions with a recommendation and a short reason. Typical questions concern:

- Product framing or language when the conversation and project do not establish it.
- Clone and commercial-use choices; a project license alone does not answer these platform settings.
- Activity choice for every new app: which candidate to join, or none. Include actual dates, rules, fit and gaps, and a recommendation; never ask a generic opt-in question without candidates.
- Similar hardware variants, optional accessories, unknown quantities, or a custom product fallback.
- Missing deployable files, an unsupported action, or a choice to upload an explicitly incomplete draft.
- Cover selection or an asset with unclear redistribution rights.

Do not re-ask known answers, ask the user to supply information already in the project, or block unrelated local work while waiting. A proposed default is a recommendation, not a recorded user decision.
