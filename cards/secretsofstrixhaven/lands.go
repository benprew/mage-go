package secretsofstrixhaven

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage"
	. "github.com/benprew/mage-go/pkg/mage/core"
)

func init() {
	registerLands()
}

// slowLand builds the "Innistrad slow land" pattern: enters tapped unless the
// controller controls two or more other lands. Produces color1 (primary, used by
// autotap) or color2 via two separate mana abilities.
func slowLand(name string, color1, color2 Color) Card {
	return NewLand(name,
		WithAbility(ETBEffect(FuncEffect(
			"enter tapped unless you control two or more other lands",
			EffectProperties{},
			func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
				perm := g.MutablePermanent(sourceID)
				if perm == nil {
					return nil
				}
				// Count lands the controller controls OTHER than this permanent.
				otherLands := g.CountBattlefield(And(IsLand, NotID(sourceID), ControlledBy(controller)))
				if otherLands < 2 {
					perm.Tapped = true
				}
				return nil
			},
		))),
		WithManaAbility(color1),
		WithManaAbility(color2),
	)
}

// surveilLand builds the "Strixhaven campus survey land" pattern: always enters
// tapped; produces color1 (primary) or color2 (secondary) via two separate mana
// abilities; activated surveil ability costing {manaCost}, {T}.
//
// Two separate ManaAbility instances are used (one per color) so that the
// harness's ActivateAbility picker — which prefers non-mana activated abilities
// in its first pass — will correctly find the surveil ability when called without
// a ChooseManaColor override.
func surveilLand(name, manaCost string, color1, color2 Color) Card {
	return NewLand(name,
		WithKeyword(EntersTapped),
		WithManaAbility(color1),
		WithManaAbility(color2),
		WithActivatedAbility(
			Surveil(Fixed(1)),
			ManaCostOf(manaCost),
			WithCost(Tap()),
		),
	)
}

