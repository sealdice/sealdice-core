package dice

import (
	"fmt"
	"testing"
)

const times = 10000

func TestRollExportPic(t *testing.T) {
	m := map[int]int{}
	//v := make(plotter.Values, times)
	for i := 0; i < times; i++ {
		value := Roll(100)
		//v[i] = float64(value)
		m[value]++
	}

	//p := plot.New()
	//p.Title.Text = "Normal Roll"
	//
	//h, err := plotter.NewHist(v, 25)
	//if err != nil {
	//	panic(err)
	//}
	//h.Normalize(1)
	//p.Add(h)
	//
	//_ = os.MkdirAll("../dist", 0755)
	//if err := p.Save(4*vg.Inch, 4*vg.Inch, "../dist/test_roll.png"); err != nil {
	//	panic(err)
	//}
	fmt.Printf("%+v", m)
}

func TestRollGaussExportPic(t *testing.T) {
	m := map[int]int{}
	//v := make(plotter.Values, times)
	for i := 0; i < times; i++ {
		value := RollGauss(100)
		//v[i] = float64(value)
		m[value]++
	}

	//p := plot.New()
	//p.Title.Text = "Gauss Roll"
	//
	//h, err := plotter.NewHist(v, 25)
	//if err != nil {
	//	panic(err)
	//}
	//h.Normalize(1)
	//p.Add(h)
	//
	//_ = os.MkdirAll("../dist", 0755)
	//if err := p.Save(4*vg.Inch, 4*vg.Inch, "../dist/test_roll_gauss.png"); err != nil {
	//	panic(err)
	//}

	fmt.Printf("%+v", m)
}
