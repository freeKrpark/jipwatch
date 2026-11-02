package main

// District is a 시군구 the API takes as LAWD_CD.
type District struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

var districts = []District{
	{"11110", "종로구"}, {"11140", "중구"}, {"11170", "용산구"}, {"11200", "성동구"},
	{"11215", "광진구"}, {"11230", "동대문구"}, {"11260", "중랑구"}, {"11290", "성북구"},
	{"11305", "강북구"}, {"11320", "도봉구"}, {"11350", "노원구"}, {"11380", "은평구"},
	{"11410", "서대문구"}, {"11440", "마포구"}, {"11470", "양천구"}, {"11500", "강서구"},
	{"11530", "구로구"}, {"11545", "금천구"}, {"11560", "영등포구"}, {"11590", "동작구"},
	{"11620", "관악구"}, {"11650", "서초구"}, {"11680", "강남구"}, {"11710", "송파구"},
	{"11740", "강동구"},
}

func (a *App) Districts() []District { return districts }

func districtName(code string) string {
	for _, d := range districts {
		if d.Code == code {
			return d.Name
		}
	}
	return code
}
