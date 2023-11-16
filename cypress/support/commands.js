Cypress.Commands.add('login', (username) => {
    cy.readFile('./password.txt').then((password) => {
        cy.get('input[data-test="username"]').type(username)
        cy.get('input[data-test="password"]').type(`${password}{enter}`)
    })
})

Cypress.Commands.add('sendmessage', (message) => {
    cy.get('input[data-test="message"]').type(`${message}{enter}`)

    cy.get('[data-test="chatlog"]').should('contain', message)
})

Cypress.Commands.add('changeroom', (room) => {
    cy.get('input[data-test="chatroom"]').type(`${room}{enter}`)

    cy.get('[data-test="chatlog"]').should('contain', ` has entered ${room}`)
})

Cypress.Commands.add('becomecluegiver', () => {
    cy.get('select[data-test="role"]').select('cluegiver')
})

Cypress.Commands.add('startgame', () => {
    cy.get('input[data-test="newgame"]').should('not.be.hidden')
    cy.get('input[data-test="abort"]').should('be.hidden')
    cy.get('[data-test="team"]').should('not.be.disabled')
    cy.get('[data-test="role"]').should('not.be.disabled')

    cy.get('input[data-test="newgame"]').click()

    cy.get('input[data-test="newgame"]').should('be.hidden')
    cy.get('input[data-test="abort"]').should('not.be.hidden')
    cy.get('[data-test="team"]').should('be.disabled')
    cy.get('[data-test="role"]').should('be.disabled')
})

Cypress.Commands.add('sortcards', (how) => {
    cy.get('select[data-test="sort"]').select(how)
})

Cypress.Commands.add('giveclue', (clue, num) => {
    cy.get('input[data-test="number"]').type(`{backspace}${num}`)
    cy.get('input[data-test="giveclue"').type(`${clue}{enter}`)

    cy.get('[data-test="clueheader"]').should('contain', clue)
    if (num > 0) {
        cy.get('[data-test="numguess"]').should('contain', num + 1)
    } else {
        cy.get('[data-test="clueheader"]').should('contain', 'unlimited guesses')
    }
    cy.get('input[data-test="giveclue"').should('be.disabled')
})