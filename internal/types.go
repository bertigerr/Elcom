package internal

type ItemSource string

const (
	SourceEmailText      ItemSource = "email_text"
	SourceEmailHTMLTable ItemSource = "email_html_table"
	SourceXLSX           ItemSource = "xlsx"
	SourcePDF            ItemSource = "pdf"
)

type ExtractionItem struct {
	LineNo     int
	Source     ItemSource
	RawLine    string
	NameOrCode *string
	Qty        *float64
	Unit       *string
	Meta       map[string]any
}

type MatchStatus string

type MatchReason string

const (
	MatchOK       MatchStatus = "OK"
	MatchReview   MatchStatus = "REVIEW"
	MatchNotFound MatchStatus = "NOT_FOUND"

	ReasonCode   MatchReason = "CODE"
	ReasonHeader MatchReason = "HEADER"
	ReasonFuzzy  MatchReason = "FUZZY"
	ReasonNone   MatchReason = "NONE"
)

type ProductFlatCodes struct {
	Elcom        *string `json:"elcom,omitempty"`
	Manufacturer *string `json:"manufacturer,omitempty"`
	Raec         *string `json:"raec,omitempty"`
	PC           *string `json:"pc,omitempty"`
	Etm          *string `json:"etm,omitempty"`
}

type ProductRecord struct {
	ID                 int
	SyncUID            *string
	Header             string
	Articul            *string
	UnitHeader         *string
	ManufacturerHeader *string
	MultiplicityOrder  *float64
	AnalogCodes        []string
	FlatCodes          ProductFlatCodes
	UpdatedAt          *string
	RawJSON            string
}

type MatchCandidate struct {
	ID      int     `json:"id"`
	SyncUID *string `json:"syncUid"`
	Header  string  `json:"header"`
	Score   float64 `json:"score"`
}

type MatchProduct struct {
	ID         *int             `json:"id"`
	SyncUID    *string          `json:"syncUid"`
	Header     *string          `json:"header"`
	Articul    *string          `json:"articul"`
	UnitHeader *string          `json:"unitHeader"`
	FlatCodes  ProductFlatCodes `json:"flatCodes"`
}

type MatchResult struct {
	Status     MatchStatus      `json:"status"`
	Confidence float64          `json:"confidence"`
	Reason     MatchReason      `json:"reason"`
	Product    *MatchProduct    `json:"product"`
	Candidates []MatchCandidate `json:"candidates"`
}

type EmailRow struct {
	ID         int
	Provider   string
	MessageID  string
	Subject    string
	Sender     string
	ReceivedAt string
	Hash       string
	Status     string
	RawRef     string
}

type FetchedMailMessage struct {
	Provider   string
	MessageID  string
	Subject    string
	From       string
	ReceivedAt string
	Raw        []byte
}

type MatchExportRow struct {
	InputLineNo      int
	Source           string
	RawLine          string
	ParsedNameOrCode *string
	ParsedQty        *float64
	ParsedUnit       *string
	MatchStatus      string
	Confidence       float64
	MatchReason      string
	ProductID        *int
	ProductSyncUID   *string
	ProductHeader    *string
	ProductArticul   *string
	UnitHeader       *string
	FlatElcom        *string
	FlatManufacturer *string
	FlatRaec         *string
	FlatPC           *string
	FlatEtm          *string
	Candidate2Header *string
	Candidate2Score  *float64
}
