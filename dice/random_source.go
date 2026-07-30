package dice

import (
	"encoding/binary"
	"fmt"
	"io"
	randv2 "math/rand/v2"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
	gmrand "github.com/emmansun/gmsm/rand"
	wr "github.com/mroth/weightedrand/v3"
	ds "github.com/sealdice/dicescript"
	ctrdrbg "github.com/sixafter/aes-ctr-drbg"
	"go.uber.org/zap"
)

type DiceRandomMode string

const (
	DiceRandomModePCG    DiceRandomMode = "pcg"
	DiceRandomModeGM     DiceRandomMode = "gm"
	DiceRandomModeNIST   DiceRandomMode = "nist"
	DiceRandomModeCrypto DiceRandomMode = "crypto"
)

const (
	nistReseedInterval = 5 * time.Minute
	nistReseedRequests = 4096
	gmEntropyMixSize   = 32
	gojaRandScale      = 1.0 / (1 << 53)
)

type diceRandomModeSpec struct {
	label       string
	algorithm   string
	standard    string
	shortDesc   string
	description string
}

type readerDiceSource struct {
	reader      io.Reader
	mu          sync.Mutex
	fallback    ds.DiceSource
	spec        diceRandomModeSpec
	logger      *zap.SugaredLogger
	runtimeNote string
}

type pcgDiceSource struct {
	mu  sync.Mutex
	pcg *randv2.PCG
}

func (s *readerDiceSource) Uint64() uint64 {
	s.mu.Lock()
	if s.reader == nil {
		fallback := s.fallback
		if fallback == nil {
			fallback = newPCGDiceSource(generateRandSeed())
			s.fallback = fallback
		}
		s.mu.Unlock()
		return fallback.Uint64()
	}

	var data [8]byte
	if _, err := io.ReadFull(s.reader, data[:]); err != nil {
		fallback := s.fallback
		if fallback == nil {
			fallback = newPCGDiceSource(generateRandSeed())
			s.fallback = fallback
		}
		s.reader = nil
		spec := s.spec
		logger := s.logger
		s.mu.Unlock()
		if logger != nil {
			logger.Errorf(
				"[随机源][降级] %s 模式读取失败，算法=%s，标准/口径=%s，特点=%s。已自动切换到 PCG 默认模式。错误: %v",
				spec.label,
				spec.algorithm,
				spec.standard,
				spec.description,
				err,
			)
		}
		return fallback.Uint64()
	}
	s.mu.Unlock()
	return binary.BigEndian.Uint64(data[:])
}

func newPCGDiceSource(seed uint64) ds.StatefulDiceSource {
	return &pcgDiceSource{pcg: randv2.NewPCG(seed, seed)}
}

func (s *pcgDiceSource) Uint64() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pcg.Uint64()
}

func (s *pcgDiceSource) MarshalBinary() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pcg.MarshalBinary()
}

func (s *pcgDiceSource) UnmarshalBinary(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pcg == nil {
		s.pcg = randv2.NewPCG(0, 0)
	}
	return s.pcg.UnmarshalBinary(data)
}

func normalizeDiceRandomMode(raw string) DiceRandomMode {
	switch DiceRandomMode(strings.ToLower(strings.TrimSpace(raw))) {
	case DiceRandomModeGM:
		return DiceRandomModeGM
	case DiceRandomModeNIST:
		return DiceRandomModeNIST
	case DiceRandomModeCrypto:
		return DiceRandomModeCrypto
	default:
		return DiceRandomModePCG
	}
}

var supportedDiceRandomModes = []DiceRandomMode{
	DiceRandomModePCG,
	DiceRandomModeGM,
	DiceRandomModeNIST,
	DiceRandomModeCrypto,
}

func parseDiceRandomModeStrict(raw string) (DiceRandomMode, bool) {
	switch DiceRandomMode(strings.ToLower(strings.TrimSpace(raw))) {
	case DiceRandomModePCG:
		return DiceRandomModePCG, true
	case DiceRandomModeGM:
		return DiceRandomModeGM, true
	case DiceRandomModeNIST:
		return DiceRandomModeNIST, true
	case DiceRandomModeCrypto:
		return DiceRandomModeCrypto, true
	default:
		return "", false
	}
}

