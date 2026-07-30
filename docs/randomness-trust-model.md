# 为什么我们不迷信“大气噪声”？

> 副标题：从信任根、四种随机模式到海豹当前实现的可审计边界
>
> 本文基于 `sealdice-core-newui` 当前工作树撰写，核对日期为 2026-07-30。
> 依赖版本以 `go.mod` 为准：`github.com/emmansun/gmsm v0.44.1`、`github.com/sixafter/aes-ctr-drbg v1.16.0`、`github.com/sealdice/dicescript v0.0.0-20260729052003-c4c99fe00cb6`。

## 摘要

在线跑团工具的公平性，最终落在随机数是否可被信任。很多讨论会把 `random.org` 一类“大气噪声”服务直接等同于“更真实”“更安全”的随机数来源，但这类说法忽略了一个更底层的问题：你凭什么相信自己真的连到了它，以及返回值没有在链路中被替换？

海豹当前的答案不是“押注某一个神秘终极随机源”，而是把信任边界公开出来。就 2026-07-30 的代码而言，海豹提供的是 `pcg`、`crypto`、`nist`、`gm` 四种可切换模式，并把样本生成、统计检测脚本、运行时查询命令和关键实现都放进了仓库。我们能证明的是“代码当前做了什么、依赖当前承诺了什么、脚本当前如何复测”；我们不能证明的是“只要写上‘真随机’三个字，一切就自动安全了”。

## 1. 口径约定：把项目代码、依赖代码和工程判断分开说

为了避免把“依赖做到的事”误写成“海豹仓库自己从头实现的事”，本文统一使用三种口径：

- **代码事实**：能在海豹当前仓库直接定位到的实现。
- **依赖事实**：海豹直接调用的上游模块，在当前版本源码或文档中明确给出的行为。
- **工程判断**：基于以上两类事实得出的解释、边界和风险判断。

这不是咬文嚼字，而是避免文档虚胖的最低要求。

## 2. 为什么“远程真随机”不能成为你的信任根

“我不信任本机熵源，所以去请求远端 TRNG 服务”在密码学工程上并不自洽。原因可以写成一条信任链：

\[
\text{可信远端随机数}
\Rightarrow
\text{可信 TLS 会话}
\Rightarrow
\text{可信本地 CSPRNG}
\]

如果你怀疑本地密码学随机数已经不可信，那么你同样没有充分理由相信：

1. TLS 握手里的临时随机量是安全生成的。
2. 你连到的对端真的是目标服务，而不是被劫持、重放或代理的链路。
3. 拿回来的字节确实是“原装远端输出”，而不是中间人替换过的内容。

因此，更准确的工程表述是：

- **工程判断**：远端 TRNG 可以是一个已可信本地系统的补充输入。
- **工程判断**：它不能替代本地信任根，更不能在“不信本地”的前提下充当最终裁判。

所以，问题不是“我要不要迷信大气噪声”，而是“我的信任根到底落在哪里”。

## 3. 海豹当前实现的不是一个神话，而是四种模式

### 3.1 模式总览

当前仓库定义如下：

```go
const (
    DiceRandomModePCG    DiceRandomMode = "pcg"
    DiceRandomModeGM     DiceRandomMode = "gm"
    DiceRandomModeNIST   DiceRandomMode = "nist"
    DiceRandomModeCrypto DiceRandomMode = "crypto"
)
```

- **代码事实**：模式定义在 `dice/random_source.go`。
- **代码事实**：默认配置是 `pcg`，见 `dice/dice_config_default.go`。
- **代码事实**：运行时可通过 `.randalgo` 和 `.randalgo set <模式>` 查询与切换。

| 模式 | 当前代码入口 | 核心实现 | 初始化/熵来源 | 运行期失败语义 |
| --- | --- | --- | --- | --- |
| `pcg` | `newPCGDiceSource(generateRandSeed())` | `math/rand/v2.PCG` | 本地运行时信息经 FNV-1a 混合 | 无外部 reader；不是密码学安全 RNG |
| `crypto` | `ds.NewCryptoDiceSource()` | `crypto/rand.Reader` | Go 标准库 OS CSPRNG | 底层读失败时 `panic`，海豹当前没有本地降级 |
| `nist` | `ctrdrbg.NewReader(...)` | AES-CTR-DRBG | 依赖内部系统熵；启动时额外尝试混入 32 字节 `gmrand` 输出做一次 `Reseed` | 构造失败上抛；运行期 `Read` 返回错误时会记录日志并降级到 `PCG` |
| `gm` | `gmrand.Reader` | SM3 Hash DRBG | 依赖内部多源熵池、健康检测和 SM3 条件化 | SeaDice wrapper 仅能处理“返回错误”的路径；依赖的不可恢复故障语义是 `panic` |

