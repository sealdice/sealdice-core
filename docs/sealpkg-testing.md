# Sealpkg Testing Guide

## Recommended test package IDs

Use the canonical `author/package` format in tests:

- `test/simple-script`
- `test/config-package`
- `test/deck-package`
- `test/reply-package`
- `test/helpdoc-package`
- `test/template-package`
- `test/permission-package`

## Expected local paths

```text
data/packages/<author>/<package>@<version>.sealpkg
data/extensions/<author>/<package>/_userdata/
cache/packages/<author>/<package>/
```

## Suggested checks

- Install a package from a local `.sealpkg` file.
- Install a package from a URL.
- Enable and disable a package.
- Verify package scripts load from `scripts/`.
- Verify decks, replies, help documents, and templates load from the extracted package cache.
- Verify `_userdata/` survives package upgrades.
- Verify `src/` archives are rejected.
- Verify old `author@package` IDs are rejected.
- Verify Chinese package IDs such as `作者/扩展` are accepted.

## Useful commands

```bash
# focused metadata tests
go test ./dice/sealpkg

# broader package and dice validation
go test ./dice/...
```
