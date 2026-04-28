package main

import (
	"fmt"
	"strings"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// mismatch records a single state or legal-action difference.
//
// Warning is set true when the diff was observed mid-stack (either side has
// a non-empty stack at comparison time). These are typically transient: CR
// 603.3b lets the controller pick any order for simultaneous triggers, so
// engines can briefly disagree on intermediate state while resolving the
// stack. They tend to converge by the time the stack empties — a warning is
// useful to surface but not a failure on its own.
type mismatch struct {
	Game    int
	Turn    int
	Step    string
	Field   string
	Go      string
	XMage   string
	Warning bool
}

func (m mismatch) String() string {
	tag := ""
	if m.Warning {
		tag = " [warn]"
	}
	return fmt.Sprintf("T%d %s%s: %s: go=%s xmage=%s", m.Turn, m.Step, tag, m.Field, m.Go, m.XMage)
}

// canonicalStep converts a mage-go PhaseStep to the canonical wire name.
func canonicalStep(ps core.PhaseStep) string {
	switch ps {
	case core.Untap:
		return "untap"
	case core.Upkeep:
		return "upkeep"
	case core.Draw:
		return "draw"
	case core.PrecombatMain:
		return "precombat_main"
	case core.BeginCombat:
		return "begin_combat"
	case core.DeclareAttackers:
		return "declare_attackers"
	case core.DeclareBlockers:
		return "declare_blockers"
	case core.FirstStrikeDamage:
		return "first_strike_damage"
	case core.CombatDamage:
		return "combat_damage"
	case core.EndCombat:
		return "end_combat"
	case core.PostcombatMain:
		return "postcombat_main"
	case core.EndStep:
		return "end_step"
	case core.Cleanup:
		return "cleanup"
	}
	return "unknown"
}

// compareStates compares two cvStates and returns all mismatches. Diffs that
// occur while either engine has a non-empty stack are flagged as warnings
// (Warning=true), since they're typically transient ordering effects.
func compareStates(goState, xmageState *cvState) []mismatch {
	if goState == nil || xmageState == nil {
		return []mismatch{{Field: "state", Go: fmt.Sprintf("%v", goState), XMage: fmt.Sprintf("%v", xmageState)}}
	}

	midStack := len(goState.Stack) > 0 || len(xmageState.Stack) > 0
	var mm []mismatch
	turn := goState.Turn
	step := goState.Step

	if goState.Turn != xmageState.Turn {
		mm = append(mm, mismatch{Turn: turn, Step: step, Field: "turn",
			Go: fmt.Sprintf("%d", goState.Turn), XMage: fmt.Sprintf("%d", xmageState.Turn)})
	}
	if goState.Step != xmageState.Step {
		mm = append(mm, mismatch{Turn: turn, Step: step, Field: "step",
			Go: goState.Step, XMage: xmageState.Step})
	}
	if goState.ActivePlayerIdx != xmageState.ActivePlayerIdx {
		mm = append(mm, mismatch{Turn: turn, Step: step, Field: "active_player_idx",
			Go: fmt.Sprintf("%d", goState.ActivePlayerIdx), XMage: fmt.Sprintf("%d", xmageState.ActivePlayerIdx)})
	}

	for i := range 2 {
		prefix := fmt.Sprintf("player[%d]", i)
		gp := goState.Players[i]
		xp := xmageState.Players[i]

		if gp.Life != xp.Life {
			mm = append(mm, mismatch{Turn: turn, Step: step, Field: prefix + ".life",
				Go: fmt.Sprintf("%d", gp.Life), XMage: fmt.Sprintf("%d", xp.Life)})
		}

		goHand := strings.Join(gp.Hand, ", ")
		xmHand := strings.Join(xp.Hand, ", ")
		if goHand != xmHand {
			mm = append(mm, mismatch{Turn: turn, Step: step, Field: prefix + ".hand",
				Go: goHand, XMage: xmHand})
		}

		goNames := permNamesOnly(gp.Battlefield)
		xmNames := permNamesOnly(xp.Battlefield)
		if goNames != xmNames {
			goBF := permNames(gp.Battlefield)
			xmBF := permNames(xp.Battlefield)
			mm = append(mm, mismatch{Turn: turn, Step: step, Field: prefix + ".battlefield",
				Go: goBF, XMage: xmBF})
		}

		goGY := strings.Join(gp.Graveyard, ", ")
		xmGY := strings.Join(xp.Graveyard, ", ")
		if goGY != xmGY {
			mm = append(mm, mismatch{Turn: turn, Step: step, Field: prefix + ".graveyard",
				Go: goGY, XMage: xmGY})
		}

		if gp.LibrarySize != xp.LibrarySize {
			mm = append(mm, mismatch{Turn: turn, Step: step, Field: prefix + ".library_size",
				Go: fmt.Sprintf("%d", gp.LibrarySize), XMage: fmt.Sprintf("%d", xp.LibrarySize)})
		}
	}

	if len(goState.Stack) != len(xmageState.Stack) {
		goStack := strings.Join(goState.Stack, ", ")
		xmStack := strings.Join(xmageState.Stack, ", ")
		mm = append(mm, mismatch{Turn: turn, Step: step, Field: "stack",
			Go: goStack, XMage: xmStack})
	}

	if midStack {
		for i := range mm {
			mm[i].Warning = true
		}
	}
	return mm
}

// parseStep converts a canonical wire step name back to a PhaseStep.
func parseStep(s string) (core.PhaseStep, bool) {
	switch s {
	case "untap":
		return core.Untap, true
	case "upkeep":
		return core.Upkeep, true
	case "draw":
		return core.Draw, true
	case "precombat_main":
		return core.PrecombatMain, true
	case "begin_combat":
		return core.BeginCombat, true
	case "declare_attackers":
		return core.DeclareAttackers, true
	case "declare_blockers":
		return core.DeclareBlockers, true
	case "first_strike_damage":
		return core.FirstStrikeDamage, true
	case "combat_damage":
		return core.CombatDamage, true
	case "end_combat":
		return core.EndCombat, true
	case "postcombat_main":
		return core.PostcombatMain, true
	case "end_step":
		return core.EndStep, true
	case "cleanup":
		return core.Cleanup, true
	}
	return 0, false
}

func permNamesOnly(perms []cvPermanent) string {
	names := make([]string, len(perms))
	for i, p := range perms {
		names[i] = p.Name
	}
	return strings.Join(names, ", ")
}

func permNames(perms []cvPermanent) string {
	names := make([]string, len(perms))
	for i, p := range perms {
		suffix := ""
		if p.Tapped {
			suffix = "(T)"
		}
		names[i] = p.Name + suffix
	}
	return strings.Join(names, ", ")
}
