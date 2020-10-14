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
		}
	}(&totalsUnit)
	go unitCounter(rnCnd.DiffUnitChan, rnCnd.TotalUnitChan)

	lastFlatTgs := flattenTargets(last)
	for _, ct := range current.Targets {
		if targetIsAllowed(rnCnd, &ct) {
			if lt, ok := lastFlatTgs[ct.Name]; ok {
				diffTargets(rnCnd, &lt, &ct)
			} else {
				// inspectTargetFiles(rnCnd, &ct)
				diffTargets(rnCnd, &Target{}, &ct)
			}
		}
	}

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
		covMsg = fmt.Sprintf(covMsg, "lowered")
	} else if lastPct < currentPct {
		covMsg = fmt.Sprintf(covMsg, "heightened")
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
	if lastPct > currentPct+cfg.Tolerance || currentPct == 0 {
		fmt.Printf("::error::%s\n", covMsg)
		os.Exit(1)
	}
	fmt.Println(covMsg)
	os.Exit(0)
}

func diffTargets(rnCnd *RunConditions, lastTgt, currentTgt *Target) {
	ltgtFlatFiles := flattenFiles(lastTgt)
	for _, curFile := range currentTgt.Files {
		if pathIsAllowed(rnCnd, &curFile) {
			if lstFile, ok := ltgtFlatFiles[curFile.Name]; ok {
				rnCnd.DiffUnitChan <- DiffUnit{
					LastExecutableLines:    lstFile.ExecutableLines,
					LastCoveredLines:       lstFile.CoveredLines,
					CurrentExecutableLines: curFile.ExecutableLines,
					CurrentCoveredLines:    curFile.CoveredLines,
				}

				lfsFlatFuncs := flattenFuncs(&lstFile)
				for _, curFunc := range curFile.Functions {
					if lstFunc, ok := lfsFlatFuncs[curFunc.Name]; ok {
						warnFuncCov(curFile.Path, &lstFunc, &curFunc)
					} else {
						warnZeroFuncCov(curFile.Path, &curFunc)
					}
				}
			} else {
				rnCnd.DiffUnitChan <- DiffUnit{
					LastExecutableLines:    0,
					LastCoveredLines:       0,
					CurrentExecutableLines: curFile.ExecutableLines,
					CurrentCoveredLines:    curFile.CoveredLines,
				}

				inspectFileFuncs(&curFile)
			}
		}
	}
}

func inspectFileFuncs(file *File) {
	for _, ff := range file.Functions {
		warnZeroFuncCov(file.Path, &ff)
	}
}

func warnFuncCov(filePth string, lstFunc, curFunc *Function) {
	if lstFunc.LineCoverage > curFunc.LineCoverage {
		fmt.Printf(
			"::warning file=%s,line=%d::%s coverage is lowered to %d%% (was %d%%).\n",
			strings.TrimPrefix(filePth, getWorkdir()),
			curFunc.LineNumber,
			curFunc.Name,
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
			"::warning file=%s,line=%d::%s coverage is 0%%.\n",
			strings.TrimPrefix(filePth, getWorkdir()),
			curFunc.LineNumber,
			curFunc.Name,
		)
	}
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
	return elInList(rnCnd.Config.FilterTargets, tgt.Name) == rnCnd.Config.InvertFilter
}

func pathIsAllowed(rnCnd *RunConditions, fl *File) bool {
	if rnCnd.Regexp != nil {
		return elemAllowedWithRegex(rnCnd, fl.Path)
	}
	return elSubInList(rnCnd.Config.FilterPaths, fl.Path) == rnCnd.Config.InvertFilter
}

func elInList(elList []string, elName string) bool {
	for _, el := range elList {
		if strings.Contains(elName, el) {
			return true
		}
	}
	return false
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
	return (rnCnd.Regexp.Find([]byte(el)) != nil) == rnCnd.Config.InvertFilter
}

func unitCounter(duChan, totalChan chan DiffUnit) {
	totalDu := DiffUnit{}
	for {
		select {
		case u := <-duChan:
			totalDu.CurrentCoveredLines += u.CurrentCoveredLines
			totalDu.CurrentExecutableLines += u.CurrentExecutableLines
			totalDu.LastCoveredLines += u.LastCoveredLines
			totalDu.LastExecutableLines += u.LastExecutableLines
		case totalChan <- totalDu:
		}
	}
}
