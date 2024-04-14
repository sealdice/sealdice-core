package dice

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	wr "github.com/mroth/weightedrand"
	"github.com/xuri/excelize/v2"
)

type NamesGenerator struct {
	NamesInfo map[string]map[string][]string
}

func (ng *NamesGenerator) Load() {
	_ = os.MkdirAll("./data/names", 0755)

	nameInfo := map[string]map[string][]string{}
	ng.NamesInfo = nameInfo

	for _, fn := range []string{"./data/names/names.xlsx", "./data/names/names-dnd.xlsx"} {
		f, err := excelize.OpenFile(fn)
		if err != nil {
			fmt.Println("加载names信息出错", fn, err)
			continue
		}

		for _, sheetName := range f.GetSheetList() {
			words := map[string][]string{}
			columns, err := f.Cols(sheetName)
			if err == nil {
				for columns.Next() {
					column, _ := columns.Rows()
					if len(column) > 0 {
						// 首行为标题，如“男性名” 其他行为内容，如”济民 珍祥“
						name := column[0]
						var values []string
						for _, i := range column[1:] {
							if i == "" {
								break
							}
							values = append(values, i)
						}
						// values := column[1:] // 注意行数是以最大行数算的，所以会出现很多空行，不能这样取
						words[name] = values
					}
				}
			}
			nameInfo[sheetName] = words
		}

		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}
}

func (ng *NamesGenerator) NameGenerate(rule string) string {
	// 规则说明:
	// 基本形式为 {sheetName:columnName} 例如 {中文:姓氏}
	// 权重扩展 {中文:姓氏@姓氏权重}
	// 位置扩展 {英文:名字} ({英文:名字中文#英文:名字.index}) 意思是在“名字中文”这一列中取值，行数与“名字”这一列的行数相同

	re := regexp.MustCompile(`\{[^}]+}`)
	tmpVars := map[string]int{}

	getList := func(inner string) []string {
		// TODO: 可在ng加缓存优化速度
		sp := strings.Split(inner, ":")
		if len(sp) > 1 {
			m, exists := ng.NamesInfo[sp[0]]
			if exists {
				lst, exists := m[sp[1]]
				if exists {
					return lst
				}
			}
		}
		return []string{}
	}

	getIntList := func(inner string) []int {
		// TODO: 可在ng加缓存优化速度
		lst := getList(inner)
		var result []int
		for _, i := range lst {
			weight, err := strconv.Atoi(i)
			if err != nil {
				_ = fmt.Errorf("权重转换出错，并非整数: %s, 来自 %s", i, rule)
				weight = 1
			}
			result = append(result, weight)
		}
		return result
	}

	parseWeight := func(inner string) (c *wr.Chooser, restText string, err error) {
		// TODO: 可加缓存，避免每次解析
		sp := strings.SplitN(inner, "@", 2)
		var choices []wr.Choice
		if len(sp) > 1 {
			lst := getList(sp[0])
			weightLst := getIntList(sp[1])

			// 取最小的，防止越界
			length := len(lst)
			length2 := len(weightLst)
			if length > length2 {
				length = length2
			}

			for index := 0; index < length; index++ {
				choices = append(choices, wr.NewChoice(index, uint(weightLst[index])))
			}
			restText = sp[0]
		} else {
			// 这里注意一点，如果遇到 {英文:名字中文#英文:名字.index} 这样的格式，choices会是空的
			// 但是没关系，因为不需要生成带权随机器
			lst := getList(inner)
			for index := range lst {
				choices = append(choices, wr.NewChoice(index, 1))
			}
			restText = inner
		}

		if len(choices) != 0 {
			c, err = wr.NewChooser(choices...)
		}
		return
	}

	parseInner := func(inner string, c *wr.Chooser) string {
		sp := strings.Split(inner, "#")
		if len(sp) > 1 {
			// 读取位置流程
			index := tmpVars[sp[1]]
			lst := getList(sp[0])
			if index < len(lst) {
				return lst[index]
			}
		} else {
			// 正常流程
			lst := getList(inner)
			if len(lst) == 0 {
				tmpVars[inner+".index"] = 0
				return ""
			}
			index := c.Pick().(int) // 取得权重
			tmpVars[inner+".index"] = index
			return lst[index]
		}
		return ""
	}

	result := ""
	lastLeft := 0
	for _, i := range re.FindAllStringIndex(rule, -1) {
		var c *wr.Chooser
		var err error
		inner := rule[i[0]+1 : i[1]-1]
		result += rule[lastLeft:i[0]]
		c, inner, err = parseWeight(inner)
		if err != nil {
			result += "<语句错误>"
		} else {
			result += parseInner(inner, c)
		}
		lastLeft = i[1]
	}

	result += rule[lastLeft:]
	return result
}