func getDiceRandomModeSpec(mode DiceRandomMode) diceRandomModeSpec {
	switch mode {
	case DiceRandomModeGM:
		return diceRandomModeSpec{
			label:     "GM 国密",
			algorithm: "SM3 Hash DRBG",
			standard:  "GM/T 0105-2021（并结合 SP 800-90B 健康检测）",
			shortDesc: "GM 国密，SM3 Hash DRBG",
			description: "符合国内商密行业标准 GM/T 0105-2021 的国密随机方案，采用 SM3 Hash DRBG，并结合多源熵输入、" +
				"已知答案自检与 SP 800-90B 健康检测机制；兼容国密/商密体系，在生成质量、规范一致性和安全设计方面表现极为突出" +
				"，但计算、初始化开销相对较高。",
		}
	case DiceRandomModeNIST:
		return diceRandomModeSpec{
			label:     "NIST",
			algorithm: "AES-CTR-DRBG",
			standard:  "NIST SP 800-90A Rev.1（启用 prediction resistance、自检、连续健康检测与主动重播种策略）",
			shortDesc: "NIST，AES-CTR-DRBG",
			description: "基于 NIST SP 800-90A Rev.1 的 AES-CTR-DRBG 增强方案，本质上以 AES 计数器模式维护内部状态，采用 AES-256、实例级 personalization、" +
				"prediction resistance、已知答案自检、连续健康检测、主动重播种与密钥轮换策略，" +
				"在标准一致性、持续保护与审计可解释性方面表现突出。",
		}
	case DiceRandomModeCrypto:
		return diceRandomModeSpec{
			label:     "系统级随机数",
			algorithm: "操作系统原生随机数接口",
			standard:  "Linux 默认 getrandom(2)，老版本回退 /dev/urandom；Windows 使用 ProcessPrng API DRBG",
			shortDesc: "操作系统随机源",
			description: "基于操作系统提供的密码学安全随机源，直接调用系统级熵池能力，来源可靠、与平台安全机制天然集成。" +
				"Linux 默认使用 getrandom(2) 获取随机字节，较老系统可能回退 /dev/urandom；" +
				"Windows 使用 ProcessPrng API DRBG 从系统安全子系统取数。" +
				"速度一般，依赖操作系统原生安全能力。",
		}
	case DiceRandomModePCG:
		fallthrough
	default:
		return diceRandomModeSpec{
			label:     "默认 PCG",
			algorithm: "PCG",
			standard:  "通过 TestU01、PractRand 等常见随机性测试",
			shortDesc: "高性能置换同余发生器",
			description: "基于置换同余发生器的高性能随机算法，程序结合多维运行时信息生成初始种子以保证其不可预测性，" +
				"以状态推进与输出置换设计，兼顾速度与分布质量。" +
				"通过 TestU01、PractRand 等常见随机性测试，在很多维度上比传统 rand() 或部分老牌发生器更均匀稳定；支持极高吞吐随机调用。",
		}
	}
}

func (d *Dice) newGojaRandSource() goja.RandSource {
	return func() float64 {
		if d == nil {
			return float64(randSource.Uint64()>>11) * gojaRandScale
		}
		return float64(d.getSystemDiceSource().Uint64()>>11) * gojaRandScale
	}
}

