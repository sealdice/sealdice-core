# Repository Guidelines

## Project Structure & Module Organization

This is the SealDice administration UI, built with Vue 3, TypeScript, Naive UI, and Vite. Application code is under `src/`: `pages/` contains file-routed page entries, `components/<domain>/` contains domain UI, `components/shared/` contains reusable presentation components, and `features/` owns business logic, composables, and state. Keep layouts in `layouts/`, routing and navigation policy in `router/`, and API configuration in `api/`. Read `src/ARCHITECTURE.md` before making cross-cutting changes. Static assets live in `src/assets/` and `public/`.

Do not edit `src/api/generated/` or `openapi.json` by hand. Regenerate them with `pnpm run generate-api` when the backend contract changes.

## Build, Test, and Development Commands

- `pnpm install` installs the pinned workspace dependencies (Node `^20.19` or `>=22.12`).
- `pnpm dev` starts Vite at `http://localhost:5175`; set `VITE_API_PROXY_TARGET` for a non-default backend.
- `pnpm run type-check` runs `vue-tsc --build`.
- `pnpm run test` runs the Vitest suite once; use `pnpm run test:watch` while developing.
- `pnpm run build` regenerates the client, type-checks, and produces the production bundle. Use `build-only` only when generated API files already exist.
- `pnpm run lint` applies Oxlint and ESLint fixes; `pnpm run format` formats `src/` with Prettier.

## Coding Style & Naming Conventions

Use Vue Composition API with `<script setup lang="ts">`. Follow existing directory boundaries and keep pages focused on composition; move reusable or stateful domain logic to `features/` or domain components. Use PascalCase for Vue component filenames (`ConnectEditDialog.vue`), camelCase for TypeScript modules (`queryKeys.ts`), and colocate tests as `*.test.ts`. Prettier requires two spaces, single quotes, semicolons, trailing commas, and a 100-character print width. Vue blocks must be ordered `template`, `script`, then `style`.

## Testing Guidelines

Vitest tests sit beside the code they cover, for example `src/router/routeMeta.test.ts`. Add focused tests for parsers, state, query adapters, route behavior, and regressions. Run the relevant test file during development, then `pnpm run type-check` and `pnpm run test` before review; run a build for changes affecting configuration, generated APIs, or bundling.

## Commit & Pull Request Guidelines

Use Conventional Commit-style subjects, commonly with a Chinese summary: `fix(layout): 修正页面标题对齐` or `refactor(ui): 收拢实时通道`. Keep commits scoped to one concern. Pull requests should describe the behavior change, list validation performed, link the relevant issue when available, and include before/after screenshots for visible UI changes. Preserve unrelated working-tree changes.
