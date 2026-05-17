# SealPack User Guide

A `.sealpack` file is a SealDice extension package. A single package can include scripts, decks, replies, help documents, templates, and assets.

## Install methods

- Install from the extension UI.
- Install from a local `.sealpack` file.
- Install from a direct package URL.

## What happens on install

SealDice stores package data in three places:

```text
data/packages/<author>/<package>@<version>.sealpack
data/extensions/<author>/<package>/_userdata/
cache/packages/<author>/<package>/
```

- `data/packages/...` stores the package artifact.
- `_userdata/` stores user config and runtime data.
- `cache/packages/...` stores the extracted runtime files and can be rebuilt.

## Updates and uninstall

- Upgrading a package keeps `_userdata/`.
- A full uninstall removes the package artifact, cache, and `_userdata/`.
- A keep-data uninstall removes the artifact and cache but keeps `_userdata/`.

## Safety

- Prefer packages from trusted sources.
- Review requested permissions before enabling a package.
- Modern packages use the `author/package` ID format and do not use the old `author@package` format.
