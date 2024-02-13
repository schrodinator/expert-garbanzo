[expert-garbanzo.com](https://expert-garbanzo.com)

This started as a chat app using websockets, based on ProgrammingPercy's tutorial ([YouTube](https://www.youtube.com/watch?v=pKpKv9MKN-E) / [GitHub](https://github.com/percybolmer/websocketsgo)).

Now it's a game, a knockoff of [CodeNames](https://boardgamegeek.com/boardgame/178900/codenames), with AI players courtesy of ChatGPT 3.5.

### Running locally
#### Files You'll Need
* `server.crt`: server certificate (for https)
* `server.key`: server key (for https)
* `gpt-secretkey.txt`: OpenAI ChatGPT API key (to use [AI ChatBot players](#ai-players))
* `wordlist.txt`: _Optional_. Custom word list. One word per line. Must contain at least 25 words.

If you do not have a server certificate and key, generate a self-signed certificate and key by running `gencert.bash` in Linux shell. This creates the files `server.crt` and `server.key`.

#### Linux
Add the above [Files You'll Need](#files-youll-need) to the directory `external`.

Start the server: `go run !(*_test).go`

In a browser, navigate to: `https://localhost:8080`

#### Docker
Alternately, use the provided Dockerfile to create a container. Inside the `expert-garbanzo` top-level directory, run:

`docker build -t expert-garbanzo .`

Then, from any directory:
`docker run -v <path on host>:/usr/src/app/external -p <host port>:8080 expert-garbanzo`
* `<path on host>` is the full path to the directory on the host machine containing the [Files You'll Need](#files-youll-need)
* `<host port>` is the port you want to expose on the host machine. Use 8080 to access the game at `https://localhost:8080`

### Starting a Game

To play a game, go to any chat room except for the lobby. The lobby is intended to meet people who want to play. Make up a room name for your group (or just yourself).

A game must have (at least) one guesser and (at least) one cluegiver per team. These roles may be played by ChatBots. You may elect to play a cooperative game by filling both roles on only one team.

Multiple players with the same role are permitted. You may elect to give yourself a (possibly unhelpful) helper bot by selecting a bot to fill the same role as you.

### AI Players

AI players are powered by OpenAI ChatGPT 3.5. Using AI players requires an API key, which can be obtained at https://platform.openai.com/api-keys. Save the secret key in a file called `gpt-secretkey.txt`.

### Tests
#### Backend
`go test -short` runs all tests except those with real calls to OpenAI ChatGPT. There are tests with mocks that cover the same functionality as the skipped tests.

#### Frontend
Start the server locally (see above) and run Playwright **without** parallelism: `npx playwright test --workers=1`

Or execute the tests for each browser type individually in `npx playwright test --ui`
