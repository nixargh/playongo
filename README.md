# playongo
Music Library Server

## Build
`go get -d`

`go build`

## Run
Scan library:

`./playongo -musicDir ~/Music -scan`

Scan reads files metadata using goroutines (multiple threads), so read threads number and speed of reading depends on number of cores. By default Go limits the number of OS thread using **GOMAXPROCS** variable, which by default is equals to the number of cores.

Start HTTP server:

`./playongo -musicDir ~/Music`

## Test
Get songs list:

`curl -s http://localhost:12345/songs | jq .`

Choose some and request it by ID:

`curl -s http://localhost:12345/songs/478e442ab8450fce9d5ad01b9535327d |jq .`

Look for all songs of album (or any other attribute):

`curl -s http://localhost:12345/songs/album/Skald | jq .`

Download:

`curl -v http://localhost:12345/static/Wardruna/2018%20-%20Skald/07%20Ormagardskvedi.mp3 --output /tmp/Ormagardskvedi.mp3`
