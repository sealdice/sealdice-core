package dice

import (
	"fmt"
	"github.com/xuri/excelize/v2"
	"os"
	"strconv"
)

type local struct {
	surname    map[string]float64
	maleName   []string
	femaleName []string
	firstName  []string
}

type NamesGenerator struct {
	names      map[string]local
	aliasNames map[string]string
}

func (ng *NamesGenerator) Load() {
	_ = os.MkdirAll("./data/names", 0755)
	ng.names = make(map[string]local)
	ng.aliasNames = make(map[string]string)
	for _, fn := range []string{"./data/names/names.xlsx", "./data/names/names-dnd.xlsx"} {
		f, err := excelize.OpenFile(fn)
		if err != nil {
			fmt.Println("加载names信息出错", fn, err)
			continue
		}

		for _, sheetName := range f.GetSheetList() {
			var l local
			l.surname = make(map[string]float64)
			cols, _ := f.GetCols(sheetName)
			for i, col := range cols {
				cols[i] = col[1:]
			}
			switch sheetName {
			case "中文":
				l.maleName = append(l.maleName, cols[0]...)
				l.femaleName = append(l.femaleName, cols[1]...)
				for i, s := range cols[2] {
					w, _ := strconv.ParseFloat(cols[3][i], 64)
					l.surname[s] = w
				}
			case "英文":
				for i, s := range cols[0] {
					l.firstName = append(l.firstName, s)
					ng.aliasNames[s] = cols[1][i]
				}
				for i, s := range cols[2] {
					l.surname[s] = 1
					ng.aliasNames[s] = cols[3][i]
				}
			case "日文":
				ng.c6(&cols, &l)
			case "DND地精":
				for i, s := range cols[0] {
					l.maleName = append(l.maleName, s)
					ng.aliasNames[s] = cols[1][i]
				}
				for i, s := range cols[2] {
					l.femaleName = append(l.femaleName, s)
					ng.aliasNames[s] = cols[3][i]
				}
			case "DND海族":
				// 暂时没有女名 和 姓
				for i, s := range cols[0] {
					l.firstName = append(l.firstName, s)
					ng.aliasNames[s] = cols[1][i]
				}
			case "DND兽人":
				ng.c6(&cols, &l)
			case "DND矮人":
				ng.c6(&cols, &l)
			case "DND精灵":
				ng.c6(&cols, &l)
			case "DND受国人":
				ng.c6(&cols, &l)
			case "DND莱瑟曼人":
				ng.c6(&cols, &l)
			case "DND卡林珊人":
				ng.c6(&cols, &l)
			}
			l.removeEmptyStrings()
			ng.names[sheetName] = l
		}
	}
}

func (l *local) removeEmptyStrings() {
	one := func(sl []string) []string {
		var res []string
		for _, str := range sl {
			if str != "" {
				res = append(res, str)
			}
		}
		return res
	}
	l.maleName = one(l.maleName)
	l.femaleName = one(l.femaleName)
	l.firstName = one(l.firstName)
}

func (ng *NamesGenerator) c6(col *[][]string, l *local) {
	cols := *col
	for i, s := range cols[0] {
		l.maleName = append(l.maleName, s)
		ng.aliasNames[s] = cols[1][i]
	}
	for i, s := range cols[2] {
		l.femaleName = append(l.femaleName, s)
		ng.aliasNames[s] = cols[3][i]
	}
	for i, s := range cols[4] {
		ng.aliasNames[s] = cols[5][i]
		l.surname[s] = 1
	}
}
