package dice

import (
	"fmt"
	"github.com/xuri/excelize/v2"
	"math/rand"
	"os"
	"regexp"
	"strings"
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
						//values := column[1:] // 注意行数是以最大行数算的，所以会出现很多空行，不能这样取
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
	re := regexp.MustCompile(`\{[^}]+}`)
	tmpVars := map[string]int{}

	getList := func(inner string) []string {
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

	parseInner := func(inner string) string {
		sp := strings.Split(inner, "#")
		if len(sp) > 1 {
			index := tmpVars[sp[1]]
			lst := getList(sp[0])
			if index < len(lst) {
				return lst[index]
			}
		} else {
			lst := getList(inner)
			if len(lst) == 0 {
				tmpVars[inner+".index"] = 0
				return ""
			}
			index := rand.Int() % len(lst)
			tmpVars[inner+".index"] = index
			return lst[index]
		}
		return ""
	}

	result := ""
	lastLeft := 0
	for _, i := range re.FindAllStringIndex(rule, -1) {
		inner := rule[i[0]+1 : i[1]-1]
		result += rule[lastLeft:i[0]]
		result += parseInner(inner)
		lastLeft = i[1]
	}

	result += rule[lastLeft:]
	return result
}
