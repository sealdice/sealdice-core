package random

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	ds "github.com/sealdice/dicescript"
	"go.uber.org/zap"
)

var ErrGMDisabled = errors.New("国密源未启用（默认禁用；启用后约 +32MB 内存用于熵采集，保证 SM3 随机数质量）")

type GlobalRand struct {
	mu         sync.RWMutex
	sources    map[Mode]ds.DiceSource
	errs       map[Mode]error
	activeMode Mode
	gmEnabled  bool
}

func NewEmptyGlobalOwner() *GlobalRand {
	return &GlobalRand{
		sources:    make(map[Mode]ds.DiceSource),
		errs:       make(map[Mode]error),
		activeMode: ModePCG,
	}
}

func NewGlobalOwner(logger *zap.SugaredLogger) *GlobalRand {
	owner := NewEmptyGlobalOwner()
	owner.RegisterSource(ModePCG, NewPCGSource(generateRandSeed()))
	for _, mode := range []Mode{ModeNIST, ModeCRNG} {
		src, err := NewSourceForMode(mode, logger)
		if err != nil {
			owner.RegisterSourceError(mode, err)
			if logger != nil {
				logger.Errorf("[随机源] %s 模式初始化失败: %v", ModeSpecFor(mode).Label, err)
			}
			continue
		}
		owner.RegisterSource(mode, src)
	}
	if err := owner.RegisterHybridSource(); err != nil {
		owner.RegisterSourceError(ModeHybrid, err)
	}
	owner.activeMode = ModePCG
	return owner
}

func (g *GlobalRand) RegisterSource(mode Mode, src ds.DiceSource) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.registerSourceLocked(mode, src)
}

func (g *GlobalRand) RegisterSourceError(mode Mode, err error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.registerSourceErrorLocked(mode, err)
}

func (g *GlobalRand) registerSourceLocked(mode Mode, src ds.DiceSource) {
	if g.sources == nil {
		g.sources = make(map[Mode]ds.DiceSource)
	}
	if g.errs == nil {
		g.errs = make(map[Mode]error)
	}
	if src == nil {
		delete(g.sources, mode)
		if mode == ModeGM {
			g.gmEnabled = false
		}
		g.errs[mode] = fmt.Errorf("random source %s is nil", mode)
		return
	}
	g.sources[mode] = src
	delete(g.errs, mode)
	if mode == ModeGM {
		g.gmEnabled = true
	}
}

func (g *GlobalRand) registerSourceErrorLocked(mode Mode, err error) {
	if g.errs == nil {
		g.errs = make(map[Mode]error)
	}
	if err == nil {
		delete(g.errs, mode)
		return
	}
	if g.sources != nil {
		delete(g.sources, mode)
	}
	g.errs[mode] = err
}

func (g *GlobalRand) sourceAvailableLocked(mode Mode) ds.DiceSource {
	src := g.sources[mode]
	if src == nil {
		return nil
	}
	if checker, ok := src.(sourceAvailability); ok && !checker.Available() {
		return nil
	}
	return src
}

func (g *GlobalRand) sourceErrorLocked(mode Mode) error {
	if err := g.errs[mode]; err != nil {
		return err
	}
	if mode == ModeGM && !g.gmEnabled {
		return ErrGMDisabled
	}
	return sourceErrorFromSource(g.sources[mode])
}

func fallbackModesAfter(preferred Mode) []Mode {
	modes := SupportedModes()
	if preferred == "" {
		return modes
	}

	index := -1
	for i, mode := range modes {
		if mode == preferred {
			index = i
			break
		}
	}
	if index < 0 {
		return modes
	}

	rotated := make([]Mode, 0, len(modes)-1)
	rotated = append(rotated, modes[index+1:]...)
	rotated = append(rotated, modes[:index]...)
	return rotated
}

func (g *GlobalRand) pickAvailableSourceLocked(preferred Mode) (Mode, ds.DiceSource) {
	if src := g.sourceAvailableLocked(preferred); src != nil {
		return preferred, src
	}
	for _, mode := range fallbackModesAfter(preferred) {
		if src := g.sourceAvailableLocked(mode); src != nil {
			return mode, src
		}
	}
	return "", nil
}

func (g *GlobalRand) RegisterHybridSource() error {
	g.mu.RLock()
	available := make(map[Mode]ds.DiceSource, len(g.sources))
	for _, mode := range HybridBaseModes() {
		if src := g.sources[mode]; src != nil {
			if checker, ok := src.(sourceAvailability); ok && !checker.Available() {
				continue
			}
			available[mode] = src
		}
	}
	g.mu.RUnlock()

	hybrid, err := buildHybridSourceFromAvailable(available)

	g.mu.Lock()
	defer g.mu.Unlock()
	if err != nil {
		if g.errs == nil {
			g.errs = make(map[Mode]error)
		}
		g.errs[ModeHybrid] = err
		delete(g.sources, ModeHybrid)
		return err
	}
	if g.sources == nil {
		g.sources = make(map[Mode]ds.DiceSource)
	}
	g.sources[ModeHybrid] = hybrid
	delete(g.errs, ModeHybrid)
	return nil
}

