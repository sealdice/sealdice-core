package main

import (
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type analysisRow struct {
	TestName   string
	PassCount  int
	TotalCount int
	PassRate   float64
	IsPassed   bool
}

type aggregateRow struct {
	TestName          string
	Rounds            int
	PassedRounds      int
	AveragePassRate   float64
	MinPassRate       float64
	MaxPassRate       float64
	AveragePassCount  float64
	AverageTotalCount float64
}

type modeReport struct {
	Mode          string
	Rounds        int
	AllPassed     bool
	RowsByTest    map[string]*aggregateRow
	OrderedTests  []string
	SummaryLine   string
	SampleCount   int
	BitsPerSample int
}

type modeComparison struct {
	Mode              string
	TotalTests        int
	StablePassTests   int
	AveragePassRate   float64
	LowestPassRate    float64
	LowestPassRateKey string
}

func main() {
	inDir := flag.String("in", "", "input directory containing round-*/<mode>-analysis.csv")
	outFile := flag.String("out", "", "output markdown summary file")
	profile := flag.String("profile", "poweron", "report profile name")
	rounds := flag.Int("rounds", 1, "number of rounds that were generated")
	samples := flag.Int("samples", 20, "sample files per round")
	bits := flag.Int("bits", 1000000, "bits per sample file")
	flag.Parse()

	if *inDir == "" || *outFile == "" {
		fmt.Fprintln(os.Stderr, "-in and -out are required")
		os.Exit(1)
	}

	reports, err := collectModeReports(*inDir, *rounds, *samples, *bits)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	md := buildMarkdown(*profile, reports)
	if err := os.WriteFile(*outFile, []byte(md), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", *outFile, err)
		os.Exit(1)
	}
}

func collectModeReports(inDir string, rounds int, samples int, bits int) ([]modeReport, error) {
	modeMap := map[string]*modeReport{}

	for round := 1; round <= rounds; round++ {
		pattern := filepath.Join(inDir, fmt.Sprintf("round-%d", round), "*-analysis.csv")
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("glob %s: %w", pattern, err)
		}
		sort.Strings(matches)
		for _, match := range matches {
			mode := strings.TrimSuffix(filepath.Base(match), "-analysis.csv")
			rows, err := readAnalysisCSV(match)
			if err != nil {
				return nil, fmt.Errorf("read %s: %w", match, err)
			}

			report := modeMap[mode]
			if report == nil {
				report = &modeReport{
					Mode:          mode,
					AllPassed:     true,
					RowsByTest:    map[string]*aggregateRow{},
					SampleCount:   samples,
					BitsPerSample: bits,
				}
				modeMap[mode] = report
			}
			report.Rounds++

			for _, row := range rows {
				agg := report.RowsByTest[row.TestName]
				if agg == nil {
					agg = &aggregateRow{
						TestName:    row.TestName,
						MinPassRate: row.PassRate,
						MaxPassRate: row.PassRate,
					}
					report.RowsByTest[row.TestName] = agg
					report.OrderedTests = append(report.OrderedTests, row.TestName)
				}
				agg.Rounds++
				if row.IsPassed {
					agg.PassedRounds++
				}
				agg.AveragePassRate += row.PassRate
				agg.AveragePassCount += float64(row.PassCount)
				agg.AverageTotalCount += float64(row.TotalCount)
				if row.PassRate < agg.MinPassRate {
					agg.MinPassRate = row.PassRate
				}
				if row.PassRate > agg.MaxPassRate {
					agg.MaxPassRate = row.PassRate
				}
				if !row.IsPassed {
					report.AllPassed = false
				}
			}
		}
	}

	modes := make([]string, 0, len(modeMap))
	for mode := range modeMap {
		modes = append(modes, mode)
	}
	sort.Strings(modes)

	reports := make([]modeReport, 0, len(modes))
	for _, mode := range modes {
		report := modeMap[mode]
		sort.Strings(report.OrderedTests)
		for _, testName := range report.OrderedTests {
			agg := report.RowsByTest[testName]
			agg.AveragePassRate /= float64(agg.Rounds)
			agg.AveragePassCount /= float64(agg.Rounds)
			agg.AverageTotalCount /= float64(agg.Rounds)
		}
		if report.AllPassed {
			report.SummaryLine = "所有检测项目在本次汇总范围内均满足通过判定。"
		} else {
			report.SummaryLine = "存在未在所有轮次中稳定通过的检测项目，需结合分析报告与样本轮次一起解读。"
		}
		reports = append(reports, *report)
	}
	return reports, nil
}

