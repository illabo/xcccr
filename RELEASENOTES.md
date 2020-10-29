
### Hotfix (0.3.2)
- Added missing warning message for file coverage % when reduced.
- Some warnings weren't visible. Seem GitHub Actions warnings should be oneliners. Removed newline symbols.
- Some code duplications cleaned.

### Hotfix (0.3.1)
- "Increased" and "Decreased" messages was swapped. 

### Fixed (0.3.0)
- File paths for warnings was always relative to working dir only. `ProjectPath` setting in config and `proj` flag was completely ignored.
- Removed some code duplication.
- Data race is cured by adding a wait group to wait for all file reports being processed. Sometimes looping through all the files was ended prior the last report is collected from channel leading to incomplete counts. No more.

### Changed (0.3.0)
- Now warnings for functions coverage aren't shown per line, but instead are included in file coverage warnings. It still includes xccov messages with the line numbers.
- New filter added to mute warnings for paths without excluding the path from coverage report.
- Filters may be inverted individually with new flags and config settings. Please check readme and config example for usage.
- No breaking changes were introduced.