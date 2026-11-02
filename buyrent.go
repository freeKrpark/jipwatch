package main

import (
	"math"
	"strconv"
)

type BuyRentInput struct {
	Price       int     `json:"price"`
	Jeonse      int     `json:"jeonse"`
	Cash        int     `json:"cash"`
	Area        int     `json:"area"`
	LoanRate    float64 `json:"loanRate"`
	JeonseRate  float64 `json:"jeonseRate"`
	OppRate     float64 `json:"oppRate"`
	MaintRate   float64 `json:"maintRate"`
	Years       int     `json:"years"`
	PropertyTax int     `json:"propertyTax"`
}

type CostLine struct {
	Label  string `json:"label"`
	Amount int    `json:"amount"`
}

type Sensitivity struct {
	OppRate   float64 `json:"oppRate"`
	BreakEven float64 `json:"breakEven"`
}

type BuyRentResult struct {
	Loan         int           `json:"loan"`
	AcquireCost  int           `json:"acquireCost"`
	PropertyTax  int           `json:"propertyTax"`
	TaxEstimated bool          `json:"taxEstimated"`
	Buy          []CostLine    `json:"buy"`
	BuyTotal     int           `json:"buyTotal"`
	Rent         []CostLine    `json:"rent"`
	RentTotal    int           `json:"rentTotal"`
	BreakEven    float64       `json:"breakEven"`
	Sensitivity  []Sensitivity `json:"sensitivity"`
}

func acquisitionTax(price, area int) int {
	eok := float64(price) / 10000
	var rate float64
	switch {
	case eok <= 6:
		rate = 1
	case eok <= 9:
		rate = eok*2/3 - 3
	default:
		rate = 3
	}
	tax := float64(price) * rate / 100
	tax += tax / 10
	if area > 85 {
		tax += float64(price) * 0.2 / 100
	}
	return int(math.Round(tax))
}

func brokerFee(amount int, sale bool) int {
	eok := float64(amount) / 10000
	var rate float64
	switch {
	case sale && eok >= 15:
		rate = 0.7
	case sale && eok >= 12:
		rate = 0.6
	case sale && eok >= 9:
		rate = 0.5
	case sale:
		rate = 0.4
	case eok >= 15:
		rate = 0.6
	case eok >= 12:
		rate = 0.5
	case eok >= 6:
		rate = 0.4
	default:
		rate = 0.3
	}
	return int(math.Round(float64(amount) * rate / 100))
}

func propertyTaxEstimate(price int) int {
	base := float64(price) * 0.7 * 0.6
	var tax float64
	switch {
	case base <= 6000:
		tax = base * 0.001
	case base <= 15000:
		tax = 6 + (base-6000)*0.0015
	case base <= 30000:
		tax = 19.5 + (base-15000)*0.0025
	default:
		tax = 57 + (base-30000)*0.004
	}
	return int(math.Round(tax + base*0.0014 + tax/5))
}

func buyRent(in BuyRentInput) BuyRentResult {
	r := BuyRentResult{AcquireCost: acquisitionTax(in.Price, in.Area) + brokerFee(in.Price, true)}
	r.PropertyTax = in.PropertyTax
	if r.PropertyTax <= 0 {
		r.PropertyTax, r.TaxEstimated = propertyTaxEstimate(in.Price), true
	}
	years := in.Years
	if years < 1 {
		years = 1
	}

	need := in.Price + r.AcquireCost
	r.Loan = max(0, need-in.Cash)
	equity := need - r.Loan - r.AcquireCost // cash sitting in the home itself
	if equity < 0 {
		equity = 0
	}
	yearly := func(base int, rate float64) int { return int(math.Round(float64(base) * rate / 100)) }

	buyLines := func(opp float64) []CostLine {
		return []CostLine{
			{"대출 이자", yearly(r.Loan, in.LoanRate)},
			{"집에 묶인 현금의 기회비용", yearly(equity, opp)},
			{"보유세", r.PropertyTax},
			{"유지보수", yearly(in.Price, in.MaintRate)},
			{fmtYears("취득세·중개수수료", years), int(math.Round(float64(r.AcquireCost) / float64(years)))},
		}
	}
	rentLines := func(opp float64) []CostLine {
		own := min(in.Cash, in.Jeonse)
		return []CostLine{
			{"보증금에 묶인 현금의 기회비용", yearly(own, opp)},
			{"전세대출 이자", yearly(in.Jeonse-own, in.JeonseRate)},
			{"중개수수료 (2년마다)", brokerFee(in.Jeonse, false) / 2},
		}
	}
	total := func(lines []CostLine) int {
		n := 0
		for _, l := range lines {
			n += l.Amount
		}
		return n
	}
	breakEven := func(opp float64) float64 {
		if in.Price <= 0 {
			return 0
		}
		return float64(total(buyLines(opp))-total(rentLines(opp))) / float64(in.Price) * 100
	}

	r.Buy, r.Rent = buyLines(in.OppRate), rentLines(in.OppRate)
	r.BuyTotal, r.RentTotal = total(r.Buy), total(r.Rent)
	r.BreakEven = breakEven(in.OppRate)
	for _, opp := range []float64{3, 5, 7} {
		r.Sensitivity = append(r.Sensitivity, Sensitivity{opp, breakEven(opp)})
	}
	return r
}

func fmtYears(label string, years int) string {
	return label + " ÷ " + strconv.Itoa(years) + "년"
}

type Growth struct {
	Since   string  `json:"since"`
	All     float64 `json:"all"`
	Recent5 float64 `json:"recent5"`
}

func growth(points []TrendPoint) Growth {
	var valid []TrendPoint
	for _, p := range points {
		if p.Median > 0 {
			valid = append(valid, p)
		}
	}
	if len(valid) < 2 {
		return Growth{}
	}
	first, last := valid[0], valid[len(valid)-1]
	g := Growth{Since: first.Month, All: cagr(first, last)}
	if from := addMonths(last.Month, -60); from >= first.Month {
		for _, p := range valid {
			if p.Month >= from {
				g.Recent5 = cagr(p, last)
				break
			}
		}
	}
	return g
}

func cagr(a, b TrendPoint) float64 {
	months := monthsBetween(a.Month, b.Month)
	if months <= 0 || a.Median <= 0 {
		return 0
	}
	return (math.Pow(float64(b.Median)/float64(a.Median), 12/float64(months)) - 1) * 100
}

func monthsBetween(a, b string) int {
	ya, _ := strconv.Atoi(a[:4])
	ma, _ := strconv.Atoi(a[5:7])
	yb, _ := strconv.Atoi(b[:4])
	mb, _ := strconv.Atoi(b[5:7])
	return (yb-ya)*12 + mb - ma
}
