A chat app using websockets. 

Based on ProgrammingPercy's tutorial: [YouTube](https://www.youtube.com/watch?v=pKpKv9MKN-E) / [GitHub](https://github.com/percybolmer/websocketsgo)

### Running locally

Install third-party Go libraries:
- `go get github.com/gorilla/websocket`
- `go get github.com/google/uuid`

Generate a self-signed certificate by running `gencert.bash`

Start the server: `go run !(*_test).go`

In a browser, navigate to: `https://localhost:8080`

### Tests

Start the server locally (see above) and run Playwright **without** parallelism: `npx playwright test --workers=1`

Or execute the tests for each browser type individually in `npx playwright test –ui`

When running tests in parallel, one will usually randomly fail to log in; rerunning the test on its own is successful.