package core

type StorageCardData struct {
	Value       string `ch:"Value" json:"value"`
	Unit        string `ch:"Unit" json:"unit"`
	Delta       string `ch:"Delta" json:"delta"`
	DeltaColour string `ch:"DeltaColour" json:"deltaColour"`
}

type StorageStats struct {
	DiskName    string  `ch:"DiskName"`
	FreeBytes   uint64  `ch:"FreeBytes"`
	TotalBytes  uint64  `ch:"TotalBytes"`
	UsedPercent float64 `ch:"UsedPercent"`
}

type StorageOutlook struct {
	CurrentTotalBytes         uint64
	ProjectedSteadyStateBytes uint64
	IsSteadyState             bool
	DaysOfHistory             int
}
