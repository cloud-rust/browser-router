#!/usr/bin/env bash
# End-to-end check inside a Linux desktop-less container: register the router as default
# browser, open links via xdg-open, then look up in which browser/profile each one landed.
set -u
R=/poc/build/linux-$(uname -m | sed -e s/aarch64/arm64/ -e s/x86_64/amd64/)/router
mkdir -p ~/bin ~/ffp && cp "$R" ~/bin/router
C=~/.config/browser-router-poc; mkdir -p $C
cat > $C/config.json <<J
{
  "rules": [
    {"name": "development", "match": ["^https?://(localhost|127\\\\.0\\\\.0\\\\.1|\\\\[::1\\\\]|[^/]+\\\\.localhost)(:\\\\d+)?([/?#]|\$)"],
     "target": {"browser": "firefox", "args": ["--profile", "$HOME/ffp"]}},
    {"name": "acme", "match": ["acme"],
     "target": {"browser": "vivaldi", "profile": "Acme", "user_data_dir": "$HOME/vud", "args": ["--no-sandbox", "--disable-gpu", "--no-first-run"]}}
  ],
  "fallback": {"browser": "firefox", "args": ["--profile", "$HOME/ffp"]}
}
J
# A real profile named "Acme": create it once so Local State maps the name to its directory.
vivaldi --user-data-dir=$HOME/vud --no-sandbox --disable-gpu --no-first-run --profile-directory="Profile 1" about:blank >/dev/null 2>&1 & sleep 8
pkill -TERM -f user-data-dir=; sleep 4
python3 - <<P
import json; p="$HOME/vud/Local State"; s=json.load(open(p))
s["profile"]["info_cache"]["Profile 1"]["name"]="Acme"; json.dump(s, open(p,"w"))
P
~/bin/router register || { echo "register failed"; exit 1; }
echo "default browser: $(xdg-settings get default-web-browser)"
python3 -m http.server 3000 --bind 127.0.0.1 >/dev/null 2>&1 &
sleep 1; T=$(date +%s)
for u in "http://localhost:3000/?probe=$T-dev" "https://example.com/acme/?probe=$T-acme" "https://example.org/?probe=$T-other"; do
  ~/bin/router test "$u" | head -3 | tr '\n' ' '; echo; xdg-open "$u"; sleep 8
done
sleep 15; pkill -TERM firefox; pkill -TERM -f user-data-dir=; sleep 6
echo "--- router.log"; cat $C/router.log
echo "--- landed"
cp ~/ffp/places.sqlite /tmp/p.sqlite; cp ~/ffp/places.sqlite-wal /tmp/p.sqlite-wal 2>/dev/null
sqlite3 /tmp/p.sqlite "select url from moz_places where url like '%probe=$T-%'" | sed "s|.*probe=$T-||; s|^|  [firefox]: |"
for h in ~/vud/*/History; do cp "$h" /tmp/h; sqlite3 /tmp/h "select url from urls where url like '%probe=$T-%'" | sed "s|.*probe=$T-||; s|^|  [vivaldi $(basename "$(dirname "$h")")]: |"; done
