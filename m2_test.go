package main

import (
	"math"
	"testing"

	"github.com/freekrpark/jipwatch/internal/store"
)

func m2points(start string, n int, m2start, m2step float64) ([]TrendPoint, []store.EcosPoint) {
	trend := make([]TrendPoint, n)
	ecos := make([]store.EcosPoint, n)
	for i := range n {
		m := addMonths(start, i)
		trend[i] = TrendPoint{Month: m, Median: 100000, Sales: 5, Radius: 3}
		ym := m[:4] + m[5:7]
		ecos[i] = store.EcosPoint{YM: ym, Value: m2start + float64(i)*m2step}
	}
	return trend, ecos
}

func TestM2ProportionalGrowthFlat(t *testing.T) {
	// Price and M2 both stay constant → adjusted is exactly constant → Z = 0.
	trend, m2data := m2points("2024-01", 24, 3000, 0)
	a := m2analysis(trend, m2data, nil)

	if !a.Enough || len(a.Points) != 24 {
		t.Fatalf("enough %v, points %d", a.Enough, len(a.Points))
	}
	if a.Z != 0 {
		t.Fatalf("constant price / constant M2 should yield z=0, got %.2f", a.Z)
	}
	// All adjusted values should equal the nominal price.
	for _, p := range a.Points {
		if p.Adjusted != float64(p.Nominal) {
			t.Fatalf("adjusted %f != nominal %d", p.Adjusted, p.Nominal)
		}
	}
}

func TestM2GrowthMatchesM2Flat(t *testing.T) {
	// Price doubles from 100k to 200k, M2 doubles from 3000 to 6000.
	// Each step multiplies by the same factor → adjusted stays constant.
	trend := make([]TrendPoint, 24)
	m2data := make([]store.EcosPoint, 24)
	for i := range 24 {
		m := addMonths("2024-01", i)
		factor := 1.0 + float64(i)/23.0 // 1.0 → 2.0
		price := int(100000 * factor)
		m2val := 3000.0 * factor
		trend[i] = TrendPoint{Month: m, Median: price, Sales: 5, Radius: 3}
		m2data[i] = store.EcosPoint{YM: m[:4] + m[5:7], Value: m2val}
	}
	a := m2analysis(trend, m2data, nil)

	if !a.Enough {
		t.Fatal("should have enough months")
	}
	// Int truncation of price introduces tiny noise; the adjusted range is
	// under 1 만원 on a 100,000 base, so the spread is near-zero and Z
	// amplifies rounding.  Assert adjusted values are close to the base.
	base := a.Points[0].Adjusted
	for _, p := range a.Points {
		if math.Abs(p.Adjusted-base)/base > 0.001 {
			t.Fatalf("month %s adjusted %.0f drifted >0.1%% from base %.0f", p.Month, p.Adjusted, base)
		}
	}
}

func TestM2PriceOutpacesM2(t *testing.T) {
	// Price rises 50% in last 6 months while M2 stays flat → adjusted spikes → high Z.
	trend := make([]TrendPoint, 24)
	m2data := make([]store.EcosPoint, 24)
	for i := range 24 {
		m := addMonths("2024-01", i)
		price := 100000
		if i >= 18 {
			price = 150000
		}
		trend[i] = TrendPoint{Month: m, Median: price, Sales: 5, Radius: 3}
		m2data[i] = store.EcosPoint{YM: m[:4] + m[5:7], Value: 3000.0}
	}
	a := m2analysis(trend, m2data, nil)

	if a.Z < 1 {
		t.Fatalf("price outpacing M2 should show positive Z, got %.2f", a.Z)
	}
}

func TestM2WithRateOverlay(t *testing.T) {
	trend := []TrendPoint{{Month: "2024-01", Median: 100000, Sales: 5, Radius: 3}}
	m2data := []store.EcosPoint{{YM: "202401", Value: 3000}}
	rateData := []store.EcosPoint{{YM: "202401", Value: 3.5}}
	a := m2analysis(trend, m2data, rateData)

	if len(a.Points) != 1 || a.Points[0].Rate != 3.5 {
		t.Fatalf("rate overlay: %+v", a.Points)
	}
	if a.Points[0].Nominal != 100000 || a.Points[0].M2 != 3000 {
		t.Fatalf("values: %+v", a.Points[0])
	}
}

func TestM2NoData(t *testing.T) {
	trend := []TrendPoint{{Month: "2024-01", Median: 100000}}
	a := m2analysis(trend, nil, nil)
	if len(a.Points) != 0 || a.Enough {
		t.Fatalf("no M2 data should be empty: %+v", a)
	}
}

func TestM2NotEnoughMonths(t *testing.T) {
	trend, m2data := m2points("2024-01", 6, 3000, 0)
	a := m2analysis(trend, m2data, nil)
	if a.Enough {
		t.Fatal("6 months should not be enough")
	}
	if len(a.Points) != 6 {
		t.Fatalf("got %d points", len(a.Points))
	}
}
