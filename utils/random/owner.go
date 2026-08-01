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

type GlobalRand struct {
	mu         sync.RWMutex
	sources    map[Mode]ds.DiceSource
	errs       map[Mode]error
	activeMode Mode
}

func NewEmptyGlobalOwner() *GlobalRand {
	return &GlobalRand{
		sources:    map[Mode]ds.DiceSource{},
		errs:       map[Mode]error{},
		activeMode: ModePCG,
	}
}

func NewGlobalOwner(logger *zap.SugaredLogger) *GlobalRand {
	owner := NewEmptyGlobalOwner()
	owner.RegisterSource(ModePCG, NewPCGSource(generateRandSeed()))
	for _, mode := range []Mode{ModeGM, ModeNIST, ModeCRNG} {
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
		g.sources = map[Mode]ds.DiceSource{}
	}
	if g.errs == nil {
		g.errs = map[Mode]error{}
	}
	if src == nil {
		delete(g.sources, mode)
		g.errs[mode] = fmt.Errorf("random source %s is nil", mode)
		return
	}
	g.sources[mode] = src
	delete(g.errs, mode)
}

func (g *GlobalRand) registerSourceErrorLocked(mode Mode, err error) {
	if g.errs == nil {
		g.errs = map[Mode]error{}
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
			g.errs = map[Mode]error{}
		}
		g.errs[ModeHybrid] = err
		delete(g.sources, ModeHybrid)
		return err
	}
	if g.sources == nil {
		g.sources = map[Mode]ds.DiceSource{}
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
	if err := g.errs[mode]; err != nil {
		return err
	}
	return sourceErrorFromSource(g.sources[mode])
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

func (g *GlobalRand) snapshotSources() (map[Mode]ds.DiceSource, map[Mode]error, Mode) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	sources := make(map[Mode]ds.DiceSource, len(g.sources))
	for mode, src := range g.sources {
		sources[mode] = src
	}
	errs := make(map[Mode]error, len(g.errs))
	for mode, err := range g.errs {
		errs[mode] = err
	}
	return sources, errs, g.activeMode
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

func (g *GlobalRand) ReportGetText(points int64) string {
	sources, errs, _ := g.snapshotSources()
	lines := []string{fmt.Sprintf("随机源单次骰点测速 D%d", points)}
	for _, mode := range SupportedModes() {
		src := sources[mode]
		if src == nil {
			if err := errs[mode]; err != nil {
				lines = append(lines, fmt.Sprintf("%s: 不可用 (%v)", mode, err))
				continue
			}
			lines = append(lines, fmt.Sprintf("%s: 不可用 (mode %s source unavailable)", mode, mode))
			continue
		}
		if checker, ok := src.(sourceAvailability); ok && !checker.Available() {
			if err := sourceErrorFromSource(src); err != nil {
				lines = append(lines, fmt.Sprintf("%s: 不可用 (%v)", mode, err))
				continue
			}
			lines = append(lines, fmt.Sprintf("%s: 不可用 (mode %s source unavailable)", mode, mode))
			continue
		}

		start := time.Now()
		value := ds.Roll(src, ds.IntType(points), 0)
		elapsed := time.Since(start)
		lines = append(lines, fmt.Sprintf("%s: 出目=%d 耗时=%s", mode, value, elapsed))
	}
	return strings.Join(lines, "\n")
}

func (g *GlobalRand) ReportStatusText(configuredMode Mode) string {
	g.mu.Lock()
	effectiveMode, src := g.currentSourceLocked()
	g.mu.Unlock()
	initErr := g.InitError(configuredMode)
	if effectiveMode == "" || src == nil {
		if initErr == nil {
			initErr = errors.New("no available random source")
		}
		return fmt.Sprintf(
			"当前随机模式: %s\n当前生效模式: 无可用随机源\n回退原因: %v",
			ModeSpecFor(configuredMode).Label,
			initErr,
		)
	}
	if initErr == nil || effectiveMode == configuredMode {
		return formatModeCommandText(configuredMode, src)
	}

	effectiveText := formatModeCommandText(effectiveMode, src)
	effectiveHeader := "当前随机模式: " + ModeSpecFor(effectiveMode).Label + "\n"
	effectiveText = strings.TrimPrefix(effectiveText, effectiveHeader)

	return fmt.Sprintf(
		"当前随机模式: %s\n当前生效模式: %s\n回退原因: %v\n%s",
		ModeSpecFor(configuredMode).Label,
		ModeSpecFor(effectiveMode).Label,
		initErr,
		effectiveText,
	)
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
