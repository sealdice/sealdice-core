package random

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	randv2 "math/rand/v2"
	"strings"
	"sync"
	"time"

	gmrand "github.com/emmansun/gmsm/rand"
	ds "github.com/sealdice/dicescript"
	ctrdrbg "github.com/sixafter/aes-ctr-drbg"
	"go.uber.org/zap"
)

const (
	nistReseedInterval = 5 * time.Minute
	nistReseedRequests = 4096
	gmEntropyMixSize   = 32
)

type sourceAvailability interface {
	Available() bool
}

type sourceErrorProvider interface {
	SourceError() error
}

type runtimeNoteProvider interface {
	RuntimeNote() string
}

type readerSourceState struct {
	reader      io.Reader
	mu          sync.Mutex
	failed      bool
	failErr     error
	spec        ModeSpec
	logger      *zap.SugaredLogger
	runtimeNote string
}

func (s *readerSourceState) Uint64() uint64 {
	s.mu.Lock()
	if s.failed || s.reader == nil {
		s.mu.Unlock()
		return 0
	}

	var data [8]byte
	if _, err := io.ReadFull(s.reader, data[:]); err != nil {
		s.failed = true
		s.failErr = err
		s.reader = nil
		spec := s.spec
		logger := s.logger
		note := strings.TrimSpace(s.runtimeNote)
		s.mu.Unlock()
		if logger != nil {
			logger.Errorf(
				"[随机源][降级] %s 模式读取失败，算法=%s，标准/口径=%s，特点=%s。该源已标记为不可用，后续将由全局随机源选择其他可用源。错误: %v",
				spec.Label,
				spec.Algorithm,
				spec.Standard,
				spec.Description,
				err,
			)
			if note != "" {
				logger.Debugf("[随机源] %s 模式运行时说明: %s", spec.Label, note)
			}
		}
		return 0
	}
	s.mu.Unlock()
	return binary.BigEndian.Uint64(data[:])
}

func (s *readerSourceState) Available() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return !s.failed && s.reader != nil
}

func (s *readerSourceState) SourceError() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.failErr
}

func (s *readerSourceState) RuntimeNote() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return strings.TrimSpace(s.runtimeNote)
}

type gmSource struct {
	readerSourceState
}

type nistSource struct {
	readerSourceState
}

type pcgSource struct {
	mu  sync.Mutex
	pcg *randv2.PCG
}

func NewPCGSource(seed uint64) ds.StatefulDiceSource {
	return &pcgSource{pcg: randv2.NewPCG(seed, seed)}
}

func (s *pcgSource) Uint64() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pcg.Uint64()
}

func (s *pcgSource) MarshalBinary() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pcg.MarshalBinary()
}

func (s *pcgSource) UnmarshalBinary(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pcg == nil {
		s.pcg = randv2.NewPCG(0, 0)
	}
	return s.pcg.UnmarshalBinary(data)
}

type hybridSource struct {
	sources     []ds.DiceSource
	runtimeNote string
}

func (s *hybridSource) Uint64() uint64 {
	var value uint64
	for _, src := range s.sources {
		if src == nil {
			continue
		}
		value ^= src.Uint64()
	}
	return value
}

func (s *hybridSource) RuntimeNote() string {
	return strings.TrimSpace(s.runtimeNote)
}

func SourceRuntimeNote(mode Mode, src ds.DiceSource) string {
	if src == nil {
		return ""
	}
	if noter, ok := src.(runtimeNoteProvider); ok {
		note := strings.TrimSpace(noter.RuntimeNote())
		if note == "" {
			return ""
		}
		if mode == ModeNIST {
			return "熵补充: " + note
		}
		return note
	}
	return ""
}

func buildHybridSourceFromAvailable(available map[Mode]ds.DiceSource) (ds.DiceSource, error) {
	sources := make([]ds.DiceSource, 0, len(available))
	labels := make([]string, 0, len(available))
	for _, mode := range HybridBaseModes() {
		src := available[mode]
		if src == nil {
			continue
		}
		if avail, ok := src.(sourceAvailability); ok && !avail.Available() {
			continue
		}
		sources = append(sources, src)
		labels = append(labels, ModeSpecFor(mode).Label)
	}

	if len(sources) == 0 {
		return nil, errors.New("no available random source to build hybrid mode")
	}

	return &hybridSource{
		sources:     sources,
		runtimeNote: "当前混合源包含: " + strings.Join(labels, ", "),
	}, nil
}

// NewSourceForMode creates one concrete random source for a mode.
func NewSourceForMode(mode Mode, logger *zap.SugaredLogger) (ds.DiceSource, error) {
	spec := ModeSpecFor(mode)
	switch mode {
	case ModeGM:
		return &gmSource{
			readerSourceState: readerSourceState{
				reader: gmrand.Reader,
				spec:   spec,
				logger: logger,
			},
		}, nil
	case ModeNIST:
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
		return &nistSource{
			readerSourceState: readerSourceState{
				reader:      reader,
				spec:        spec,
				logger:      logger,
				runtimeNote: runtimeNote,
			},
		}, nil
	case ModeCRNG:
		return ds.NewCryptoDiceSource(), nil
	case ModePCG:
		fallthrough
	default:
		return NewPCGSource(generateRandSeed()), nil
	}
}
