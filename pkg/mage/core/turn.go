package core

// PhaseStep represents a phase or step in a turn.
type PhaseStep int

const (
	Untap PhaseStep = iota
	Upkeep
	Draw
	PrecombatMain
	BeginCombat
	DeclareAttackers
	DeclareBlockers
	FirstStrikeDamage
	CombatDamage
	EndCombat
	PostcombatMain
	EndStep
	Cleanup
)

func (ps PhaseStep) String() string {
	switch ps {
	case Untap:
		return "Untap"
	case Upkeep:
		return "Upkeep"
	case Draw:
		return "Draw"
	case PrecombatMain:
		return "Precombat Main"
	case BeginCombat:
		return "Begin Combat"
	case DeclareAttackers:
		return "Declare Attackers"
	case DeclareBlockers:
		return "Declare Blockers"
	case FirstStrikeDamage:
		return "First Strike Damage"
	case CombatDamage:
		return "Combat Damage"
	case EndCombat:
		return "End of Combat"
	case PostcombatMain:
		return "Postcombat Main"
	case EndStep:
		return "End Step"
	case Cleanup:
		return "Cleanup"
	default:
		return "Unknown"
	}
}

// IsMainPhase returns true for main phases.
func (ps PhaseStep) IsMainPhase() bool {
	return ps == PrecombatMain || ps == PostcombatMain
}

// StepHasPriority returns true if players receive priority during this step.
// Only Untap has no priority round.
func (ps PhaseStep) StepHasPriority() bool {
	return ps != Untap
}

// AllSteps returns the normal step order for a turn.
func AllSteps() []PhaseStep {
	return []PhaseStep{
		Untap,
		Upkeep,
		Draw,
		PrecombatMain,
		BeginCombat,
		DeclareAttackers,
		DeclareBlockers,
		FirstStrikeDamage,
		CombatDamage,
		EndCombat,
		PostcombatMain,
		EndStep,
		Cleanup,
	}
}
