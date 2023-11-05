package main

import (
	"reflect"
	"testing"
)

// Calling Change() on a Role should return a Role.
func TestRoleChangeType(t *testing.T) {
	var myRole Role
	myRole = cluegiver
	roleChange := myRole.Change()
	if reflect.TypeOf(roleChange) != reflect.TypeOf(myRole) {
		t.Errorf("type of roleChange: %v", reflect.TypeOf(roleChange))
	}
}

// Calling Change() on a cluegiver Role should return
// the guesser Role.
func TestRoleChangeValue0(t *testing.T) {
	myRole := cluegiver
	roleChange := myRole.Change()
	if roleChange != guesser {
		t.Errorf("value of roleChange: %v", roleChange)
	}
}

// Calling Change() on a guesser Role should return
// the cluegiver Role.
func TestRoleChangeValue1(t *testing.T) {
	myRole := guesser
	roleChange := myRole.Change()
	if roleChange != cluegiver {
		t.Errorf("value of roleChange: %v", roleChange)
	}
}

// Calling Change() on a Team should return a Team.
func TestTeamChangeType(t *testing.T) {
	var myTeam Team
	myTeam = red
	teamChange := myTeam.Change()
	if reflect.TypeOf(teamChange) != reflect.TypeOf(myTeam) {
		t.Errorf("type of teamChange: %v", reflect.TypeOf(teamChange))
	}
}

// Calling Change() on the red Team should return blue.
func TestTeamChangeValue0(t *testing.T) {
	myTeam := red
	teamChange := myTeam.Change()
	if teamChange != blue {
		t.Errorf("value of teamChange: %v", teamChange)
	}
}

// Calling Change() on the blue Team should return red.
func TestTeamChangeValue1(t *testing.T) {
	myTeam := blue
	teamChange := myTeam.Change()
	if teamChange != red {
		t.Errorf("value of teamChange: %v", teamChange)
	}
}