### 3.2 绑定关系并不完全对称

模式名字相同，不代表所有调用点都以同一种方式绑定随机源。当前 `MsgContext.getDiceSource()` 的关键逻辑是：

```text
if ctx already has a dice source:
    return it

if ctx.Dice.mode == nist:
    ctx.diceRandSrc = ctx.Dice.getSystemDiceSource()
else:
    ctx.diceRandSrc = ctx.Dice.newDiceSource()

return ctx.diceRandSrc
```

这意味着：

- **代码事实**：`nist` 在消息上下文里会复用 `Dice` 级别的共享随机源。
- **代码事实**：`pcg`、`gm`、`crypto` 在消息上下文里默认是“按上下文新建 source”。
- **工程判断**：这基本是在用实现手段承认 `nist` 的初始化/重播种开销更高，因此避免为每个上下文反复构造。

## 4. 四种模式分别在做什么

### 4.1 `crypto`：最短路径地走 Go 标准库 CSPRNG

当前 `crypto` 模式不是海豹自己手写 `getrandom(2)` 或 Windows API 包装，而是直接委托给 `dicescript.NewCryptoDiceSource()`，底层读取 `crypto/rand.Reader`。

可以把它抽象成：

```text
if mode == crypto:
    source = crypto/rand.Reader
    on Uint64():
        read 8 bytes
        if read fails:
            panic
```

- **代码事实**：海豹仓库本身没有为 `crypto` 模式包一层本地 fallback。
- **依赖事实**：`dicescript` 当前实现中，`cryptoDiceSource.Uint64()` 读失败会 `panic`。
- **依赖事实**：Go 文档说明 `crypto/rand.Reader` 在 Linux 默认使用 `getrandom(2)`，在旧 Linux 回退 `/dev/urandom`，在 Windows 使用 `ProcessPrng` API。

因此，更严谨的文案应该是：

> `crypto` 模式是“直接走 Go 标准库系统随机源”，不是“海豹自己实现了跨平台系统调用层”；其失败语义当前也不是“自动降级”，而是沿用依赖的 `panic` 路径。

### 4.2 `nist`：AES-CTR-DRBG，加了启动时一次 GM 额外 `Reseed`

当前 `nist` 模式构造逻辑如下：

```go
reader, err := ctrdrbg.NewReader(
    ctrdrbg.WithKeySize(ctrdrbg.KeySize256),
    ctrdrbg.WithPersonalization([]byte("sealdice-nist")),
    ctrdrbg.WithPredictionResistance(true),
    ctrdrbg.WithSelfTests(true),
    ctrdrbg.WithContinuousHealthTest(true),
    ctrdrbg.WithEnableKeyRotation(true),
    ctrdrbg.WithReseedInterval(5 * time.Minute),
    ctrdrbg.WithReseedRequests(4096),
)

buf := make([]byte, 32)
n, err := gmrand.Read(buf)
if err == nil && n == len(buf) {
    _ = reader.Reseed(buf)
}
```

把海豹层面的初始化行为写成伪代码，会更容易理解：

```text
reader = NewCTRDRBG(
    key_size = 256,
    personalization = "sealdice-nist",
    prediction_resistance = true,
    self_tests = true,
    continuous_health_test = true,
    enable_key_rotation = true,
    reseed_interval = 5 minutes,
    reseed_requests = 4096,
)

try:
    gm_extra = gmrand.read(32 bytes)
    reader.reseed(gm_extra)
catch:
    log warning and continue with native NIST path
```

这里要特别注意三个实现细节：

1. **代码事实**：海豹只额外混入了一次 `gmrand` 的 32 字节输出，不是自己手写了一整套“多源熵池 + 90B 健康检测 + 128x32 GFSR”。
2. **依赖事实**：`aes-ctr-drbg` 的 `Read` 实现里，如果 `PredictionResistance=true`，则每次 `Read` 之前都会先从系统熵源做一次 reseed。
3. **工程判断**：因此，海豹这里的 5 分钟/4096 次阈值虽然被配置了，但在当前 `PredictionResistance=true` 的分支下并不是主导路径；真正主导运行时行为的是“每次读前重播种”。

也就是说，启动时的这次 GM 额外 `Reseed` 更像是“首次状态扩充”，而不是“后续每次输出都持续经过 GM 熵池”。

如果只写成数学抽象，CTR-DRBG 输出可以概括为：

