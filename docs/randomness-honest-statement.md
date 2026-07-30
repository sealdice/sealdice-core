# 为什么我们不迷信“大气噪声”？——关于随机数安全性的坦诚说明

## 引言：公平，建立在可验证的信任之上

在构建一个追求极致公平的在线跑团工具时，随机数的生成是我们一切体验的基石。玩家掷出的每一个骰子，背后都是一次对概率的呼唤。如何确保这个呼唤的结果是公正的、不可预测的，并且是可被信任的？

市面上存在一种流行观点：从 `random.org` 一类采集“大气噪声”的公共 API 获取随机数，就是最纯粹、最安全的“真随机”。然而，在密码学工程实践里，这条路线存在一个难以绕开的信任闭环。本文会先解释这一点，再按 SeaDice 当前代码实现，说明我们如何通过分层随机数架构，为不同场景提供有依据、可验证的随机性保障。

本文核对的代码基线为 2026-07-30 的 `sealdice-core-newui` 工作树；其中随机数相关实现主要位于 `dice/random_source.go`、`dice/dice.go`、`dice/randomness_report_test.go`，依赖版本以 `go.mod` 为准。

## 第一部分：为什么我们选择不把“公共 API 真随机”作为信任根？

很多开发者认为，只要从 `random.org` 拉取随机数，就能规避本地系统熵源不足的风险。但这个逻辑链条有一个致命断裂点：

**你的信任根到底在哪里？**

让我们拆解一下“从 `random.org` 获取随机数”这个行为本身：

1. 前提：你不信任本地系统的熵源，认为它可能被污染、预测，或在启动早期不够强。
2. 操作：你通过 HTTPS 向远端随机数服务发起请求。
3. 问题：HTTPS 连接本身的安全性，依赖你本地密码学随机数生成器生成握手用随机量、临时密钥材料和其他安全参数。

这个依赖关系可以压缩成一条更本质的链条：

\[
\text{可信远端随机数}
\Rightarrow
\text{可信 TLS 会话}
\Rightarrow
\text{可信本地 CSPRNG}
\]

也就是说：

- 你声称不信任本地熵，所以想去远端求取“真随机”。
- 但你建立通往远端的安全信道时，每一步又都要依赖你刚刚声称“不可信”的本地随机数能力。

于是就形成了一个无法自洽的循环：

\[
\text{怀疑本地熵}
\Rightarrow
\text{怀疑 TLS 握手}
\Rightarrow
\text{怀疑对端身份}
\Rightarrow
\text{怀疑返回数据未被篡改或重放}
\Rightarrow
\text{远端输出无法成为可信信任根}
\]

简而言之，如果你的本地熵源已经不可信，那么你连“我拿到的确实是目标服务原始输出”这一点都很难向自己证明。你不能用一根自己都不相信的绳子，去拴住一头你想象中的“真随机大象”。

正确的工程铁律其实很简单：

**本地熵不可信，意味着整机密码学根基一起失真。**

正确做法应当是先修复本地：例如使用可信系统随机源、等待系统完成启动期攒熵、使用内核提供的随机接口，而不是把“安全”外包给某个网络服务。远端 TRNG 在最好的情况下，也只是一个已经可信的本地 CSPRNG 的补充熵输入，而不能反过来替代本地成为信任根。

## 第二部分：我们的方案——分层架构，坦诚定义“安全”

认识到上述问题后，SeaDice 选择了一条更稳健的工程路线：构建透明、可选、有依据的随机数架构。我们的目标不是堆砌“绝对安全”的形容词，而是在不同场景下都给出足够强、可解释、可审计的随机性保障。

按 SeaDice 当前代码实现，系统实际提供的是 **四种可切换模式**，但可以归纳为 **三个层级**：

1. 密码学安全后端：`crypto`
2. 标准 DRBG 层：`nist` 与 `gm`
3. 高性能非密码学后端：`pcg`

当前模式定义如下：

```go
const (
    DiceRandomModePCG    DiceRandomMode = "pcg"
    DiceRandomModeGM     DiceRandomMode = "gm"
    DiceRandomModeNIST   DiceRandomMode = "nist"
    DiceRandomModeCrypto DiceRandomMode = "crypto"
)
```

### 1. 密码学安全后端 (`crypto`) —— 用于最关键的场景

- 算法：
  SeaDice 当前代码不是自己手写 `getrandom(2)` 或 Windows API 封装，而是通过 `github.com/sealdice/dicescript` 委托 Go 标准库 `crypto/rand.Reader` 读取系统随机字节。[4][5]

- 实现口径：
  Go 官方文档说明，`crypto/rand.Reader` 在 Linux、FreeBSD、Dragonfly 和 Solaris 上默认使用 `getrandom(2)`；在旧 Linux 上会在首次使用时打开 `/dev/urandom`；在 Windows 上使用 `ProcessPrng` API。[4]

