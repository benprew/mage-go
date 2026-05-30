package combatsolver

import (
	"time"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/interactive/eval"
)

// Profile is the personality input the solver needs. It mirrors the subset of
// ai.WeightedPersonality relevant to combat decisions, kept here to avoid an
// import cycle with the parent ai package.
type Profile struct {
	eval.Weights

	Aggression     float64
	BlockThreshold float64
}

// Options controls solver behaviour for a single Solve call.
type Options struct {
	Profile Profile

	// Deadline, if non-zero, bounds wall-clock time for the search. The solver
	// returns its best-so-far result when the deadline passes; DeadlineHit is
	// set on the Result.
	Deadline time.Time

	// BeamK caps the number of attacker subsets fully evaluated at L1. Zero
	// means no beam (full enumeration). Phase 2 default is 0; phase-2-bis adds
	// a beam once benchmarks show it's needed.
	BeamK int
}

// TrickPlan describes one combat-eligible instant or activated ability the
// solver expects to fire during the post-blockers response window.
type TrickPlan struct {
	// PermanentID is the on-battlefield source for an activated ability.
	// Zero UUID when the trick is a spell from hand (use CardID instead).
	PermanentID uuid.UUID

	// CardID is the spell-from-hand source. Zero UUID for activated abilities.
	CardID uuid.UUID

	// AbilityIdx is the index into the source's activated abilities; -1 when
	// the trick is a spell from hand.
	AbilityIdx int

	Targets []uuid.UUID
	XValue  int

	// ManaCost is the approximate mana commitment for this trick (CMC for
	// spells, mana cost portion of activation cost for abilities). Returned so
	// callers can weigh hold-vs-cast against other plays this turn.
	ManaCost int
}

// Result is the solver output.
type Result struct {
	// Attackers is populated by SolveAttack: the attacker subset to declare.
	Attackers []uuid.UUID

	// Blocks is populated by SolveDefense: blocker assignments to declare.
	Blocks []mage.BlockAssignment

	// HeldTricks lists the combat-eligible instants/abilities the solver
	// assumed would fire during the post-blockers response window. Callers
	// hold these (don't cast them sorcery-speed in the main phase) and fire
	// them when the blockers-declared priority window opens.
	HeldTricks []TrickPlan

	// Score is the leaf evaluation (eval.Evaluate) after combat resolves under
	// the chosen line. From the AI's perspective: positive = good.
	Score int

	// Nodes is the number of leaf evaluations performed. Diagnostic.
	Nodes int

	// DeadlineHit is true when the solver returned its best-so-far due to a
	// deadline rather than fully exploring the tree.
	DeadlineHit bool
}
