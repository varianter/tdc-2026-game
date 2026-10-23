# TDC Game

Flere spill som kan spilles ved å trykke en enkelt knapp.

## Get started

Du kan kjøre standspillet fra CLI med:

```bash
go run .
```

Om du ønsker å starte et spesifikt spill (for utvikling, bytt ut `tdcrunner` med ditt spill-navn):

```bash
go run . -game=tdcrunner
```

## Kjøre på stand-PC-en (ferdigbygget binary)

I stedet for å kjøre `go run .` fra Releases. 

Last ned riktig variant 😏:

- `tdcgame-macos-arm64` – Apple Silicon (M1/M2/M3/M4)
- `tdcgame-macos-amd64` – Intel-Mac

Binaryen er ikke signert av Apple, så Gatekeeper blokkerer den første gangen. Fjern
karantene-flagget, gjør den kjørbar og start:

```bash
xattr -dr com.apple.quarantine ./tdcgame-macos-arm64
chmod +x ./tdcgame-macos-arm64
./tdcgame-macos-arm64
```

(Alternativt: høyreklikk → Åpne første gang.)
