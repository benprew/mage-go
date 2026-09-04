// Package combatsolver calculates combat decisions for the AI.
//
// The package calculates three decisions:
//   - Which creatures attack.
//   - Which creatures block.
//   - Which instant spells and activated abilities the AI activates after
//     blockers are declared.
//
// The solver uses a minimax algorithm across the combat phase:
//
//	L1 attacker chooses attacker subset A
//	  L2 defender chooses blocks B*(A) that maximize defender evaluation
//	    L3 attacker activates responses R*(A,B) that maximize attacker evaluation
//	      ExecuteCombatDamage; evaluate position
//
// If a creature has first strike or double strike, L3 runs two times. It runs
// before normal combat damage, and it runs during normal combat damage.
//
// When the AI defends, the analysis changes roles. The attacking creatures are
// fixed. The AI selects blockers at L1, then selects responses at L2.
//
// The solver does not simulate spells from the hand of the opponent. The
// solver simulates activated abilities of the opponent, including abilities
// with discard costs. Future updates will add probability models for unknown
// cards in the hand of the opponent.
//
// Both [HeuristicStrategy] and [SearchStrategy] call this solver.
//
// This package imports only pkg/mage and pkg/mage/interactive/eval. This
// limitation prevents import cycles with the ai package.
package combatsolver
