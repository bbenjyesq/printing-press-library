# Phase 4.95 local code review — chabad-pp-cli
Review path: direct subagent dispatch (correctness, security, maintainability reviewers in parallel for round 1; one combined three-lens reviewer for round 2). Working dir is not a git repo, so diff-shaped review skills did not fit.
Autofix summary: 22 findings autofixed in-place across 3 rounds (no commits; working tree is not under git). Regression tests added: TestReviewFixes, TestEventsOverlappingKeepsTrueEnds, TestRoshChodeshBetweenLongWindow.
Convergence: findings cleared at round 3 (round-2 reviewer confirmed all round-1 fixes; its 3 low findings fixed in round 3 and verified by build/vet/tests).
Not applied (style-only, noted): named constants for location type 1/2 and the "2006-01-02" layout (~50 uses); shared withOccasionsStore helper; single Rosh Chodesh merge helper for holidays/ahead.
Template-shape retro candidates (not patched): internal/cli/helpers.go dead helpers endpointPositionalArgs, retainCLIQueryParams (generator emission).
Out-of-scope retro candidates: none.
Surface-to-user findings: none.
/simplify: not run (scope kept to reviewer findings).
