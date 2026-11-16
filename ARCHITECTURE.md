# Workbench architecture decision — 2026-11-16

## Decision

Retain Go and Wails. Introduce React + TypeScript for the workbench shell. Migrate service UI controllers gradually; do not rewrite working SQLite/media/Word/credential code to change the layout. No backend Rust migration in v0.4.8.

The shell owns the topbar, left navigation/history column, main workspace slot, optional evidence inspector and settings entry. Existing controllers own their material/answer/player DOM islands and requests. `Slot` preserves element identity; React must not reconcile an island's interior. This is an explicit migration boundary, not a claim that the entire frontend is React. Layout CSS is separate from service styles. Generated bundle is embedded in Go; production requires neither npm nor Node. CI checks TypeScript and regenerates bundle on Windows and Mac.

Tasks are separate sessions and independent job objects, persisted in module_task. Snapshot store remains for whole-workbench import/recovery. Lecture SQLite is read-only; user materials and tasks are separate. Optimistic revision conflict creates a restored copy rather than overwriting a newer record. No automatic replay of interrupted model calls.

## Primary-source comparison

These are selected representative projects, not a measured market-share claim.

| Project | Observed stack | Relevant tradeoff |
|---|---|---|
| MaClaw | Go/Wails/React | Closest to our existing native stack; module/session separation is useful. |
| AnythingLLM | Vite/React; desktop downstream Electron wrapper | Rich JS ecosystem; own Chromium/Node runtime costs and update responsibility. |
| Dify | React/Next.js frontend | Useful web application patterns; SSR/server deployment is not needed by this local desktop shell. |
| Open WebUI | SvelteKit | Another viable component framework; not proof React is universally superior. |
| Tauri | Rust + system WebView | Small runtime distribution, strong native integration; migration requires retesting platform behavior and backend services. |

Wails also uses the system WebView. Switching to Rust alone will not improve document layout, network model latency, or SQLite queries. Electron supplies a more consistent Chromium runtime; Wails/Tauri rely on per-platform WebViews and need cross-platform tests. Actual RAM/latency comparisons require the same workload and were not benchmarked in this study.

Sources inspected:
- https://github.com/RapidAI/MaClaw/blob/main/README_EN.md
- https://github.com/Mintplex-Labs/anything-llm/blob/master/CONTRIBUTING.md
- https://github.com/langgenius/dify/blob/main/web/README.md
- https://github.com/open-webui/open-webui/blob/main/package.json
- https://v3.wails.io/concepts/architecture/
- https://v2.tauri.app/concept/architecture/
- https://www.electronjs.org/docs/latest/tutorial/process-model

## SlideMuse evaluation (not integrated in this release)

SlideMuse is an agent skill and Python production pipeline for presentations, not a full teaching competition knowledge workbench. MIT license allows reuse with copyright/license retained; bundled third-party code must keep its own notices. It relies on host capabilities for visual comprehension and image generation. Do not execute its sponsor onboarding or installer just to research the repository.

Useful future integration boundary: a PPT production service accepting page-level semantic specifications (claims, exact text, data and source IDs), separate from visual Scene geometry; four style previews; editable chart/table/text generation; change dependencies; affected-page rerendering and version-bound validation. Numeric changes propagate only through explicitly declared bindings. Current user reports and evidence must remain the semantic source, never inferred from generated images.

Competition-scenarios includes teaching pain points, curriculum redesign, evaluation, evidence and transferability, but is not official competition scoring evidence. Full editable reconstruction requires render-and-review and cannot guarantee hidden details or complete fidelity. No image-generation key or sponsor service is automatically selected.

Sources:
- https://github.com/helloo1568/slidemuse
- https://github.com/helloo1568/slidemuse/blob/main/LICENSE
- https://github.com/helloo1568/slidemuse/blob/main/docs/competition-scenarios.md
- https://github.com/helloo1568/slidemuse/blob/main/references/page-spec.md
- https://github.com/helloo1568/slidemuse/blob/main/references/data-bindings.md
