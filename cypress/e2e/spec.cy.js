/// <reference types="cypress" />

describe('example to-do app', () => {
    beforeEach(() => {
      cy.visit('/')
    })

    it('starts a game as the guesser', () => {
        const username = 'redguess'
        const room = 'redroom'
        let i = 0

        cy.login(username)
        cy.sendmessage('meet me in ' + room)
        cy.changeroom(room)

        cy.get('input[data-test="endturn"]').should('be.hidden')

        cy.startgame()

        for (i = 0; i < 25; i++) {
            cy.get(`#card-${i}`).should('have.class', 'white')
        }
    })

    it('starts a game as the cluegiver', () => {
        const username = 'redclue'
        const room = 'roomy'
        let i = 0

        cy.login(username)
        cy.sendmessage('meet me in ' + room)
        cy.changeroom(room)
        cy.becomecluegiver()
        cy.startgame()

        cy.sortcards('keep-sorted')
        for (i = 0; i < 9; i++) {
            cy.get(`#card-${i}`).should('have.class', 'red')
        }
        for (i = 9; i < 17; i++) {
            cy.get(`#card-${i}`).should('have.class', 'blue')
        }
        cy.get(`#card-17`).should('have.class', 'black')
        for (i = 18; i < 25; i++) {
            cy.get(`#card-${i}`).should('have.class', 'neutral')
        }

        cy.giveclue('avocado', 1)
    })
})