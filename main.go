package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

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

func main() {
	lstPth := flag.String("lst", "", "Pass the path of last report to compare current report to")
	curPth := flag.String("cur", "", "Pass the path of current report")

	flag.Parse()

	var pipeIn []byte
	info, _ := os.Stdin.Stat()
	if info.Mode()&os.ModeNamedPipe != 0 {
		reader := bufio.NewReader(os.Stdin)
		for {
			ln, _, err := reader.ReadLine()
			if err != nil && err == io.EOF {
				break
			}
			pipeIn = append(pipeIn, ln...)
		}
	}

	var curXccRep XCCoverageReport
	var err error
	if *curPth != "" {
		curXccRep, err = readReportFile(*curPth)
		if err != nil {
			log.Fatal(err)
		}
	}
	// piped in file content always overrides passed with a 'cur' flag
	if len(pipeIn) > 0 {
		if err = json.Unmarshal(pipeIn, &curXccRep); err != nil {
			log.Fatal(err)
		}
	}

	var lstXccRep XCCoverageReport
	if *lstPth != "" {
		lstXccRep, err = readReportFile(*lstPth)
		if err != nil {
			log.Fatal(err)
		}
	}

	performReport(&lstXccRep, &curXccRep)
}

func readReportFile(filePath string) (XCCoverageReport, error) {
	filePath = fmt.Sprintf("%s%s", getWorkdir(), filePath)
	var report XCCoverageReport
	_, err := os.Stat(filePath)
	if err != nil {
		return report, err
	}
	fByt, err := ioutil.ReadFile(filePath)
	if err != nil {
		return report, err
	}
	err = json.Unmarshal(fByt, &report)

	return report, err
}

func performReport(last, current *XCCoverageReport) {
	lastFlatTgs := flattenTargets(last)
	for _, ct := range current.Targets {
		if lt, ok := lastFlatTgs[ct.Name]; ok {
			diffTargets(&lt, &ct)
		} else {
			inspectTargetFiles(&ct)
		}
	}
	if last.LineCoverage > current.LineCoverage {
		fmt.Printf(
			"::error::Code coverage is lowered to %d%% (was %d%%).\n",
			percentify(current.LineCoverage),
			percentify(last.LineCoverage),
		)
		os.Exit(1)
	}
	fmt.Printf(
		"Coverage is %d%%. Tested %d lines of %d.\n",
		percentify(current.LineCoverage),
		current.CoveredLines,
		current.ExecutableLines,
	)
	os.Exit(0)
}

func diffTargets(lastTgt, currentTgt *Target) {
	ltgtFlatFiles := flattenFiles(lastTgt)
	for _, curFile := range currentTgt.Files {
		if lstFile, ok := ltgtFlatFiles[curFile.Name]; ok {
			lfsFlatFuncs := flattenFuncs(&lstFile)
			for _, curFunc := range curFile.Functions {
				if lstFunc, ok := lfsFlatFuncs[curFunc.Name]; ok {
					warnFuncCov(curFile.Path, &lstFunc, &curFunc)
				} else {
					warnZeroFuncCov(curFile.Path, &curFunc)
				}
			}
		} else {
			inspectFileFuncs(&curFile)
		}
	}
}

func inspectTargetFiles(target *Target) {
	for _, fl := range target.Files {
		inspectFileFuncs(&fl)
	}
}

func inspectFileFuncs(file *File) {
	for _, ff := range file.Functions {
		warnZeroFuncCov(file.Path, &ff)
	}
}

func warnZeroFileCov(fPath string) {
	fmt.Printf("::warning file=%s::File coverage is 0%%.\n", fPath)
}

func warnFuncCov(filePth string, lstFunc, curFunc *Function) {
	if lstFunc.LineCoverage > curFunc.LineCoverage {
		fmt.Printf(
			"::warning file=%s,line=%d::Func coverage is lowered to %d%% (was %d%%).\n",
			filePth,
			curFunc.LineNumber,
			percentify(curFunc.LineCoverage),
			percentify(lstFunc.LineCoverage),
		)
		return
	}
	warnZeroFuncCov(filePth, curFunc)
}

func warnZeroFuncCov(filePth string, curFunc *Function) {
	if percentify(curFunc.LineCoverage) == 0 {
		fmt.Printf(
			"::warning file=%s,line=%d::Func coverage is 0%%.\n",
			filePth,
			curFunc.LineNumber,
		)
	}
}
func percentify(linecov float64) int {
	return int(linecov * 100)
}

func flattenTargets(covR *XCCoverageReport) map[string]Target {
	flatTargets := map[string]Target{}
	for _, t := range covR.Targets {
		flatTargets[t.Name] = t
	}
	return flatTargets
}

func flattenFiles(tgtR *Target) map[string]File {
	flatFiles := map[string]File{}
	for _, f := range tgtR.Files {
		flatFiles[f.Name] = f
	}
	return flatFiles
}

func flattenFuncs(filR *File) map[string]Function {
	flatFuncs := map[string]Function{}
	for _, f := range filR.Functions {
		flatFuncs[f.Name] = f
	}
	return flatFuncs
}

func getWorkdir() (dir string) {
	dir, _ = os.Getwd()
	if dir == "" {
		dir, _ = os.UserHomeDir()
	}
	if dir == "" {
		dir = "~/"
	}
	if strings.HasSuffix(dir, "/") == false {
		dir = dir + "/"
	}
	return
}

func appendSlash(pathStr string) string {
	if strings.HasSuffix(pathStr, "/") {
		return pathStr
	}
	return pathStr + "/"
}