func registerLands() {

	// Deathcap Glade
	// Land
	// This land enters tapped unless you control two or more other lands.
	// {T}: Add {B} or {G}.
	Register("Deathcap Glade", func() Card {
		return slowLand("Deathcap Glade", Black, Green)
	})

	// Dreamroot Cascade
	// Land
	// This land enters tapped unless you control two or more other lands.
	// {T}: Add {G} or {U}.
	Register("Dreamroot Cascade", func() Card {
		return slowLand("Dreamroot Cascade", Green, Blue)
	})

	// Fields of Strife
	// Land
	// This land enters tapped.
	// {T}: Add {R} or {W}.
	// {2}{R}{W}, {T}: Surveil 1. (Look at the top card of your library. You may put it into your graveyard.)
	Register("Fields of Strife", func() Card {
		return surveilLand("Fields of Strife", "{2}{R}{W}", Red, White)
	})

	// Forum of Amity
	// Land
	// This land enters tapped.
	// {T}: Add {W} or {B}.
	// {2}{W}{B}, {T}: Surveil 1. (Look at the top card of your library. You may put it into your graveyard.)
	Register("Forum of Amity", func() Card {
		return surveilLand("Forum of Amity", "{2}{W}{B}", White, Black)
	})

	// Great Hall of the Biblioplex
	// Land
	// {T}: Add {C}.
	// {T}, Pay 1 life: Add one mana of any color. Spend this mana only to cast an instant or sorcery spell.
	// {5}: If this land isn't a creature, it becomes a 2/4 Wizard creature with "Whenever you cast an
	// instant or sorcery spell, this creature gets +1/+0 until end of turn." It's still a land.
	Register("Great Hall of the Biblioplex", func() Card {
		// XXX: The second ability "Add one mana of any color. Spend this mana only to cast an instant
		// or sorcery spell." is only partially implemented — the mana restriction ("spend only on
		// instant or sorcery") is not enforceable by the engine, which has no per-mana-pool tagging
		// for spend restrictions beyond the existing artifact-only and creature-only flags.
		// The ability produces any-color mana without the spending restriction.
		//
		// XXX: The third ability "{5}: If this land isn't a creature, it becomes a 2/4 Wizard
		// creature with 'Whenever you cast an instant or sorcery spell, this creature gets +1/+0
		// until end of turn. It's still a land.'" is not implemented. The engine supports
		// AnimateTargetLand for spells/ETB effects targeting other lands, but there is no
		// mechanism for a land to animate itself via an activated ability with a conditional
		// ("if this isn't a creature") check, and the granted triggered ability on the animated
		// land is beyond the current animation options (AnimateLandOptions has no trigger slot).
		return NewLand("Great Hall of the Biblioplex",
			WithManaAbility(Colorless),
			WithActivatedAbility(
				FuncEffect(
					"add one mana of any color",
					EffectProperties{},
					func(g *Game, _, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						c := p.ChooseManaColor("add mana of any color")
						p.ManaPool().Add(c, 1)
						return nil
					},
				),
				Tap(),
				WithCost(LifePayCost(1)),
			),
		)
	})

	// Paradox Gardens
	// Land
	// This land enters tapped.
	// {T}: Add {G} or {U}.
	// {2}{G}{U}, {T}: Surveil 1. (Look at the top card of your library. You may put it into your graveyard.)
	Register("Paradox Gardens", func() Card {
		return surveilLand("Paradox Gardens", "{2}{G}{U}", Green, Blue)
	})

	// Petrified Hamlet
	// Land
	// When this land enters, choose a land card name.
	// Activated abilities of sources with the chosen name can't be activated unless they're mana abilities.
	// Lands with the chosen name have "{T}: Add {C}."
	// {T}: Add {C}.
	Register("Petrified Hamlet", func() Card {
		// The chosen name is stored in ChosenSubtype (repurposed for land card names).
		//
		// XXX: The static ability "Activated abilities of sources with the chosen name can't be
		// activated unless they're mana abilities" requires the engine to gate ability activation
		// by card name — a form of ability-activation lock that the engine doesn't support
		// for arbitrary card names chosen at runtime.
		//
		// XXX: The static ability "Lands with the chosen name have '{T}: Add {C}'" would require
		// granting a new mana ability to all permanents matching the chosen name — a form of
		// dynamic GrantActivatedAbilityToAll keyed on a runtime-chosen string. The engine's
		// GrantActivatedAbilityToAll continuous effect uses a PermanentFilter, but there's no
		// mechanism to bind a Named(perm.ChosenSubtype) filter that closes over the source's
		// runtime state and survives the per-cycle reset correctly.
		//
		// The {T}: Add {C} basic mana ability and the ETB "choose a land card name" are implemented.
		chooseNameETB := ETBEffect(FuncEffect(
			"choose a land card name",
			EffectProperties{},
			func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
				perm := g.MutablePermanent(sourceID)
				if perm == nil {
					return nil
				}
				p := g.GetPlayer(controller)
				if p == nil {
					return nil
				}
				// Offer all registered card names so the test harness (and
				// interactive players) can select from the full pool. The harness
				// requires a non-empty options slice to consume a queued choice.
				chosen := p.ChooseString(RegisteredCardNames(), "choose a land card name")
				perm.ChosenSubtype = chosen
				return nil
			},
		))
		return NewLand("Petrified Hamlet",
			WithAbility(chooseNameETB),
			WithManaAbility(Colorless),
		)
	})

	// Shattered Sanctum
	// Land
	// This land enters tapped unless you control two or more other lands.
	// {T}: Add {W} or {B}.
	Register("Shattered Sanctum", func() Card {
		return slowLand("Shattered Sanctum", White, Black)
	})

	// Skycoach Waypoint
	// Land
	// {T}: Add {C}.
	// {3}, {T}: Target creature becomes prepared. (Only creatures with prepare spells can become prepared.)
	Register("Skycoach Waypoint", func() Card {
		preparedFilter := NewPermanentFilter("creature with a prepare spell",
			func(p *Permanent, g *Game) bool {
				return p != nil && p.Card != nil && HasPreparedSpell(p.Card)
			})
		return NewLand("Skycoach Waypoint",
			WithManaAbility(Colorless),
			WithActivatedAbility(
				FuncEffect("target creature becomes prepared",
					EffectProperties{},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						target := targets[0]
						perm := g.FindPermanent(target)
						if perm == nil {
							return nil
						}
						if !HasPreparedSpell(perm.Card) {
							return nil
						}
						g.SetPrepared(target, true)
						return nil
					},
				),
				ManaCostOf("{3}"),
				WithCost(Tap()),
				WithTarget(TargetCreature(preparedFilter)),
			),
		)
	})

	// Spectacle Summit
	// Land
	// This land enters tapped.
	// {T}: Add {U} or {R}.
	// {2}{U}{R}, {T}: Surveil 1. (Look at the top card of your library. You may put it into your graveyard.)
	Register("Spectacle Summit", func() Card {
		return surveilLand("Spectacle Summit", "{2}{U}{R}", Blue, Red)
	})

	// Stormcarved Coast
	// Land
	// This land enters tapped unless you control two or more other lands.
	// {T}: Add {U} or {R}.
	Register("Stormcarved Coast", func() Card {
		return slowLand("Stormcarved Coast", Blue, Red)
	})

	// Sundown Pass
	// Land
	// This land enters tapped unless you control two or more other lands.
	// {T}: Add {R} or {W}.
	Register("Sundown Pass", func() Card {
		return slowLand("Sundown Pass", Red, White)
	})

	// Terramorphic Expanse
	// Land
	// {T}, Sacrifice this land: Search your library for a basic land card, put it onto the battlefield tapped, then shuffle.
	// Already registered in cards/jumpstart; this stub prevents duplicate-registration panic.
	// The jumpstart registration takes precedence (it's a reprint).

	// Titan's Grave
	// Land
	// This land enters tapped.
	// {T}: Add {B} or {G}.
	// {2}{B}{G}, {T}: Surveil 1. (Look at the top card of your library. You may put it into your graveyard.)
	Register("Titan's Grave", func() Card {
		return surveilLand("Titan's Grave", "{2}{B}{G}", Black, Green)
	})

}
