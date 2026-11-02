package molit

import "testing"

func sample() item {
	return item{
		AptSeq: "11110-0001", AptNm: "테스트아파트", SggCd: "11110",
		UmdNm: "테스트동", Jibun: "1", ExcluUseAr: "84.95", Floor: "8",
		DealAmount: "110,000", DealYear: "2023", DealMonth: "4", DealDay: "5",
		DealingGbn: "중개거래", CdealType: " ", CdealDay: " ", RgstDate: "23.08.17",
		BuildYear: "2011",
	}
}

func TestItemTrade(t *testing.T) {
	got, err := sample().trade()
	if err != nil {
		t.Fatal(err)
	}
	want := Trade{
		AptSeq: "11110-0001", AptNm: "테스트아파트", SggCd: "11110",
		UmdNm: "테스트동", Jibun: "1", Area: 84.95, Floor: 8, Price: 110000,
		Date: "2023-04-05", RegisteredOn: "2023-08-17", BuildYear: 2011,
	}
	if got != want {
		t.Fatalf("\n got %+v\nwant %+v", got, want)
	}
}

func TestItemTradeCanceled(t *testing.T) {
	it := sample()
	it.CdealType, it.CdealDay, it.RgstDate = "O", "23.04.27", " "
	got, err := it.trade()
	if err != nil {
		t.Fatal(err)
	}
	if !got.Canceled || got.CanceledOn != "2023-04-27" || got.RegisteredOn != "" {
		t.Fatalf("canceled: %+v", got)
	}
}

func TestItemTradeDirect(t *testing.T) {
	it := sample()
	it.DealingGbn = "직거래"
	got, err := it.trade()
	if err != nil {
		t.Fatal(err)
	}
	if !got.Direct {
		t.Fatalf("direct: %+v", got)
	}
}

func TestItemTradeRejectsBadFields(t *testing.T) {
	cases := map[string]func(*item){
		"amount in words": func(it *item) { it.DealAmount = "약 11억" },
		"month 13":        func(it *item) { it.DealMonth = "13" },
		"blank aptSeq":    func(it *item) { it.AptSeq = " " },
		"rgstDate format": func(it *item) { it.RgstDate = "2023-08-17" },
		"blank area":      func(it *item) { it.ExcluUseAr = " " },
	}
	for name, mod := range cases {
		it := sample()
		mod(&it)
		if _, err := it.trade(); err == nil {
			t.Errorf("%s: want error", name)
		}
	}
}
