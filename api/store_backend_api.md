# SealDice Store Backend API

## Base URL

All backend store endpoints are rooted at:

```text
<baseUrl>/dice/api/store
```

Example:

```text
https://example.com/dice/api/store
```

Protocol version: `2.0`

## Unified package DTO

Store list endpoints now exchange a package-centric DTO. Public responses should use nested `storeAssets` and `download` objects. SealDice validates known fields that affect package identity, display, and installation, but ignores unknown extension fields so backend additions do not require an immediate client update. SealDice now uses a single active backend source, so backend identity is not part of the package DTO.

```json
{
  "id": "author/package",
  "version": "1.2.3",
  "name": "Demo Package",
  "authors": ["Alice", "Bob"],
  "description": "Package description",
  "license": "MIT",
  "homepage": "https://example.com/pkg",
  "repository": "https://github.com/example/pkg",
  "keywords": ["coc", "tools"],
  "contents": ["scripts", "decks", "helpdoc"],
  "seal": {
    "minVersion": "1.5.0",
    "maxVersion": "2.0.0"
  },
  "dependencies": {
    "author/base": ">=1.0.0"
  },
  "storeAssets": {
    "readme": "docs/README.md",
    "icon": "assets/icon.png",
    "banner": "assets/banner.png",
    "screenshots": ["assets/shot-1.png"],
    "category": "rules"
  },
  "download": {
    "url": "/dice/api/store/packages/author/package/1.2.3/package@1.2.3.sealpack",
    "zipUrl": "/dice/api/store/packages/author/package/1.2.3/package@1.2.3.zip",
    "hash": {
      "sha256": "abcdef..."
    },
    "size": 10240,
    "releaseTime": 1710000000,
    "updateTime": 1710500000,
    "downloadCount": 1234
  },
  "installed": false
}
```

### Field notes

- `id`: package ID in `author/package` format.
- `version`: semantic version string.
- `contents`: allowed values are `scripts`, `decks`, `reply`, `helpdoc`, `templates`.
- `dependencies`: map of package ID to semver constraint.
- `storeAssets`: package presentation assets shown in the store UI.
- `download.url`: absolute URL or store-rooted relative path to a `.sealpack` file.
- `download.zipUrl`: optional absolute URL or store-rooted relative path to a `.zip` artifact.
- `download.size`: package artifact size in bytes.
- `download.hash`: optional integrity hashes keyed by algorithm. If provided, it must include `sha256`; unsupported-only hash sets are rejected. Omit the object when no client-side integrity verification is available.
- `download.releaseTime` / `download.updateTime`: Unix timestamps in seconds.
- `download.downloadCount`: public download counter.
- `installed`: local-only field computed by the SealDice client; backend responses may omit it.
- Unknown fields: allowed at response, package, `storeAssets`, `download`, and other object levels. Older clients may ignore these fields.

## Endpoints

### 1. Backend info

`GET /info`

Response example:

```json
{
  "name": "Official Store",
  "protocolVersions": ["2.0"],
  "announcement": "Welcome",
  "sign": "base64-signature"
}
```

### 2. Recommendations

`GET /recommend`

Response example:

```json
{
  "result": true,
  "data": [
    {
      "id": "author/package",
      "version": "1.2.3",
      "name": "Demo Package",
      "authors": ["Alice"],
      "description": "Package description",
      "license": "MIT",
      "homepage": "https://example.com/pkg",
      "repository": "https://github.com/example/pkg",
      "keywords": ["coc"],
      "contents": ["scripts"],
      "seal": {
        "minVersion": "1.5.0",
        "maxVersion": "2.0.0"
      },
      "dependencies": {},
      "storeAssets": {
        "readme": "docs/README.md",
        "icon": "assets/icon.png",
        "banner": "assets/banner.png",
        "screenshots": [],
        "category": "rules"
      },
      "download": {
        "url": "/dice/api/store/packages/author/package/1.2.3/package@1.2.3.sealpack",
        "zipUrl": "/dice/api/store/packages/author/package/1.2.3/package@1.2.3.zip",
        "hash": {
          "sha256": "abcdef..."
        },
        "size": 10240,
        "releaseTime": 1710000000,
        "updateTime": 1710500000,
        "downloadCount": 1234
      }
    }
  ],
  "err": ""
}
```

### 3. Paged packages

`GET /page`

Query parameters:

| Name | Type | Required | Notes |
| --- | --- | --- | --- |
| `content` | string | no | One of `scripts` / `decks` / `reply` / `helpdoc` / `templates` |
| `pageNum` | int | no | 1-based page number |
| `pageSize` | int | no | Page size |
| `author` | string | no | Author filter |
| `name` | string | no | Name filter |
| `category` | string | no | Category filter |
| `sortBy` | string | no | `updateTime`, `downloadCount`, `releaseTime`, `name` |
| `order` | string | no | `asc` or `desc` |

Example:

```text
/page?content=scripts&pageNum=1&pageSize=20&sortBy=updateTime&order=desc
```

Response example:

```json
{
  "result": true,
  "data": {
    "data": [
      {
        "id": "author/package",
        "version": "1.2.3",
        "name": "Demo Package",
        "authors": ["Alice"],
        "description": "Package description",
        "license": "MIT",
        "homepage": "https://example.com/pkg",
        "repository": "https://github.com/example/pkg",
        "keywords": ["coc"],
        "contents": ["scripts"],
        "seal": {
          "minVersion": "1.5.0",
          "maxVersion": "2.0.0"
        },
        "dependencies": {},
        "storeAssets": {
          "readme": "docs/README.md",
          "icon": "assets/icon.png",
          "banner": "assets/banner.png",
          "screenshots": [],
          "category": "rules"
        },
        "download": {
          "url": "/dice/api/store/packages/author/package/1.2.3/package@1.2.3.sealpack",
          "zipUrl": "/dice/api/store/packages/author/package/1.2.3/package@1.2.3.zip",
          "hash": {
            "sha256": "abcdef..."
          },
          "size": 10240,
          "releaseTime": 1710000000,
          "updateTime": 1710500000,
          "downloadCount": 1234
        }
      }
    ],
    "pageNum": 1,
    "pageSize": 20,
    "next": true
  },
  "err": ""
}
```

### 4. Package files

`GET /files/{namespace}/{package}/{version}`

Response example:

```json
{
  "result": true,
  "data": [
    {
      "path": "README.md",
      "size": 1200
    },
    {
      "path": "assets/icon.png",
      "size": 4096
    }
  ],
  "err": ""
}
```

### 5. Package file preview

`GET /file/{namespace}/{package}/{version}?path=assets/icon.png`

Returns the requested package-internal file bytes with an appropriate `Content-Type`.

## Local download API contract

The SealDice local download endpoint now accepts package identity as `id` + `version`.

`POST /store/download`

Request body:

```json
{
  "id": "author/package",
  "version": "1.2.3"
}
```

Implementation note: the local client may still cache entries internally by `author/package@version`, but that cache key is not part of the public API.

## Validation rules

The SealDice client validates backend responses before exposing them locally:

1. `id` must be a valid package ID.
2. `version` must be valid semver.
3. `download.url` must resolve to a `.sealpack` URL.
4. `contents` must contain only supported content kinds.
5. dependency keys must be valid package IDs.

## Errors

Backend business errors should still use the standard wrapper:

```json
{
  "result": false,
  "err": "error message"
}
```

HTTP status guidance:

- `200`: request handled successfully; inspect `result` for business status.
- `4xx` / `5xx`: transport or server failure.
