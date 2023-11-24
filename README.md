A chat app using websockets. 

Based on ProgrammingPercy's tutorial: [YouTube](https://www.youtube.com/watch?v=pKpKv9MKN-E) / [GitHub](https://github.com/percybolmer/websocketsgo)

### Running locally

Install third-party Go libraries:
- `go get github.com/gorilla/websocket`
- `go get github.com/google/uuid`
- `go get github.com/sashabaranov/go-openai`
- `go get golang.org/x/exp/slices`

Generate a self-signed certificate by running `gencert.bash`

Save the master password in a file called `password.txt` in the top-level directory.

Start the server: `go run !(*_test).go`

In a browser, navigate to: `https://localhost:8080`

### AI Players

AI players are powered by OpenAI ChatGPT 3.5. Using AI players requires an API key, which can be obtained at https://platform.openai.com/api-keys. Save the secret key in a file called `gpt-secretkey.txt` in the top-level directory.

### Tests

Start the server locally (see above) and run Playwright **without** parallelism: `npx playwright test --workers=1`

Or execute the tests for each browser type individually in `npx playwright test –ui`

When running tests in parallel, one will usually randomly fail to log in; rerunning the test on its own is successful.