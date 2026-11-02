package molit

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Trade struct {
	AptSeq       string  `json:"aptSeq"` // complex id, e.g. "11110-0001"
	AptNm        string  `json:"aptNm"`
	SggCd        string  `json:"sggCd"` // 5-digit district code
	UmdNm        string  `json:"umdNm"`
	Jibun        string  `json:"jibun"`
	Area         float64 `json:"area"` // 전용면적 ㎡
	Floor        int     `json:"floor"`
	Price        int     `json:"price"` // 만원
	Date         string  `json:"date"`  // contract date
	Direct       bool    `json:"direct"`
	Canceled     bool    `json:"canceled"`
	CanceledOn   string  `json:"canceledOn,omitempty"`
	RegisteredOn string  `json:"registeredOn,omitempty"` // empty until registered
	BuildYear    int     `json:"buildYear"`
}

// item mirrors the XML. Every field stays a string: blanks arrive as a single
// space, amounts carry commas, and dates are split or in YY.MM.DD.
type item struct {
	AptSeq     string `xml:"aptSeq"`
	AptNm      string `xml:"aptNm"`
	SggCd      string `xml:"sggCd"`
	UmdNm      string `xml:"umdNm"`
	Jibun      string `xml:"jibun"`
	ExcluUseAr string `xml:"excluUseAr"`
	Floor      string `xml:"floor"`
	DealAmount string `xml:"dealAmount"`
	DealYear   string `xml:"dealYear"`
	DealMonth  string `xml:"dealMonth"`
	DealDay    string `xml:"dealDay"`
	DealingGbn string `xml:"dealingGbn"`
	CdealType  string `xml:"cdealType"`
	CdealDay   string `xml:"cdealDay"`
	RgstDate   string `xml:"rgstDate"`
	BuildYear  string `xml:"buildYear"`
}

type parser struct {
	err error
}

func (i item) trade() (Trade, error) {
	var p parser

	t := Trade{
		AptSeq:       clean(i.AptSeq),
		AptNm:        clean(i.AptNm),
		SggCd:        clean(i.SggCd),
		UmdNm:        clean(i.UmdNm),
		Jibun:        clean(i.Jibun),
		Area:         p.float("excluUseAr", i.ExcluUseAr),
		Floor:        p.int("floor", i.Floor),
		Price:        p.int("dealAmount", strings.ReplaceAll(i.DealAmount, ",", "")),
		Date:         p.date(i.DealYear, i.DealMonth, i.DealDay),
		Direct:       clean(i.DealingGbn) == "직거래",
		Canceled:     clean(i.CdealType) == "O",
		CanceledOn:   p.shortDate("cdealDay", i.CdealDay),
		RegisteredOn: p.shortDate("rgstDate", i.RgstDate),
		BuildYear:    p.int("buildYear", i.BuildYear),
	}

	if t.AptSeq == "" {
		p.fail("aptSeq", i.AptSeq)
	}
	if p.err != nil {
		return Trade{}, p.err
	}
	return t, nil
}

func (p *parser) fail(field, raw string) {
	if p.err == nil {
		p.err = fmt.Errorf("bad %s %q", field, raw)
	}
}

func (p *parser) int(field, raw string) int {
	n, err := strconv.Atoi(clean(raw))
	if err != nil {
		p.fail(field, raw)
	}
	return n
}

func (p *parser) float(field, raw string) float64 {
	n, err := strconv.ParseFloat(clean(raw), 64)
	if err != nil {
		p.fail(field, raw)
	}
	return n
}

func (p *parser) date(y, m, d string) string {
	s := fmt.Sprintf("%04d-%02d-%02d", p.int("dealYear", y), p.int("dealMonth", m), p.int("dealDay", d))
	if _, err := time.Parse(time.DateOnly, s); err != nil {
		p.fail("deal date", s)
	}
	return s
}

func (p *parser) shortDate(field, raw string) string {
	s := clean(raw)
	if s == "" {
		return ""
	}
	t, err := time.Parse("06.01.02", s)
	if err != nil {
		p.fail(field, raw)
		return ""
	}
	return t.Format(time.DateOnly)
}

func clean(s string) string {
	return strings.TrimSpace(s)
}
