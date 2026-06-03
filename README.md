# TDC Game

Flere spill som kan spilles ved å trykke en enkelt knapp.

# Rammen

## Spillvelger (under utvikling)

Startskjerm der man velger spill

## Spillrammeverk (under utvikling)

Et rammeverk som inneholder byggeklosser for å lage et spill.

Setter en maks varighet på spillene (forslag: 2 minutter)

### Byggeklossene

- En draw funksjon og en update funksjon
- Player som man kan bruke (med ferdig sprite, og animasjoner basert på velocity)
- Camera som følger spilleren (hvis relevant)
- Platformer
- Flag
- Coins/poenggjenstander
- Powerups
- Fiender

Hjelpefunksjoner:

- Detect collisions
- Award score (eks man traff en mynt, eller man kom forbi x=420)
- EndGame

Variabler:

- Walkspeed
- Jumpspeed
- AirControl
- JumpForce
- Gravity

Hva kan vi lage med disse:

- Mario Run (platformer, trykk for å hoppe)
- Flappybird (autorun, gravity er lav, hvis kollisjon = du dør, distance gir poeng)
- Frogger (gravity negativ, platformer "kommer rekendes", kollisjon = død? coins gir poeng)
- Ett setera

Kunstbibliotek

- Player
- Platforms
- Bakken
- Powerups
- Fiender
- Bakgrunn

# Koden

Bruker [ebiten](https://ebitengine.org/) som spillmotor. Det er en veldig enkel spillmotor for go.

Koden er delt inn i noen deler:

- `main.go` - setter opp spillet, størrelse på vindu osv
- `player.go` - logikk for hvordan spilleren beveger seg, collision detection og hvilke animasjoner som spilles
- `game.go` - selve spillet. Her styres camera, og vi tegner inn bakken, fiender og andre assets
- `sprites.go` - laste inn assets, dele inn i spritesheets og logikk for animasjoner ligger her

## Grid system

0,0 i grid-en er der spilleren starter. Positiv x er mot høyre, positiv y er oppåver.

## Kaemra

# Assets/pixelart

I mappen assets ligger pixelart.

For å editere pixelart er det enklest å bruke et program som er laget for pixelart.

- [Aseprite](https://www.aseprite.org/) - Koster 150 kroner og er veldig bra, alle youtube-tutorials bruker dette (Jeg (Anders) bruker dette). Det er open source så man kan få det gratis hvis man bygger fra source :D
- https://libresprite.github.io/#!/ Open source fork av aseprite

Best å åpne aseprite filen for å editere noe som eksiterer. Anders kan assistere med eksport og oppsett av noe nytt hvis nødvendig.

# Ide til art/konsept

- Konferansedeltager
- Løper gjennom konferansen og samler sokker
- Fiender TODO: Trenger konsept her
- Powerups er kaffe, redbull, mat, godteri, etc
