package main

import (
	"fmt"
	"os"
	"strings"
)

func performReport(rnCnd *RunConditions) {
	last := rnCnd.LastReport
	current := rnCnd.CurrentReport
	var totalsUnit DiffUnit

	go func(tu *DiffUnit) {
		for {
			totalsUnit = <-rnCnd.TotalUnitChan
			rnCnd.WaitGroup.Done()
		}
	}(&totalsUnit)
	go unitCounter(rnCnd)

	lastFlatTgs := flattenTargets(last)
	for _, ct := range current.Targets {
		if targetIsAllowed(rnCnd, &ct) {
			if lt, ok := lastFlatTgs[ct.Name]; ok {
				diffTargets(rnCnd, &lt, &ct)
			} else {
				diffTargets(rnCnd, &Target{}, &ct)
			}
		}
	}

	rnCnd.WaitGroup.Wait()

	produceVerdict(rnCnd.Config, &totalsUnit, last, current)
}

func produceVerdict(cfg *Config, tu *DiffUnit, last, current *XCCoverageReport) {
	if cfg.IncludeMasked {
		tu.LastExecutableLines = last.ExecutableLines
		tu.LastCoveredLines = last.CoveredLines
		tu.CurrentExecutableLines = current.ExecutableLines
		tu.CurrentCoveredLines = current.CoveredLines
	}
	lastPct := percentify(linesRatio(tu.LastCoveredLines, tu.LastExecutableLines))
	currentPct := percentify(linesRatio(tu.CurrentCoveredLines, tu.CurrentExecutableLines))
	covMsg := "Code coverage is %s"

	if lastPct > currentPct {
		covMsg = fmt.Sprintf(covMsg, "decreased")
	} else if lastPct < currentPct {
		covMsg = fmt.Sprintf(covMsg, "increased")
	} else {
		covMsg = fmt.Sprintf(covMsg, "not changed")
	}
	var curCovMsg, lstCovMsg string
	if cfg.MeterLOC {
		curCovMsg = fmt.Sprintf("currently covered %dLOC", tu.CurrentCoveredLines)
		lstCovMsg = fmt.Sprintf("last coverage was %dLOC", tu.LastCoveredLines)
	} else {
		curCovMsg = fmt.Sprintf("currently covered %d%%", currentPct)
		lstCovMsg = fmt.Sprintf("last coverage was %d%%", lastPct)
	}
	if len(last.Targets) == 0 {
		lstCovMsg = "last coverage is unavailable"
	}
	covMsg = fmt.Sprintf("%s: %s, %s.", covMsg, curCovMsg, lstCovMsg)
	if lastPct > currentPct+cfg.Tolerance {
		emmitError(covMsg)
	}
	currentPctZero := currentPct == 0
	if currentPctZero && cfg.ZeroWarnOnly == false {
		emmitError(covMsg)
	}
	if currentPctZero {
		covMsg = fmt.Sprintf("::warning::%s", covMsg)
	}
	exitMessage(covMsg)
}

func emmitError(message string) {
	fmt.Printf("::error::%s\n", message)
	os.Exit(1)
}

func exitMessage(message string) {
	fmt.Println(message)
	os.Exit(0)
}

