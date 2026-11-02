package molit

type Trade struct {
	AptSeq       string  `json:"aptSeq"` // complex id, e.g. "11380-8922"
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