\[
(K_{t+1}, V_{t+1}, \text{out}_t) = \operatorname{CTR\_DRBG\_Generate}(K_t, V_t, \text{additional\_input})
\]

而海豹当前 `nist` 路径对 `additional_input` 的本地显式增强，是启动期这 32 字节 `gmrand` 输出；运行期每次 `Read` 前的重播种则由依赖内部完成。

### 4.3 `gm`：多源熵池、健康检测和 SM3 Hash DRBG 主要来自依赖

这里是最容易写过头的地方。当前海豹自己做的事情，其实很克制：

- **代码事实**：`gm` 模式下，海豹直接把随机源接到 `gmrand.Reader`。
- **依赖事实**：`gmsm/rand` 当前版本文档与源码说明，它组合了三类输入：OS 随机字节、CPU jitter、运行时 hash loop noise。
- **依赖事实**：其非 OS 熵源会跑 SP 800-90B 风格健康检测，当前代码包括重复计数测试、比例测试和 lag predictor 测试。
- **依赖事实**：其熵池混合与压缩提取由依赖内部 `internal/entropy` 完成，不是海豹仓库本地实现。

因此，当前最准确的写法不是“海豹自己实现了多源熵池模块”，而是：

> 海豹在 `gm` 模式下接入了 `gmsm/rand` 提供的 SM3 Hash DRBG 路径；多源熵采集、健康检测、GFSR 熵池和 SM3 条件化由该依赖实现，海豹仓库层面主要负责模式接入、切换和上层使用。

#### 4.3.1 依赖中的熵池混合公式

`gmsm` 当前版本把熵池描述为 128 x 32-bit 的 twisted GFSR。对单个 32 位输入字的混合，源码给出的形式是：

\[
temp_j =
g \oplus p_j \oplus p_{j+1} \oplus p_{j+25}
\oplus p_{j+51} \oplus p_{j+76} \oplus p_{j+103}
\]

\[
p_j \leftarrow (temp_j \gg 3) \oplus T[temp_j \mathbin{\&} 7]
\]

其中：

- \(p_j\) 是熵池第 \(j\) 个 32 位字；
- \(g\) 是当前混入的 32 位输入；
- \(T\) 是 8 项 twist table；
- tap 位置来自源码注释中的原始多项式 \(x^{128}+x^{103}+x^{76}+x^{51}+x^{25}+x+1\)。

这部分是**依赖事实**，不是海豹本地代码事实。

#### 4.3.2 依赖中的压缩提取公式

`gmsm` 在条件化阶段实现了 `Hash_df` 风格的 SM3 压缩。源码注释可写成：

\[
\operatorname{Hash\_df}(X, L)
=
\operatorname{leftmost}_L
\Big(
\big\|_{i=1}^{\lceil L/\text{outlen}\rceil}
\operatorname{Hash}(i \parallel L \parallel X)
\Big)
\]

在当前实现里：

- 输出长度 `SeedSize = 55` 字节，即 440 bit；
- 提取后的结果会再反馈回熵池，以获得前向安全性的增强效果。

高层伪代码大致如下：

```text
os_entropy      = read 32 bytes from OS
jitter_samples  = collect CPU jitter samples with health tests
runtime_samples = collect hash-loop samples with health tests

pool.add(os_entropy)
pool.add(jitter_samples)
pool.add(runtime_samples)

seed = SM3_Hash_df(pool_bytes, 440 bits)
pool.add(seed, entropy_bits = 0)   // feedback for forward secrecy
return seed
```

#### 4.3.3 依赖中的健康测试阈值

当前依赖实现里，至少能直接定位到这些阈值：

- 重复计数测试：

\[
C = 1 + \left\lceil \frac{-\log_2 \alpha}{h} \right\rceil
  = 1 + \left\lceil \frac{20}{0.5} \right\rceil
  = 41
\]

- 比例测试：窗口 \(W=512\)，阈值近似 \(C \approx 410\)。
- Lag predictor：窗口 \(W=512\)，阈值 \(C=411\)。

这些阈值是**依赖事实**，表明依赖不是“随便混点时间戳就叫熵池”，而是确实把健康检测写进了实现。

### 4.4 `pcg`：高性能、分布质量高，但不是密码学安全随机数

当前 `pcg` 模式的关键代码只有两步：

```go
seed := generateRandSeed()
src := randv2.NewPCG(seed, seed)
```

`generateRandSeed()` 当前并不是从 `crypto/rand` 取种子，而是把这些本地运行时信息做 FNV-1a 混合：

\[
\text{seed}_{pcg}
=
\operatorname{FNV1a64}(
\text{timestamp} \parallel \text{object\_pointer} \parallel \text{pid} \parallel \text{runtime\_stack}
)
\]

