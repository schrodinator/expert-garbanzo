const { test, expect, chromium } = require('@playwright/test');
var fs = require('fs');

const chatroom = 'gameroom';
const password = fs.readFileSync('./password.txt', { encoding: 'utf8', flag: 'r' });

test('chat', async () => {
    // Cluegiver login
    const context_clue = await chromium.launch({ headless: false, slowMo: 100 });
    const page_clue = await context_clue.newPage();
    page_clue.on('websocket', ws => {
        console.log(`WebSocket opened: ${ws.url()}>`);
        ws.on('framesent', event => console.log(event.payload));
        ws.on('framereceived', event => console.log(event.payload));
        ws.on('close', () => console.log('WebSocket closed'));
    });
    await page_clue.goto('/');
    await page_clue.getByTestId('username').fill('cluegiver');
    await page_clue.getByTestId('password').fill(password);
    await page_clue.getByTestId('password').press('Enter');

    // Check that cluegiver login succeeded
    await expect(page_clue.getByTestId('chatlog')).toBeVisible();

    // Guesser login
    const context_guess = await chromium.launch({ headless: false, slowMo: 100 });
    const page_guess = await context_guess.newPage();
    page_guess.on('websocket', ws => {
        console.log(`WebSocket opened: ${ws.url()}>`);
        ws.on('framesent', event => console.log(event.payload));
        ws.on('framereceived', event => console.log(event.payload));
        ws.on('close', () => console.log('WebSocket closed'));
    });
    await page_guess.goto('/');
    await page_guess.getByTestId('username').fill('guesser');
    await page_guess.getByTestId('password').fill(password);
    await page_guess.getByTestId('password').press('Enter');

    // Check that guesser login succeeded
    await expect(page_guess.getByTestId('chatlog')).toBeVisible();

    // Cluegiver sends a chat message
    const msg = 'Meet me in ' + chatroom;
    await page_clue.getByTestId('message').fill(msg);
    await page_clue.getByTestId('message').press('Enter');

    // Assert that the message is received in the guesser's chat view
    await expect(page_guess.getByTestId('chatlog')).toContainText(msg);

    // Close all pages and contexts
    await page_clue.close();
    await page_guess.close();
    await context_clue.close();
    await context_guess.close();
});