func (g *GlobalRand) SetActive(mode Mode) (Mode, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if src := g.sourceAvailableLocked(mode); src != nil {
		g.activeMode = mode
		return mode, nil
	}

	fallbackMode, fallbackSrc := g.pickAvailableSourceLocked(mode)
	if fallbackSrc != nil {
		g.activeMode = fallbackMode
		if initErr := g.sourceErrorLocked(mode); initErr != nil {
			return fallbackMode, initErr
		}
		return fallbackMode, fmt.Errorf("mode %s source unavailable", mode)
	}

	if initErr := g.sourceErrorLocked(mode); initErr != nil {
		return "", initErr
	}
	return "", fmt.Errorf("mode %s source unavailable", mode)
}

func sourceErrorFromSource(src ds.DiceSource) error {
	if src == nil {
		return nil
	}
	if checker, ok := src.(sourceErrorProvider); ok {
		return checker.SourceError()
	}
	return nil
}

func (g *GlobalRand) InitError(mode Mode) error {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.sourceErrorLocked(mode)
}

func (g *GlobalRand) GMEnabled() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.gmEnabled
}

// EnableGM 显式启用国密源，并让 hybrid 与 NIST 使用它。
func (g *GlobalRand) EnableGM(logger *zap.SugaredLogger) error {
	g.mu.RLock()
	enabled := g.gmEnabled
	g.mu.RUnlock()
	if enabled {
		return nil
	}

	gm, err := NewSourceForMode(ModeGM, logger)
	if err != nil {
		return err
	}
	nist, err := newSourceForMode(ModeNIST, logger, true)
	if err != nil {
		if logger != nil {
			logger.Warnf("[随机源] 重建 NIST 源以混入国密熵失败，将继续使用原 NIST 源: %v", err)
		}
		nist = nil
	}

	g.mu.Lock()
	if g.sources == nil {
		g.sources = make(map[Mode]ds.DiceSource)
	}
	if g.errs == nil {
		g.errs = make(map[Mode]error)
	}
	g.sources[ModeGM] = gm
	delete(g.errs, ModeGM)
	if nist != nil {
		g.sources[ModeNIST] = nist
		delete(g.errs, ModeNIST)
	}
	g.gmEnabled = true
	g.mu.Unlock()

	return g.RegisterHybridSource()
}

// DisableGM 关闭国密源。已分配的熵采集内存只有重启进程才会释放。
func (g *GlobalRand) DisableGM(logger *zap.SugaredLogger) error {
	g.mu.RLock()
	enabled := g.gmEnabled
	g.mu.RUnlock()
	if !enabled {
		return nil
	}

	nist, err := newSourceForMode(ModeNIST, logger, false)
	if err != nil {
		if logger != nil {
			logger.Warnf("[随机源] 重建 NIST 源以移除国密熵失败，将继续使用原 NIST 源: %v", err)
		}
		nist = nil
	}

	g.mu.Lock()
	delete(g.sources, ModeGM)
	delete(g.errs, ModeGM)
	if nist != nil {
		g.sources[ModeNIST] = nist
		delete(g.errs, ModeNIST)
	}
	g.gmEnabled = false
	if g.activeMode == ModeGM {
		g.activeMode = ModePCG
	}
	g.mu.Unlock()

	return g.RegisterHybridSource()
}

func (g *GlobalRand) CurrentMode() Mode {
	g.mu.Lock()
	defer g.mu.Unlock()
	mode, src := g.currentSourceLocked()
	if src == nil {
		return ""
	}
	return mode
}

func (g *GlobalRand) currentSourceLocked() (Mode, ds.DiceSource) {
	mode, src := g.pickAvailableSourceLocked(g.activeMode)
	if src != nil && mode != "" && mode != g.activeMode {
		g.activeMode = mode
	}
	return mode, src
}

func (g *GlobalRand) Uint64() uint64 {
	g.mu.Lock()
	_, src := g.currentSourceLocked()
	g.mu.Unlock()
	if src == nil {
		panic("global random source owner has no active source")
	}
	return src.Uint64()
}

func (g *GlobalRand) ReportGetText(mode Mode, points int64) string {
	lines := []string{fmt.Sprintf("随机源单次骰点测速 D%d", points)}

	if mode == "" {
		g.mu.Lock()
		currentMode, src := g.currentSourceLocked()
		gmEnabled := g.gmEnabled
		g.mu.Unlock()
		if src == nil {
			lines = append(lines, "当前生效模式: 无可用随机源")
		} else {
			lines = append(lines, formatSourceGetLine(currentMode, src, points))
		}
		if !gmEnabled {
			lines = append(lines, formatGMDisabledLine())
		}
		return strings.Join(lines, "\n")
	}

	g.mu.RLock()
	src := g.sources[mode]
	err := g.sourceErrorLocked(mode)
	gmEnabled := g.gmEnabled
	g.mu.RUnlock()

	if mode == ModeGM && !gmEnabled {
		return strings.Join(append(lines, formatGMDisabledLine()), "\n")
	}
	if src == nil || !sourceAvailable(src) {
		return strings.Join(append(lines, formatSourceUnavailableLine(mode, err)), "\n")
	}
	return strings.Join(append(lines, formatSourceGetLine(mode, src, points)), "\n")
}

