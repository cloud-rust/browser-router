# mac — die macOS-App

Repo: `../CLAUDE.md` (browser-router).

macOS schickt Links als Apple Events an eine App, nicht an ein Programm; diese kleine App nimmt sie an und
ruft das Go-Programm. `scripts/mac-app.sh` baut daraus mit `swiftc` das ad hoc signierte Bundle.

| Datei | Inhalt |
|---|---|
| `main.swift` | nimmt den Link an und startet `browser-router` |
| `Info.plist` | Bundle-Angaben, Platzhalter `VERSION` |
