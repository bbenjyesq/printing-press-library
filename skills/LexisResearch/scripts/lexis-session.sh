#!/bin/bash
# lexis-session.sh — check the Lexis+ session the Lexis skills use, and renew it automatically.
#
#   lexis-session.sh check    Is the saved session live? (one Lexis+ lookup of a known citation)
#   lexis-session.sh login    Open the sign-in window, wait for it to close by itself, save the session.
#   lexis-session.sh ensure   check; if the session has expired, login; then check again.
#
# Shared by /lexispull, /lexis-citecheck, and /LexisResearch (identical copy in each skill's scripts/).
#
# How sign-in works: press-auth opens its own Chrome window (a separate profile, so the user signs in
# there even when their everyday Chrome is signed in) at the Lexis+ sign-in page, set to return to the
# Lexis+ research home. As soon as the research search box appears, press-auth saves the cookies and
# closes the window. The CLI is then pointed at that fresh session. Nobody presses Ctrl-C, and nobody
# types a command after signing in.
#
# Output: one line starting LEXIS_SESSION_OK, LEXIS_SESSION_EXPIRED, LEXIS_SESSION_LOGIN, or
# LEXIS_SESSION_ERROR, then details. Exit 0 = live session; 4 = expired / sign-in not completed;
# 3 = tools missing; 1 = other failure (network, Lexis+ unavailable).

CLI="${LEXIS_PLUS_CLI:-$HOME/printing-press/library/lexis-plus/lexis-plus-pp-cli}"
PRESS_AUTH="$HOME/go/bin/press-auth"
export PATH="$HOME/go/bin:$PATH"
PROBE="27 Cal.3d 167"   # Tameny v. Atlantic Richfield Co.: stable, always in Lexis+
LOGIN_URL="https://signin.lexisnexis.com/lnaccess/app/signin?aci=lp&back=https%3A%2F%2Fplus.lexis.com%2Fzhome"
DONE_SELECTOR='textarea[placeholder*="citation"], input[placeholder*="citation"], [placeholder*="Shepardize"], #searchTerms'
LOGIN_WAIT="${LEXIS_LOGIN_WAIT:-540}"   # seconds to wait for the user to finish signing in

need() {
  if [ ! -x "$CLI" ]; then
    echo "LEXIS_SESSION_ERROR: the Lexis+ CLI was not found at $CLI"; exit 3
  fi
  if [ ! -x "$PRESS_AUTH" ]; then
    echo "LEXIS_SESSION_ERROR: press-auth was not found at $PRESS_AUTH"; exit 3
  fi
}

# check: 0 live, 4 expired, 1 anything else (with the CLI's message).
check() {
  local out code
  out="$("$CLI" cite check "$PROBE" --agent 2>&1)"
  code=$?
  if [ $code -eq 0 ] && printf '%s' "$out" | grep -q '"status": "recognized"'; then
    echo "LEXIS_SESSION_OK: Lexis+ session is live"
    return 0
  fi
  if [ $code -eq 4 ] || printf '%s' "$out" | grep -q "session missing or expired"; then
    echo "LEXIS_SESSION_EXPIRED: the Lexis+ session has expired"
    return 4
  fi
  echo "LEXIS_SESSION_ERROR: Lexis+ could not be checked (exit $code): $(printf '%s' "$out" | grep -m1 -i 'error' | cut -c1-300)"
  return 1
}

login() {
  echo "LEXIS_SESSION_LOGIN: opening the Lexis+ sign-in window. Sign in there; it closes by itself when the Lexis+ search page loads."
  "$PRESS_AUTH" login plus.lexis.com --force --login-url "$LOGIN_URL" --complete-selector "$DONE_SELECTOR" >/dev/null 2>&1 &
  local pid=$! waited=0
  while kill -0 "$pid" 2>/dev/null; do
    if [ "$waited" -ge "$LOGIN_WAIT" ]; then
      kill "$pid" 2>/dev/null; sleep 1; kill -9 "$pid" 2>/dev/null
      echo "LEXIS_SESSION_EXPIRED: sign-in was not completed within $((LOGIN_WAIT / 60)) minutes; the window was closed. Run the request again to retry."
      return 4
    fi
    sleep 2; waited=$((waited + 2))
  done
  wait "$pid"
  local code=$?
  if [ $code -ne 0 ]; then
    echo "LEXIS_SESSION_ERROR: the sign-in window closed without saving a session (press-auth exit $code)"
    return 4
  fi
  # Point the CLI at the fresh session (it keeps its own copy, which would otherwise stay stale).
  if ! "$CLI" auth login --chrome >/dev/null 2>&1; then
    echo "LEXIS_SESSION_ERROR: signed in, but the Lexis+ CLI could not load the new session"
    return 1
  fi
  check
}

need
case "${1:-ensure}" in
  check)  check; exit $? ;;
  login)  login; exit $? ;;
  ensure)
    check; code=$?
    [ $code -eq 4 ] || exit $code
    login; exit $? ;;
  *) sed -n '2,6p' "$0"; exit 2 ;;
esac