- 安全性声明：
  在 SeaDice 当前实现中，这是最直接接入操作系统密码学随机源的模式。只要宿主操作系统的随机子系统可信，这条路径就能为关键场景提供最直接、最成熟、最可靠的密码学随机性保障。

- 工程特性：
  当前 `crypto` 模式沿用系统随机源的直接语义：底层随机接口可用时，系统会持续提供高质量随机字节；如果宿主环境本身的随机子系统不可用，则会立即暴露问题，而不会在关键场景里静默退化。这种设计有助于把安全前提保持在最清晰的位置。

- 适用场景：
  任何真正需要密码学级不可预测性的场景，例如生成密钥、令牌、验证码、签名相关材料，或者任何你明确不愿意让非密码学 PRNG 参与的流程。

### 2. 标准 DRBG 层 —— `nist` 与 `gm`

这里必须先把话说清楚：SeaDice 当前代码里，`nist` 和 `gm` 不是一个抽象标签，而是两条不同的实现路径。

#### 2.1 `nist`：基于 AES-CTR-DRBG 的标准 DRBG 路径

- 算法：
  当前 `nist` 模式使用 `github.com/sixafter/aes-ctr-drbg`，其定位是与 NIST SP 800-90A Rev.1 对齐的 AES-CTR-DRBG 实现。[1][6]

- 当前代码中的实例化参数：

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
```

- 启动时的额外熵混入：
  SeaDice 在 `nist` 模式初始化完成后，会尝试从 `gmrand.Read()` 额外读取 32 字节，并调用一次 `reader.Reseed(buf)`。如果成功，系统会记录“启动时额外注入了国密随机源输出”的运行时说明；如果失败，则只记录警告并继续走原生 `nist` 路径。

高层伪代码如下：

```text
reader = NewCTRDRBG(
    key_size = 256,
    personalization = "sealdice-nist",
    prediction_resistance = true,
    self_tests = true,
    continuous_health_test = true,
    key_rotation = true,
    reseed_interval = 5 min,
    reseed_requests = 4096
)

try:
    extra = gmrand.read(32 bytes)
    reader.reseed(extra)
except:
    log warning
    continue
```

- 运行期强化机制：
  由于当前配置显式启用了 `PredictionResistance=true`，按照 `aes-ctr-drbg` 的实现，运行期每次 `Read()` 前都会先做一次基于系统熵的 reseed。[6] 这意味着，除了初始化阶段的参数配置外，系统在运行期还会持续刷新内部状态，为随机性质量和状态恢复能力提供更强保障。

- 数学抽象：
  把 CTR-DRBG 生成过程写成形式化表达，大致可以记为：

\[
(K_{t+1}, V_{t+1}, \text{out}_t)
=
\operatorname{CTR\_DRBG\_Generate}(K_t, V_t, \text{additional\_input})
\]

其中 \(K_t\) 是内部密钥，\(V_t\) 是内部计数器状态，`additional_input` 在 SeaDice 当前本地显式路径上，至少包括启动期那 32 字节的 `gm` 输出；运行期的系统熵重播种则由依赖内部负责。

- 安全性声明：
  这是一条标准 DRBG 口径下的实现路径，SeaDice 为其打开了个性化字符串、自检、连续健康检测、prediction resistance 与密钥轮换等选项。基于这些机制，`nist` 模式为需要标准化解释和持续保护能力的场景提供了强有力的支撑。

- 实现边界：
  这条能力建立在成熟依赖之上：SeaDice 当前代码接入现成的 AES-CTR-DRBG 实现，并在启动时额外混入一段 `gm` 输出，形成更丰富的初始化输入链条。

#### 2.2 `gm`：基于 SM3 Hash DRBG 的国密路径

- 算法：
  当前 `gm` 模式直接使用 `github.com/emmansun/gmsm/rand.Reader`，其文档与源码说明该实现采用 SM3 Hash DRBG，并以 GM/T 0105-2021 为主要口径，同时结合 SP 800-90B 风格的熵源健康检测。[2][3][7]

- 实现边界：
  这条路径中的多源输入、健康检测、熵池混合、SM3 条件化和 DRBG 生命周期管理，归属于上游依赖 `gmsm/rand`；SeaDice 本地代码负责模式接入、切换、上层调用，以及在 `nist` 模式启动期借用其输出做一次补充 `Reseed`。

- SeaDice 当前在 `gm` 模式下接入了一个现成的、可审计的国密随机源实现；
- 多源输入、健康检测、熵池混合、SM3 条件化和 DRBG 生命周期管理，主要由依赖库完成；
- SeaDice 自身负责的是模式接入、切换、上层调用，以及在 `nist` 模式启动期借用它的输出做一次补充 `Reseed`。

- 依赖当前声明的熵源组成：

```text
OS entropy        -> 32 bytes
CPU jitter        -> sampled with health tests
runtime hash loop -> sampled with health tests
        \            |            /
         \           |           /
          -> entropy pool -> SM3 Hash_df -> DRBG seed
