# browser-router

Ein kleiner Router, der sich als Standardbrowser einträgt und jeden Link nach Regeln weitergibt: an einen
Browser, an ein Vivaldi-Profil oder an eine App. So landet ein Link aus Mail, Teams oder dem Terminal
(`az login`) im richtigen Profil, und jedes Vivaldi-Profil hält nur die Logins eines Mandanten.

Läuft auf macOS (arm64), Linux (amd64, arm64) und Windows (amd64). Ein Go-Programm erledigt die Arbeit; auf
macOS kommt eine kleine App dazu, weil macOS Links als Apple Events an eine App schickt.

## Installation

Mit [mise](https://mise.jdx.dev) aus den GitHub-Releases:

```bash
mise use -g github:cloud-rust/browser-router
```

Oder ein Archiv aus den [Releases](https://github.com/cloud-rust/browser-router/releases) entpacken und
`browser-router` in den `PATH` legen. Das macOS-Archiv enthält neben dem Programm `Browser Router.app`
(ad hoc signiert); `browser-router register` kopiert die App nach `~/Applications`.

Danach einmal:

```bash
browser-router config init     # Beispielkonfiguration schreiben, dann anpassen
browser-router register        # als Standardbrowser anbieten
browser-router doctor          # alles prüfen
```

`register` fragt nur, wenn der Router noch nicht Standard ist: macOS zeigt seinen Bestätigungsdialog, Windows
öffnet die Einstellungen (dort wählt man „Browser Router“), Linux stellt direkt um. `register --no-prompt`
installiert und aktualisiert nur (App, Desktop-Datei, Registry) und fragt nie; das ist für automatische Läufe.

## Konfiguration

`browser-router config path` zeigt die Datei: `~/Library/Application Support/browser-router/config.json`
(macOS), `~/.config/browser-router/config.json` (Linux), `%APPDATA%\browser-router\config.json` (Windows).
Die Umgebungsvariable `BROWSER_ROUTER_CONFIG` setzt einen anderen Pfad.

```json
{
  "rules": [
    {"name": "development",
     "regex": ["^https?://(localhost|127\\.0\\.0\\.1|\\[::1\\]|[^/?#]+\\.localhost)(:\\d+)?([/?#]|$)"],
     "target": {"browser": "firefox"}},
    {"name": "acme", "urls": ["acme.atlassian.net", "*.acme.com", "github.com/acme"],
     "match": ["*login.microsoftonline.com/<tenant-id>/*"],
     "target": {"browser": "vivaldi", "profile": "Acme"}},
    {"name": "spotify", "match": ["*spotify.com*"], "target": {"browser": "app", "app": "Spotify"}}
  ],
  "fallback": {"browser": "safari"}
}
```

- Zuerst zählen die `urls` aller Regeln: der **genaueste Eintrag** gewinnt, egal in welcher Regel er steht. Passt
  keiner, gewinnt die **erste Regel**, deren `match` oder `regex` passt; passt auch das nicht, geht der Link an
  `fallback`.
- `urls`: Seiten einer Regel als Host, wahlweise mit Pfadanfang: `acme.atlassian.net`, `github.com/acme`, und mit
  `*.` davor der Host samt allen Subdomains (`*.acme.com`). Geprüft werden nur Host und Pfad, nie Query oder Titel
  einer Seite: Eine Wiki-Seite „Globex-Workshop“ auf `acme.atlassian.net` bleibt bei Acme. Ein Pfadanfang endet an
  einem `/` (`github.com/acme` deckt nicht `github.com/acme-corp`). Genauer heißt: ein Host ohne `*.` vor einem mit,
  ein längerer Host vor einem kürzeren, dann der längere Pfad. Derselbe Eintrag in zwei Regeln ist ein Fehler.
- `match`: Muster wie bei Finicky. `*` steht für beliebig viele Zeichen, sonst gilt jedes Zeichen wörtlich, Groß-
  und Kleinschreibung zählen nicht. Das Muster deckt die ganze URL ab. Ohne `://` im Muster darf ein Schema davor
  stehen: `localhost*` passt auf `http://localhost:3000/`.
- `regex`: reguläre Ausdrücke (Go-Syntax) gegen die ganze URL, ohne Groß/klein.
- `target.browser`: `vivaldi` (mit `profile`), `firefox`, `safari`, `chrome` oder `app` (mit `app`: App-Name auf
  macOS, Programmpfad sonst). `args` hängt Argumente vor die URL.
- `profile`: der angezeigte Name des Vivaldi-Profils („Acme“) oder sein Ordner („Profile 1“).

Regeln, die nie raten:

- Kein Profil ohne Angabe: Vivaldi würde sonst das zuletzt benutzte Profil nehmen. Ein unbekanntes Profil schickt
  den Link an `fallback`, eine kaputte Konfiguration an den eingebauten Rückfall (Safari auf macOS, sonst Firefox).
- Fehlt der Browser einer Regel, geht der Link an `fallback`. Ein Link geht nie verloren.
- `router.log` neben der Konfiguration hält je Link Regel, Ziel und nur den Host, nie die ganze URL (Links können
  Tokens enthalten).

## Anmeldungen von Kommandozeilen-Werkzeugen

Die Anmeldeseite von `az login` oder `gh auth login --web` verrät nicht, für welchen Mandanten man sich anmeldet. Wer
es weiß (z. B. weil der Aufruf im Ordner des Mandanten passiert), gibt den Mandanten mit:

```json
"clients": {
  "acme": {"browser": "vivaldi", "profile": "Acme"}
}
```

```bash
browser-router --client acme https://login.microsoftonline.com/…
BROWSER_ROUTER_CLIENT=acme browser-router https://…   # dasselbe als Umgebungsvariable
```

Ein Link mit bekanntem Mandanten geht an dessen Ziel, vor allen Regeln. Ein unbekannter Mandant oder ein fehlendes
Profil lässt den Link wie gewohnt durch die Regeln laufen. Werkzeuge wie `az` und `gh` öffnen Links über `BROWSER`,
das nur einen Programmpfad ohne Argumente nimmt; daher die Umgebungsvariable. Ein kleines Skript als `BROWSER`, das
`BROWSER_ROUTER_CLIENT` setzt und den Router aufruft, reicht.

## Vivaldi-Profile

```bash
browser-router profiles list
browser-router profiles ensure Acme Globex      # fehlende anlegen
browser-router profiles ensure --dry-run Acme    # nur melden
```

`ensure` übernimmt vorhandene Profile über ihren Namen. Fehlende legt es in Vivaldis `Local State` an, nur bei
geschlossenem Vivaldi (Vivaldi schreibt die Datei beim Beenden zurück), und öffnet jedes neue Profil einmal: Der
allererste Start eines Profils zeigt eine Begrüßung, die sonst den ersten Link verschlucken würde.

Exit-Codes aller Befehle: 0 in Ordnung, 1 Fehler, 2 falscher Aufruf, 3 jetzt nicht möglich (Vivaldi läuft oder
wurde nie gestartet).

## Befehle

| Befehl | Zweck |
|---|---|
| `browser-router <url>` | Link weiterleiten (so ruft ihn das Betriebssystem auf) |
| `browser-router test <url>` | zeigen, wohin ein Link ginge, ohne ihn zu öffnen |
| `browser-router register [--no-prompt]` | installieren, aktuell halten, als Standard anbieten |
| `browser-router unregister [<browser>]` | Registrierung entfernen; macOS: `<bundle id>` wieder als Standard (Vorgabe Safari) |
| `browser-router status` | Version, Konfiguration, Standardbrowser |
| `browser-router doctor` | Konfiguration, Browser, Profile und Registrierung prüfen; Exit-Code 1 bei Problemen |
| `browser-router profiles list\|ensure` | Vivaldi-Profile |
| `browser-router config path\|init\|check` | Konfiguration |

## Hinweise je Plattform

- **macOS:** Vivaldi wird mit `open -n -a Vivaldi --args --profile-directory=…` gestartet; ohne `-n` verliert ein
  laufendes Vivaldi den Link.
- **Linux:** `register` schreibt `~/.local/share/applications/browser-router.desktop` und setzt den Standard über
  `xdg-settings`. Nach einem Update mit neuem Programmpfad hält `register --no-prompt` die Datei aktuell.
- **Windows:** Den Standardbrowser setzt nur der Nutzer per Klick in den Einstellungen. Konfigurationen dürfen ein
  UTF-8-BOM haben (PowerShell 5 schreibt eines).

## Entwicklung

Ohne lokales Go, im Container (Docker, z. B. über Colima mit `DOCKER_CONTEXT=colima`):

```bash
scripts/build.sh    # gofmt, vet, Tests, Builds für alle Ziele; auf macOS zusätzlich die App
scripts/e2e.sh      # Ende-zu-Ende unter Linux: Profil anlegen, registrieren, Links öffnen, Ziel prüfen
```

Die Windows-Skripte `scripts/win-*.ps1` prüfen dasselbe in einer angemeldeten Desktop-Sitzung.

Die CI (`.github/workflows/ci.yml`) testet jeden Push und Pull Request, führt den Linux-Ende-zu-Ende-Test aus und
baut die Mac-App. Ein Tag `vX.Y.Z` erzeugt das Release mit `browser-router_X.Y.Z_<os>_<arch>.tar.gz` (Windows:
`.zip`) und `checksums.txt`.
