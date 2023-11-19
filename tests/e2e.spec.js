const { test, expect, chromium } = require('@playwright/test');
const exp = require('constants');
var fs = require('fs');

const chatroom = 'gameroom';
const password = fs.readFileSync('./password.txt', { encoding: 'utf8', flag: 'r' });

test('play two-player game', async () => {
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

    // Assert that the message is received by the guesser
    await expect(page_guess.getByTestId('chatlog')).toContainText(msg);

    // Assert the new game button is hidden in the lobby
    await expect(page_guess.getByTestId('newgame')).toBeHidden();

    // Move into the new chat room
    await page_clue.getByTestId('chatroom').fill(chatroom);
    await page_clue.getByTestId('chatroom').press('Enter');
    await page_guess.getByTestId('chatroom').fill(chatroom);
    await page_guess.getByTestId('chatroom').press('Enter');

    // Assert a change room notification is printed in the chat log
    await expect(page_clue.getByTestId('chatlog')).toContainText('cluegiver has entered ' + chatroom);
    await expect(page_clue.getByTestId('chatlog')).toContainText('guesser has entered ');

    // Assert button / field visibilities and states
    await expect(page_guess.getByTestId('newgame')).toBeVisible();
    await expect(page_guess.getByTestId('abort')).toBeHidden();
    await expect(page_guess.getByTestId('team')).toBeEnabled();
    await expect(page_guess.getByTestId('role')).toBeEnabled();
    await expect(page_guess.getByTestId('clueheader')).toBeHidden();
    await expect(page_guess.getByTestId('numguess')).toBeHidden();
    await expect(page_guess.getByTestId('giveclue')).toBeHidden();
    await expect(page_guess.getByTestId('endturn')).toBeHidden();

    // Cluegiver assumes the clue giver role
    await page_clue.getByTestId('role').selectOption('Clue Giver');

    // Cluegiver starts new game
    await page_clue.getByTestId('newgame').click()

    // Assert button / field visibilities and states
    await expect(page_guess.getByTestId('newgame')).toBeHidden();
    await expect(page_guess.getByTestId('abort')).toBeVisible();
    await expect(page_guess.getByTestId('team')).toBeDisabled();
    await expect(page_guess.getByTestId('role')).toBeDisabled();
    await expect(page_guess.getByTestId('endturn')).toBeHidden();
    await expect(page_guess.getByTestId('numguess')).toContainText("It's red cluegiver's turn");

    await expect(page_clue.getByTestId('giveclue')).toBeVisible();
    await expect(page_clue.getByTestId('giveclue')).toBeEnabled();
    await expect(page_clue.getByTestId('endturn')).toBeHidden();

    // Sort cards and check colors
    await page_clue.getByTestId('sort').selectOption('Color - Keep Sorted');
    let i = 0;
    for (i = 0; i < 9; i++) {
        await expect(page_clue.locator(`#card-${i}`)).toHaveClass(/red/);
    }
    for (i = 9; i < 17; i++) {
        await expect(page_clue.locator(`#card-${i}`)).toHaveClass(/blue/);
    }
    await expect(page_clue.locator(`#card-17`)).toHaveClass(/black/);
    for (i = 18; i < 25; i++) {
        await expect(page_clue.locator(`#card-${i}`)).toHaveClass(/neutral/);
    }

    for (i = 0; i < 25; i++) {
        await expect(page_guess.locator(`#card-${i}`)).toHaveClass(/white/);
    }

    // Give clue
    await page_clue.getByTestId('giveclue').fill('avocado');
    await page_clue.getByTestId('number').press('Backspace');
    await page_clue.getByTestId('number').fill('1');
    await page_clue.getByTestId('number').press('Enter');
    await expect(page_clue.getByTestId('giveclue')).toBeDisabled();

    // Guesser's turn
    await expect(page_guess.getByTestId('clueheader')).toContainText('avocado');
    await expect(page_guess.getByTestId('numguess')).toContainText('2');
    await expect(page_guess.getByTestId('endturn')).toBeVisible();

    // Close all pages and contexts
    await page_clue.close();
    await page_guess.close();
    await context_clue.close();
    await context_guess.close();
});