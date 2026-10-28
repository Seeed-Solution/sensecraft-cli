# Write and review the app for its readers

Read this before drafting any human-facing app content. Read the [field reference](../../references/app-fields.md) before filling fields and the entire [tool-block reference](../../references/tool-blocks.md) before writing deployment instructions. This guide supplements their schema and device rules; it does not replace them.

## Give each field a purpose

| Content | Reader's question | Writing criteria |
|---|---|---|
| `app_name` | What is this app called? | Preserve its established name; propose renaming rather than adding unexplained device/marketing suffixes. Max 60 UTF-16 code units. |
| `excerpt` | Is this useful to me? | One or two clear sentences: implemented capability, audience/use case, useful outcome. Max 300 UTF-16 code units. Omit changelog, deployment sequence, framework inventory, and unsupported performance claims. Mention hardware only when it helps the reader understand the app. |
| Introduction blocks | What will I be able to do, and what should I know first? | At most two short paragraphs that add context or essential limitations. Do not repeat the excerpt verbatim or paste a developer README. |
| Step title | What action comes next? | A concrete verb/action. No manual step number; the UI supplies it. |
| Step instructions | What do I do here, and how do I know it worked? | Necessary prerequisites, actual tool/button/input, and an observable outcome. Put the relevant tool immediately after the instruction that uses it. |
| BOM note | Why do I need this part? | Its role in this app, quantity/variant clarification, and optional status where relevant. No copied catalog advertising. |

Use the user's/project's established language; ask if a consequential audience/language choice is unclear. Explain unfamiliar product-specific terms only where the reader needs them. Delete filler and superlatives whose meaning or evidence cannot be stated.

Example, assuming source evidence confirms the behavior:

- Weak excerpt: “An innovative ESP32-S3 + FOMO + Arduino project. Flash the bin and connect USB. v1.2.”
- Useful excerpt: “Count jars passing a small conveyor and view the running total on the display. Built for makers testing a simple packing-line counter.”

Do not reuse the example's claims for another project. Avoid specific accuracy, speed, compatibility, or ease-of-use promises without evidence.

## Review the deployment as a complete journey

Before serializing blocks, outline each necessary user action with prerequisites, supported tool, verified inputs, expected result, and readiness. Choose tools because they support the project, rather than inserting every available tool type.

Start from a reader with the listed hardware and finish with the promised function running. Cover required connection/preparation, deploy/flash/install, per-user setup, first use, and stop/reset only where the project supports and needs them. Usually a few distinct actions suffice; do not force a step count or omit necessary actions to meet one.

Cross-check the plan against the real artifacts and device configuration:

- Is the exact board/variant supported by this tool and artifact? Are SKU, device type, address, architecture, and version sourced correctly?
- Can the reader carry out every action through the documented page tools and explicit preparation? Developer compilation commands are not a substitute for an available deployment tool.
- Does each required tool have usable inputs, or a documented prompt that allows the reader to supply them? A known local artifact awaiting this session's upload differs from an artifact nobody has provided.
- Are the instructions consistent with actual configuration prompts, reset behavior, and the tool's expected output? Do not invent a Wi-Fi setup screen, login flow, stop command, or success message.
- Are missing inputs visible as gaps? Empty URLs, invented IDs, placeholder containers, and instructions saying “deploy” without a functioning path do not establish a deployable app.

A needed but incomplete tool stays in the local draft. Resolve the missing input, or obtain the user's decision to save an incomplete draft and report it as such. Configuring a block does not execute it on hardware.

## Run an editorial pass on the assembled manifest

Review the actual final values as continuous reader-facing content, including compiled text and the order of tools. This pass is separate from running the validator.

1. **Truth:** Trace claims and compatibility statements to inspected files or user decisions. Remove roadmap features and unsupported promises. Preserve meaningful limitations.
2. **Clarity:** Resolve vague references such as “configure it” or “click deploy” into the actual object, input, and action. Keep terms and device names consistent.
3. **Flow:** Check prerequisites appear before use, the tool is in the right step, success can be observed, and the reader reaches the promised outcome.
4. **Economy:** Remove repeated introductions, duplicate steps, internal development history, and spec/menu dumps. Keep enough detail to complete each action.
5. **Presentation:** Inspect HTML/plain-text content after Markdown compilation. Check step titles, links/media, code vs prose, and accidental raw Markdown. Ensure BOM role notes and gallery order match the plan.

Fix defects before showing the final review. Record material corrections and any remaining gap in `.sensecraft/<manifest-stem>.review.md`. Preserve brief source pointers and BOM lookup evidence so a resumed session can reassess the decisions.

Present the concrete result before upload: final name/excerpt and permission choices; selected activity name/ID with dates, requirements and fit evidence, or reason for no participation; ordered steps with tool types; BOM variants/quantities and confirmation status; selected cover/assets; manifest link and remaining deployment/publishing gaps. Activity selection must follow [app-fields.md](../../references/app-fields.md#activity-selection-during-app-authoring): do not claim eligibility from a title alone or silently change permissions to qualify. Record remaining entry deliverables and distinguish selection, accepted join, verified association, and completed competition submission. The review file can carry the full copy and evidence when chat would become unwieldy. Ask only for outstanding material decisions or required authorization; do not add a ceremonial approval round for an already authorized create.

Record readiness precisely: **format validated**, **catalog verified**, **editorially reviewed**, and **deployment inputs complete** are separate findings. Include validation warnings, custom/unverified BOM rows, pending uploads versus missing artifacts, and untested hardware execution in the review. A coverless private app can have complete deployment inputs. A private app with a missing firmware URL cannot be described as ready to deploy. Run all CLI checks yourself; ask the user for unresolved decisions or required browser/device actions.
