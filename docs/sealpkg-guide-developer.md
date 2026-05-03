# Sealpkg Developer Guide

`sealpkg` is the canonical SealDice package format.

## Package layout

A package is a zip archive with the `.sealpkg` extension. The archive root must contain `info.toml` and may contain the following directories:

```text
my-extension/
|- info.toml
|- scripts/
|- decks/
|- reply/
|- helpdoc/
|- templates/
|- assets/
`- README.md
```

`src/` is not supported.

## Package ID

`package.id` uses the canonical `author/package` format.

Examples:

```toml
id = "alice/awesome-dice"
id = "sealdice/official-coc7"
id = "作者/扩展包"
```

Rules:

- `author` and `package` may use Unicode letters, digits, `-`, and `_`.
- Chinese is allowed.
- Each segment is 1-64 characters.
- `/` is reserved as the separator and cannot appear inside a segment.
- Spaces, `.`, `..`, and backslashes are invalid.

## Minimal `info.toml`

```toml
[package]
id = "yourname/my-extension"
name = "My Extension"
version = "1.0.0"
authors = ["Your Name"]
license = "MIT"
description = "Example package"
homepage = "https://example.com/my-extension"
repository = "https://github.com/example/my-extension"
keywords = ["dice", "tool"]

[package.seal]
min_version = "1.5.0"

[dependencies]
"sealdice/base-utils" = ">=1.0.0"

[permissions]
network = false
file_read = ["assets/*"]
file_write = ["_userdata/*"]

[contents]
scripts = ["scripts/*.js"]
decks = ["decks/*.toml"]
reply = ["reply/*.yaml"]
helpdoc = ["helpdoc/*.md"]
templates = ["templates/*.yaml"]

[store]
readme = "README.md"
icon = "assets/icon.png"
screenshots = ["assets/screen-1.png"]
category = "tool"

[config.mode]
type = "string"
default = "simple"
```

## Field summary

- `[package]`: identity and display metadata.
- `[package.seal]`: minimum and optional maximum SealDice version.
- `[dependencies]`: map of `author/package` to semver constraints.
- `[permissions]`: runtime capability declarations.
- `[contents]`: package resource globs. Paths must stay under their matching top-level directory.
- `[store]`: package-local store presentation assets.
- `[config.*]`: typed config schema consumed by the package manager UI.

## Packaging

Create a zip from the package root and rename it to `.sealpkg`.

```bash
cd my-extension
zip -r ../my-extension.sealpkg .
```

## Local storage layout

After installation, core stores package data like this:

```text
data/packages/<author>/<package>@<version>.sealpkg
cache/packages/<author>/<package>/
data/extensions/<author>/<package>/_userdata/
```

Only one version is active at a time. Upgrades replace the active cache and keep `_userdata`.

## Notes

- Package scripts are loaded from `scripts/` inside enabled packages.
- `src/` archives and old `author@package` IDs are rejected.
