package molit

import (
	"errors"
	"os"
	"testing"
)

func page(total string, rows ...string) []byte {
	b := `<?xml version="1.0" encoding="utf-8" standalone="yes"?><response><script/>` +
		`<header><resultCode>000</resultCode><resultMsg>OK</resultMsg></header><body><items>`
	for _, r := range rows {
		b += r
	}
	return []byte(b + `</items><numOfRows>1000</numOfRows><pageNo>1</pageNo><totalCount>` + total + `</totalCount></body></response>`)
}

const row = `<item><aptDong>101</aptDong><aptNm>테스트아파트</aptNm><aptSeq>11110-0001</aptSeq>` +
	`<buildYear>2011</buildYear><cdealDay> </cdealDay><cdealType> </cdealType><dealAmount>110,000</dealAmount>` +
	`<dealDay>5</dealDay><dealMonth>4</dealMonth><dealYear>2023</dealYear><dealingGbn>중개거래</dealingGbn>` +
	`<excluUseAr>84.95</excluUseAr><floor>8</floor><jibun>1</jibun><rgstDate>23.08.17</rgstDate>` +
	`<sggCd>11110</sggCd><umdCd>10100</umdCd><umdNm>테스트동</umdNm></item>`

func TestParse(t *testing.T) {
	trades, total, err := Parse(page("1500", row, row))
	if err != nil {
		t.Fatal(err)
	}
	if len(trades) != 2 || total != 1500 {
		t.Fatalf("got %d trades, total %d", len(trades), total)
	}
	if tr := trades[0]; tr.AptSeq != "11110-0001" || tr.Price != 110000 || tr.Date != "2023-04-05" {
		t.Fatalf("trade: %+v", tr)
	}
}

const emptyMonth = `<response>
<script/>
<header>
<resultCode>000</resultCode>
<resultMsg>OK</resultMsg>
</header>
<body>
<items/>
<numOfRows>10</numOfRows>
<pageNo>1</pageNo>
<totalCount>0</totalCount>
</body>
</response>`

func TestParseEmptyMonth(t *testing.T) {
	trades, total, err := Parse([]byte(emptyMonth))
	if err != nil || len(trades) != 0 || total != 0 {
		t.Fatalf("got %v, %d, %v", trades, total, err)
	}
}

// The gateway's own format for a key the API does not know.
const keyError = `<?xml version="1.0" encoding="UTF-8"?>
<OpenAPI_ServiceResponse>
<cmmMsgHeader>
  <errMsg>SERVICE_KEY_IS_NOT_REGISTERED_ERROR</errMsg>
  <returnAuthMsg>등록되지 않은 서비스키</returnAuthMsg>
  <returnReasonCode>30</returnReasonCode>
</cmmMsgHeader>
</OpenAPI_ServiceResponse>`

func TestParseKeyError(t *testing.T) {
	_, _, err := Parse([]byte(keyError))
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "30" {
		t.Fatalf("want APIError 30, got %v", err)
	}
}

func TestParseResultCodeError(t *testing.T) {
	body := `<response><header><resultCode>03</resultCode><resultMsg>NO_DATA</resultMsg></header></response>`
	_, _, err := Parse([]byte(body))
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "03" {
		t.Fatalf("want APIError 03, got %v", err)
	}
}

// Nothing that is not a success response may read as zero trades.
func TestParseRejectsUnknownBodies(t *testing.T) {
	bodies := map[string]string{
		"empty":      "",
		"html":       "<html><body>502 Bad Gateway</body></html>",
		"plain text": "Unexpected errors",
		"truncated":  string(page("1", row))[:200],
		"no total":   `<response><header><resultCode>000</resultCode></header><body><items></items></body></response>`,
		"bad row":    string(page("1", `<item><aptSeq>11110-0001</aptSeq><dealAmount>abc</dealAmount></item>`)),
	}
	for name, b := range bodies {
		if _, _, err := Parse([]byte(b)); err == nil {
			t.Errorf("%s: want error", name)
		}
	}
}

// The full response saved from the real API. Kept out of git; skipped when absent.
func TestParseSavedResponse(t *testing.T) {
	body, err := os.ReadFile("testdata/sample.xml")
	if errors.Is(err, os.ErrNotExist) {
		t.Skip("testdata/sample.xml not present")
	}
	if err != nil {
		t.Fatal(err)
	}
	trades, total, err := Parse(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(trades) != total {
		t.Fatalf("%d trades, total %d", len(trades), total)
	}
}
