package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
)

func readConfig(path string) (*Config, error) {
	var cfg Config
	_, err := toml.DecodeFile(path, &cfg)
	return &cfg, err
}

func prepareRunConditions() (rnCnd *RunConditions, err error) {
	rnCnd = &RunConditions{}

	dumbValue := dumbValue()

	lstPth := flag.String(
		"lst",
		"",
		"Pass the path of last report to compare current report to.",
	)
	curPth := flag.String(
		"cur",
		"",
		"Pass the path of current report.",
	)
	tolerance := flag.String("tol",
		dumbValue,
		"Percent to be tolerated before error. E.g. last is 70%, current is 65% and tol is 10 means no error.",
	)
	projRoot := flag.String(
		"proj",
		getWorkdir(),
		"Project root to get relative paths. If empty and missing from config working dir would be used.",
	)
	mute := flag.String(
		"nw",
		"",
		"Comma separated strings of paths not to warn. When a path has one of the values of this list warnings for the files on path would be muted. Coverage would still be metered.",
	)
	filtr := flag.String("rg",
		dumbValue,
		"Regex filter. Applied to paths.",
	)
	fInvert := flag.Bool(
		"i",
		false,
		"Invert filter. Applied to -rg and to targets/paths filter lists in config. If no filters supplied prevents reporting allowing nothing.",
	)
	selInvert := flag.String(
		"si",
		"",
		"Selectively invert filters. When `-i` is set it is equivalent to `-si=prtw` and always have precedence over this flag. Values here are:\n`p` — inverts paths filter,\n`r` — inverts regex,\n`t` — inverts targets filter,\n`w` — inverts warnings filter.",
	)
	includeMasked := flag.Bool(
		"m",
		false,
		"Include masked files coverage in total metrics. If true warnings would be produced only for unfiltered targets/paths, but total coverage would be calculated for all the files.",
	)
	meterLOC := flag.Bool(
		"loc",
		false,
		"Count covered LOCs diff instead of percent coverage.",
	)
	zeroWarn := flag.Bool(
		"z",
		false,
		"Don't error on 0%% coverage, produce warning only.",
	)
	cfgPth := flag.String(
		"cfg",
		"",
		"Path to config.",
	)

	flag.Parse()

	if *cfgPth == "" {
		*cfgPth = ".xcccr.toml"
	}
	cfg, err := readConfig(*cfgPth)
	if err != nil && !os.IsNotExist(err) {
		return
	}
	if *tolerance != dumbValue {
		i, e := strconv.Atoi(*tolerance)
		if i < 0 || i > 100 {
			e = errors.New("out of allowed range")
		}
		if e != nil {
			err = fmt.Errorf("tolerance -tol must be an int in range 0 to 100. %w", err)
			return
		}
		cfg.Tolerance = i
	}
	if *mute != "" {
		cfg.FilterWarnPaths = strings.Split(*mute, ",")
	}
	if *filtr != dumbValue {
		cfg.FilterPattern = *filtr
	}
	if *fInvert {
		cfg.InvertFilter = true
	}
	if cfg.InvertFilter {
		cfg.InvertPathFilter = true
		cfg.InvertRegexp = true
		cfg.InvertTargetFilter = true
		cfg.InvertWarnpathFilter = true
	}
	for _, c := range *selInvert {
		if c == 'p' {
			cfg.InvertPathFilter = true
		}
		if c == 'r' {
			cfg.InvertRegexp = true
		}
		if c == 't' {
			cfg.InvertTargetFilter = true
		}
		if c == 'w' {
			cfg.InvertWarnpathFilter = true
		}
	}
	if *includeMasked {
		cfg.IncludeMasked = true
	}
	if *meterLOC {
		cfg.MeterLOC = true
	}
	if *zeroWarn {
		cfg.ZeroWarnOnly = true
	}
	cfg.ProjectPath = *projRoot
	if cfg.FilterPattern != "" {
		re, e := regexp.Compile(cfg.FilterPattern)
		if e != nil {
			err = fmt.Errorf("filter pattern (aka regexp) is invalid: %w", e)
			return
		}
		rnCnd.Regexp = re
	}

	rnCnd.Config = cfg
	rnCnd.DiffUnitChan = make(chan DiffUnit)
	rnCnd.TotalUnitChan = make(chan DiffUnit)

	curXccRep, err := getCurrentCoverage(curPth)
	if err != nil {
		return
	}
	rnCnd.CurrentReport = curXccRep

	lstXccRep, err := getLastCoverage(lstPth)
	if err != nil {
		return
	}
	rnCnd.LastReport = lstXccRep

	rnCnd.WaitGroup = &sync.WaitGroup{}

	return
}

func getCurrentCoverage(curPth *string) (curXccRep *XCCoverageReport, err error) {
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

	if *curPth != "" {
		curXccRep, err = readReportFile(*curPth)
		if err != nil {
			return
		}
	}
	// piped in file content always overrides passed with a 'cur' flag
	if len(pipeIn) > 0 {
		if err = json.Unmarshal(pipeIn, &curXccRep); err != nil {
			return
		}
	}
	if *curPth == "" && len(pipeIn) == 0 {
		log.Fatal("No data passed to programme to report coverage.")
	}
	return
}

func getLastCoverage(lstPth *string) (lstXccRep *XCCoverageReport, err error) {
	lstXccRep = &XCCoverageReport{}
	if *lstPth != "" {
		lstXccRep, err = readReportFile(*lstPth)
		notExist := os.IsNotExist(err)
		if err != nil && notExist == false {
			return
		}
		if notExist {
			err = nil
			fmt.Println("Previous report file not found. Continuing with current report only.")
		}
	}
	return
}

func readReportFile(filePath string) (report *XCCoverageReport, err error) {
	report = &XCCoverageReport{}
	filePath = fmt.Sprintf("%s%s", getWorkdir(), filePath)
	_, err = os.Stat(filePath)
	if err != nil {
		return
	}
	fByt, err := ioutil.ReadFile(filePath)
	if err != nil {
		return
	}
	err = json.Unmarshal(fByt, report)

	return
}

func dumbValue() string {
	return randomStr(12)
}
