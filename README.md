# playongo
Music Library Server

## Build
`go get`

`go build`

## Run
Scan library:

`./playongo -musicDir ~/Music -scan`

Start HTTP server:

`./playongo -musicDir ~/Music`

## Test
Get songs list:

`curl http://localhost:12345/songs`

Choose some and request it by ID:

`curl http://localhost:12345/songs/478e442ab8450fce9d5ad01b9535327d`

Download:

`curl -v http://localhost:12345/static/Wardruna/2018%20-%20Skald/07%20Ormagardskvedi.mp3 --output /tmp/Ormagardskvedi.mp3`
