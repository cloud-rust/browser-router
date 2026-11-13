# Plan: Browser-Router in devenv integrieren

Stand 2026-10-24. Grundlage ist der Prototyp in diesem Repo (Branch `poc/browser-router`), getestet auf
macOS, Linux und Windows 11 als Standardbrowser.

## Ziel

Jeder Link, der von außen kommt (Mail, Teams, Terminal, `az login`), landet im richtigen Browser:

- Mandanten-URLs im Vivaldi-Profil des Mandanten, damit dort nur dessen Logins liegen
- Entwicklung (localhost, `127.0.0.1`, `[::1]`, `*.localhost`) in Firefox
- Apps über ihr Schema (`spotify:` …), soweit gewünscht
- alles andere im persönlichen Browser, frei wählbar (Safari, Firefox, …)

Die Regeln für die Mandanten entstehen aus den Mandantendateien (`clients/<key>.yml`, Feld `browser`). Die
festen Regeln (Entwicklung, Apps, Rückfall) liegen im Layer Persönlich.

## Entscheidung: eigenes Repo

Der Router wird ein **eigenes Repo** (Vorschlag: `cloud-rust/browser-router`) und nicht Teil der Engine.

- Er ist eine eigenständige Anwendung mit eigener Toolchain (Go, Swift für die Mac-Hülle), eigenen Tests
  und eigenen Releases je Betriebssystem. Die Engine ist Ansible und Python.
- Er läuft auch ohne devenv, z. B. auf Windows. Native Windows liegt laut ARCHITECTURE §11 außerhalb von devenv.
- devenv behandelt ihn wie jedes andere Werkzeug: installieren, Konfiguration schreiben, aufrufen.
  Installiert wird über mise aus den GitHub-Releases (`github:`-Backend), also ohne eigenen Downloader
  und ohne Build auf dem Rechner (ARCHITECTURE §8).

Arbeitsteilung:

| Teil | Wohin |
|---|---|
| Alles Browser-Spezifische: Regeln auswerten, Vivaldi-Profile auflösen und anlegen, Standardbrowser registrieren, Mac-App | Repo `browser-router` |
| Installation, Erzeugen der Router-Konfiguration aus den Mandanten, Aufruf von `router profiles ensure` und `router register` | devenv, zunächst Layer Persönlich (Entscheidung „Browser-Teil vorerst persönlich“) |
| Feld `browser` in der Mandantendatei | wie bisher frei für die Layer; Beschreibung in der Vorlage angleichen |

## Erkenntnisse aus den Tests, die der Router einhalten muss

1. Vivaldi-Profile nur über `--profile-directory`. Auf macOS immer `open -n -a Vivaldi --args …`, weil ein
   einfaches `open -a` bei laufendem Vivaldi den Link verliert.
2. Ein noch nie geöffnetes Profil verschluckt die erste URL (Begrüßung). Neue Profile also einmal öffnen.
3. Ohne Profilangabe landet ein Link im zuletzt benutzten Profil. Unbekanntes Profil oder kaputte
   Konfiguration → Rückfallbrowser, nie raten, nie einen Link verlieren.
4. Windows: Standardbrowser setzt nur der Nutzer per Klick. Tests brauchen eine angemeldete Desktop-Sitzung.
5. Konfigurationen können ein UTF-8-BOM haben.
6. `xdg-settings` lehnt einen Programmpfad in Anführungszeichen ab.

## Schritte

**0. Warten** bis die laufende Arbeit an den Mandantendateien (Engine-Branch `feat/client-files`) zusammengeführt
ist. Darauf aufbauen.

**1. Repo `browser-router` (aus dem Prototyp)**
- Prototyp mit Historie übernehmen; GitHub-Repo `cloud-rust/browser-router` **öffentlich** anlegen (vom Nutzer freigegeben).
- Muster wie bei Finicky als Glob (`*acme*`) zusätzlich zu Regex, damit `browser.match` aus den Mandanten
  unverändert passt.
- `router profiles ensure <Name>…`: fehlende Vivaldi-Profile anlegen (nur bei geschlossenem Vivaldi),
  bestehende über den Namen übernehmen, neue einmal öffnen.
- `router register` / `router status`: auf macOS die App nach `~/Applications` legen, bei LaunchServices
  anmelden und den Dialog nur auslösen, wenn der Router noch nicht Standard ist.
- `router doctor`: Konfiguration, Browser, Profile, Standardbrowser prüfen.
- CI: Tests, Builds für darwin-arm64 (App als Zip, ad-hoc signiert), linux-amd64/arm64, windows-amd64;
  Release per Tag. Die Linux-Ende-zu-Ende-Prüfung aus `scripts/e2e-linux.sh` in die CI.

**2. devenv (Layer Persönlich, Engine nur falls nötig)**
- Paket `browser-router` über mise (`github:`-Backend) in `devenv-personal/packages.yml` und `select.yml`.
  Prüfen, ob die Engine das `github:`-Backend schon unterstützt; sonst kleine Engine-Erweiterung.
- Task erzeugt `~/.config/browser-router/config.json` (Pfad je OS) aus den zusammengeführten Mandanten
  (`browser`) und einem festen Teil im Layer Persönlich (Entwicklung → Firefox, Apps, Rückfall Safari).
- Task ruft `router profiles ensure` für die Profile der Mandanten auf dem Rechner auf. Fehlt ein Layer
  (z. B. Globex auf dem privaten Mac), fehlen auch dessen Profile und Regeln.
- Finicky wird abgelöst, sobald der Router auf dem Mac läuft: Task für `finicky.js` und Paket entfernen, Router
  als Standardbrowser eintragen. Bis dahin bleibt `finicky.js` unverändert. Der tägliche Lauf löst nie einen Dialog aus.
- Kommentar zum Feld `browser` in `roles/clients/files/client-template.yml` an das Router-Format angleichen.

**3. Tests**
- Router: Unit-Tests, Linux-Ende-zu-Ende in CI.
- devenv: `task test:unit`, `task test:container`; Bootstrap-Tests zuletzt.
- Mac: einmal mit dem Nutzer (Dialog), danach Testlinks wie im Prototyp.

**4. Doku:** README im Router-Repo; ARCHITECTURE §9 (Browser-Teil), `devenv-personal/NEUER-MAC.md`.

## Entscheidungen des Nutzers (2026-10-24)

- Router-Repo ist **öffentlich**.
- **Finicky wird abgelöst**, sobald der eigene Router läuft.
- Persönlicher Browser (Rückfall) ist **Safari**.
- **Firefox ohne Profile**: Entwicklung geht an Firefox, wie er ist; keine Firefox-Profilunterstützung nötig.
- Offen bleibt nur WSL: Links aus WSL gehen an den Windows-Standardbrowser; ein Router auf Windows liegt
  außerhalb von devenv.
