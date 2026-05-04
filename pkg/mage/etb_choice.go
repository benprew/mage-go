package mage

import (
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"

	"github.com/google/uuid"
)

// "As ~ enters the battlefield, choose ___" — CR 614.12.
//
// Per CR 614.12, an "as it enters" replacement effect makes the controller
// pick a value (color, creature type, opponent, etc.) at the moment the
// permanent enters. The choice is recorded on the permanent itself
// (Permanent.ChosenColor / ChosenPlayer / ChosenSubtype) so that the
// permanent's other abilities can consult it later.
//
// Implementation: each constructor below returns an *ETBEffectAbility,
// which PutOnBattlefield runs unconditionally during ETB resolution
// BEFORE firing the EvtEntersBattlefield event. This ordering means the
// stored choice is already in place when "when this enters" triggers
// fire — exactly what CR 614.12 requires.

// ETBChooseColor returns an ETB replacement that asks the controller to
// choose a color and stores it on the source permanent's ChosenColor field.
//
// Use this for cards like "As ~ enters, choose a color." Reading code
// later consults Permanent.ChosenColor (e.g. via ChangeAttachedSubTypesByChosenColor
// or a custom mana ability).
func ETBChooseColor(reason string) *ETBEffectAbility {
	return ETBEffect(FuncEffect(
		"as ~ enters, "+reason,
		EffectProperties{},
		func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
			perm := g.FindPermanent(sourceID)
			if perm == nil {
				return nil
			}
			p := g.GetPlayer(controller)
			if p == nil {
				return nil
			}
			perm.ChosenColor = p.ChooseManaColor(reason)
			return nil
		}))
}

// ETBChooseColorOtherThan returns an ETB replacement that asks the controller
// to choose a color other than `excluded`, storing the result on
// Permanent.ChosenColor. Used by the Thriving land cycle ("As this enters,
// choose a color other than {its color}.").
//
// The excluded color is filtered out of the candidate list before the choice
// is presented, and the implementation re-prompts (via the player's
// ChooseManaColor) until a legal color is returned. Per CR 614.12, this
// happens during the ETB replacement step.
func ETBChooseColorOtherThan(reason string, excluded Color) *ETBEffectAbility {
	return ETBEffect(FuncEffect(
		"as ~ enters, "+reason,
		EffectProperties{},
		func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
			perm := g.FindPermanent(sourceID)
			if perm == nil {
				return nil
			}
			p := g.GetPlayer(controller)
			if p == nil {
				return nil
			}
			// Loop until the player picks a valid color (i.e. not the
			// excluded one and not Colorless/AnyColor sentinels). Most
			// player implementations honor the constraint on first try;
			// the loop guards against scripted choices supplying an
			// illegal value.
			for tries := 0; tries < 8; tries++ {
				c := p.ChooseManaColor(reason)
				if c != excluded && c != Colorless && c != AnyColor {
					perm.ChosenColor = c
					return nil
				}
			}
			// Fallback: pick the first legal basic color.
			for _, c := range []Color{White, Blue, Black, Red, Green} {
				if c != excluded {
					perm.ChosenColor = c
					return nil
				}
			}
			return nil
		}))
}

// ETBChooseOpponent returns an ETB replacement that asks the controller to
// choose an opponent, storing the result on Permanent.ChosenPlayer.
// In a 2-player game this is mechanically forced but we still set the field
// at ETB-replacement time (per CR 614.12) so that downstream effects
// (Nyxathid's P/T-by-opponent's-hand-size, Black Vise's upkeep trigger,
// Jihad's color-protection) read a consistent value from the moment of
// entry forward.
func ETBChooseOpponent() *ETBEffectAbility {
	return ETBEffect(FuncEffect(
		"as ~ enters, choose an opponent",
		EffectProperties{},
		func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
			perm := g.FindPermanent(sourceID)
			if perm == nil {
				return nil
			}
			opp := g.GetOpponent(controller)
			if opp != nil {
				perm.ChosenPlayer = opp.PlayerID()
			}
			return nil
		}))
}

// ETBChooseCreatureType returns an ETB replacement that asks the controller
// to choose one creature type from `options`, storing the result on
// Permanent.ChosenSubtype. Used by cards like Herald's Horn ("As ~ enters,
// choose a creature type.").
//
// The caller supplies the candidate list (typically the engine's curated
// creature-type vocabulary). The chosen string is written verbatim to
// ChosenSubtype and is later compared case-sensitively against
// card.SubTypes(), matching how engine subtype filters operate.
func ETBChooseCreatureType(options []string) *ETBEffectAbility {
	return ETBEffect(FuncEffect(
		"as ~ enters, choose a creature type",
		EffectProperties{},
		func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
			perm := g.FindPermanent(sourceID)
			if perm == nil {
				return nil
			}
			p := g.GetPlayer(controller)
			if p == nil {
				return nil
			}
			if len(options) == 0 {
				return nil
			}
			perm.ChosenSubtype = p.ChooseString(options, "choose a creature type")
			return nil
		}))
}
