const { test, expect, chromium } = require('@playwright/test');
const exp = require('constants');
var fs = require('fs');

const chatroom = 'gameroom';
const password = fs.readFileSync('./password.txt', { encoding: 'utf8', flag: 'r' });

test('play two-player game', async () => {
    // Cluegiver login
    const context_clue = await chromium.launch({ headless: false, slowMo: 100 });
    const page_clue = await context_clue.newPage();
    page_clue.on('console', (msg) => {
        console.log('cluegiver: ' + msg.text());
    });
    page_clue.on('websocket', ws => {
        console.log(`WebSocket opened: ${ws.url()}>`);
        ws.on('framesent', event => console.log(event.payload));
        ws.on('framereceived', event => console.log(event.payload));
        ws.on('close', () => console.log('WebSocket closed'));
    });
    page_clue.on('dialog', async alert => {
        const msg = alert.message();
        console.log('cluegiver: ' + msg);
        await expect(msg).toBe('Red Team uncovers the Black Card. Red Team loses!');
        await alert.dismiss();
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
    page_guess.on('console', (msg) => {
        console.log('guesser: ' + msg.text());
    });
    page_guess.on('websocket', ws => {
        console.log(`WebSocket opened: ${ws.url()}>`);
        ws.on('framesent', event => console.log(event.payload));
        ws.on('framereceived', event => console.log(event.payload));
        ws.on('close', () => console.log('WebSocket closed'));
    });
    page_guess.on('dialog', async alert => {
        const msg = alert.message();
        console.log('guesser: ' + msg);
        await expect(msg).toBe('Red Team uncovers the Black Card. Red Team loses!');
        await alert.dismiss();
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
    await expect(page_clue.getByTestId('chatlog')).toContainText(`cluegiver has entered room "${chatroom}"`);
    await expect(page_clue.getByTestId('chatlog')).toContainText('guesser has entered ');

    // Assert button / field visibilities and states
    await expect(page_guess.getByTestId('newgame')).toBeVisible();
    await expect(page_guess.getByTestId('abort')).toBeHidden();
    await expect(page_guess.getByTestId('team')).toBeEnabled();
    await expect(page_guess.getByTestId('role')).toBeEnabled();
    await expect(page_guess.getByTestId('clue')).toBeEmpty();
    await expect(page_guess.getByTestId('numguess')).toBeEmpty();
    await expect(page_guess.getByTestId('giveclue')).toBeHidden();
    await expect(page_guess.getByTestId('endturn')).toBeHidden();
    await expect(page_guess.getByTestId('redscore')).toBeEmpty();
    await expect(page_guess.getByTestId('bluescore')).toBeEmpty();

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
    await expect(page_guess.getByTestId('numguess')).toBeEmpty();
    await expect(page_guess.getByTestId('turn')).toContainText("Red");
    await expect(page_guess.getByTestId('turn')).toContainText("Cluegiver");

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
    let clue = 'avocado';
    await page_clue.getByTestId('giveclue').fill(clue);
    await page_clue.getByTestId('number').press('Backspace');
    await page_clue.getByTestId('number').fill('1');
    await page_clue.getByTestId('number').press('Enter');
    await expect(page_clue.getByTestId('giveclue')).toBeDisabled();

    // Guesser's turn
    await expect(page_guess.getByTestId('turn')).toContainText("Red");
    await expect(page_guess.getByTestId('turn')).toContainText("Guesser");
    await expect(page_guess.getByTestId('clue')).toContainText(clue);
    await expect(page_guess.getByTestId('numguess')).toHaveText('2');
    await expect(page_guess.getByTestId('endturn')).toBeVisible();

    // Find two red cards for guesser
    let word = "";
    for (i = 0; i < 2; i++) {
        word = await page_clue.locator('#card-0').textContent();
        let j = 0;
        while (j < 25) {
            const card = page_guess.locator(`#card-${j}`);
            if (await card.textContent() === word) {
                await card.click();
                if (i === 0) {
                    await expect(page_guess.getByTestId('numguess')).toHaveText('1');
                }
                await expect(page_guess.getByTestId('redscore')).toHaveText(`${8-i}`);
                await expect(page_guess.getByTestId('bluescore')).toHaveText('8');
                // Cluegiver's cards are kept sorted, with guessed cards at the end
                const cluecard = page_clue.locator('#card-24');
                await expect(cluecard).toContainText(word);
                await expect(cluecard).toHaveClass(/guessed-red/);
                break;
            }
            j++;
        }
        /* There are 9 red cards. If all are at the end of the alphabetically
           sorted list (extreme scenario), the first red card will be card #
           24 - 9 = 15 (indexed from 0) and the second will be card # 16.*/
        await expect(j < 17).toBeTruthy();
    }

    // Guesser has exhausted all guesses. Should be cluegiver's turn.
    await expect(page_guess.getByTestId('endturn')).toBeHidden();
    await expect(page_guess.getByTestId('numguess')).toBeEmpty();
    await expect(page_guess.getByTestId('turn')).toContainText("Red");
    await expect(page_guess.getByTestId('turn')).toContainText("Cluegiver");

    // Give clue with unlimited guesses
    clue = 'banana';
    await expect(page_clue.getByTestId('giveclue')).toBeEnabled();
    await page_clue.getByTestId('giveclue').fill(clue);
    await page_clue.getByTestId('number').press('Backspace');
    await page_clue.getByTestId('number').fill('0');
    await page_clue.getByTestId('number').press('Enter');
    await expect(page_clue.getByTestId('giveclue')).toBeDisabled();

    // Guesser's turn
    await expect(page_guess.getByTestId('turn')).toContainText("Red");
    await expect(page_guess.getByTestId('turn')).toContainText("Guesser");
    await expect(page_guess.getByTestId('clue')).toContainText(clue);
    await expect(page_guess.getByTestId('numguess')).toContainText('\u221E');
    await expect(page_guess.getByTestId('endturn')).toBeVisible();

    // Guesser guesses a blue card
    word = await page_clue.locator('#card-7').textContent();
    let j = 0;
    while (j < 25) {
        const card = page_guess.locator(`#card-${j}`);
        if (await card.textContent() === word) {
            await card.click();
            await expect(page_guess.getByTestId('redscore')).toHaveText('7');
            await expect(page_guess.getByTestId('bluescore')).toHaveText('7');
            // Cluegiver's cards are kept sorted, with guessed cards at the end
            const cluecard = page_clue.locator('#card-24');
            await expect(cluecard).toContainText(word);
            await expect(cluecard).toHaveClass(/guessed-blue/);
            break;
        }
        j++;
    }
    /* There are 8 blue cards. If all are at the end of the alphabetically
       sorted list (extreme scenario), the first blue card will be card #
       24 - 8 = 16 (indexed from 0).*/
    await expect(j < 17).toBeTruthy();

    // Guesser guessed wrong color. Should be cluegiver's turn.
    await expect(page_guess.getByTestId('endturn')).toBeHidden();
    await expect(page_guess.getByTestId('numguess')).toBeEmpty();
    await expect(page_guess.getByTestId('turn')).toContainText("Red");
    await expect(page_guess.getByTestId('turn')).toContainText("Cluegiver");

    // Give clue with unlimited guesses
    clue = 'pear';
    await expect(page_clue.getByTestId('giveclue')).toBeEnabled();
    await page_clue.getByTestId('giveclue').fill(clue);
    await page_clue.getByTestId('number').press('Backspace');
    await page_clue.getByTestId('number').fill('0');
    await page_clue.getByTestId('number').press('Enter');
    await expect(page_clue.getByTestId('giveclue')).toBeDisabled();

    // Guesser's turn
    await expect(page_guess.getByTestId('turn')).toContainText("Red");
    await expect(page_guess.getByTestId('turn')).toContainText("Guesser");
    await expect(page_guess.getByTestId('clue')).toContainText(clue);
    await expect(page_guess.getByTestId('numguess')).toContainText('\u221E');
    await expect(page_guess.getByTestId('endturn')).toBeVisible();

    // Guesser ends turn
    await page_guess.getByTestId('endturn').click();

    // Should be cluegiver's turn.
    await expect(page_guess.getByTestId('endturn')).toBeHidden();
    await expect(page_guess.getByTestId('numguess')).toBeEmpty();
    await expect(page_guess.getByTestId('turn')).toContainText("Red");
    await expect(page_guess.getByTestId('turn')).toContainText("Cluegiver");


    // Give clue with unlimited guesses
    clue = 'raspberry';
    await expect(page_clue.getByTestId('giveclue')).toBeEnabled();
    await page_clue.getByTestId('giveclue').fill(clue);
    await page_clue.getByTestId('number').press('Backspace');
    await page_clue.getByTestId('number').fill('0');
    await page_clue.getByTestId('number').press('Enter');
    await expect(page_clue.getByTestId('giveclue')).toBeDisabled();

    // Guesser guesses three red cards
    for (i = 0; i < 3; i++) {
        word = await page_clue.locator('#card-0').textContent();
        j = 0;
        while (j < 25) {
            const card = page_guess.locator(`#card-${j}`);
            if (await card.textContent() === word) {
                await card.click();
                await expect(page_guess.getByTestId('redscore')).toHaveText(`${6 - i}`);
                await expect(page_guess.getByTestId('bluescore')).toHaveText('7');
                await expect(page_guess.getByTestId('numguess')).toContainText('\u221E');
                // Card #24 is the blue guessed card
                const cluecard = page_clue.locator('#card-23');
                await expect(cluecard).toContainText(word);
                await expect(cluecard).toHaveClass(/guessed-red/);
                break;
            }
            j++;
        }
        await expect(j < 23).toBeTruthy();
    }

    // Guesser guesses neutral card
    // Loc: 24 - 4 (remaining red) - 7 (remaining blue) - 1 (black)
    word = await page_clue.locator('#card-12').textContent();
    j = 0;
    while (j < 25) {
        const card = page_guess.locator(`#card-${j}`);
        if (await card.textContent() === word) {
            await card.click();
            await expect(page_guess.getByTestId('redscore')).toHaveText('4');
            await expect(page_guess.getByTestId('bluescore')).toHaveText('7');
            const cluecard = page_clue.locator('#card-24');
            await expect(cluecard).toContainText(word);
            await expect(cluecard).toHaveClass(/guessed-neutral/);
            break;
        }
        j++;
    }

    // Guesser guessed wrong color. Should be cluegiver's turn.
    await expect(page_guess.getByTestId('endturn')).toBeHidden();
    await expect(page_guess.getByTestId('numguess')).toBeEmpty();
    await expect(page_guess.getByTestId('turn')).toContainText("Red");
    await expect(page_guess.getByTestId('turn')).toContainText("Cluegiver");

    // Give clue with 4 guesses
    clue = 'raspberry';
    await expect(page_clue.getByTestId('giveclue')).toBeEnabled();
    await page_clue.getByTestId('giveclue').fill(clue);
    await page_clue.getByTestId('number').press('Backspace');
    await page_clue.getByTestId('number').fill('3');
    await page_clue.getByTestId('number').press('Enter');
    await expect(page_clue.getByTestId('giveclue')).toBeDisabled();

    // Guesser's turn
    await expect(page_guess.getByTestId('turn')).toContainText("Red");
    await expect(page_guess.getByTestId('turn')).toContainText("Guesser");
    await expect(page_guess.getByTestId('clue')).toContainText(clue);
    await expect(page_guess.getByTestId('numguess')).toContainText('4');
    await expect(page_guess.getByTestId('endturn')).toBeVisible();

    // Guesser guesses death card
    // Loc: 4 (remaining red) + 7 (remaining blue)
    word = await page_clue.locator('#card-11').textContent();
    j = 0;
    while (j < 25) {
        const card = page_guess.locator(`#card-${j}`);
        if (await card.textContent() === word) {
            await card.click();
            break;
        }
        j++;
    }
    await expect(page_clue.getByTestId('turn')).toBeEmpty();
    await expect(page_clue.getByTestId('clue')).toBeEmpty();
    await expect(page_clue.getByTestId('numguess')).toBeEmpty();
    await expect(page_clue.getByTestId('abort')).toHaveValue('End Game');

    // Close all pages and contexts
    await page_clue.close();
    await page_guess.close();
    await context_clue.close();
    await context_guess.close();
});