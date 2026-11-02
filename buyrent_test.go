package main

import (
	"math"
	"testing"
)

func TestAcquisitionTax(t *testing.T) {
	cases := []struct {
		price, area, want int
	}{
		{50000, 84, 550},    // 5억: 1% + 교육세 0.1%
		{75000, 84, 1650},   // 7.5억: 2% + 0.2%
		{120000, 84, 3960},  // 12억: 3% + 0.3%
		{120000, 114, 4200}, // above 85㎡: + 농특세 0.2%
	}
	for _, c := range cases {
		if got := acquisitionTax(c.price, c.area); got != c.want {
			t.Errorf("acquisitionTax(%d, %d) = %d, want %d", c.price, c.area, got, c.want)
		}
	}
}

func TestBrokerFee(t *testing.T) {
	if got := brokerFee(120000, true); got != 720 { // 12억 sale: 0.6%
		t.Errorf("sale fee %d", got)
	}
	if got := brokerFee(60000, false); got != 240 { // 6억 lease: 0.4%
		t.Errorf("lease fee %d", got)
	}
	if got := brokerFee(50000, false); got != 150 { // 5억 lease: 0.3%
		t.Errorf("lease fee %d", got)
	}
}

// A worked example, checked by hand: a 12억 home against a 6억 전세 with
// 8.5억 cash, a 4.5% loan, 3% on cash, 0.5% upkeep, kept 10 years.
func TestBuyRentWorkedExample(t *testing.T) {
	r := buyRent(BuyRentInput{
		Price: 120000, Jeonse: 60000, Cash: 85000, Area: 84,
		LoanRate: 4.5, OppRate: 3, MaintRate: 0.5, Years: 10,
	})
	// Need 12억 + 4,680만 (취득세 3,960 + 수수료 720); 8.5억 cash leaves a 3억 9,680만 loan.
	if r.AcquireCost != 4680 || r.Loan != 39680 {
		t.Fatalf("acquire %d, loan %d", r.AcquireCost, r.Loan)
	}
	// Owning: interest 1,786 + cash in the home 8억 320만 × 3% 2,410 + tax
	// + upkeep 600 + 468 a year of buying costs.
	wantBuy := 1786 + 2410 + r.PropertyTax + 600 + 468
	if r.BuyTotal != wantBuy || !r.TaxEstimated {
		t.Fatalf("buy %d, want %d: %+v", r.BuyTotal, wantBuy, r.Buy)
	}
	// Renting: 6억 of cash × 3% 1,800, no 전세대출, 240 fee every two years.
	if r.RentTotal != 1800+120 {
		t.Fatalf("rent %d: %+v", r.RentTotal, r.Rent)
	}
	want := float64(wantBuy-1920) / 120000 * 100
	if math.Abs(r.BreakEven-want) > 0.001 {
		t.Fatalf("break-even %.3f, want %.3f", r.BreakEven, want)
	}
	// A higher return on cash makes owning, which ties up more of it, dearer.
	if len(r.Sensitivity) != 3 || !(r.Sensitivity[0].BreakEven < r.Sensitivity[2].BreakEven) {
		t.Fatalf("sensitivity %+v", r.Sensitivity)
	}
}

func TestBuyRentJeonseGapIsBorrowed(t *testing.T) {
	r := buyRent(BuyRentInput{Price: 100000, Jeonse: 60000, Cash: 40000, LoanRate: 5, JeonseRate: 4, OppRate: 3, Years: 10})
	if r.Rent[0].Amount != 1200 || r.Rent[1].Amount != 800 {
		t.Fatalf("cash 4억 × 3%% and a 2억 전세대출 × 4%%: %+v", r.Rent)
	}
}

func TestPropertyTaxGivenIsUsed(t *testing.T) {
	r := buyRent(BuyRentInput{Price: 120000, Jeonse: 60000, Cash: 85000, PropertyTax: 300, Years: 10})
	if r.PropertyTax != 300 || r.TaxEstimated {
		t.Fatalf("%+v", r)
	}
}

func TestGrowth(t *testing.T) {
	// Doubling over ten years is about 7.18% a year; the last five years
	// rose from 150 to 200, about 5.92% a year.
	points := []TrendPoint{
		{Month: "2016-01", Median: 100},
		{Month: "2021-01", Median: 150},
		{Month: "2026-01", Median: 200},
	}
	g := growth(points)
	if g.Since != "2016-01" || math.Abs(g.All-7.177) > 0.01 || math.Abs(g.Recent5-5.922) > 0.01 {
		t.Fatalf("%+v", g)
	}
	if g := growth(points[1:2]); g.All != 0 {
		t.Fatalf("one point is no growth: %+v", g)
	}
}