func (g *GlobalRand) ReportAllGetText(points int64) string {
	g.mu.RLock()
	sources := make(map[Mode]ds.DiceSource, len(g.sources))
	for mode, src := range g.sources {
		sources[mode] = src
	}
	errs := make(map[Mode]error, len(g.errs))
	for mode, err := range g.errs {
		errs[mode] = err
	}
	gmEnabled := g.gmEnabled
	g.mu.RUnlock()

	lines := []string{fmt.Sprintf("随机源单次骰点测速 D%d", points)}
	for _, mode := range SupportedModes() {
		if mode == ModeGM && !gmEnabled {
			lines = append(lines, formatGMDisabledLine())
			continue
		}
		src := sources[mode]
		if src == nil || !sourceAvailable(src) {
			err := errs[mode]
			if err == nil {
				err = sourceErrorFromSource(src)
			}
			lines = append(lines, formatSourceUnavailableLine(mode, err))
			continue
		}
		lines = append(lines, formatSourceGetLine(mode, src, points))
	}
	return strings.Join(lines, "\n")
}

func sourceAvailable(src ds.DiceSource) bool {
	if src == nil {
		return false
	}
	if checker, ok := src.(sourceAvailability); ok && !checker.Available() {
		return false
	}
	return true
}

func formatSourceGetLine(mode Mode, src ds.DiceSource, points int64) string {
	start := time.Now()
	value := ds.Roll(src, ds.IntType(points), 0)
	return fmt.Sprintf("%s: 出目=%d 耗时=%s", mode, value, time.Since(start))
}

func formatSourceUnavailableLine(mode Mode, err error) string {
	if err != nil {
		return fmt.Sprintf("%s: 不可用 (%v)", mode, err)
	}
	return fmt.Sprintf("%s: 不可用 (mode %s source unavailable)", mode, mode)
}

func formatGMDisabledLine() string {
	return "gm: 未启用（默认禁用；启用后约 +32MB 内存用于熵采集，保证 SM3 随机数质量）"
}

func (g *GlobalRand) ReportStatusText(configuredMode Mode) string {
	g.mu.Lock()
	effectiveMode, src := g.currentSourceLocked()
	g.mu.Unlock()
	initErr := g.InitError(configuredMode)

	var text string
	if effectiveMode == "" || src == nil {
		if initErr == nil {
			initErr = errors.New("no available random source")
		}
		text = fmt.Sprintf(
			"当前随机模式: %s\n当前生效模式: 无可用随机源\n回退原因: %v",
			ModeSpecFor(configuredMode).Label,
			initErr,
		)
	} else if initErr == nil || effectiveMode == configuredMode {
		text = formatModeCommandText(configuredMode, src)
	} else {
		effectiveText := formatModeCommandText(effectiveMode, src)
		effectiveHeader := "当前随机模式: " + ModeSpecFor(effectiveMode).Label + "\n"
		effectiveText = strings.TrimPrefix(effectiveText, effectiveHeader)
		text = fmt.Sprintf(
			"当前随机模式: %s\n当前生效模式: %s\n回退原因: %v\n%s",
			ModeSpecFor(configuredMode).Label,
			ModeSpecFor(effectiveMode).Label,
			initErr,
			effectiveText,
		)
	}
	return text + "\n" + g.gmStatusLine()
}

func (g *GlobalRand) gmStatusLine() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.gmEnabled {
		return "国密源: 已启用（约 32MB 内存用于熵采集；关闭请执行 .randalgo set gm off 并重启海豹）"
	}
	return "国密源: 未启用（默认禁用；启用后约 +32MB 内存用于熵采集，保证 SM3 随机数质量）"
}

func (g *GlobalRand) LogActiveMode(logger *zap.SugaredLogger) {
	if logger == nil {
		return
	}
	g.mu.Lock()
	mode, src := g.currentSourceLocked()
	g.mu.Unlock()
	if src == nil {
		return
	}

	spec := ModeSpecFor(mode)
	details := spec.Description
	if note := SourceRuntimeNote(mode, src); note != "" {
		details += " " + note
	}
	logger.Infof(
		"[随机源] 当前使用 %s 模式：算法=%s；标准/口径=%s；特点=%s",
		spec.Label,
		spec.Algorithm,
		spec.Standard,
		details,
	)
}

func formatModeCommandText(mode Mode, src ds.DiceSource) string {
	spec := ModeSpecFor(mode)
	description := spec.Description
	if note := SourceRuntimeNote(mode, src); note != "" {
		description += "\n" + note
	}
	return fmt.Sprintf(
		"当前随机模式: %s\n算法: %s\n规范: %s\n特点: %s",
		spec.Label,
		spec.Algorithm,
		spec.Standard,
		description,
	)
}
