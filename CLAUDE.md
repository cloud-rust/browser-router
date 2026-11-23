# browser-router

Projekt: `../CLAUDE.md` (local-infra).

Router, der sich als Standardbrowser einträgt und jeden Link nach Regeln an einen Browser, ein
Vivaldi-Profil oder eine App weitergibt; macOS, Linux, Windows. Bedienung und Konfiguration: [README.md](README.md).

## Karte

| Ordner / Datei | Inhalt |
|---|---|
| `*.go` im Root | das Go-Programm: Befehle (`main.go`), Regeln (`router.go`), Profile, Registrierung je Plattform |
| `mac/` | kleine macOS-App, weil macOS Links als Apple Events an eine App schickt |
| `scripts/` | Bauen, App packen, Ende-zu-Ende-Tests je Plattform |
| `contracts/` | Format der Konfiguration, die andere (devenv-personal) schreiben |
| `spec/`, `docs/` | was gilt; Doku |

## Regeln

- Tests: `go test ./...`; ein geänderter Befehl ändert seine Zeile in der README im selben Commit.
- Ändert sich das Format der Konfiguration, ändert sich der Vertrag unter `contracts/` mit.
