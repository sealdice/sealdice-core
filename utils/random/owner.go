package random

import (
	"fmt"
	"strings"
	"sync"
	"time"

	ds "github.com/sealdice/dicescript"
	"go.uber.org/zap"
)

type GlobalOwner struct {
	mu         sync.RWMutex
	sources    map[Mode]ds.DiceSource
	errs       map[Mode]error
	activeMode Mode
}

func NewEmptyGlobalOwner() *GlobalOwner {
	return &GlobalOwner{
		sources:    map[Mode]ds.DiceSource{},
		errs:       map[Mode]error{},
		activeMode: ModePCG,
	}
}

func NewGlobalOwner(logger *zap.SugaredLogger) *GlobalOwner {
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

func (g *GlobalOwner) RegisterSource(mode Mode, src ds.DiceSource) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.registerSourceLocked(mode, src)
}

func (g *GlobalOwner) RegisterSourceError(mode Mode, err error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.registerSourceErrorLocked(mode, err)
}

func (g *GlobalOwner) registerSourceLocked(mode Mode, src ds.DiceSource) {
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

func (g *GlobalOwner) registerSourceErrorLocked(mode Mode, err error) {
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

func (g *GlobalOwner) RegisterHybridSource() error {
	g.mu.RLock()
	available := make(map[Mode]ds.DiceSource, len(g.sources))
	for _, mode := range HybridBaseModes() {
		if src := g.sources[mode]; src != nil {
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

func (g *GlobalOwner) SetActive(mode Mode) (Mode, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if src := g.sources[mode]; src != nil {
		g.activeMode = mode
		return mode, nil
	}

	initErr := g.errs[mode]
	if src := g.sources[ModePCG]; src != nil {
		g.activeMode = ModePCG
		if initErr == nil {
			initErr = fmt.Errorf("mode %s source unavailable", mode)
		}
		return ModePCG, initErr
	}

	panic("global random source owner has no PCG fallback")
}

func (g *GlobalOwner) InitError(mode Mode) error {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.errs[mode]
}

func (g *GlobalOwner) CurrentMode() Mode {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if src := g.sources[g.activeMode]; src != nil {
		return g.activeMode
	}
	if src := g.sources[ModePCG]; src != nil {
		return ModePCG
	}
	for _, mode := range SupportedModes() {
		if src := g.sources[mode]; src != nil {
			return mode
		}
	}
	return ""
}

func (g *GlobalOwner) currentSourceLocked() ds.DiceSource {
	if src := g.sources[g.activeMode]; src != nil {
		return src
	}
	if src := g.sources[ModePCG]; src != nil {
		return src
	}
	for _, mode := range SupportedModes() {
		if src := g.sources[mode]; src != nil {
			return src
		}
	}
	return nil
}

func (g *GlobalOwner) snapshotSources() (map[Mode]ds.DiceSource, map[Mode]error, Mode) {
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

func (g *GlobalOwner) Uint64() uint64 {
	g.mu.RLock()
	src := g.currentSourceLocked()
	g.mu.RUnlock()
	if src == nil {
		panic("global random source owner has no active source")
	}
	return src.Uint64()
}

func (g *GlobalOwner) ReportGetText(points int64) string {
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

		start := time.Now()
		value := ds.Roll(src, ds.IntType(points), 0)
		elapsed := time.Since(start)
		lines = append(lines, fmt.Sprintf("%s: 出目=%d 耗时=%s", mode, value, elapsed))
	}
	return strings.Join(lines, "\n")
}

func (g *GlobalOwner) ReportStatusText(configuredMode Mode) string {
	sources, _, effectiveMode := g.snapshotSources()
	src := sources[effectiveMode]
	initErr := g.InitError(configuredMode)
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

func (g *GlobalOwner) LogActiveMode(logger *zap.SugaredLogger) {
	if logger == nil {
		return
	}
	g.mu.RLock()
	mode := g.activeMode
	src := g.currentSourceLocked()
	g.mu.RUnlock()
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
