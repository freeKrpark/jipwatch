package main

import (
	"testing"

	"github.com/freekrpark/jipwatch/internal/molit"
)

var sample = []molit.Trade{
	{AptSeq: "11110-0002", AptNm: "둘째아파트", Date: "2023-04-03", BuildYear: 2015},
	{AptSeq: "11110-0001", AptNm: "첫째아파트", Date: "2023-04-09", BuildYear: 2011},
	{AptSeq: "11110-0001", AptNm: "첫째아파트", Date: "2023-04-01", Direct: true},
	{AptSeq: "11110-0001", AptNm: "첫째아파트", Date: "2023-04-09", Canceled: true},
}

func TestSummarize(t *testing.T) {
	s := summarize(sample)
	if s.Total != 4 || s.Canceled != 1 || s.Direct != 1 {
		t.Fatalf("counts: %+v", s)
	}
	if len(s.Complexes) != 2 || s.Complexes[0].AptSeq != "11110-0001" || s.Complexes[0].Sales != 3 {
		t.Fatalf("busiest first: %+v", s.Complexes)
	}
}

func TestSummarizeEmptyIsNotNil(t *testing.T) {
	if s := summarize(nil); s.Complexes == nil {
		t.Fatal("complexes must be an empty slice so the frontend gets []")
	}
	if c := complexTrades(nil, "x"); c == nil {
		t.Fatal("trades must be an empty slice so the frontend gets []")
	}
}

func TestComplexTradesByDate(t *testing.T) {
	got := complexTrades(sample, "11110-0001")
	if len(got) != 3 || got[0].Date != "2023-04-01" || got[2].Date != "2023-04-09" {
		t.Fatalf("%+v", got)
	}
}

func TestCheckQuery(t *testing.T) {
	if err := checkQuery("11110", "202304"); err != nil {
		t.Fatal(err)
	}
	for _, q := range [][2]string{{"1111", "202304"}, {"11110", "2023-04"}, {"11110", "202313"}} {
		if checkQuery(q[0], q[1]) == nil {
			t.Errorf("%v: want error", q)
		}
	}
}
