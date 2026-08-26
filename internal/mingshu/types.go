package mingshu

type BirthInfo struct {
	Name   string `json:"name"`
	Year   int    `json:"year"`
	Month  int    `json:"month"`
	Day    int    `json:"day"`
	Hour   int    `json:"hour"`
	Minute int    `json:"minute"`
	City   string `json:"city"`
	Gender string `json:"gender"`
}

type OrderRequest struct {
	Birth    BirthInfo `json:"birth"`
	Email    string    `json:"email"`
	Currency string    `json:"currency"`
	Lang     string    `json:"lang"`
}

type SectionField struct {
	Title    string `json:"title"`
	Data     string `json:"data"`
	Analysis string `json:"analysis"`
	Advice   string `json:"advice"`
}

type FortuneSection struct {
	SectionField
	Phases []FortunePhase `json:"phases"`
}

type FortunePhase struct {
	Age      string `json:"age"`
	Pillar   string `json:"pillar"`
	Analysis string `json:"analysis"`
	Advice   string `json:"advice"`
}

type ReportSections struct {
	Personality SectionField   `json:"personality"`
	Career      SectionField   `json:"career"`
	Love        SectionField   `json:"love"`
	Health      SectionField   `json:"health"`
	Fortune     FortuneSection `json:"fortune"`
	Milestones  SectionField   `json:"milestones"`
}

type ReportData struct {
	Birth    string         `json:"birth"`
	Engine   map[string]any `json:"engine"`
	Sections ReportSections `json:"sections"`
}

type ReviewSection struct {
	Status string `json:"status"` // "pass" or "fail"
	Reason string `json:"reason,omitempty"`
}

type ReviewResult struct {
	Sections map[string]ReviewSection `json:"sections"` // key = section name
	AllPass  bool                     `json:"all_pass"`
}
