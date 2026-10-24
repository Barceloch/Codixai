# Feature: Phase 7.1 — Audit interface (Wails v2 + Svelte 5 + Vite + Tailwind + daisyUI)

Objective: implement P7-06 / 7.1 from the specification — the audit UI bound to `application.RunAudit`, bilingual (English/Spanish), light/dark theme selector, with pnpm and frozen lockfile discipline.

Binding:
- ADR 0011: pnpm is the package manager; Svelte/Vite/Tailwind/daisyUI land now (not in the vanilla spike). Exact versions pinned at init. `{@html}` forbidden; `svelte/no-at-html-tags` enabled.
- ADR 0010: single binary CLI+UI via `-tags ui`.
- Architecture invariant: `domain`/`application` never import Wails/ui; frontend types generated from Go or consumed via the Wails bridge (no hand-written duplicate types).

Scope:

- [x] T1 — Bootstrap frontend with `pnpm create vite` (Svelte + TypeScript), add Tailwind CSS + daisyUI, lock versions, configure `frontend:install`/`frontend:build` in `wails.json` (pnpm), generate bindings (`wails generate module` / bindings).
  - `frontend/` with Svelte 5.57.0, Vite 8.3.0, TypeScript 5.9.3, Tailwind 4.3.3, daisyUI 5.7.42; exact versions and `pnpm-lock.yaml`.
  - `wails.json` uses `pnpm install --frozen-lockfile` and `pnpm run build`; `wailsjsdir ./frontend/src/lib`.
  - Bindings generated with `go run -tags "ui bindings" ./cmd/codixai ui --root .`.
- [x] T2 — i18n (English/Spanish), persisted locale, language selector in UI.
  - `frontend/src/lib/i18n.svelte.ts`, `localStorage` key `codixai.locale`, selector in `Header.svelte`.
- [x] T3 — Theme (light/dark) with daisyUI/themes, system preference + manual selector, persisted.
  - `frontend/src/lib/theme.svelte.ts` and the pre-paint script of `index.html`; daisyUI `light`/`dark` themes.
- [x] T4 — Audit screen: repo root picker/dialog, origin selector (`--worktree`/`--staged`/`--base`), run/cancel, findings list (filter/sort by rule/severity/epistemic status/language/path), finding detail (evidence, hunk context, support level, limitations, recommendation), export JSON/SARIF via save dialog.
  - `App.svelte` and `frontend/src/lib/components/`; `ui.App.AuditAsync`/`CancelAudit`/`ChooseRepository`/`Export`.
- [x] T5 — Parity test: UI output equals `--format json` for the same change (automated check), tests pass, docs updated.
  - `tests/integration/ui_parity_test.go`; docs updated in `docs/tasks/phase-7.md`, `README.md` and `CHANGELOG.md`.

Out of scope: verification UI (7.2), structural viz (7.3), local API (7.4).

Verification (real results):
- `pnpm --dir frontend check` → 0 errors, 0 warnings.
- `pnpm --dir frontend lint` → exit 0.
- `pnpm --dir frontend build` → `dist/index.html` + two assets.
- `go build -tags ui ./...`, `go vet -tags ui ./...` → exit 0.
- `go test -tags ui -count=1 ./ui/ ./tests/integration/` → ok.
- `go build -tags "ui production webkit2_41" ./cmd/codixai` → exit 0.
- `CODIXAI_UI_AUTOQUIT=3s ./codixai ui --root .` → exit 0; the window opened and closed.
- Not verified: the visual review of the window (layout, theme contrast, keyboard navigation) by a person, and the default WebKitGTK 4.0 link (the system library misses `libjxl.so.0.11`).
