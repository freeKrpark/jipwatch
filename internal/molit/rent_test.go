package molit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Rows shaped like the API's: a new 전세, a renewal with the old terms, and
// a 월세 reported without 신규/갱신, which happens in recent months too.
const (
	newJeonse = `<item><aptNm>테스트아파트</aptNm><aptSeq>11110-0001</aptSeq><buildYear>2011</buildYear>` +
		`<contractTerm>25.11~27.11</contractTerm><contractType>신규</contractType><dealDay>8</dealDay>` +
		`<dealMonth>9</dealMonth><dealYear>2025</dealYear><deposit>57,750</deposit><excluUseAr>84.9</excluUseAr>` +
		`<floor>6</floor><jibun>1</jibun><monthlyRent>0</monthlyRent><preDeposit> </preDeposit>` +
		`<preMonthlyRent> </preMonthlyRent><roadnm>테스트로 1</roadnm><sggCd>11110</sggCd>` +
		`<umdNm>테스트동</umdNm><useRRRight> </useRRRight></item>`
	renewal = `<item><aptNm>테스트아파트</aptNm><aptSeq>11110-0001</aptSeq><buildYear>2011</buildYear>` +
		`<contractTerm>25.09~27.09</contractTerm><contractType>갱신</contractType><dealDay>2</dealDay>` +
		`<dealMonth>9</dealMonth><dealYear>2025</dealYear><deposit>126,000</deposit><excluUseAr>149</excluUseAr>` +
		`<floor>13</floor><jibun>1</jibun><monthlyRent>0</monthlyRent><preDeposit>120,000</preDeposit>` +
		`<preMonthlyRent> </preMonthlyRent><sggCd>11110</sggCd><umdNm>테스트동</umdNm>` +
		`<useRRRight>사용</useRRRight></item>`
	unknownMonthly = `<item><aptNm>둘째아파트</aptNm><aptSeq>11110-0002</aptSeq><buildYear>2000</buildYear>` +
		`<contractTerm> </contractTerm><contractType> </contractType><dealDay>26</dealDay>` +
		`<dealMonth>9</dealMonth><dealYear>2025</dealYear><deposit>20,000</deposit><excluUseAr>60</excluUseAr>` +
		`<floor>7</floor><jibun>2</jibun><monthlyRent>150</monthlyRent><preDeposit> </preDeposit>` +
		`<preMonthlyRent> </preMonthlyRent><sggCd>11110</sggCd><umdNm>테스트동</umdNm>` +
		`<useRRRight> </useRRRight></item>`
)

func TestParseRents(t *testing.T) {
	rents, total, err := ParseRents(page("217", newJeonse, renewal, unknownMonthly))
	if err != nil {
		t.Fatal(err)
	}
	if len(rents) != 3 || total != 217 {
		t.Fatalf("got %d rents, total %d", len(rents), total)
	}
	n, r, u := rents[0], rents[1], rents[2]
	if n.Deposit != 57750 || n.Monthly != 0 || n.Kind != "신규" || n.Term != "25.11~27.11" ||
		n.Date != "2025-09-08" || n.PreDeposit != 0 || n.RenewalRight {
		t.Fatalf("new 전세: %+v", n)
	}
	if r.Kind != "갱신" || r.PreDeposit != 120000 || r.PreMonthly != 0 || !r.RenewalRight {
		t.Fatalf("renewal: %+v", r)
	}
	if u.Kind != "" || u.Term != "" || u.Deposit != 20000 || u.Monthly != 150 {
		t.Fatalf("blank kind: %+v", u)
	}
}

func TestParseRentsBadAmount(t *testing.T) {
	bad := `<item><aptSeq>11110-0001</aptSeq><deposit>abc</deposit></item>`
	if _, _, err := ParseRents(page("1", bad)); err == nil {
		t.Fatal("want error")
	}
}

// Rents go to the rent endpoint, and its gateway errors come through the
// same way as the sales API's.
func TestRentsUsesRentEndpoint(t *testing.T) {
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		if path != "/rent" {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(keyError))
			return
		}
		w.Write(page("1", newJeonse))
	}))
	defer srv.Close()
	c := testClient(srv.URL+"/trade", 10)
	c.RentEndpoint = srv.URL + "/rent"
	rents, err := c.Rents(context.Background(), "11110", "202509")
	if err != nil || len(rents) != 1 || path != "/rent" {
		t.Fatalf("got %d rents from %s: %v", len(rents), path, err)
	}
}
