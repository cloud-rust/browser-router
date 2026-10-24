#!/usr/bin/env bash
# End-to-end test inside a desktop-less Linux container (scripts/e2e.sh): create a Vivaldi profile
# with the router, register the router as default browser, open links via xdg-open, then check in
# which browser and profile each link landed. Exits non-zero on any mismatch.
set -u
if [ -z "${DISPLAY:-}" ]; then Xvfb :99 -screen 0 1280x1024x24 -nolisten tcp >/dev/null 2>&1 & export DISPLAY=:99; sleep 2; fi
fail() { echo "FAIL: $*"; failed=1; }
failed=0
arch=$(uname -m | sed -e s/aarch64/arm64/ -e s/x86_64/amd64/)
mkdir -p ~/bin ~/ffp && cp /src/build/linux-$arch/browser-router ~/bin/browser-router
R=~/bin/browser-router
UDD=~/vud
VARGS=(--no-sandbox --disable-gpu --no-first-run)
export BROWSER_ROUTER_CONFIG=~/.config/browser-router/config.json
stop_vivaldi() { pkill -TERM -f user-data-dir=; for _ in $(seq 20); do pgrep -f user-data-dir= >/dev/null || return 0; sleep 1; done; pkill -KILL -f user-data-dir=; sleep 1; }

echo "--- profiles"
$R profiles ensure --user-data-dir $UDD Acme; rc=$?
[ $rc = 3 ] || fail "ensure without Local State: rc $rc, want 3"
# setsid: an exited Vivaldi must not linger as a zombie child of this shell, which looks alive
setsid vivaldi --user-data-dir=$UDD "${VARGS[@]}" about:blank >/dev/null 2>&1 & sleep 8; stop_vivaldi
$R profiles ensure --user-data-dir $UDD Acme -- "${VARGS[@]}" || fail "ensure Acme"
sleep 10
$R profiles ensure --user-data-dir $UDD Other; rc=$?
[ $rc = 3 ] || fail "ensure while Vivaldi runs: rc $rc, want 3"
stop_vivaldi
$R profiles list --user-data-dir $UDD
acme_dir=$($R profiles list --user-data-dir $UDD | awk -F'\t' '$1=="Acme"{print $2}')
[ -n "$acme_dir" ] || fail "Acme missing from Local State after Vivaldi ran"
[ -f "$UDD/$acme_dir/Preferences" ] || fail "Vivaldi did not open $acme_dir once"
$R profiles ensure --user-data-dir $UDD Acme | grep -q '^exists: Acme' || fail "second ensure did not find Acme"

mkdir -p "$(dirname "$BROWSER_ROUTER_CONFIG")"
cat > "$BROWSER_ROUTER_CONFIG" <<J
{
  "rules": [
    {"name": "development", "regex": ["^https?://(localhost|127\\\\.0\\\\.0\\\\.1|\\\\[::1\\\\]|[^/?#]+\\\\.localhost)(:\\\\d+)?([/?#]|\$)"],
     "target": {"browser": "firefox", "args": ["--profile", "$HOME/ffp"]}},
    {"name": "acme", "match": ["*acme*"],
     "target": {"browser": "vivaldi", "profile": "Acme", "user_data_dir": "$UDD", "args": ["--no-sandbox", "--disable-gpu", "--no-first-run"]}}
  ],
  "fallback": {"browser": "firefox", "args": ["--profile", "$HOME/ffp"]}
}
J
$R config check || fail "config check"

echo "--- register"
xdg-settings set default-web-browser firefox.desktop || fail "set firefox as default"
$R register --no-prompt || fail "register --no-prompt"
[ "$(xdg-settings get default-web-browser)" != browser-router.desktop ] || fail "register --no-prompt changed the default"
$R register || fail "register"
[ "$(xdg-settings get default-web-browser)" = browser-router.desktop ] || fail "router is not the default browser"
$R register | grep -q 'default browser: yes' || fail "second register did not see the router as default"

echo "--- links"
python3 -m http.server 3000 --bind 127.0.0.1 >/dev/null 2>&1 &
sleep 1; T=$(date +%s)
for u in "http://localhost:3000/?probe=$T-dev" "https://example.com/acme/?probe=$T-acme" "https://example.org/?probe=$T-other"; do
  $R test "$u" | head -3 | tr '\n' ' '; echo; xdg-open "$u"; sleep 8
done
sleep 10; pkill -TERM firefox; stop_vivaldi; sleep 4
echo "--- router.log"; cat ~/.config/browser-router/router.log
grep -q probe ~/.config/browser-router/router.log && fail "router.log contains a full URL"

echo "--- landed"
landed=$(
  cp ~/ffp/places.sqlite /tmp/p.sqlite; cp ~/ffp/places.sqlite-wal /tmp/p.sqlite-wal 2>/dev/null
  sqlite3 /tmp/p.sqlite "select url from moz_places where url like '%probe=$T-%'" | sed "s|.*probe=$T-||; s|^|firefox:|"
  for h in $UDD/*/History; do
    cp "$h" /tmp/h; rm -f /tmp/h-wal; cp "$h-wal" /tmp/h-wal 2>/dev/null
    sqlite3 /tmp/h "select url from urls where url like '%probe=$T-%'" | sed "s|.*probe=$T-||; s|^|vivaldi $(basename "$(dirname "$h")"):|"
  done | sort -u)
echo "$landed"
want=$(printf '%s\n' "firefox:dev" "firefox:other" "vivaldi $acme_dir:acme" | sort)
[ "$(echo "$landed" | sort)" = "$want" ] || fail "links landed wrong; want: $(echo $want)"

[ $failed = 0 ] && echo "PASS" || exit 1