对应伪代码：

```text
timestamp = now.UnixNano()
obj_ptr   = address of a temporary object
pid       = os.Getpid()
stack     = runtime.Stack(all goroutines)

seed = FNV1a64(timestamp || obj_ptr || pid || stack)
pcg  = NewPCG(seed, seed)
```

这部分需要把话说实：

- **代码事实**：当前 PCG 种子不是来自 OS CSPRNG。
- **代码事实**：它使用的是 FNV-1a 64 位哈希混合本地运行时信息，且 `NewPCG` 的两个 64 位参数都用了同一个 `seed`。
- **工程判断**：这足以减少“每次启动都用固定种子”这类低级问题，但不能把它写成“密码学安全初始化”。
- **依赖事实**：Go `math/rand/v2` 文档明确说明它适用于仿真/统计用途，不应用于安全敏感场景。

因此，`pcg` 更准确的定位是：

> 高性能、统计质量好的通用伪随机后端；对普通跑团体验足够，但不应被包装成 CSPRNG。

### 4.5 骰点区间映射显式避免了模偏

很多系统会直接把大整数 `% n` 映射到 `[0, n)`。这样写快，但如果原始随机空间不是 `n` 的整数倍，就会产生模偏。海豹当前使用的是拒绝采样：

```go
func randUint64n(src ds.DiceSource, n uint64) uint64 {
    if n&(n-1) == 0 {
        return src.Uint64() & (n - 1)
    }
    ceiling := ^uint64(0) - ^uint64(0)%n
    for {
        v := src.Uint64()
        if v < ceiling {
            return v % n
        }
    }
}
```

对应数学形式为：

\[
\text{ceiling} = 2^{64} - (2^{64} \bmod n)
\]

\[
v \sim U(\{0,\dots,\text{ceiling}-1\})
\Longrightarrow
v \bmod n \sim U(\{0,\dots,n-1\})
\]

这意味着只有采样值落在完整可整除的接受域内时，才做 `% n`。这不是宣传词，而是一个很具体、很有价值的公平性细节。

### 4.6 JS 扩展没有偷偷维护第二套 RNG

当前 Goja `Math.random()` 绑定如下：

```go
func (d *Dice) newGojaRandSource() goja.RandSource {
    return func() float64 {
        if d == nil {
            return float64(randSource.Uint64()>>11) * gojaRandScale
        }
        return float64(d.getSystemDiceSource().Uint64()>>11) * gojaRandScale
    }
}
```

- **代码事实**：JS 扩展使用的不是另一套平行 RNG。
- **代码事实**：它跟随当前 `Dice` 的系统随机源配置。
- **工程判断**：至少在“骰点”和“脚本扩展”之间，海豹没有故意制造两套不一致的随机口径。

## 5. 失败语义比算法名字更重要

如果你只看 `PCG`、`CTR-DRBG`、`SM3 Hash DRBG` 这些名词，会漏掉一个真正影响系统可信性的事实：失败时到底怎样处理。

### 5.1 当前行为矩阵

| 模式 | 构造失败 | 运行时读取失败 | 备注 |
| --- | --- | --- | --- |
| `pcg` | 无外部 reader 构造失败路径 | 无 | 非 CSPRNG |
| `crypto` | 依赖构造简单 | 依赖当前会 `panic` | 海豹当前没有包装成本地 fallback |
| `nist` | `NewReader` 失败会沿 `newDiceSource()` 路径上抛 | `readerDiceSource` 能在 `Read` 返回 error 时记录日志并降级到 `PCG` | 启动期 `gmrand.Read` 额外混入失败仅记录警告并继续 |
| `gm` | 直接接入共享 `Reader` | SeaDice wrapper 只能处理返回 error 的情况；依赖文档声明不可恢复故障会 `panic` | 因此不能简单写成“总会自动降级” |

### 5.2 这意味着什么

- **工程判断**：当前系统并不是“所有密码学模式都统一 fail-stop”。
- **工程判断**：它也不是“所有异常都统一自动降级”。
- **工程判断**：更准确的说法是：不同模式的失败语义取决于海豹本地 wrapper 和上游依赖的组合。

这正是“可审计”比“最安全”三个字更重要的原因。

## 6. 当前仓库能证明什么，不能证明什么

### 6.1 当前仓库能直接支撑的内容