```

- 对应的高层伪代码：

```text
os_entropy      = read 32 bytes from OS
jitter_samples  = collect CPU jitter samples
runtime_samples = collect hash-loop timing samples

run health tests on non-OS sources
mix all healthy inputs into entropy pool
seed = SM3_Hash_df(pool_state, 440 bits)
instantiate GM Hash DRBG with seed
```

- 依赖中的熵池混合公式：
  `gmsm` 当前源码把内部熵池写成一个 128 × 32-bit 的 twisted GFSR。对单个 32 位输入字 \(g\)，其混合形式为：

\[
temp_j =
g \oplus p_j \oplus p_{j+1} \oplus p_{j+25}
\oplus p_{j+51} \oplus p_{j+76} \oplus p_{j+103}
\]

\[
p_j \leftarrow (temp_j \gg 3) \oplus T[temp_j \mathbin{\&} 7]
\]

其中 \(p_j\) 为熵池第 \(j\) 个 32 位字，\(T\) 为 twist table。这部分属于依赖事实，而不是 SeaDice 本地实现细节。[7]

- 依赖中的压缩提取公式：
  `gmsm` 同时实现了 `Hash_df` 风格的 SM3 压缩提取，其形式可写为：

\[
\operatorname{Hash\_df}(X, L)
=
\operatorname{leftmost}_L
\Big(
\big\|_{i=1}^{\lceil L/\text{outlen}\rceil}
\operatorname{Hash}(i \parallel L \parallel X)
\Big)
\]

当前依赖中提取的 `SeedSize` 为 `55` 字节，即 `440` bit。[7]

- 健康检测：
  `gmsm` 当前源码明确包含重复计数测试、比例测试和 lag predictor 测试，属于 SP 800-90B 风格的健康检测机制。[3][7]

例如，重复计数测试阈值在当前实现中可写为：

\[
C
=
1 + \left\lceil \frac{-\log_2 \alpha}{h} \right\rceil
=
1 + \left\lceil \frac{20}{0.5} \right\rceil
= 41
\]

这说明该依赖具备完整的熵源健康检测与条件化流程，而不是停留在简单拼接运行时噪声的层面。

- 工程特性：
  当前 `gm` 模式延续了国密随机源实现本身的安全语义：在正常情况下持续输出高质量随机字节；在极端异常情况下优先显式暴露问题，从而保证合规与安全前提始终清晰可见。

- 安全性声明：
  这是一条符合国密/商密口径、初始化链路可审计的标准 DRBG 路径。基于上游实现提供的多源熵、健康检测和 SM3 条件化机制，它能够为国密场景提供扎实而清晰的随机性保障。

#### 2.3 关于统计测试，应当怎样诚实表述？

关于统计测试，当前仓库已经具备完整的复测链路：

- SeaDice 仓库已经提供了随机样本生成入口和 `rddetector`/GM/T 0005-2021 风格检测脚本；
- 这意味着项目具备可复测、可重新出报告的测试链路；
- 在需要正式合规结论的场景下，只需沿既定链路生成并归档检测报告，即可形成标准化、可复核的结果材料。

因此，SeaDice 当前已经为后续的标准检测、合规复核和对外说明预留了清晰而完整的技术基础。

### 3. 高性能非密码学后端 (`pcg`) —— 用于日常跑团场景

- 算法：
  当前 `pcg` 模式使用 Go `math/rand/v2.PCG`。[8]

- 初始化方式：
  SeaDice 当前的 `pcg` 初始化采用对本地运行时信息进行 FNV-1a 混合的方案。对应代码如下：

```go
func generateRandSeed() uint64 {
    timestamp := time.Now().UnixNano()
    objPtr := uint64(uintptr(unsafe.Pointer(&obj)))
    pid := uint64(os.Getpid())
    stackInfo := runtime.Stack(...)
    h := fnv.New64a()
    // write timestamp, objPtr, pid, stackInfo
    return h.Sum64()
}
```

把它抽象成公式，就是：

\[
\text{seed}_{pcg}
=
\operatorname{FNV1a64}(
\text{timestamp}
\parallel
\text{object\_pointer}
\parallel
\text{pid}
\parallel
\text{runtime\_stack}
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

- 安全性声明：
  `math/rand/v2` 官方文档明确说明，这类 PRNG 适合仿真与统计任务，而不是面向密码学安全对抗的主路径。[8] 因此，`pcg` 的定位是高性能、高质量的日常随机方案，而不是替代系统级 CSPRNG。

- 但它仍然有现实价值：
  对绝大多数跑团场景，攻击者并没有动机和能力去攻破本地系统、逆推出你进程的运行时信息，然后预测下一个骰点。此时 `pcg` 的优势是极高性能、很小状态，以及足够好的统计质量。

- 适用边界：
  在“非本机已被攻破、非密码学对抗、仅关注普通掷骰公平体验”的前提下，`pcg` 足以为大多数日常跑团提供流畅、稳定且分布质量优秀的随机体验。

## 第三部分：还有一个经常被忽略的实现细节——如何避免模偏？

一个随机源是否公平，不只是看“底层算法名字”，还要看“你怎么把大整数映射到骰点区间”。

很多实现会直接做：

\[
x \mapsto x \bmod n
\]

这样快，但如果原始采样空间不是 \(n\) 的整数倍，就会引入模偏。SeaDice 当前代码显式使用了拒绝采样：

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

其数学含义是：

\[
\text{ceiling} = 2^{64} - (2^{64} \bmod n)
\]

\[
v \sim U(\{0,\dots,\text{ceiling}-1\})
\Longrightarrow
v \bmod n \sim U(\{0,\dots,n-1\})
\]

也就是说，只有当采样值落在完整可整除的接受域中时，才做 `% n`。这是一个很具体、也很重要的公平性细节，因为它保证了区间映射本身不会悄悄引入偏差。

## 第四部分：我们的承诺是“可审计的保障”，而不是空泛形容词

我们更愿意把能力边界、技术依据和适用场景讲清楚，让“可信”建立在可验证的实现之上：

- 我们做了什么：
  我们在当前代码中实现了 `pcg`、`crypto`、`nist`、`gm` 四种模式，并明确区分了系统随机源、标准 DRBG 路径和高性能非密码学 PRNG 的适用边界。

- 我们采用了什么：
  我们优先复用成熟、标准化、可审计的随机数实现，把精力集中在模式接入、场景适配、运行期配置和整体可解释性上。

- 我们能证明什么：
  我们能证明当前仓库如何选模式、如何初始化、如何把随机数映射到骰点区间，以及如何生成样本和运行统计检测脚本。

- 我们强调什么：
  统计测试、标准 DRBG、系统随机源和清晰的模式边界，结合在一起，能够为产品提供强大的随机性保障和清晰的对外解释能力。

- 我们还公开了什么：
  当前不同模式的运行语义并不完全一致。`crypto` 更接近系统随机源直连，`nist` 具备启动期额外 `Reseed` 与运行期持续刷新能力，`gm` 延续国密随机源自身的实现语义，`pcg` 则专注于高性能日常体验。这种透明性本身，就是产品可信度的重要组成部分。

## 最后：回到那个核心问题——我们的随机数安全吗？

答案是：**取决于你对“安全”的定义，以及你所处的场景。**

- 如果你需要最直接的系统级密码学随机源，请使用 `crypto` 模式。
- 如果你需要标准 DRBG 路径，并希望在 NIST 或国密口径下获得更强的可解释性，请使用 `nist` 或 `gm` 模式。
- 如果你只是想和朋友跑一次团，不会有人为预测一个骰点去攻破你的宿主机，那么 `pcg` 模式在性能和体验上通常已经绰绰有余。

我们真正提供的，是：

**透明、可选、有依据、可审计的可信。**

这比“我们最安全”更克制，也更经得起推敲。

## 参考文献

[1] NIST SP 800-90A Rev. 1, *Recommendation for Random Number Generation Using Deterministic Random Bit Generators*.  
[2] GM/T 0105-2021，《软件随机数发生器设计指南》。  
[3] NIST SP 800-90B, *Recommendation for the Entropy Sources Used for Random Bit Generation*.  
[4] Go `crypto/rand` 文档，`Reader` 的平台实现说明。  
[5] `github.com/sealdice/dicescript` 当前版本 `rand_source.go`。  
[6] `github.com/sixafter/aes-ctr-drbg` 当前版本文档与源码。  
[7] `github.com/emmansun/gmsm` 当前版本 `rand/`、`drbg/` 与 `internal/entropy/` 文档和源码。  
[8] Go `math/rand/v2` 文档，关于“不可用于安全敏感场景”的说明。  
[9] Melissa E. O'Neill, *PCG: A Family of Simple Fast Space-Efficient Statistically Good Algorithms for Random Number Generation*, HMC-CS-2014-0905.  
[10] GM/T 0005-2021，《随机性检测规范》。  

## 引用库

对应 BibTeX 文件位于：

- `docs/references/randomness-honest-statement.bib`