func newDiceSourceForMode(mode DiceRandomMode, logger *zap.SugaredLogger) (ds.DiceSource, error) {
	spec := getDiceRandomModeSpec(mode)
	switch mode {
	case DiceRandomModeGM:
		return &readerDiceSource{reader: gmrand.Reader, spec: spec, logger: logger}, nil
	case DiceRandomModeNIST:
		runtimeNote := "系统熵路径。"
		reader, err := ctrdrbg.NewReader(
			ctrdrbg.WithKeySize(ctrdrbg.KeySize256),
			ctrdrbg.WithPersonalization([]byte("sealdice-nist")),
			ctrdrbg.WithPredictionResistance(true),
			ctrdrbg.WithSelfTests(true),
			ctrdrbg.WithContinuousHealthTest(true),
			ctrdrbg.WithEnableKeyRotation(true),
			ctrdrbg.WithReseedInterval(nistReseedInterval),
			ctrdrbg.WithReseedRequests(nistReseedRequests),
		)
		if err != nil {
			return nil, err
		}

		buf := make([]byte, gmEntropyMixSize)
		n := 0
		readErr := func() (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("read gm entropy panic: %v", r)
				}
			}()
			n, err = gmrand.Read(buf)
			return err
		}()
		if readErr == nil && n == len(buf) {
			if err := reader.Reseed(buf); err != nil {
				if logger != nil {
					logger.Warnf("[随机源] NIST 模式混入 GM 国密熵源失败，将继续使用原生 NIST 熵路径。错误: reseed nist reader with gm entropy: %v", err)
				}
			} else {
				runtimeNote = "启动时额外注入了基于 GM/T 0105-2021、SM3 Hash DRBG 的国密随机源输出，作为 reseed seed material 的附加输入，在密码学上进一步扩充了熵输入强度。"
			}
		} else if logger != nil {
			if readErr != nil {
				logger.Warnf("[随机源] NIST 模式混入 GM 国密熵源失败，将继续使用原生 NIST 熵路径。错误: read gm entropy: %v", readErr)
			} else {
				logger.Warnf("[随机源] NIST 模式混入 GM 国密熵源失败，将继续使用原生 NIST 熵路径。错误: read gm entropy: short read %d/%d", n, len(buf))
			}
		}
		clear(buf)
		return &readerDiceSource{reader: reader, spec: spec, logger: logger, runtimeNote: runtimeNote}, nil
	case DiceRandomModeCrypto:
		return ds.NewCryptoDiceSource(), nil
	case DiceRandomModePCG:
		fallthrough
	default:
		return newPCGDiceSource(generateRandSeed()), nil
	}
}

func (d *Dice) getDiceRandomMode() DiceRandomMode {
	return normalizeDiceRandomMode(d.Config.DiceRandomMode)
}

func (d *Dice) newDiceSource() ds.DiceSource {
	src, err := newDiceSourceForMode(d.getDiceRandomMode(), d.Logger)
	if err != nil {
		panic(fmt.Errorf("dice random mode %q init failed: %w", d.getDiceRandomMode(), err))
	}
	return src
}

func (d *Dice) getSystemDiceSource() ds.DiceSource {
	mode := d.getDiceRandomMode()

	d.randomSourceMu.Lock()
	defer d.randomSourceMu.Unlock()

	if d.systemDiceSource == nil || d.systemDiceMode != mode {
		d.systemDiceSource = d.newDiceSource()
		d.systemDiceMode = mode
		d.logDiceRandomMode(mode, d.systemDiceSource)
	}
	return d.systemDiceSource
}

func (d *Dice) logDiceRandomMode(mode DiceRandomMode, src ds.DiceSource) {
	if d == nil || d.Logger == nil {
		return
	}
	spec := getDiceRandomModeSpec(mode)
	details := spec.description
	if mode == DiceRandomModeNIST {
		if readerSrc, ok := src.(*readerDiceSource); ok {
			if note := strings.TrimSpace(readerSrc.runtimeNote); note != "" {
				details += " 熵补充: " + note
			}
		}
	}
	d.Logger.Infof(
		"[随机源] 当前使用 %s 模式：算法=%s；标准/口径=%s；特点=%s",
		spec.label,
		spec.algorithm,
		spec.standard,
		details,
	)
}

func formatDiceRandomModeCommandText(mode DiceRandomMode, src ds.DiceSource) string {
	spec := getDiceRandomModeSpec(mode)
	description := spec.description
	if mode == DiceRandomModeNIST {
		if readerSrc, ok := src.(*readerDiceSource); ok {
			if note := strings.TrimSpace(readerSrc.runtimeNote); note != "" {
				description += "\n熵补充: " + note
			}
		}
	}
	return fmt.Sprintf(
		"当前随机模式: %s\n算法: %s\n规范: %s\n特点: %s",
		spec.label,
		spec.algorithm,
		spec.standard,
		description,
	)
}

func formatSupportedDiceRandomModesText() string {
	lines := make([]string, 0, len(supportedDiceRandomModes))
	for _, mode := range supportedDiceRandomModes {
		spec := getDiceRandomModeSpec(mode)
		lines = append(lines, fmt.Sprintf("%s // %s", mode, spec.shortDesc))
	}
	return strings.Join(lines, "\n")
}