func diffTargets(rnCnd *RunConditions, lastTgt, currentTgt *Target) {
	ltgtFlatFiles := flattenFiles(lastTgt)
	// Wait for every file coverage to be reported.
	rnCnd.WaitGroup.Add(len(currentTgt.Files))
	for _, curFile := range currentTgt.Files {
		if pathIsAllowed(rnCnd, &curFile) == false {
			// Path is excluded from coverage report.
			// Don't expect coverage unit. Try next.
			rnCnd.WaitGroup.Done()
			continue
		}

		var lfsFlatFuncs map[string]Function
		lstFile := ltgtFlatFiles[curFile.Name]
		curFilePth := strings.TrimPrefix(curFile.Path, rnCnd.Config.ProjectPath)

		lfsFlatFuncs = flattenFuncs(&lstFile)
		rnCnd.DiffUnitChan <- DiffUnit{
			LastExecutableLines:    lstFile.ExecutableLines,
			LastCoveredLines:       lstFile.CoveredLines,
			CurrentExecutableLines: curFile.ExecutableLines,
			CurrentCoveredLines:    curFile.CoveredLines,
		}

		if warnIsAllowed(rnCnd, &curFile) == false {
			// Don't warn, count coverage only.
			continue
		}

		fileReport := "\n"
		if curFile.CoveredLines < lstFile.CoveredLines {
			var warnMsg string
			if rnCnd.Config.MeterLOC {
				warnMsg = fmt.Sprintf("covered %dLOC, was %dLOC.", curFile.CoveredLines, lstFile.CoveredLines)
			}
			fileReport = fmt.Sprintf("File coverage is reduced: %s\n", warnMsg)
		}
		if curFile.CoveredLines == 0 {
			fileReport = "File is not covered.\n"
		}

		for _, curFunc := range curFile.Functions {
			if lstFunc, ok := lfsFlatFuncs[curFunc.Name]; ok {
				fileReport = fmt.Sprintf("%s%s", fileReport, warnFuncCov(&lstFunc, &curFunc))
			} else {
				fileReport = fmt.Sprintf("%s%s", fileReport, warnZeroFuncCov(&curFunc))
			}
		}
		if fileReport != "\n" {
			fmt.Printf("::warning file=%s::%s", curFilePth, fileReport)
		}

	}
}

func warnFuncCov(lstFunc, curFunc *Function) string {
	if lstFunc.LineCoverage > curFunc.LineCoverage {
		return fmt.Sprintf(
			"%d | %s coverage is lowered to %d%% (was %d%%).\n",
			curFunc.LineNumber,
			curFunc.Name,
			percentify(curFunc.LineCoverage),
			percentify(lstFunc.LineCoverage),
		)

	}
	return warnZeroFuncCov(curFunc)
}

func warnZeroFuncCov(curFunc *Function) string {
	if percentify(curFunc.LineCoverage) == 0 {
		return fmt.Sprintf(
			"%d | %s is not covered.\n",
			curFunc.LineNumber,
			curFunc.Name,
		)
	}
	return ""
}

func linesRatio(cov, exec int) float64 {
	if exec == 0 {
		return 0
	}
	return float64(cov) / float64(exec)
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

func targetIsAllowed(rnCnd *RunConditions, tgt *Target) bool {
	if rnCnd.Regexp != nil {
		return true // Allow any target and let regex sortout filtering on paths.
	}
	return elSubInList(rnCnd.Config.FilterTargets, tgt.Name) == rnCnd.Config.InvertTargetFilter
}

func pathIsAllowed(rnCnd *RunConditions, fl *File) bool {
	if rnCnd.Regexp != nil {
		return elemAllowedWithRegex(rnCnd, fl.Path)
	}
	return elSubInList(rnCnd.Config.FilterPaths, fl.Path) == rnCnd.Config.InvertPathFilter
}

func warnIsAllowed(rnCnd *RunConditions, fl *File) bool {
	return elSubInList(rnCnd.Config.FilterWarnPaths, fl.Path) == rnCnd.Config.InvertWarnpathFilter
}

func elSubInList(elList []string, elName string) bool {
	for _, el := range elList {
		if strings.Contains(elName, el) {
			return true
		}
	}
	return false
}

func elemAllowedWithRegex(rnCnd *RunConditions, el string) bool {
	if rnCnd.Regexp == nil {
		return true
	}
	return (rnCnd.Regexp.Find([]byte(el)) != nil) == rnCnd.Config.InvertRegexp
}

func unitCounter(rnCnd *RunConditions) {
	totalDu := DiffUnit{}
	for {
		u := <-rnCnd.DiffUnitChan
		totalDu.CurrentCoveredLines += u.CurrentCoveredLines
		totalDu.CurrentExecutableLines += u.CurrentExecutableLines
		totalDu.LastCoveredLines += u.LastCoveredLines
		totalDu.LastExecutableLines += u.LastExecutableLines

		rnCnd.TotalUnitChan <- totalDu
	}
}
