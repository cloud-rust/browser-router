# router — das Go-Programm

Repo: `../CLAUDE.md` (browser-router).

Der Router selbst: nimmt jeden Link entgegen und gibt ihn nach Regeln weiter; ein Modul (`go.mod`), gebaut
mit `go build -C router` oder im Container über `scripts/build.sh`.

## Karte

| Datei | Inhalt |
|---|---|
| `main.go` | Befehle (`<url>`, `test`, `register`, `doctor`, `profiles`, `config`) |
| `router.go`, `urls.go`, `redirect.go` | Regeln und Weiterleitung |
| `profiles.go`, `vivaldi.go` | Vivaldi-Profile |
| `register.go`, `macos.go`, `launch.go`, `proc_*.go` | Registrierung und Start je Plattform |
| `config.go`, `doctor.go` | Konfiguration, Prüfung |
| `*_test.go` | Tests (`go -C router test ./...`) |
