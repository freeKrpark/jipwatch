package molit

import "strings"

type Rent struct {
	AptSeq  string  `json:"aptSeq"`
	AptNm   string  `json:"aptNm"`
	SggCd   string  `json:"sggCd"`
	UmdNm   string  `json:"umdNm"`
	Jibun   string  `json:"jibun"`
	Area    float64 `json:"area"`
	Floor   int     `json:"floor"`
	Deposit int     `json:"deposit"`
	Monthly int     `json:"monthly"`
	Date    string  `json:"date"` // contract date

	Kind         string `json:"kind"`
	Term         string `json:"term"`
	PreDeposit   int    `json:"preDeposit"`
	PreMonthly   int    `json:"preMonthly"`
	RenewalRight bool   `json:"renewalRight"`
	BuildYear    int    `json:"buildYear"`
}

type rentItem struct {
	AptSeq         string `xml:"aptSeq"`
	AptNm          string `xml:"aptNm"`
	SggCd          string `xml:"sggCd"`
	UmdNm          string `xml:"umdNm"`
	Jibun          string `xml:"jibun"`
	ExcluUseAr     string `xml:"excluUseAr"`
	Floor          string `xml:"floor"`
	Deposit        string `xml:"deposit"`
	MonthlyRent    string `xml:"monthlyRent"`
	DealYear       string `xml:"dealYear"`
	DealMonth      string `xml:"dealMonth"`
	DealDay        string `xml:"dealDay"`
	ContractType   string `xml:"contractType"`
	ContractTerm   string `xml:"contractTerm"`
	PreDeposit     string `xml:"preDeposit"`
	PreMonthlyRent string `xml:"preMonthlyRent"`
	UseRRRight     string `xml:"useRRRight"`
	BuildYear      string `xml:"buildYear"`
}

func (i rentItem) name() string { return clean(i.AptNm) }

func (i rentItem) rent() (Rent, error) {
	var p parser

	r := Rent{
		AptSeq:       clean(i.AptSeq),
		AptNm:        clean(i.AptNm),
		SggCd:        clean(i.SggCd),
		UmdNm:        clean(i.UmdNm),
		Jibun:        clean(i.Jibun),
		Area:         p.float("excluUseAr", i.ExcluUseAr),
		Floor:        p.int("floor", i.Floor),
		Deposit:      p.amount("deposit", i.Deposit),
		Monthly:      p.amount("monthlyRent", i.MonthlyRent),
		Date:         p.date(i.DealYear, i.DealMonth, i.DealDay),
		Kind:         clean(i.ContractType),
		Term:         clean(i.ContractTerm),
		PreDeposit:   p.optional("preDeposit", i.PreDeposit),
		PreMonthly:   p.optional("preMonthlyRent", i.PreMonthlyRent),
		RenewalRight: clean(i.UseRRRight) == "사용",
		BuildYear:    p.int("buildYear", i.BuildYear),
	}

	if r.AptSeq == "" {
		p.fail("aptSeq", i.AptSeq)
	}
	if p.err != nil {
		return Rent{}, p.err
	}
	return r, nil
}

// amount reads a sum in 만원, which may carry commas.
func (p *parser) amount(field, raw string) int {
	return p.int(field, strings.ReplaceAll(raw, ",", ""))
}

// optional is amount for a field that is blank when it does not apply.
func (p *parser) optional(field, raw string) int {
	if clean(raw) == "" {
		return 0
	}
	return p.amount(field, raw)
}
