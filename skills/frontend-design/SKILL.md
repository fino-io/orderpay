---
name: frontend-design
description: Use when designing or reshaping the generated frontend business UI in a fino project. Apply it to frontend pages, workflows, dashboards, forms, empty states, and visual polish. Do not trigger for proto/backend work, generated API/client code, release packaging, CI repair, or ordinary non-visual bug fixes.
license: Complete terms in LICENSE.txt
---

# Frontend Design

Approach this as the design lead for a generated business application that needs to become a usable product interface. The goal is not a marketing page or decorative redesign; the goal is a clear, efficient, domain-specific UI that helps users complete real workflows.

## Ground it in the product

Before designing, identify the concrete product domain, the user role, and the screen's single job. Use the generated API/client shape as the source of truth for what data and actions exist, then design the smallest interface that supports the workflow.

If the brief is vague, choose conservative defaults for an operational tool: dense but readable information, restrained visual styling, predictable navigation, and controls that are easy to scan and repeat.

For an existing frontend, reuse its application shell, control vocabulary, and semantic theme tokens before introducing another visual system. Keep product names, navigation destinations, concrete palettes, and screen-specific rules in project code or documentation; this skill records reusable design decisions. Use `fino` when available for architecture, generated ownership, API integration, and code style.

## Design principles

Business UI should feel intentional, not templated. Use color, typography, layout, and motion to clarify hierarchy and state. Avoid oversized heroes, decorative card-heavy sections, visual filler, and one-note palettes.

Structure is information. Tables, filters, segmented controls, tabs, forms, detail panels, and empty states should match the user's task. Use cards only for repeated items, modals, and genuinely framed tools. Do not put cards inside cards. Numbered markers should represent a real sequence, and labels should convey a real category or scope; avoid adding eyebrows, arrows, or other recurring decoration without a task-specific purpose.

Typography carries hierarchy. Use a practical type scale, readable line lengths, and compact headings inside tool surfaces. Reserve hero-scale type for true first-run or landing contexts.

Motion should serve orientation. Prefer small, predictable transitions for state changes. Respect reduced motion. Favor motion that responds to user actions over repeated section entrances or ambient effects that compete with the working area.

## Layout and interaction

- Keep global navigation and theme controls in a stable application shell. Put the page title, scope, and primary action above its working area; distinguish global destinations from tabs that change the current view.
- For data-heavy pages, arrange context and filters before summaries and detailed results. Include summary metrics only when they help a decision; make their time range, scope, and units clear. Choose sections, tables, or a list/detail layout for the actual workflow rather than forcing every page into a dashboard grid.
- Keep filters together and show the active scope. Distinguish a dataset with no entries from filters with no matches, and offer the appropriate creation or reset action. Preserve useful selection and filter context while opening and closing details.
- Keep primary actions easy to find and secondary row actions quiet. Use links for navigation and buttons for state changes. Give icon-only controls accessible names, indicate selected tabs or pressed states, and keep all controls reachable by keyboard.
- Keep form labels visible, put validation beside the relevant input, retain entered values after failure, and show pending and success feedback at the action's location. For destructive or unsaved changes, use confirmation or undo appropriate to the cost of losing work.
- Adapt columns and filter rows to available space. Use shrinkable grid tracks and `min-width: 0` where needed; give wide tables their own horizontal scroll region. On narrow screens, stack regions or open details as a separate view while keeping a clear return path. Truncated content must remain available through an accessible detail or expansion control.
- Use semantic theme tokens for light/dark variants where supported. Pair status colors with text or icons; verify readable contrast and visible focus in each theme. Preserve the existing visual identity unless a redesign is requested.

## Process

Work in two passes.

First, write a compact design plan:

- Audience: the primary user and the job they need to finish.
- Layout: the main regions and how users move through them.
- Controls: filters, actions, forms, tabs, menus, and status surfaces needed for the workflow.
- Visual system: existing semantic tokens or a compact palette, typography roles, spacing density, focus/hover/error states.
- Signature: one domain-specific visual or interaction choice that makes the UI feel purpose-built without adding clutter.

Then critique the plan before coding. If any choice would fit any generic admin dashboard, revise it toward the actual domain or remove it.

## Implementation guidance

Use generated frontend conventions first. Keep generated API/client code authoritative. Do not hand-edit files marked as generated; change the source contract or generator upstream instead.

Build complete states for the workflow: loading, populated, empty, error, disabled, validation, and success feedback. Text should name what users recognize and what actions do. Use consistent action names in buttons, pending states, and results: “Save changes” → “Saving…” → “Changes saved.” Errors should explain the problem and next step; empty states should guide the appropriate action. Match the product language and tone, without filler or implementation jargon.

Keep responsive behavior explicit. Text must not overflow buttons, tables, cards, or panels. Controls should keep stable dimensions so hover states, labels, icons, and loading text do not shift layout.

Use familiar controls: icon buttons for tools, swatches for color, segmented controls for modes, toggles or checkboxes for binary settings, sliders/steppers/inputs for numbers, menus for option sets, tabs for views, and clear text buttons for commands.

## Self-critique

Before finishing, review desktop and narrow-screen behavior using the available evidence. Check that hierarchy is clear, controls are discoverable, data can be scanned, text fits, focus states exist, and the interface does not rely on decoration to explain itself. Use browser automation only when explicitly requested; otherwise review the implementation and supplied visuals, and report visual or interaction checks that remain unverified.
