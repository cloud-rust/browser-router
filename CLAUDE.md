# browser-router

Projekt: `../CLAUDE.md` (local-infra).

Router, der sich als Standardbrowser einträgt und jeden Link nach Regeln an einen Browser, ein
Vivaldi-Profil oder eine App weitergibt; macOS, Linux, Windows. Bedienung und Konfiguration: [README.md](README.md).

## Karte

| Ordner / Datei | Inhalt |
|---|---|
| `router/` | das Go-Programm: [`router/CLAUDE.md`](router/CLAUDE.md) |
| `mac/` | kleine macOS-App, weil macOS Links als Apple Events an eine App schickt: [`mac/CLAUDE.md`](mac/CLAUDE.md) |
| `scripts/` | Bauen, App packen, Ende-zu-Ende-Tests je Plattform |
| `contracts/` | Format der Konfiguration, die andere (devenv-personal) schreiben |
| `spec/`, `docs/` | was gilt; Doku |

## Regeln

- Tests: `go -C router test ./...` oder `scripts/build.sh` im Container; ein geänderter Befehl ändert seine Zeile in der README im selben Commit.
- Ändert sich das Format der Konfiguration, ändert sich der Vertrag unter `contracts/` mit.
