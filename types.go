package main

import "regexp"

// Report
type XCCoverageReport struct {
	CoveredLines    int      `json:"coveredLines"`
	LineCoverage    float64  `json:"lineCoverage"`
	Targets         []Target `json:"targets"`
	ExecutableLines int      `json:"executableLines"`
}
type Function struct {
	CoveredLines    int     `json:"coveredLines"`
	LineCoverage    float64 `json:"lineCoverage"`
	LineNumber      int     `json:"lineNumber"`
	ExecutionCount  int     `json:"executionCount"`
	Name            string  `json:"name"`
	ExecutableLines int     `json:"executableLines"`
}
type File struct {
	CoveredLines    int        `json:"coveredLines"`
	LineCoverage    float64    `json:"lineCoverage"`
	Path            string     `json:"path"`
	Functions       []Function `json:"functions"`
	Name            string     `json:"name"`
	ExecutableLines int        `json:"executableLines"`
}
type Target struct {
	CoveredLines     int     `json:"coveredLines"`
	LineCoverage     float64 `json:"lineCoverage"`
	Files            []File  `json:"files"`
	Name             string  `json:"name"`
	ExecutableLines  int     `json:"executableLines"`
	BuildProductPath string  `json:"buildProductPath"`
}

// App config
type Config struct {
	ProjectPath   string
	FilterTargets []string
	FilterPaths   []string
	FilterPattern string
	InvertFilter  bool
	IncludeMasked bool
	ZeroWarnOnly  bool
	MeterLOC      bool
	Tolerance     int
}
type RunConditions struct {
	LastReport    *XCCoverageReport
	CurrentReport *XCCoverageReport
	Config        *Config
	Regexp        *regexp.Regexp
	DiffUnitChan  chan DiffUnit
	TotalUnitChan chan DiffUnit
}

// Coverage diff unit
type DiffUnit struct {
	LastExecutableLines    int
	LastCoveredLines       int
	CurrentExecutableLines int
	CurrentCoveredLines    int
}
