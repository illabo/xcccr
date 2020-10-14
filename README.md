## Xcode Coverage for Commit Reporter (xcccr)
This dead simple tool only works with Apple's `xccov` generated code coverage reports in json format. It takes two reports, diffs and spits out warnings and errors in GitHub Actions compatible format.

### CLI usage
 Flag   | Description
--------|------------
 `-cur` | For path of "current report" file. A report to be checked.
 `-lst` | For path of "last report" file. A report to be base for determining whether the coverage metrics is lowered or became higher in "current" report.
 `-tol` | Percent to be tolerated before error. E.g. last is 70%, current is 65% and tol is 10 means no error.
 `-proj` | Project root to get relative paths. If empty and missing from config working dir would be used. Usually in GitHub Actions you just want the working dir.
 `-rg` | Regex filter, pass it in quotes e.g. `-rg="[\s\S]\.swift"`. Matching paths would be ignored. When used with `-i` the result is opposite: only matching paths would be visible to reporter. Please note that if regex is used targets and paths filtering set in config has no effect.
 `-i` | Invert filter. Applied to -rg and to targets/paths filter lists in config. If no filters supplied prevents reporting allowing nothing.
 `-m` | Include masked files coverage in total metrics. If true warnings would be produced only for unfiltered targets/paths, but total coverage would be  reported for all the files as listed in original xccov json.
 `-loc` | Count covered LOCs diff instead of percent coverage.
 `-cfg` | Path to config. Parameters passed with the flags have precedence and overrides the values stored in config. Config passed with this flag overrides default config at `.xcccr.toml`. If there aren't any config default values are used.


 If there aren't last report or it isn't needed it's ok to pass only the current one. Current report also may be piped in to xcccr like that `cat report.json | ./xcccr`  

 ### Config values

Please check `.xcccr.toml.example` for reference.