- **代码事实**：四种模式、配置项、运行时命令、区间映射、Goja 绑定都能直接从仓库核对。
- **代码事实**：样本生成入口在 `dice/randomness_report_test.go`，默认支持 `pcg,crypto,nist,gm` 四种模式。
- **代码事实**：仓库已经提供 `rddetector` 报告脚本与汇总逻辑，脚本入口与 profile 说明在 `scripts/randomness/README.md`。

样本生成最直接的复现实验命令是：

```bash
SEALDICE_RANDOMNESS_GENERATE=1 \
SEALDICE_RANDOMNESS_OUT_DIR=temp/randomness/manual \
SEALDICE_RANDOMNESS_MODES=pcg,crypto,nist,gm \
SEALDICE_RANDOMNESS_SAMPLES=20 \
SEALDICE_RANDOMNESS_BITS=1000000 \
go test ./dice -run TestGenerateRandomnessSamples -count=1
```

### 6.2 当前仓库不能诚实宣称的内容

以下表述在当前工作树下都不应直接写进对外文案：

1. “海豹自己手写实现了完整的 NIST / 国密多源熵池模块。”
2. “四种模式在任何异常情况下都保持同等级密码学强度。”
3. “PCG 模式的种子初始化已经达到密码学安全。”
4. “仓库内已经随附现成的、可直接引用的 NIST SP 800-22/GM/T 0005-2021 全套通过报告。”

第 4 点尤其要保守：

- **代码事实**：仓库提供了生成与检测脚本。
- **代码事实**：当前工作树下并没有现成的 `docs/randomness-reports/...` 报告文件可直接引用。
- **工程判断**：因此更稳妥的说法应当是“具备复测链路”，而不是“已经在仓库内随附正式报告并可直接引用结果”。

## 7. 面向不同场景，应该怎样选模式

| 场景 | 更合适的模式 | 原因 |
| --- | --- | --- |
| 日常跑团、性能优先 | `pcg` | 吞吐高、分布质量好、默认体验轻量，但不是 CSPRNG |
| 想最短路径地走系统随机源 | `crypto` | 直接委托 Go `crypto/rand.Reader` |
| 想采用 NIST 风格 DRBG，并接受更高开销 | `nist` | AES-CTR-DRBG、启用 prediction resistance、启动期会额外混入一次 `gmrand` 输出 |
| 需要国密/商密语境实现口径 | `gm` | 直接接入 SM3 Hash DRBG 路径；多源熵与健康检测来自 `gmsm/rand` |

对普通玩家而言，可以把这张表压缩成一句话：

> 不是所有场景都需要“最高规格词汇”，但所有场景都值得拥有“说清楚边界”的文档。

## 8. 结论：真正值得信任的，不是“真随机”三个字

海豹当前随机数方案最值得强调的地方，不是“我们比所有人更安全”，而是：

1. 它把随机模式公开成了可配置、可查询、可切换的实现。
2. 它把样本生成与统计检测脚本留在了仓库里。
3. 它没有把“依赖做到的事”冒充成“项目自己从头实现的事”。
4. 它也暴露了自己的工程边界，例如 `crypto` / `gm` / `nist` 并不共享同一种失败语义。

如果你真正关心公平，那么比“迷信大气噪声”更重要的问题是：

> 这个系统有没有把自己的信任根、实现路径、降级策略、失败语义和证据链讲明白？

在这个问题上，可审计，永远比形容词更重要。

## 参考文献与引用库

### 8.1 仓库内定位

- `dice/random_source.go`
- `dice/dice.go`
- `dice/randomness_report_test.go`
- `scripts/randomness/README.md`
- `go.mod`

### 8.2 外部标准与上游资料

1. NIST SP 800-90A Rev. 1, *Recommendation for Random Number Generation Using Deterministic Random Bit Generators*.
2. NIST SP 800-90B, *Recommendation for the Entropy Sources Used for Random Bit Generation*.
3. NIST SP 800-22 Rev. 1a, *A Statistical Test Suite for Random and Pseudorandom Number Generators for Cryptographic Applications*.
4. GM/T 0105-2021, 《软件随机数发生器设计指南》.
5. GM/T 0005-2021, 《随机性检测规范》.
6. Melissa E. O'Neill, *PCG: A Family of Simple Fast Space-Efficient Statistically Good Algorithms for Random Number Generation*, HMC-CS-2014-0905.
7. Go `crypto/rand` 文档与源码说明。
8. Go `math/rand/v2` 文档说明。
9. `github.com/sixafter/aes-ctr-drbg` 当前版本文档与源码。
10. `github.com/emmansun/gmsm/rand` 当前版本文档与源码。

### 8.3 BibTeX 引用库

可直接复用的 BibTeX 文件位于：

- `docs/references/randomness-trust-model.bib`