func readAnalysisCSV(path string) ([]analysisRow, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) <= 1 {
		return nil, errors.New("no analysis rows")
	}

	header := rows[0]
	index := make(map[string]int, len(header))
	for i, col := range header {
		normalized := strings.TrimSpace(strings.TrimPrefix(col, "\ufeff"))
		index[normalized] = i
	}

	nameColumn := "检测项目"
	if _, ok := index[nameColumn]; !ok {
		if _, ok := index["检测项目（含参数）"]; ok {
			nameColumn = "检测项目（含参数）"
		}
	}

	required := []string{nameColumn, "通过数", "检测数", "通过率", "是否通过"}
	for _, key := range required {
		if _, ok := index[key]; !ok {
			return nil, fmt.Errorf("missing required column %q", key)
		}
	}

	out := make([]analysisRow, 0, len(rows)-1)
	for _, row := range rows[1:] {
		if len(row) == 0 {
			continue
		}
		passCount, err := strconv.Atoi(strings.TrimSpace(row[index["通过数"]]))
		if err != nil {
			return nil, fmt.Errorf("parse pass count: %w", err)
		}
		totalCount, err := strconv.Atoi(strings.TrimSpace(row[index["检测数"]]))
		if err != nil {
			return nil, fmt.Errorf("parse total count: %w", err)
		}
		passRate, err := strconv.ParseFloat(strings.TrimSpace(row[index["通过率"]]), 64)
		if err != nil {
			return nil, fmt.Errorf("parse pass rate: %w", err)
		}

		passedText := strings.TrimSpace(row[index["是否通过"]])
		isPassed := passedText == "true" || passedText == "TRUE" || passedText == "是" || passedText == "通过"

		out = append(out, analysisRow{
			TestName:   strings.TrimSpace(row[index[nameColumn]]),
			PassCount:  passCount,
			TotalCount: totalCount,
			PassRate:   passRate,
			IsPassed:   isPassed,
		})
	}

	return out, nil
}

func buildMarkdown(profile string, reports []modeReport) string {
	var b strings.Builder
	b.WriteString("# SealDice 随机性检测汇总报告\n\n")
	b.WriteString(fmt.Sprintf("- 检测配置：`%s`\n", profile))
	if len(reports) > 0 {
		b.WriteString(fmt.Sprintf("- 样本规模：每轮每模式 `%d` 个文件，每文件 `%d` bit\n", reports[0].SampleCount, reports[0].BitsPerSample))
	}
	b.WriteString("- 检测工具：`rddetector`（GM/T 0005-2021 15 项检测）\n\n")
	if len(reports) > 0 {
		required := requiredPassCount(reports[0].SampleCount)
		b.WriteString(fmt.Sprintf(
			"说明：在当前配置下，单项检测默认要求每轮至少 `%d/%d` 个样本满足规范阈值（约 `98.1%%`）。因此低于该阈值的结果会被记为未满通过，但仍应结合样本规模、轮次和项目类型一起解读。\n\n",
			required,
			reports[0].SampleCount,
		))
	}

	comparisons := buildComparisons(reports)
	if len(comparisons) > 0 {
		b.WriteString("## 四种模式总览对比\n\n")
		b.WriteString("| 模式 | 检测项目数 | 各轮稳定通过项目数 | 平均单项通过率 | 最低单项通过率 | 最低项 |\n")
		b.WriteString("| --- | ---: | ---: | ---: | ---: | --- |\n")
		for _, item := range comparisons {
			b.WriteString(fmt.Sprintf(
				"| `%s` | %d | %d | %.4f | %.4f | %s |\n",
				item.Mode,
				item.TotalTests,
				item.StablePassTests,
				item.AveragePassRate,
				item.LowestPassRate,
				item.LowestPassRateKey,
			))
		}
		b.WriteString("\n")
	}

	for _, report := range reports {
		b.WriteString(fmt.Sprintf("## `%s`\n\n", report.Mode))
		b.WriteString(report.SummaryLine + "\n\n")
		b.WriteString("| 检测项目 | 通过轮次 | 平均通过率 | 最低通过率 | 最高通过率 | 平均通过样本数 |\n")
		b.WriteString("| --- | ---: | ---: | ---: | ---: | ---: |\n")
		for _, testName := range report.OrderedTests {
			row := report.RowsByTest[testName]
			b.WriteString(fmt.Sprintf(
				"| %s | %d/%d | %.4f | %.4f | %.4f | %.2f/%.2f |\n",
				row.TestName,
				row.PassedRounds,
				row.Rounds,
				row.AveragePassRate,
				row.MinPassRate,
				row.MaxPassRate,
				row.AveragePassCount,
				row.AverageTotalCount,
			))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func requiredPassCount(samples int) int {
	if samples <= 0 {
		return 0
	}
	alpha := 0.01
	s := float64(samples)
	threshold := s * (1 - alpha - 3*math.Sqrt((alpha*(1-alpha))/s))
	return int(math.Ceil(threshold))
}

func buildComparisons(reports []modeReport) []modeComparison {
	out := make([]modeComparison, 0, len(reports))
	for _, report := range reports {
		if len(report.OrderedTests) == 0 {
			continue
		}
		item := modeComparison{
			Mode:              report.Mode,
			TotalTests:        len(report.OrderedTests),
			LowestPassRate:    1.0,
			LowestPassRateKey: report.OrderedTests[0],
		}
		for _, testName := range report.OrderedTests {
			row := report.RowsByTest[testName]
			item.AveragePassRate += row.AveragePassRate
			if row.PassedRounds == report.Rounds {
				item.StablePassTests++
			}
			if row.MinPassRate < item.LowestPassRate {
				item.LowestPassRate = row.MinPassRate
				item.LowestPassRateKey = row.TestName
			}
		}
		item.AveragePassRate /= float64(item.TotalTests)
		out = append(out, item)
	}

	sort.SliceStable(out, func(i, j int) bool {
		order := map[string]int{"pcg": 0, "crypto": 1, "nist": 2, "gm": 3}
		oi, oki := order[out[i].Mode]
		oj, okj := order[out[j].Mode]
		if oki && okj {
			return oi < oj
		}
		return out[i].Mode < out[j].Mode
	})

	return out
}
