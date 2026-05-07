// Package combatsolver computes joint-optimal combat decisions for the AI:
// which attackers to declare, which blocks to assign, and which combat-eligible
// instants and activated abilities to fire in the post-blockers response window.
//
// The solver is a staged minimax over the natural priority structure of an MTG
// combat phase:
//
//	L1 attacker chooses attacker subset A
//	  L2 defender chooses blocks B*(A) maximising defender eval
//	    L3 attacker activates trick-response R*(A,B) maximising attacker eval
//	      ExecuteCombatDamage; eval position
//
// When any creature in combat has first strike or double strike, L3 fires twice:
// once before normal damage (so first-strike-only damage is observable) and once
// at normal damage.
//
// Defender-side analysis swaps the roles: opponent's attackers are fixed,
// the AI picks blocks at L1, then tricks at L2.
//
// Per spec, the solver does not model spells from the opponent's hand. It does
// model the opponent's activated abilities, including those with hand-cost
// components (cycling, discard-to-buff). Probabilistic modelling of unknown
// opponent tricks is a phase-2 follow-up.
//
// The solver is called as a subroutine by both [HeuristicStrategy] and (in
// phase 4) [SearchStrategy] in the parent ai/ package.
//
// Dependencies are intentionally limited to pkg/mage and pkg/mage/interactive/eval
// to keep the package free of import cycles with ai/.
package combatsolver