func formatDiceRandomModeHelpText() string {
	return strings.Join([]string{
		"查看随机算法:",
		".randalgo // 查看当前随机算法、对应规范和简介",
		".randalgo set <模式> // 设置随机模式，仅Master可用",
		"支持的模式:",
		formatSupportedDiceRandomModesText(),
	}, "\n")
}

func formatDiceRandomModeSetSuccessText(mode DiceRandomMode) string {
	return fmt.Sprintf("已切换随机模式为 %s，使用 .randalgo 查看详情", mode)
}

func formatDiceRandomModeSetMissingModeText() string {
	return "请提供随机模式。\n支持的模式:\n" + formatSupportedDiceRandomModesText()
}

func formatDiceRandomModeSetInvalidModeText(raw string) string {
	return fmt.Sprintf("不支持的随机模式: %s\n支持的模式:\n%s", raw, formatSupportedDiceRandomModesText())
}

func (d *Dice) Roll(points int) int {
	if points <= 0 {
		return 0
	}
	return int(ds.Roll(d.getSystemDiceSource(), ds.IntType(points), 0))
}

func (d *Dice) Roll64(points int64) int64 {
	return DiceRoll64x(d.getSystemDiceSource(), points)
}

func (ctx *MsgContext) getDiceSource() ds.DiceSource {
	if ctx.diceRandSrc != nil {
		ctx._v1Rand = ctx.diceRandSrc
		return ctx.diceRandSrc
	}
	if ctx.Dice != nil {
		if ctx.Dice.getDiceRandomMode() == DiceRandomModeNIST {
			ctx.diceRandSrc = ctx.Dice.getSystemDiceSource()
		} else {
			ctx.diceRandSrc = ctx.Dice.newDiceSource()
		}
	} else {
		ctx.diceRandSrc = randSource
	}
	ctx._v1Rand = ctx.diceRandSrc
	return ctx.diceRandSrc
}

func (ctx *MsgContext) getChooserRand() *randv2.Rand {
	src := normalizeDiceSource(ctx.getDiceSource())
	if ctx.chooserRand == nil || ctx.chooserSrc != src {
		ctx.chooserSrc = src
		ctx.chooserRand = randv2.New(src)
	}
	return ctx.chooserRand
}

func (ctx *MsgContext) Roll(points int) int {
	if points <= 0 {
		return 0
	}
	return int(ds.Roll(ctx.getDiceSource(), ds.IntType(points), 0))
}

func (ctx *MsgContext) Roll64(points int64) int64 {
	return DiceRoll64x(ctx.getDiceSource(), points)
}

func (d *Dice) RandIntn(n int) int {
	return randIntnFromSource(d.getSystemDiceSource(), n)
}

func (ctx *MsgContext) RandIntn(n int) int {
	return randIntnFromSource(ctx.getDiceSource(), n)
}

func (ctx *MsgContext) Shuffle(n int, swap func(i, j int)) {
	shuffleWithSource(ctx.getDiceSource(), n, swap)
}

func (d *Dice) Shuffle(n int, swap func(i, j int)) {
	shuffleWithSource(d.getSystemDiceSource(), n, swap)
}

type chooserWeight interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

func pickChooserWithRand[T any, W chooserWeight](chooser *wr.Chooser[T, W], rand *randv2.Rand) T {
	if chooser == nil {
		var zero T
		return zero
	}
	return chooser.PickWith(rand)
}

func randIntnFromSource(src ds.DiceSource, n int) int {
	if n <= 0 {
		panic("invalid bound")
	}
	return int(randUint64n(src, uint64(n)))
}

func randUint64n(src ds.DiceSource, n uint64) uint64 {
	if n == 0 {
		panic("invalid bound")
	}
	src = normalizeDiceSource(src)
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

func shuffleWithSource(src ds.DiceSource, n int, swap func(i, j int)) {
	for i := n - 1; i > 0; i-- {
		j := randIntnFromSource(src, i+1)
		swap(i, j)
	}
}

func normalizeDiceSource(src ds.DiceSource) ds.DiceSource {
	if isNilDiceSource(src) {
		return randSource
	}
	return src
}

func isNilDiceSource(src ds.DiceSource) bool {
	if src == nil {
		return true
	}
	v := reflect.ValueOf(src)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
