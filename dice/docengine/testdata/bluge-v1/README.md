# Legacy Bluge index

`index/` is a real persisted index generated with
`github.com/fy0/bluge@v0.0.0-20260713113642-38c63a5ebb28`, the dependency used
before the v0.8.0 upgrade. It contains zapx-bluge v1 segments and the application's
old `schema_version` marker (`2`). The generator closes and reopens the index
with the old library and checks that both documents were persisted.

The two documents use the same fields as the old help search engine:

| ID | Source | Title | Content |
| --- | --- | --- | --- |
| `legacy-help` | `data/helpdoc/upgrade.json` | 旧缓存重建验证 | 升级后帮助内容仍可检索 |
| `legacy-removed` | `data/helpdoc/removed.json` | 已删除旧条目 | 此条目不应在重建后保留 |

Both belong to group `default` and package `upgrade`.

`TestHelpManagerRebuildsLegacyBlugeCache` copies the fixture into a temporary
working directory. It verifies that the new Bluge library rejects the original
index, then calls `HelpManager.Load` with valid metadata matching an unchanged
help source. It checks automatic rebuilding, search and numeric-ID lookup,
removal of obsolete entries, and cache reuse after a second load. Normal tests
do not run the generator or download an older dependency.

To regenerate, use a separate temporary Go module so the application dependency
is not changed. Use absolute paths for the generator and a new output directory:

```sh
go mod init bluge-v1-fixture
go get github.com/fy0/bluge@v0.0.0-20260713113642-38c63a5ebb28
go run /path/to/bluge-v1/generate.go /path/to/new-index
```

The generator refuses to use another Bluge version or overwrite an existing
directory. Replace `index/` with the generated files after verifying the result.
