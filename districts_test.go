package main

import "testing"

func TestDistricts(t *testing.T) {
	if len(districts) != 25 {
		t.Fatalf("%d districts, want Seoul's 25", len(districts))
	}
	seen := map[string]bool{}
	for _, d := range districts {
		if len(d.Code) != 5 || d.Code[:2] != "11" || d.Name == "" || seen[d.Code] {
			t.Errorf("bad district %+v", d)
		}
		seen[d.Code] = true
	}
}

func TestDistrictName(t *testing.T) {
	if got := districtName("11380"); got != "은평구" {
		t.Fatalf("got %q", got)
	}
	if got := districtName("41135"); got != "41135" {
		t.Fatalf("unlisted code should come back as is, got %q", got)
	}
}
