This started as a chat app using websockets, based on ProgrammingPercy's tutorial ([YouTube](https://www.youtube.com/watch?v=pKpKv9MKN-E) / [GitHub](https://github.com/percybolmer/websocketsgo)).

Now it's a game, a knockoff of [CodeNames](https://boardgamegeek.com/boardgame/178900/codenames), with AI players courtesy of ChatGPT 3.5.

### Running locally
#### Linux
Install third-party Go libraries:
- `go get github.com/gorilla/websocket`
- `go get github.com/google/uuid`
- `go get github.com/sashabaranov/go-openai`

Generate a self-signed certificate by running `gencert.bash`

Start the server: `go run !(*_test).go`

In a browser, navigate to: `https://localhost:8080`

#### Docker
Alternately, use the provided Dockerfile to create a container. From the directory containing the Dockerfile and source code:

`docker build -t expert-garbanzo .`

`docker run -p 8080:8080 expert-garbanzo`

### Starting a Game

To play a game, go to any chat room except for the lobby. The lobby is intended to meet people who want to play. Make up a room name for your group (or just yourself).

A game must have (at least) one guesser and (at least) one cluegiver per team. These roles may be played by ChatBots. You may elect to play a cooperative game by filling both roles on only one team.

Multiple players with the same role are permitted. You may elect to give yourself a (possibly unhelpful) helper bot by selecting a bot to fill the same role as you.

### AI Players

AI players are powered by OpenAI ChatGPT 3.5. Using AI players requires an API key, which can be obtained at https://platform.openai.com/api-keys. Save the secret key in a file called `gpt-secretkey.txt` in the top-level directory.

### Tests
#### Backend
`go test -short` runs all tests except those with real calls to OpenAI ChatGPT. There are tests with mocks that cover the same functionality as the skipped tests.

#### Frontend
Start the server locally (see above) and run Playwright **without** parallelism: `npx playwright test --workers=1`

Or execute the tests for each browser type individually in `npx playwright test --ui`