package main

type Proxy struct {
	ID                    int64   `gorm:"index;primary_key" json:"id"`
	IP                    string  `json:"ip"`
	Port                  string  `json:"port"`
	Content               string  `gorm:"unique_index:unique_content;" json:"content"`
	AssessTimes           int64   `json:"assess_times"`
	SuccessTimes          int64   `json:"success_times"`
	AvgResponseTime       float64 `json:"avg_response_time"`
	ContinuousFailedTimes int64   `json:"continuous_failed_times"`
	Score                 float64 `json:"score"`
	UpdateTime            int64   `json:"update_time"`
}
