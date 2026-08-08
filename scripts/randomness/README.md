# SealDice 随机性样本与检测脚本

这个目录提供两类能力：

1. 生成符合 `rddetector` 输入要求的随机样本；
2. 使用本地 `rddetector_linux_amd64.tar.gz` 生成 CSV 报告与 Markdown 汇总。

## 1. 生成样本

样本生成入口放在 `dice/randomness_report_test.go` 中，默认不会在普通测试里执行。只有显式设置环境变量后才会运行。

示例：

```bash
SEALDICE_RANDOMNESS_GENERATE=1 \
SEALDICE_RANDOMNESS_OUT_DIR=temp/randomness/manual \
SEALDICE_RANDOMNESS_MODES=pcg,crypto,nist,gm \
SEALDICE_RANDOMNESS_SAMPLES=20 \
SEALDICE_RANDOMNESS_BITS=1000000 \
go test ./dice -run TestGenerateRandomnessSamples -count=1
```

关键环境变量：

- `SEALDICE_RANDOMNESS_OUT_DIR`：输出目录
- `SEALDICE_RANDOMNESS_MODES`：模式列表，默认 `pcg,crypto,nist,gm`
- `SEALDICE_RANDOMNESS_SAMPLES`：每种模式生成多少个样本
- `SEALDICE_RANDOMNESS_BITS`：每个样本多少 bit，支持 `20000`、`1000000`

## 2. 运行 rddetector 报告脚本

仓库假设本地已经有：

`docs/rddetector_linux_amd64.tar.gz`

脚本会自动：

1. 解压 `rddetector`
2. 生成随机样本
3. 调用 `rddetector`
4. 输出原始 CSV 报告
5. 生成 Markdown 汇总

### quick

```bash
bash scripts/randomness/run_rddetector_reports.sh quick
```

- 每模式 20 个样本
- 每样本 20,000 bit

### poweron

```bash
bash scripts/randomness/run_rddetector_reports.sh poweron
```

- 每模式 20 个样本
- 每样本 1,000,000 bit

### factory

```bash
bash scripts/randomness/run_rddetector_reports.sh factory
```

- 每模式 50 个样本
- 每样本 1,000,000 bit

### strict-0005

这是严格对齐 `GM/T 0005-2021` 样本集口径的模式，默认使用：

- 每模式 `1000` 个样本
- 每样本 `1,000,000 bit`

```bash
bash scripts/randomness/run_rddetector_reports.sh strict-0005
```

该模式适合做正式标准口径报告。

### 其他严格档位

```bash
bash scripts/randomness/run_rddetector_reports.sh strict-0005-20k
bash scripts/randomness/run_rddetector_reports.sh strict-0005-100m
```

- `strict-0005-20k`：`1000 x 20,000 bit`
- `strict-0005-100m`：`1000 x 100,000,000 bit`

其中 `strict-0005-100m` 计算与存储成本很高，实际运行前应先确认机器资源。

## 3. 输出位置

脚本默认输出到：

- `docs/randomness-reports/<profile>/round-1/*.csv`
- `docs/randomness-reports/<profile>/summary.md`

原始样本默认放在：

- `temp/randomness/<profile>/samples/`

## 4. 多轮运行

如果需要多轮重复采样，可以使用：

```bash
ROUNDS=3 bash scripts/randomness/run_rddetector_reports.sh poweron
```

脚本会按轮次输出多组报告，并在 `summary.md` 中汇总各轮结果。

## 5. 调试覆盖

正式 profile 已经内置标准样本数和 bit 长度。如果只想做流程验证，可以临时覆盖：

```bash
SAMPLES_OVERRIDE=2 BITS_OVERRIDE=20000 \
bash scripts/randomness/run_rddetector_reports.sh strict-0005
```

这只适合冒烟测试，不应作为正式标准结果引用。

## 6. 脚本自测

可用下面的命令检查 profile 解析是否符合预期：

```bash
bash scripts/randomness/run_rddetector_reports_test.sh
```
