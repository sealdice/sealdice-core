# Sealpkg Internals

## Core model

`sealpkg.Manifest` is the parsed form of `info.toml`.

Key runtime fields:

```go
type Instance struct {
    Manifest     *Manifest
    State        PackageState
    InstallTime  time.Time
    InstallPath  string // cache/packages/<author>/<package>/
    SourcePath   string // data/packages/<author>/<package>@<version>.sealpkg
    UserDataPath string // data/extensions/<author>/<package>/_userdata/
    Config       map[string]interface{}
}
```

## On-disk layout

```text
data/
|- packages/
|  `- <author>/<package>@<version>.sealpkg
`- extensions/
   `- <author>/<package>/_userdata/

cache/
`- packages/
   `- <author>/<package>/
      |- info.toml
      |- scripts/
      |- decks/
      |- reply/
      |- helpdoc/
      |- templates/
      `- assets/
```

## Install flow

```text
Install(pkgPath)
|- parse info.toml from the archive
|- validate package id and semver
|- reject src/ archives
|- check SealDice version and package dependencies
|- copy artifact to data/packages/<author>/<package>@<version>.sealpkg
|- extract to cache/packages/<author>/<package>/
|- create data/extensions/<author>/<package>/_userdata/
`- persist state
```

## Runtime behavior

- Enabled package scripts are loaded from `scripts/`.
- Decks, replies, help documents, and templates are loaded directly from enabled package directories.
- Config is validated against `[config.*]` schemas.
- `_userdata/` remains outside the cache so upgrades can rebuild runtime files without losing user data.
