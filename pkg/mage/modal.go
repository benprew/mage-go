package mage

import (
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"github.com/google/uuid"
)

// CR 700.2 — Modal spells and abilities. A spell or ability is modal if it
// has two or more options preceded by "Choose one —", "Choose two —",
// "Choose one or both —", etc. The controller of the spell or ability
// chooses the desired number of modes as the spell or ability is being cast
// or activated. Each chosen mode defines its own targets (if any) and its
// own one-shot effect. Targets and additional costs are chosen only for
// the chosen modes (CR 700.2d, 601.2c).
//
// Engine model:
//   - Mode bundles a label, a list of Target requirements, and a list of
//     Effect resolutions specific to that mode.
//   - A *ModalSpellAbility is a SpellAbility whose Modes() list defines the
//     options. Building one with NewModalSpell([]Mode{…}) sets the card up
//     to prompt Player.ChooseMode at cast time, gather targets only for
//     the chosen mode, and resolve only that mode's effects.
//   - The chosen mode index is stored on the StackObject (ModeChoice). The
//     chosen mode's targets are stored on StackObject.Targets (and also on
//     StackObject.ModalTargets[ModeChoice]).
//   - At resolution, the executor runs only the effects of Modes()[ModeChoice].
//     The shared ModeValue() / currentMode plumbing remains so legacy
//     SetModes-style cards continue to work alongside the new API.
//
// Modal triggered abilities (Trusty Retriever, Entomber Exarch ETB) are
// supported via ModalTriggerEffect, which prompts ChooseMode at trigger
// resolution and dispatches to the chosen mode's effect.

// Mode describes one option of a modal spell or ability. Targets is the
// list of Target requirements specific to this mode (zero or more); the
// controller is prompted only for these when this mode is chosen. Effects
// is the list of Effect actions that resolve when this mode is chosen.
// Label is the human-readable mode text shown to the player when prompting
// ChooseMode (matching what's printed on the card).
type Mode struct {
	Label   string
	Targets []Target
	Effects []Effect
}

// ModalSpellAbility extends SpellAbility with a list of modes. At cast time
// the engine prompts ChooseMode, gathers targets for the chosen mode only,
// and resolves only that mode's effects.
type ModalSpellAbility struct {
	SpellAbility
	modes []Mode
}

// Modes returns the configured modes.
func (m *ModalSpellAbility) Modes() []Mode { return m.modes }

// NewModalSpell builds a ModalSpellAbility for a spell with two or more
// modes ("Choose one — …"). Each Mode carries its own targets and effects.
//
// Example (Crushing Canopy: "Choose one — Destroy target creature with
// flying. / Destroy target enchantment."):
//
//	mage.NewModalSpell([]mage.Mode{
//	    {
//	        Label:   "Destroy target creature with flying",
//	        Targets: []mage.Target{mage.TargetCreature(mage.HasKeywordFilter(core.Flying))},
//	        Effects: []mage.Effect{mage.DestroyTarget()},
//	    },
//	    {
//	        Label:   "Destroy target enchantment",
//	        Targets: []mage.Target{mage.TargetPermanent(mage.IsEnchantment)},
//	        Effects: []mage.Effect{mage.DestroyTarget()},
//	    },
//	})
//
// At least two modes are required; passing fewer panics — modal spells
// have at least two options by definition (CR 700.2).
func NewModalSpell(modes []Mode) *ModalSpellAbility {
	if len(modes) < 2 {
		panic("NewModalSpell: a modal spell needs at least two modes (CR 700.2)")
	}
	ms := &ModalSpellAbility{
		SpellAbility: SpellAbility{
			BaseAbility: BaseAbility{
				id:          uuid.New(),
				abilityType: AbilitySpell,
			},
		},
		modes: modes,
	}
	return ms
}

// getModalSpellAbility returns the *ModalSpellAbility on a card, if any.
func getModalSpellAbility(card Card) (*ModalSpellAbility, bool) {
	if card == nil {
		return nil, false
	}
	for _, a := range card.Abilities() {
		if ms, ok := a.(*ModalSpellAbility); ok {
			return ms, true
		}
	}
	return nil, false
}

// modalLabels extracts the human-readable mode labels from a modal spell.
func modalLabels(ms *ModalSpellAbility) []string {
	out := make([]string, len(ms.modes))
	for i, m := range ms.modes {
		out[i] = m.Label
	}
	return out
}

// ModalTriggerEffect builds an Effect for a modal triggered ability (e.g.
// Trusty Retriever, Entomber Exarch ETB). When the trigger resolves, the
// controller is prompted with ChooseMode over the mode labels; the chosen
// mode's Resolve callback runs.
//
// This is the canonical pattern for modal triggers — distinct from modal
// spells because triggered abilities pick the mode at resolution time
// (CR 603.3c) rather than at cast time. Targets are still chosen when the
// trigger is put on the stack via the engine's normal trigger-target
// pathway (use ability.AddTarget for each mode's target); ModalTriggerEffect
// itself only prompts for the mode and dispatches.
func ModalTriggerEffect(reason string, modes []ModalTriggerMode) Effect {
	if len(modes) < 2 {
		panic("ModalTriggerEffect: a modal trigger needs at least two modes (CR 700.2)")
	}
	labels := make([]string, len(modes))
	for i, m := range modes {
		labels[i] = m.Label
	}
	return FuncEffect(reason, EffectProperties{},
		func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
			pl := g.GetPlayer(controller)
			if pl == nil {
				return nil
			}
			idx := pl.ChooseMode(labels, reason)
			if idx < 0 || idx >= len(modes) {
				idx = 0
			}
			return modes[idx].Resolve(g, sourceID, controller, targets)
		})
}

// ModalTriggerMode is one option of a ModalTriggerEffect. Resolve runs the
// mode's chosen effect, taking the trigger's already-chosen targets as
// input (the trigger's target prompt happens at put-on-stack time per
// CR 603.3d, before mode selection).
type ModalTriggerMode struct {
	Label   string
	Resolve func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error
}
