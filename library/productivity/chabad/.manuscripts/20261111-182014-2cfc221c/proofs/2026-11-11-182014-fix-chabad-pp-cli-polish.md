# Polish pass — chabad-pp-cli
Scorecard 92 -> 92; Verify 100% -> 100%; tools-audit 0 -> 0 pending; gosec (hand-authored) 5 -> 0.
Fixes: ICS UID hash sha1 -> sha256; narrow #nosec on user-chosen --output open and fixed-name config read; Erev Chanukah label now says light before Shabbat candles on Friday / after Shabbat on Saturday; gofmt.
Skipped (generator-owned or deliberate): 48 gosec findings in generated files; dead endpointPositionalArgs; which.go gofmt drift from dogfood sync; no Chanukah times (feed has none); Hayom Yom placeholder lesson text; no top-level parsha field in today.
ship_recommendation: ship. Live dogfood re-minted after polish edits: 160/160 pass.
