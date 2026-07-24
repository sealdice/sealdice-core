# JS Web Crypto API

本文档说明 `sealdice-core` 在 Goja 中暴露的 `crypto` / `crypto.subtle` API。

应该写手册里面的，咳咳。

另外鄙人鞭策 AI 写了个 [测试脚本](https://raw.githubusercontent.com/lyjjl/SealDice-Plugins/main/%E7%A4%BA%E4%BE%8B/CryptoAPI%E6%B5%8B%E8%AF%95.js)

## 对外入口

- 全局对象：`crypto`

## 顶层 API

### `crypto.getRandomValues(typedArray)`

- 向整数 TypedArray 填充安全随机字节。
- 支持：`Int8Array`、`Uint8Array`、`Uint8ClampedArray`、`Int16Array`、`Uint16Array`、`Int32Array`、`Uint32Array`、`BigInt64Array`、`BigUint64Array`
- 限制：单次 `byteLength <= 65536`

### `crypto.randomUUID()`

- 返回 RFC 4122 v4 UUID 字符串。

## `crypto.subtle` API

### 已实现方法

- `digest(algorithm, data)`
- `generateKey(algorithm, extractable, keyUsages)`
- `importKey(format, keyData, algorithm, extractable, keyUsages)`
- `exportKey(format, key)`
- `sign(algorithm, key, data)`
- `verify(algorithm, key, signature, data)`
- `encrypt(algorithm, key, data)`
- `decrypt(algorithm, key, data)`
- `deriveBits(algorithm, baseKey, length)`
- `deriveKey(algorithm, baseKey, derivedKeyAlgorithm, extractable, keyUsages)`
- `wrapKey(format, key, wrappingKey, wrapAlgorithm)`
- `unwrapKey(format, wrappedKey, unwrappingKey, unwrapAlgorithm, unwrappedKeyAlgorithm, extractable, keyUsages)`

### 支持的 Key 格式

- `raw`（对称密钥；以及 `ECDSA`/`ECDH`/`Ed25519`/`X25519` 公钥）
  - 兼容扩展：`Ed25519`/`X25519` 支持 raw 私钥导入（`Ed25519` 可用 32-byte seed 或 64-byte 私钥；`X25519` 为 32-byte 私钥）
  - `X25519` raw 私钥导入建议显式传 `{name:"X25519", keyType:"private"}`（或 `type:"private"`）
- `jwk`（支持 `oct` / `RSA` / `EC` / `OKP`）
  - `RSA` 私钥导入支持仅 `n/e/d`（可不带 `p/q`）
  - `exportKey("jwk")` 自动填充 `alg`（如 `A256GCM` / `HS256` / `RS256` / `ES256` / `EdDSA`）
- `pkcs1`（RSA 私钥 / 公钥）
- `sec1`（ECDSA / ECDH 私钥）
- `pkcs8`（RSA / ECDSA / ECDH / Ed25519 / X25519 私钥）
- `spki`（RSA / ECDSA / ECDH / Ed25519 / X25519 公钥）

## 算法支持（非过时）

### 摘要

- `SHA-224`
- `SHA-256`
- `SHA-384`
- `SHA-512`

### 对称加密

- `AES-CBC`
- `AES-GCM`
- `AES-CTR`
- `AES-KW`
  - `AES-GCM` 兼容非 12-byte `iv`（当 `tagLength=128`）

### 签名

- `HMAC`
- `RSASSA-PKCS1-v1_5`
- `RSA-PSS`
- `ECDSA`
- `Ed25519`
  - `ECDSA` 签名输出为 Web Crypto 常见原始格式：`r || s`（定长拼接）

### 非对称加密

- `RSA-OAEP`

### 密钥派生

- `PBKDF2`
- `HKDF`
- `ECDH`
- `X25519`

## 算法支持（已过时，不建议使用，这里是兼容考虑）

### 摘要

- `MD5`
- `SHA-1`

### 非对称加密

- `RSAES-PKCS1-v1_5`

### 对称加密

- `DES-CBC`
- `3DES-CBC`（兼容别名：`DES-EDE3-CBC` / `TripleDES-CBC`）

### 旧哈希组合

- `HMAC + MD5`
- `HMAC + SHA-1`
- `PBKDF2 + MD5`
- `PBKDF2 + SHA-1`
- `RSASSA-PKCS1-v1_5 + SHA-1`
- `RSA-PSS + SHA-1`
- `RSA-OAEP + SHA-1`

## 用法示例

```js
// digest
const d = await crypto.subtle.digest("SHA-256", new Uint8Array([1,2,3]));

// HMAC key
const hmacKey = await crypto.subtle.generateKey(
  { name: "HMAC", hash: { name: "SHA-256" }, length: 256 },
  true,
  ["sign", "verify"]
);

// AES-CBC
const aesKey = await crypto.subtle.generateKey(
  { name: "AES-CBC", length: 128 },
  true,
  ["encrypt", "decrypt"]
);
const iv = new Uint8Array(16);
crypto.getRandomValues(iv);
const ct = await crypto.subtle.encrypt({ name: "AES-CBC", iv }, aesKey, new Uint8Array([1,2,3]));
const pt = await crypto.subtle.decrypt({ name: "AES-CBC", iv }, aesKey, ct);
```
