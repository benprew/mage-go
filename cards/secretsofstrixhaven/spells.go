package secretsofstrixhaven

import (
	"github.com/google/uuid"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage"
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// fightBetween implements CR 701.13 between two arbitrary permanents. Both must
// be creatures on the battlefield; damage is dealt simultaneously.
func fightBetween(g *Game, aID, bID uuid.UUID) {
	a := g.FindPermanent(aID)
	b := g.FindPermanent(bID)
	if a == nil || b == nil {
		return
	}
	if !a.HasType(TypeCreature) || !b.HasType(TypeCreature) {
		return
	}
	aPower := a.CurrentPower(g)
	bPower := b.CurrentPower(g)
	aCard := g.FindCardAnywhere(aID)
	bCard := g.FindCardAnywhere(bID)
	if aPower > 0 && (bCard == nil || !b.HasProtectionFrom(aCard)) {
		g.DealDamageToPermanent(b, aPower, aID)
	}
	if bPower > 0 && (aCard == nil || !a.HasProtectionFrom(bCard)) {
		g.DealDamageToPermanent(a, bPower, bID)
	}
	g.FireEvent(GameEvent{
		Type:     EvtFight,
		SourceID: aID,
		TargetID: bID,
	})
}

func init() {
	registerSpells()
}

func registerSpells() {

// Ajani's Response {4}{W}
// Instant
// This spell costs {3} less to cast if it targets a tapped creature.
// Destroy target creature.
	Register("Ajani's Response", func() Card {
		return NewInstant("Ajani's Response", "{4}{W}",
			NewTargetedSpell(TargetCreature(), DestroyTarget()),
			WithTargetConditionalCostReduction(3, func(g *Game, controller uuid.UUID, c Card, targets []uuid.UUID) bool {
				for _, id := range targets {
					perm := g.FindPermanent(id)
					if perm != nil && perm.Tapped {
						return true
					}
				}
				return false
			}),
		)
	})


// Ancestral Anger {R}
// Sorcery
// Target creature gains trample and gets +X/+0 until end of turn, where X is 1 plus the number of cards named Ancestral Anger in your graveyard.
// Draw a card.
	Register("Ancestral Anger", func() Card {
		ancestralAngerFilter := NewCardFilter("Ancestral Anger", func(c Card) bool {
			return c.Name() == "Ancestral Anger"
		})
		xVal := CountZone(ZoneGraveyard, SelectController(), ancestralAngerFilter)
		return NewSorcery("Ancestral Anger", "{R}",
			NewTargetedSpell(TargetCreature(), CompositeEffects(
				"gain trample, get +X/+0 where X=1+Ancestral Angers in GY, draw a card",
				GrantKeyword(Trample),
				FuncEffect("get +X/+0 where X is 1 plus Ancestral Angers in GY",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						x := 1 + xVal.Resolve(g, sourceID, controller, targets)
						perm := g.FindPermanent(targets[0])
						if perm == nil {
							return nil
						}
						g.AddContinuousEffect(TemporaryBoost(perm.ID(), x, 0))
						return nil
					},
				),
				FuncEffect("draw a card", EffectProperties{Outcome: OutcomeBenefit, DrawCount: 1},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p != nil {
							g.PlayerDrawCard(p)
						}
						return nil
					},
				),
			)),
		)
	})


// Antiquities on the Loose {1}{W}{W}
// Sorcery
// Create two 2/2 red and white Spirit creature tokens. Then if this spell was cast from anywhere other than your hand, put a +1/+1 counter on each Spirit you control.
// Flashback {4}{W}{W} (You may cast this card from your graveyard for its flashback cost. Then exile it.)
// XXX: Flashback does not exile the card after resolution (engine feature needed).
	Register("Antiquities on the Loose", func() Card {
		return NewSorcery("Antiquities on the Loose", "{1}{W}{W}",
			NewSpellAbility(
				CreateColoredToken("Spirit Token", 2, 2, []Color{Red, White}, []CardType{TypeCreature}, []string{"Spirit"}),
				CreateColoredToken("Spirit Token", 2, 2, []Color{Red, White}, []CardType{TypeCreature}, []string{"Spirit"}),
				FuncEffect("if cast from non-hand, put +1/+1 counter on each Spirit you control",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if g.ResolvingCastZone() == ZoneHand {
							return nil
						}
						spirits := g.FilterBattlefield(And(
							ControlledBy(controller),
							HasSubType("Spirit"),
						))
						for _, spirit := range spirits {
							g.AddCountersWithReplacement(spirit, P1P1, 1, sourceID, false)
						}
						return nil
					},
				),
			),
			WithAlternateCost(ZoneGraveyard, ParseManaCost("{4}{W}{W}")),
		)
	})


// Applied Geometry {2}{G}{U}
// Sorcery
// Create a token that's a copy of target non-Aura permanent you control, except it's a 0/0 Fractal creature in addition to its other types. Put six +1/+1 counters on it.
// XXX: CloneTarget creates a copy of a creature but does not add Fractal type or 0/0 base; full copy-with-modification requires engine support.
	Register("Applied Geometry", func() Card {
		return NewSorcery("Applied Geometry", "{2}{G}{U}",
			NewTargetedSpell(
				TargetControlledPermanent(),
				FuncEffect(
					"create token copy as 0/0 Fractal creature, put six +1/+1 counters on it",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						perm := g.FindPermanent(targets[0])
						if perm == nil {
							return nil
						}
						tokenCard := perm.Card.Copy()
						newPerm := g.PutOnBattlefield(tokenCard, controller)
						if newPerm == nil {
							return nil
						}
						newPerm.GrantBaseAttr(AttrIsCreature)
						newPerm.GrantBaseAttr(AttrHasPowerToughness)
						newPerm.GrantBaseAttr(AttrCanAttack)
						newPerm.GrantBaseAttr(AttrCanBlock)
						newPerm.GrantBaseAttr(AttrSummonSick)
						newPerm.BasePTOverride = &[2]int{0, 0}
						for i := 0; i < 6; i++ {
							g.AddCountersWithReplacement(newPerm, P1P1, 1, sourceID, false)
						}
						return nil
					},
				),
			),
		)
	})


// Arcane Omens {4}{B}
// Sorcery
// Converge — Target player discards X cards, where X is the number of colors of mana spent to cast this spell.
	Register("Arcane Omens", func() Card {
		return NewSorcery("Arcane Omens", "{4}{B}",
			NewTargetedSpell(
				TargetPlayer(),
				FuncEffect(
					"target player discards X cards where X is number of colors spent",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						target := g.GetPlayer(targets[0])
						if target == nil {
							return nil
						}
						ctx := g.ResolvingCastContext()
						x := 1
						if ctx != nil {
							x = ctx.DistinctColorsSpent()
						}
						chosen := target.ChooseCardsFromHand(x, "discard", g)
						for _, card := range chosen {
							g.PlayerDiscard(target, card.ID())
						}
						return nil
					},
				),
			),
		)
	})


// Archaic's Agony {4}{R}
// Sorcery
// Converge — Archaic's Agony deals X damage to target creature, where X is the number of colors of mana spent to cast this spell. Exile cards from the top of your library equal to the excess damage dealt to that creature this way. You may play those cards until the end of your next turn.
// XXX: "Exile cards equal to excess damage, may play until end of next turn" requires exile-with-play-permission, not yet supported.
	Register("Archaic's Agony", func() Card {
		return NewSorcery("Archaic's Agony", "{4}{R}",
			NewTargetedSpell(
				TargetCreature(),
				FuncEffect(
					"deal X damage to target creature where X is colors spent",
					EffectProperties{Outcome: OutcomeDetriment, DamageValue: Fixed(1)},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						perm := g.FindPermanent(targets[0])
						if perm == nil {
							return nil
						}
						ctx := g.ResolvingCastContext()
						x := 1
						if ctx != nil {
							x = ctx.DistinctColorsSpent()
						}
						g.DealDamageToPermanent(perm, x, sourceID)
						// XXX: excess damage exile+play-until-next-turn not implemented
						return nil
					},
				),
			),
		)
	})


// Artistic Process {3}{R}{R}
// Sorcery
// Choose one —
// • Artistic Process deals 6 damage to target creature.
// • Artistic Process deals 2 damage to each creature you don't control.
// • Create a 3/3 blue and red Elemental creature token with flying. It gains haste until end of turn.
	Register("Artistic Process", func() Card {
		c := NewSorcery("Artistic Process", "{3}{R}{R}", nil)
		c.AddAbility(NewModalSpell([]Mode{
			{
				Label:   "Artistic Process deals 6 damage to target creature",
				Targets: []Target{TargetCreature()},
				Effects: []Effect{DealDamage(Fixed(6))},
			},
			{
				Label:   "Artistic Process deals 2 damage to each creature you don't control",
				Targets: []Target{},
				Effects: []Effect{FuncEffect(
					"deal 2 damage to each creature you don't control",
					EffectProperties{Outcome: OutcomeDetriment, Mass: true},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						creatures := g.FilterBattlefield(And(IsCreature, NotControlledBy(controller)))
						for _, cr := range creatures {
							g.DealDamageToPermanent(cr, 2, sourceID)
						}
						return nil
					},
				)},
			},
			{
				Label:   "Create a 3/3 blue and red Elemental creature token with flying and haste until end of turn",
				Targets: []Target{},
				Effects: []Effect{
					CreateColoredToken("Elemental Token", 3, 3, []Color{Blue, Red}, []CardType{TypeCreature}, []string{"Elemental"}, Flying),
					FuncEffect("grant haste to newly created elemental tokens",
						EffectProperties{Outcome: OutcomeBenefit},
						func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
							tokens := g.FilterBattlefield(And(ControlledBy(controller), HasSubType("Elemental"), IsCreature))
							for _, t := range tokens {
								g.AddContinuousEffect(TemporaryKeyword(t.ID(), Haste))
							}
							return nil
						},
					),
				},
			},
		}))
		return c
	})


// Banishing Betrayal {1}{U}
// Instant
// Return target nonland permanent to its owner's hand. Surveil 1. (Look at the top card of your library. You may put it into your graveyard.)
// XXX: Surveil 1 is not yet implemented in the engine (Surveil differs from Scry in that cards can go to the graveyard).
	Register("Banishing Betrayal", func() Card {
		return NewInstant("Banishing Betrayal", "{1}{U}",
			NewTargetedSpell(
				TargetPermanent(Not(IsLand)),
				ReturnToHandTarget(),
				// XXX: Surveil 1 not implemented
			),
		)
	})


// Borrowed Knowledge {2}{R}{W}
// Sorcery
// Choose one —
// • Discard your hand, then draw cards equal to the number of cards in target opponent's hand.
// • Discard your hand, then draw cards equal to the number of cards discarded this way.
	Register("Borrowed Knowledge", func() Card {
		c := NewSorcery("Borrowed Knowledge", "{2}{R}{W}", nil)
		c.AddAbility(NewModalSpell([]Mode{
			{
				Label:   "Discard your hand, then draw cards equal to the number of cards in target opponent's hand",
				Targets: []Target{TargetOpponent()},
				Effects: []Effect{FuncEffect(
					"discard hand then draw equal to opponent hand size",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						var opp Player
						if len(targets) > 0 {
							opp = g.GetPlayer(targets[0])
						}
						if opp == nil {
							opp = g.GetOpponent(controller)
						}
						drawCount := 0
						if opp != nil {
							drawCount = len(opp.Hand())
						}
						hand := append([]Card(nil), p.Hand()...)
						for _, c := range hand {
							g.PlayerDiscard(p, c.ID())
						}
						for i := 0; i < drawCount; i++ {
							g.PlayerDrawCard(p)
						}
						return nil
					},
				)},
			},
			{
				Label:   "Discard your hand, then draw cards equal to the number of cards discarded this way",
				Targets: []Target{},
				Effects: []Effect{FuncEffect(
					"discard hand then draw equal to discarded count",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						hand := append([]Card(nil), p.Hand()...)
						discarded := 0
						for _, c := range hand {
							if _, ok := g.PlayerDiscard(p, c.ID()); ok {
								discarded++
							}
						}
						for i := 0; i < discarded; i++ {
							g.PlayerDrawCard(p)
						}
						return nil
					},
				)},
			},
		}))
		return c
	})


// Brush Off {2}{U}{U}
// Instant
// This spell costs {1}{U} less to cast if it targets an instant or sorcery spell.
// Counter target spell.
// XXX: This spell costs {1}{U} less — the {1}{U} includes a colored pip; only the generic {1} portion
// can be reduced by WithTargetConditionalCostReduction (engine only reduces generic mana).
	Register("Brush Off", func() Card {
		return NewInstant("Brush Off", "{2}{U}{U}",
			NewTargetedSpell(TargetSpellOnStack(), CounterSpell()),
			// XXX: full {1}{U} reduction not possible — only generic reduction supported; reducing {1} only
			WithTargetConditionalCostReduction(1, func(g *Game, controller uuid.UUID, c Card, targets []uuid.UUID) bool {
				for _, id := range targets {
					so := g.FindStackObject(id)
					if so == nil {
						continue
					}
					if so.Card.HasType(TypeInstant) || so.Card.HasType(TypeSorcery) {
						return true
					}
				}
				return false
			}),
		)
	})


// Burrog Barrage {1}{G}
// Instant
// Target creature you control gets +1/+0 until end of turn if you've cast another instant or sorcery spell this turn. Then it deals damage equal to its power to up to one target creature an opponent controls.
// XXX: "another instant or sorcery spell" — engine only tracks instants cast this turn via GetInstantsCastThisTurn;
// sorcery cast tracking is not available. Partial implementation: checks for another instant cast this turn.
	Register("Burrog Barrage", func() Card {
		return NewInstant("Burrog Barrage", "{1}{G}",
			NewMultiTargetSpell(
				[]Target{
					TargetCreatureYouControl(),
					TargetCreatureOpponentControls(),
				},
				FuncEffect(
					"get +1/+0 if another instant cast this turn, then deal damage equal to power to opponent creature",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						yourCreature := g.FindPermanent(targets[0])
						if yourCreature == nil {
							return nil
						}
						// +1/+0 if another instant or sorcery was cast this turn
						// XXX: sorceries not tracked; only checking instants
						if g.GetInstantsCastThisTurn(controller) > 1 {
							g.AddContinuousEffect(TemporaryBoost(yourCreature.ID(), 1, 0))
						}
						// deal damage equal to power to opponent creature (up to one)
						// re-query power after any boost was registered
						g.ApplyContinuousEffects()
						if len(targets) >= 2 && targets[1] != uuid.Nil {
							oppCreature := g.FindPermanent(targets[1])
							if oppCreature != nil {
								power := yourCreature.CurrentPower(g)
								g.DealDamageToPermanent(oppCreature, power, sourceID)
							}
						}
						return nil
					},
				),
			),
		)
	})


// Chase Inspiration {U}
// Instant
// Target creature you control gets +0/+3 and gains hexproof until end of turn. (It can't be the target of spells or abilities your opponents control.)
	Register("Chase Inspiration", func() Card {
		return NewInstant("Chase Inspiration", "{U}",
			NewTargetedSpell(TargetControlledCreature(), CompositeEffects(
				"+0/+3 and hexproof until end of turn",
				Boost(Fixed(0), Fixed(3)),
				GrantKeyword(Hexproof),
			)),
		)
	})


// Chelonian Tackle {2}{G}
// Sorcery
// Target creature you control gets +0/+10 until end of turn. Then it fights up to one target creature an opponent controls. (Each deals damage equal to its power to the other.)
	Register("Chelonian Tackle", func() Card {
		return NewSorcery("Chelonian Tackle", "{2}{G}",
			NewMultiTargetSpell(
				[]Target{
					TargetCreatureYouControl(),
					TargetCreatureOpponentControls(),
				},
				FuncEffect(
					"+0/+10 until end of turn, then fight opponent creature",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						yourCreature := g.FindPermanent(targets[0])
						if yourCreature == nil {
							return nil
						}
						g.AddContinuousEffect(TemporaryBoost(yourCreature.ID(), 0, 10))
						g.ApplyContinuousEffects()
						if len(targets) >= 2 && targets[1] != uuid.Nil {
							fightBetween(g, targets[0], targets[1])
						}
						return nil
					},
				),
			),
		)
	})


// Choreographed Sparks {R}{R}
// Instant
// This spell can't be copied.
// Choose one or both —
// • Copy target instant or sorcery spell you control. You may choose new targets for the copy.
// • Copy target creature spell you control. The copy gains haste and "At the beginning of the end step, sacrifice this token."
// XXX: "This spell can't be copied" restriction is not implemented (engine cannot mark spells as uncopyable).
// XXX: "Choose one or both" is not supported (engine only supports "choose one" modal spells). Implements mode 1 only.
// XXX: Mode 2 (copy creature spell, token gains haste + sac trigger) requires engine support for permanent spell copies entering the battlefield as tokens (CR 706.12). Not implemented.
	Register("Choreographed Sparks", func() Card {
		instantOrSorcery := NewCardFilter("instant or sorcery spell", func(c Card) bool {
			return c.HasType(TypeInstant) || c.HasType(TypeSorcery)
		})
		return NewInstant("Choreographed Sparks", "{R}{R}",
			NewTargetedSpell(
				TargetOwnSpellOnStack(instantOrSorcery),
				CopySpellOnStack(),
			),
		)
	})


// Cost of Brilliance {2}{B}
// Sorcery
// Target player draws two cards and loses 2 life. Put a +1/+1 counter on up to one target creature.
	Register("Cost of Brilliance", func() Card {
		return NewSorcery("Cost of Brilliance", "{2}{B}",
			NewMultiTargetSpell(
				[]Target{
					TargetPlayer(),
					TargetUpToOneCreature(),
				},
				FuncEffect(
					"target player draws 2 and loses 2 life; put +1/+1 on up to one creature",
					EffectProperties{Outcome: OutcomeUnknown},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						targetPlayer := g.GetPlayer(targets[0])
						if targetPlayer != nil {
							g.PlayerDrawCard(targetPlayer)
							g.PlayerDrawCard(targetPlayer)
							targetPlayer.LoseLife(2)
						}
						if len(targets) >= 2 && targets[1] != uuid.Nil {
							perm := g.FindPermanent(targets[1])
							if perm != nil {
								g.AddCountersWithReplacement(perm, P1P1, 1, sourceID, false)
							}
						}
						return nil
					},
				),
			),
		)
	})


// Daydream {W}
// Sorcery
// Exile target creature you control, then return that card to the battlefield under its owner's control with a +1/+1 counter on it.
// Flashback {2}{W} (You may cast this card from your graveyard for its flashback cost. Then exile it.)
// XXX: Flashback does not exile the card after resolution (engine feature needed).
	Register("Daydream", func() Card {
		return NewSorcery("Daydream", "{W}",
			NewTargetedSpell(
				TargetControlledCreature(),
				FuncEffect(
					"exile target creature you control then return it with a +1/+1 counter",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						perm := g.FindPermanent(targets[0])
						if perm == nil {
							return nil
						}
						card := perm.Card
						owner := card.Owner()
						if owner == uuid.Nil {
							owner = perm.Controller
						}
						g.ExilePermanent(perm)
						newPerm := g.PutOnBattlefield(card, owner)
						if newPerm != nil {
							g.AddCountersWithReplacement(newPerm, P1P1, 1, sourceID, false)
						}
						return nil
					},
				),
			),
			WithAlternateCost(ZoneGraveyard, ParseManaCost("{2}{W}")),
		)
	})


// Decorum Dissertation {3}{B}{B}
// Sorcery — Lesson
// Target player draws two cards and loses 2 life.
// Paradigm (Then exile this spell. After you first resolve a spell with this name, you may cast a copy of it from exile without paying its mana cost at the beginning of each of your first main phases.)
// TODO: implement
	Register("Decorum Dissertation", func() Card {
		return NewSorcery("Decorum Dissertation", "{3}{B}{B}",
			NewSpellAbility(),
		)
	})


// Dig Site Inventory {W}
// Sorcery
// Put a +1/+1 counter on target creature you control. It gains vigilance until end of turn.
// Flashback {W} (You may cast this card from your graveyard for its flashback cost. Then exile it.)
// XXX: Flashback does not exile the card after resolution (engine feature needed).
	Register("Dig Site Inventory", func() Card {
		return NewSorcery("Dig Site Inventory", "{W}",
			NewTargetedSpell(
				TargetControlledCreature(),
				CompositeEffects(
					"put +1/+1 counter and gain vigilance until end of turn",
					AddCounters(P1P1, Fixed(1)),
					GrantKeyword(Vigilance),
				),
			),
			WithAlternateCost(ZoneGraveyard, ParseManaCost("{W}")),
		)
	})


// Dina's Guidance {1}{B}{G}
// Instant
// Search your library for a creature card, reveal it, put it into your hand or graveyard, then shuffle.
	Register("Dina's Guidance", func() Card {
		return NewInstant("Dina's Guidance", "{1}{B}{G}",
			NewSpellAbility(
				FuncEffect(
					"search library for creature card, put in hand or graveyard, then shuffle",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						lib := p.Library()
						if len(lib) == 0 {
							return nil
						}
						var candidates []Card
						for _, c := range lib {
							if c.HasType(TypeCreature) {
								candidates = append(candidates, c)
							}
						}
						if len(candidates) == 0 {
							p.ShuffleLibrary()
							return nil
						}
						chosen := p.ChooseCardFromLibrary(candidates, "search for creature", g)
						if chosen == nil {
							p.ShuffleLibrary()
							return nil
						}
						newLib := make([]Card, 0, len(lib)-1)
						for _, c := range lib {
							if c.ID() != chosen.ID() {
								newLib = append(newLib, c)
							}
						}
						p.SetLibrary(newLib)
						p.ShuffleLibrary()
						mode := p.ChooseMode([]string{"put in hand", "put in graveyard"}, "put into hand or graveyard?")
						if mode == 0 {
							p.AddToHand(chosen)
						} else {
							p.AddToGraveyard(chosen)
						}
						return nil
					},
				),
			),
		)
	})


// Dissection Practice {B}
// Instant
// Target opponent loses 1 life and you gain 1 life.
// Up to one target creature gets +1/+1 until end of turn.
// Up to one target creature gets -1/-1 until end of turn.
	Register("Dissection Practice", func() Card {
		return NewInstant("Dissection Practice", "{B}",
			NewMultiTargetSpell(
				[]Target{
					TargetOpponent(),
					TargetUpToOneCreature(),
					TargetUpToOneCreature(),
				},
				FuncEffect(
					"target opponent loses 1 life, you gain 1 life; up to one creature gets +1/+1; up to one creature gets -1/-1 until end of turn",
					EffectProperties{Outcome: OutcomeUnknown},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						opp := g.GetPlayer(targets[0])
						if opp != nil {
							opp.LoseLife(1)
						}
						p := g.GetPlayer(controller)
						if p != nil {
							g.PlayerGainLife(p, 1)
						}
						if len(targets) >= 2 && targets[1] != uuid.Nil {
							perm := g.FindPermanent(targets[1])
							if perm != nil {
								g.AddContinuousEffect(TemporaryBoost(perm.ID(), 1, 1))
							}
						}
						if len(targets) >= 3 && targets[2] != uuid.Nil {
							perm := g.FindPermanent(targets[2])
							if perm != nil {
								g.AddContinuousEffect(TemporaryBoost(perm.ID(), -1, -1))
							}
						}
						return nil
					},
				),
			),
		)
	})


// Divergent Equation {X}{X}{U}
// Instant
// Return up to X target instant and/or sorcery cards from your graveyard to your hand.
// Exile Divergent Equation.
	Register("Divergent Equation", func() Card {
		return NewInstant("Divergent Equation", "{X}{X}{U}",
			NewMultiTargetSpell(
				[]Target{
					TargetAnyNumberOfCardsInYourGraveyard(IsInstantOrSorceryCard),
				},
				FuncEffect(
					"return up to X instant and/or sorcery cards from graveyard to hand; exile self",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						x := g.XValue()
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						count := 0
						for _, tid := range targets {
							if tid == uuid.Nil {
								continue
							}
							if count >= x {
								break
							}
							card, ok := p.RemoveFromGraveyard(tid)
							if ok {
								p.AddToHand(card)
								count++
							}
						}
						// Exile Divergent Equation instead of putting it in the graveyard.
						resCard := g.GetResolvingCard()
						if resCard != nil {
							g.AddExileIfWouldGoToGraveyardThisTurn(resCard.ID(), sourceID)
						}
						return nil
					},
				),
			),
		)
	})


// Duel Tactics {R}
// Sorcery
// Duel Tactics deals 1 damage to target creature. It can't block this turn.
// Flashback {1}{R} (You may cast this card from your graveyard for its flashback cost. Then exile it.)
// XXX: Flashback does not exile the card after resolution (engine feature needed).
// XXX: "It can't block this turn" — PreventBlockingUntilEndOfCombat only prevents blocking until end of combat,
// not the full turn. For a sorcery cast during main phase, blocking happens during the following combat step.
// PreventBlockingUntilEndOfCombat is used here as the closest available approximation.
	Register("Duel Tactics", func() Card {
		return NewSorcery("Duel Tactics", "{R}",
			NewTargetedSpell(
				TargetCreature(),
				CompositeEffects(
					"Duel Tactics deals 1 damage to target creature. It can't block this turn.",
					DealDamage(Fixed(1)),
					FuncEffect(
						"target creature can't block this turn",
						EffectProperties{Outcome: OutcomeDetriment},
						func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
							if len(targets) == 0 {
								return nil
							}
							perm := g.FindPermanent(targets[0])
							if perm == nil {
								return nil
							}
							// XXX: PreventBlockingUntilEndOfCombat is an approximation; should be "this turn"
							g.AddContinuousEffect(PreventBlockingUntilEndOfCombat(perm.ID()))
							return nil
						},
					),
				),
			),
			WithAlternateCost(ZoneGraveyard, ParseManaCost("{1}{R}")),
		)
	})


// Echocasting Symposium {4}{U}{U}
// Sorcery — Lesson
// Target player creates a token that's a copy of target creature you control.
// Paradigm (Then exile this spell. After you first resolve a spell with this name, you may cast a copy of it from exile without paying its mana cost at the beginning of each of your first main phases.)
// TODO: implement
	Register("Echocasting Symposium", func() Card {
		return NewSorcery("Echocasting Symposium", "{4}{U}{U}",
			NewSpellAbility(),
		)
	})


// Efflorescence {2}{G}
// Instant
// Put two +1/+1 counters on target creature.
// Infusion — If you gained life this turn, that creature also gains trample and indestructible until end of turn.
// TODO: implement
	Register("Efflorescence", func() Card {
		return NewInstant("Efflorescence", "{2}{G}",
			NewSpellAbility(),
		)
	})


// Embrace the Paradox {3}{G}{U}
// Instant
// Draw three cards. You may put a land card from your hand onto the battlefield tapped.
	Register("Embrace the Paradox", func() Card {
		return NewInstant("Embrace the Paradox", "{3}{G}{U}",
			NewSpellAbility(
				FuncEffect(
					"draw three cards; may put a land card from hand onto battlefield tapped",
					EffectProperties{Outcome: OutcomeBenefit, DrawCount: 3},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						for i := 0; i < 3; i++ {
							g.PlayerDrawCard(p)
						}
						if !p.ChooseMayAbility("put a land card from your hand onto the battlefield tapped") {
							return nil
						}
						// Pick a land card from hand
						landFilter := NewCardFilter("land card", func(c Card) bool {
							return c.HasType(TypeLand)
						})
						chosen := g.PickFromHand(p, p, landFilter, false, "put land onto battlefield tapped")
						if chosen == nil {
							return nil
						}
						// Remove from hand
						hand := p.Hand()
						newHand := make([]Card, 0, len(hand)-1)
						for _, c := range hand {
							if c.ID() != chosen.ID() {
								newHand = append(newHand, c)
							}
						}
						p.SetHand(newHand)
						perm := g.PutOnBattlefield(chosen, controller)
						if perm != nil {
							g.TapPermanent(perm)
						}
						return nil
					},
				),
			),
		)
	})


// End of the Hunt {1}{B}
// Sorcery
// Target opponent exiles a creature or planeswalker they control with the greatest mana value among creatures and planeswalkers they control.
	Register("End of the Hunt", func() Card {
		return NewSorcery("End of the Hunt", "{1}{B}",
			NewTargetedSpell(
				TargetOpponent(),
				FuncEffect(
					"target opponent exiles their creature or planeswalker with greatest mana value",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						oppID := targets[0]
						// Find all creatures and planeswalkers the opponent controls
						candidates := g.FilterBattlefield(And(
							ControlledBy(oppID),
							Or(IsCreature, IsPlaneswalker),
						))
						if len(candidates) == 0 {
							return nil
						}
						// Find greatest mana value
						maxMV := -1
						for _, perm := range candidates {
							mv := perm.Card.ManaCost().CMC()
							if mv > maxMV {
								maxMV = mv
							}
						}
						// Collect candidates with greatest MV
						var withMaxMV []*Permanent
						for _, perm := range candidates {
							if perm.Card.ManaCost().CMC() == maxMV {
								withMaxMV = append(withMaxMV, perm)
							}
						}
						// Opponent chooses one to exile
						opp := g.GetPlayer(oppID)
						if opp == nil {
							return nil
						}
						chosen := opp.ChoosePermanent(withMaxMV, "exile a creature or planeswalker with greatest mana value", g)
						if chosen == nil {
							return nil
						}
						g.ExilePermanent(chosen)
						return nil
					},
				),
			),
		)
	})


// Erode {W}
// Instant
// Destroy target creature or planeswalker. Its controller may search their library for a basic land card, put it onto the battlefield tapped, then shuffle.
	Register("Erode", func() Card {
		return NewInstant("Erode", "{W}",
			NewTargetedSpell(
				TargetPermanent(Or(IsCreature, IsPlaneswalker)),
				FuncEffect(
					"destroy target creature or planeswalker; its controller may search library for basic land, put it onto battlefield tapped",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						perm := g.FindPermanent(targets[0])
						if perm == nil {
							return nil
						}
						permController := perm.Controller
						g.DestroyPermanent(perm)
						// Its controller may search for a basic land
						p := g.GetPlayer(permController)
						if p == nil {
							return nil
						}
						if p.ChooseMayAbility("search your library for a basic land card, put it onto the battlefield tapped") {
							searchBasicLandToBattlefieldTappedSOS(g, permController)
						}
						return nil
					},
				),
			),
		)
	})


// Essence Scatter {1}{U}
// Instant
// Counter target creature spell.
	Register("Essence Scatter", func() Card {
		creatureSpellFilter := NewCardFilter("creature spell", func(c Card) bool {
			return c.HasType(TypeCreature)
		})
		return NewInstant("Essence Scatter", "{1}{U}",
			NewTargetedSpell(TargetSpellOnStack(creatureSpellFilter), CounterSpell()),
		)
	})


// Fix What's Broken {2}{W}{B}
// Sorcery
// As an additional cost to cast this spell, pay X life.
// Return each artifact and creature card with mana value X from your graveyard to the battlefield.
// XXX: "Pay X life" as an additional cost that also sets X is not supported by the engine.
// The implementation reads X from g.XValue() (set by the test harness or cast pipeline).
// A proper XLifePayCost that requests an amount from the player and sets g.currentX is needed.
	Register("Fix What's Broken", func() Card {
		return NewSorcery("Fix What's Broken", "{2}{W}{B}",
			NewSpellAbility(
				FuncEffect(
					"return each artifact and creature card with mana value X from graveyard to battlefield",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						x := g.XValue()
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						// Pay X life (additional cost already set by the time we resolve)
						if x > 0 {
							g.PlayerLoseLife(p, x)
						}
						gy := p.Graveyard()
						var toReturn []Card
						for _, c := range gy {
							if c.ManaCost().CMC() == x && (c.HasType(TypeArtifact) || c.HasType(TypeCreature)) {
								toReturn = append(toReturn, c)
							}
						}
						for _, c := range toReturn {
							p.RemoveFromGraveyard(c.ID())
							g.PutOnBattlefield(c, controller)
						}
						return nil
					},
				),
			),
		)
	})


// Flashback {R}
// Instant
// Target instant or sorcery card in your graveyard gains flashback until end of turn. The flashback cost is equal to its mana cost. (You may cast that card from your graveyard for its flashback cost. Then exile it.)
// XXX: Granting flashback (an alternate cost) to a card in the graveyard until end of turn
// requires dynamic alternate-cost injection, which the engine does not support.
// The card is implemented as a no-op: it targets an instant/sorcery in your graveyard
// but cannot actually grant it the flashback ability.
	Register("Flashback", func() Card {
		return NewInstant("Flashback", "{R}",
			NewTargetedSpell(
				TargetCardInYourGraveyard(IsInstantOrSorceryCard),
				FuncEffect(
					"target instant or sorcery card in your graveyard gains flashback until end of turn",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						// XXX: Cannot grant flashback to the target; engine lacks dynamic alternate-cost injection.
						return nil
					},
				),
			),
		)
	})


// Flow State {1}{U}
// Sorcery
// Look at the top three cards of your library. Put one of them into your hand and the rest on the bottom of your library in any order. If there is an instant card and a sorcery card in your graveyard, instead put two of them into your hand and the rest on the bottom of your library in any order.
	Register("Flow State", func() Card {
		return NewSorcery("Flow State", "{1}{U}",
			NewSpellAbility(
				FuncEffect(
					"look at top 3; put 1 (or 2 if instant+sorcery in GY) into hand, rest on bottom in any order",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						topCards := g.RevealTopN(p, 3)
						if len(topCards) == 0 {
							return nil
						}
						// Check graveyard for both an instant and a sorcery card
						hasInstant, hasSorcery := false, false
						for _, c := range p.Graveyard() {
							if c.HasType(TypeInstant) {
								hasInstant = true
							}
							if c.HasType(TypeSorcery) {
								hasSorcery = true
							}
						}
						putToHand := 1
						if hasInstant && hasSorcery {
							putToHand = 2
						}
						// Remove top N from library
						removed := g.RemoveTopN(p, len(topCards))
						// Let player pick cards to put into hand
						var inHand []Card
						var remaining []Card
						for i := 0; i < putToHand && len(removed) > 0; i++ {
							anyFilter := NewCardFilter("any card", func(c Card) bool { return true })
							chosen := p.ChooseCardFromLibrary(removed, "put into your hand", g)
							_ = anyFilter
							if chosen == nil {
								break
							}
							inHand = append(inHand, chosen)
							newRemoved := make([]Card, 0, len(removed)-1)
							for _, c := range removed {
								if c.ID() != chosen.ID() {
									newRemoved = append(newRemoved, c)
								}
							}
							removed = newRemoved
						}
						for _, c := range inHand {
							p.AddToHand(c)
						}
						remaining = removed
						g.PutOnBottomInChosenOrder(p, remaining)
						return nil
					},
				),
			),
		)
	})


// Follow the Lumarets {1}{G}
// Sorcery
// Infusion — Look at the top four cards of your library. You may reveal a creature or land card from among them and put it into your hand. If you gained life this turn, you may instead reveal two creature and/or land cards from among them and put them into your hand. Put the rest on the bottom of your library in a random order.
// TODO: implement
	Register("Follow the Lumarets", func() Card {
		return NewSorcery("Follow the Lumarets", "{1}{G}",
			NewSpellAbility(),
		)
	})


// Foolish Fate {2}{B}
// Instant
// Destroy target creature.
// Infusion — If you gained life this turn, that creature's controller loses 3 life.
// TODO: implement
	Register("Foolish Fate", func() Card {
		return NewInstant("Foolish Fate", "{2}{B}",
			NewSpellAbility(),
		)
	})


// Fractal Anomaly {U}
// Instant
// Create a 0/0 green and blue Fractal creature token and put X +1/+1 counters on it, where X is the number of cards you've drawn this turn.
	Register("Fractal Anomaly", func() Card {
		return NewInstant("Fractal Anomaly", "{U}",
			NewSpellAbility(
				FuncEffect(
					"create 0/0 green and blue Fractal token; put X +1/+1 counters where X is cards drawn this turn",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						// Create 0/0 Fractal token via CreateColoredToken then find it to add counters
						eff := CreateColoredToken("Fractal Token", 0, 0, []Color{Green, Blue}, []CardType{TypeCreature}, []string{"Fractal"})
						_ = eff.Apply(g, sourceID, controller, nil)
						// Find the newly created token to add counters
						tokens := g.FilterBattlefield(And(
							ControlledBy(controller),
							HasSubType("Fractal"),
						))
						if len(tokens) == 0 {
							return nil
						}
						newest := tokens[len(tokens)-1]
						x := g.PlayerCardsDrawnThisTurn(controller)
						for i := 0; i < x; i++ {
							g.AddCountersWithReplacement(newest, P1P1, 1, sourceID, false)
						}
						return nil
					},
				),
			),
		)
	})


// Fractalize {X}{U}
// Instant
// Until end of turn, target creature becomes a green and blue Fractal with base power and toughness each equal to X plus 1. (It loses all other colors and creature types.)
// TODO: implement
	Register("Fractalize", func() Card {
		return NewInstant("Fractalize", "{X}{U}",
			NewSpellAbility(),
		)
	})


// Germination Practicum {3}{G}{G}
// Sorcery — Lesson
// Put two +1/+1 counters on each creature you control.
// Paradigm (Then exile this spell. After you first resolve a spell with this name, you may cast a copy of it from exile without paying its mana cost at the beginning of each of your first main phases.)
// TODO: implement
	Register("Germination Practicum", func() Card {
		return NewSorcery("Germination Practicum", "{3}{G}{G}",
			NewSpellAbility(),
		)
	})


// Glorious Decay {1}{G}
// Instant
// Choose one —
// • Destroy target artifact.
// • Glorious Decay deals 4 damage to target creature with flying.
// • Exile target card from a graveyard. Draw a card.
// TODO: implement
	Register("Glorious Decay", func() Card {
		return NewInstant("Glorious Decay", "{1}{G}",
			NewSpellAbility(),
		)
	})


// Grapple with Death {1}{B}{G}
// Sorcery
// Destroy target artifact or creature. You gain 1 life.
// TODO: implement
	Register("Grapple with Death", func() Card {
		return NewSorcery("Grapple with Death", "{1}{B}{G}",
			NewSpellAbility(),
		)
	})


// Group Project {1}{W}
// Sorcery
// Create a 2/2 red and white Spirit creature token.
// Flashback—Tap three untapped creatures you control. (You may cast this card from your graveyard for its flashback cost. Then exile it.)
// TODO: implement
	Register("Group Project", func() Card {
		return NewSorcery("Group Project", "{1}{W}",
			NewSpellAbility(),
		)
	})


// Growth Curve {G}{U}
// Sorcery
// Put a +1/+1 counter on target creature you control, then double the number of +1/+1 counters on that creature.
// TODO: implement
	Register("Growth Curve", func() Card {
		return NewSorcery("Growth Curve", "{G}{U}",
			NewSpellAbility(),
		)
	})


// Harsh Annotation {1}{W}
// Instant
// Destroy target creature. Its controller creates a 1/1 white and black Inkling creature token with flying.
// TODO: implement
	Register("Harsh Annotation", func() Card {
		return NewInstant("Harsh Annotation", "{1}{W}",
			NewSpellAbility(),
		)
	})


// Heated Argument {4}{R}
// Instant
// Heated Argument deals 6 damage to target creature. You may exile a card from your graveyard. If you do, Heated Argument also deals 2 damage to that creature's controller.
// TODO: implement
	Register("Heated Argument", func() Card {
		return NewInstant("Heated Argument", "{4}{R}",
			NewSpellAbility(),
		)
	})


// Homesickness {4}{U}{U}
// Instant
// Target player draws two cards. Tap up to two target creatures. Put a stun counter on each of them. (If a permanent with a stun counter would become untapped, remove one from it instead.)
// XXX: Stun counters not implemented (no CounterType for Stun). Draw and tap effects implemented.
	Register("Homesickness", func() Card {
		return NewInstant("Homesickness", "{4}{U}{U}",
			NewMultiTargetSpell(
				[]Target{TargetPlayer(), TargetUpToNCreatures(2)},
				FuncEffect(
					"target player draws two cards; tap up to two target creatures",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						if pid := targets[0]; pid != uuid.Nil {
							p := g.GetPlayer(pid)
							if p != nil {
								g.PlayerDrawCard(p)
								g.PlayerDrawCard(p)
							}
						}
						for _, tid := range targets[1:] {
							if tid == uuid.Nil {
								continue
							}
							perm := g.FindPermanent(tid)
							if perm != nil {
								g.TapPermanent(perm)
								// XXX: stun counter not placed — engine lacks Stun CounterType
							}
						}
						return nil
					},
				),
			),
		)
	})


// Impractical Joke {R}
// Sorcery
// Damage can't be prevented this turn. Impractical Joke deals 3 damage to up to one target creature or planeswalker.
// XXX: "Damage can't be prevented this turn" requires a global prevention-suppression flag, which the engine
// does not support. The damage effect is implemented; prevention suppression is not.
	Register("Impractical Joke", func() Card {
		return NewSorcery("Impractical Joke", "{R}",
			NewMultiTargetSpell(
				[]Target{TargetUpToNCreaturesOrPlayers(1)},
				FuncEffect(
					"Impractical Joke deals 3 damage to up to one target creature or planeswalker",
					EffectProperties{Outcome: OutcomeDetriment, DamageValue: Fixed(3)},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						// XXX: global damage prevention suppression not implemented
						for _, tid := range targets {
							if tid == uuid.Nil {
								continue
							}
							perm := g.FindPermanent(tid)
							if perm != nil {
								g.DealDamageToPermanent(perm, 3, sourceID)
								continue
							}
							p := g.GetPlayer(tid)
							if p != nil {
								g.DealDamageToPlayer(p, 3, sourceID)
							}
						}
						return nil
					},
				),
			),
		)
	})


// Improvisation Capstone {5}{R}{R}
// Sorcery — Lesson
// Exile cards from the top of your library until you exile cards with total mana value 4 or greater. You may cast any number of spells from among them without paying their mana costs.
// Paradigm (Then exile this spell. After you first resolve a spell with this name, you may cast a copy of it from exile without paying its mana cost at the beginning of each of your first main phases.)
// TODO: implement
	Register("Improvisation Capstone", func() Card {
		return NewSorcery("Improvisation Capstone", "{5}{R}{R}",
			NewSpellAbility(),
		)
	})


// Interjection {W}
// Instant
// Target creature gets +2/+2 and gains first strike until end of turn.
	Register("Interjection", func() Card {
		return NewInstant("Interjection", "{W}",
			NewTargetedSpell(
				TargetCreature(),
				CompositeEffects("target creature gets +2/+2 and gains first strike until end of turn",
					Boost(Fixed(2), Fixed(2)),
					GrantKeyword(FirstStrike),
				),
			),
		)
	})


// Killian's Confidence {W}{B}
// Sorcery
// Target creature gets +1/+1 until end of turn. Draw a card.
// Whenever one or more creatures you control deal combat damage to a player, you may pay {W/B}. If you do, return this card from your graveyard to your hand.
// TODO: implement
	Register("Killian's Confidence", func() Card {
		return NewSorcery("Killian's Confidence", "{W}{B}",
			NewSpellAbility(),
		)
	})


// Last Gasp {1}{B}
// Instant
// Target creature gets -3/-3 until end of turn.
// TODO: implement
	Register("Last Gasp", func() Card {
		return NewInstant("Last Gasp", "{1}{B}",
			NewSpellAbility(),
		)
	})


// Lorehold Charm {R}{W}
// Instant
// Choose one —
// • Each opponent sacrifices a nontoken artifact of their choice.
// • Return target artifact or creature card with mana value 2 or less from your graveyard to the battlefield.
// • Creatures you control get +1/+1 and gain trample until end of turn.
// TODO: implement
	Register("Lorehold Charm", func() Card {
		return NewInstant("Lorehold Charm", "{R}{W}",
			NewSpellAbility(),
		)
	})


// Lumaret's Favor {1}{G}
// Instant
// Infusion — When you cast this spell, copy it if you gained life this turn. You may choose new targets for the copy.
// Target creature gets +2/+4 until end of turn.
// TODO: implement
	Register("Lumaret's Favor", func() Card {
		return NewInstant("Lumaret's Favor", "{1}{G}",
			NewSpellAbility(),
		)
	})


// Mana Sculpt {1}{U}{U}
// Instant
// Counter target spell. If you control a Wizard, add an amount of {C} equal to the amount of mana spent to cast that spell at the beginning of your next main phase.
// TODO: implement
	Register("Mana Sculpt", func() Card {
		return NewInstant("Mana Sculpt", "{1}{U}{U}",
			NewSpellAbility(),
		)
	})


// Masterful Flourish {B}
// Instant
// Target creature you control gets +1/+0 and gains indestructible until end of turn. (Damage and effects that say "destroy" don't destroy it.)
// TODO: implement
	Register("Masterful Flourish", func() Card {
		return NewInstant("Masterful Flourish", "{B}",
			NewSpellAbility(),
		)
	})


// Mathemagics {X}{X}{U}{U}
// Sorcery
// Target player draws 2ˣ cards. (2⁰ = 1, 2¹ = 2, 2² = 4, 2³ = 8, 2⁴ = 16, 2⁵ = 32, and so on.)
	Register("Mathemagics", func() Card {
		return NewSorcery("Mathemagics", "{X}{X}{U}{U}",
			NewTargetedSpell(TargetPlayer(),
				FuncEffect("target player draws 2^X cards",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						p := g.GetPlayer(targets[0])
						if p == nil {
							return nil
						}
						x := g.XValue()
						count := 1 << x // 2^X
						for i := 0; i < count; i++ {
							g.PlayerDrawCard(p)
						}
						return nil
					},
				),
			),
		)
	})


// Mind Roots {1}{B}{G}
// Sorcery
// Target player discards two cards. Put up to one land card discarded this way onto the battlefield tapped under your control.
// TODO: implement
	Register("Mind Roots", func() Card {
		return NewSorcery("Mind Roots", "{1}{B}{G}",
			NewSpellAbility(),
		)
	})


// Mind into Matter {X}{G}{U}
// Sorcery
// Draw X cards. Then you may put a permanent card with mana value X or less from your hand onto the battlefield tapped.
	Register("Mind into Matter", func() Card {
		isPermanentCard := NewCardFilter("permanent card", func(c Card) bool {
			for _, t := range c.Types() {
				switch t {
				case TypeCreature, TypeArtifact, TypeEnchantment, TypeLand, TypePlaneswalker:
					return true
				}
			}
			return false
		})
		return NewSorcery("Mind into Matter", "{X}{G}{U}",
			NewSpellAbility(
				FuncEffect(
					"draw X cards, then may put a permanent card with mana value X or less from hand onto battlefield tapped",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						x := g.XValue()
						for i := 0; i < x; i++ {
							g.PlayerDrawCard(p)
						}
						// Collect eligible permanent cards from hand with mana value <= X.
						var cands []Card
						for _, c := range p.Hand() {
							if isPermanentCard.Match(c) && c.ManaCost().CMC() <= x {
								cands = append(cands, c)
							}
						}
						if len(cands) == 0 {
							return nil
						}
						if !p.ChooseMayAbility("put a permanent card with mana value X or less onto the battlefield tapped") {
							return nil
						}
						chosen := p.ChooseCardFromHand(cands, "put permanent card onto battlefield tapped", g)
						if chosen == nil {
							return nil
						}
						// Remove chosen card from hand.
						hand := p.Hand()
						newHand := make([]Card, 0, len(hand))
						for _, c := range hand {
							if c.ID() != chosen.ID() {
								newHand = append(newHand, c)
							}
						}
						p.SetHand(newHand)
						perm := g.PutOnBattlefield(chosen, controller)
						if perm != nil {
							g.TapPermanent(perm)
						}
						return nil
					},
				),
			),
		)
	})


// Molten Note {X}{R}{W}
// Sorcery
// Molten Note deals damage to target creature equal to the amount of mana spent to cast this spell. Untap all creatures you control.
// Flashback {6}{R}{W} (You may cast this card from your graveyard for its flashback cost. Then exile it.)
// TODO: implement
	Register("Molten Note", func() Card {
		return NewSorcery("Molten Note", "{X}{R}{W}",
			NewSpellAbility(),
		)
	})


// Moment of Reckoning {3}{W}{W}{B}{B}
// Sorcery
// Choose up to four. You may choose the same mode more than once.
// • Destroy target nonland permanent.
// • Return target nonland permanent card from your graveyard to the battlefield.
// TODO: implement
	Register("Moment of Reckoning", func() Card {
		return NewSorcery("Moment of Reckoning", "{3}{W}{W}{B}{B}",
			NewSpellAbility(),
		)
	})


// Muse's Encouragement {4}{U}
// Instant
// Create a 3/3 blue and red Elemental creature token with flying.
// Surveil 2. (Look at the top two cards of your library, then put any number of them into your graveyard and the rest on top of your library in any order.)
// TODO: implement
	Register("Muse's Encouragement", func() Card {
		return NewInstant("Muse's Encouragement", "{4}{U}",
			NewSpellAbility(),
		)
	})


// Oracle's Restoration {G}
// Sorcery
// Target creature you control gets +1/+1 until end of turn. You draw a card and gain 1 life.
// TODO: implement
	Register("Oracle's Restoration", func() Card {
		return NewSorcery("Oracle's Restoration", "{G}",
			NewSpellAbility(),
		)
	})


// Planar Engineering {3}{G}
// Sorcery
// Sacrifice two lands. Search your library for four basic land cards, put them onto the battlefield tapped, then shuffle.
// TODO: implement
	Register("Planar Engineering", func() Card {
		return NewSorcery("Planar Engineering", "{3}{G}",
			NewSpellAbility(),
		)
	})


// Pox Plague {B}{B}{B}{B}{B}
// Sorcery
// Each player loses half their life, then discards half the cards in their hand, then sacrifices half the permanents they control of their choice. Round down each time.
	Register("Pox Plague", func() Card {
		return NewSorcery("Pox Plague", "{B}{B}{B}{B}{B}",
			NewSpellAbility(
				FuncEffect("each player loses half their life, then discards half their hand, then sacrifices half their permanents of their choice (round down each time)",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						// Each player loses half their life (round down).
						for _, pl := range g.AllPlayers() {
							lose := pl.Life() / 2
							if lose > 0 {
								g.PlayerLoseLife(pl, lose)
							}
						}
						// Each player discards half the cards in their hand (round down).
						for _, pl := range g.AllPlayers() {
							count := len(pl.Hand()) / 2
							if count > 0 {
								chosen := pl.ChooseCardsFromHand(count, "discard half your hand", g)
								for _, c := range chosen {
									g.PlayerDiscard(pl, c.ID())
								}
							}
						}
						// Each player sacrifices half the permanents they control of their choice (round down).
						for _, pl := range g.AllPlayers() {
							var controlled []*Permanent
							for _, perm := range g.AllBattlefield() {
								if perm.Controller == pl.PlayerID() {
									controlled = append(controlled, perm)
								}
							}
							count := len(controlled) / 2
							for i := 0; i < count; i++ {
								// Rebuild candidates each iteration as prior sacrifices may have removed permanents.
								var candidates []*Permanent
								for _, perm := range g.AllBattlefield() {
									if perm.Controller == pl.PlayerID() {
										candidates = append(candidates, perm)
									}
								}
								if len(candidates) == 0 {
									break
								}
								chosen := pl.ChoosePermanent(candidates, "sacrifice a permanent", g)
								if chosen == nil {
									chosen = candidates[0]
								}
								g.Sacrifice(chosen)
							}
						}
						return nil
					}),
			),
		)
	})


// Practiced Offense {2}{W}
// Sorcery
// Put a +1/+1 counter on each creature target player controls. Target creature gains your choice of double strike or lifelink until end of turn.
// Flashback {1}{W} (You may cast this card from your graveyard for its flashback cost. Then exile it.)
// TODO: implement
	Register("Practiced Offense", func() Card {
		return NewSorcery("Practiced Offense", "{2}{W}",
			NewSpellAbility(),
		)
	})


// Prismari Charm {U}{R}
// Instant
// Choose one —
// • Surveil 2, then draw a card.
// • Prismari Charm deals 1 damage to each of one or two targets.
// • Return target nonland permanent to its owner's hand.
	Register("Prismari Charm", func() Card {
		c := NewInstant("Prismari Charm", "{U}{R}", nil)
		c.AddAbility(NewModalSpell([]Mode{
			{
				Label:   "Surveil 2, then draw a card",
				Targets: []Target{},
				Effects: []Effect{
					surveilEffect(2),
					FuncEffect("draw a card",
						EffectProperties{Outcome: OutcomeBenefit, DrawCount: 1},
						func(g *Game, _, controller uuid.UUID, _ []uuid.UUID) error {
							p := g.GetPlayer(controller)
							if p != nil {
								g.PlayerDrawCard(p)
							}
							return nil
						},
					),
				},
			},
			{
				Label:   "Prismari Charm deals 1 damage to each of one or two targets",
				Targets: []Target{TargetUpToNCreaturesOrPlayers(2)},
				Effects: []Effect{
					FuncEffect("deal 1 damage to each of the chosen targets",
						EffectProperties{Outcome: OutcomeDetriment, DamageValue: Fixed(1)},
						func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
							for _, tid := range targets {
								if tid == uuid.Nil {
									continue
								}
								perm := g.FindPermanent(tid)
								if perm != nil {
									g.DealDamageToPermanent(perm, 1, sourceID)
									continue
								}
								p := g.GetPlayer(tid)
								if p != nil {
									g.DealDamageToPlayer(p, 1, sourceID)
								}
							}
							return nil
						},
					),
				},
			},
			{
				Label:   "Return target nonland permanent to its owner's hand",
				Targets: []Target{TargetPermanent(Not(IsLand))},
				Effects: []Effect{ReturnToHandTarget()},
			},
		}))
		return c
	})


// Procrastinate {X}{U}
// Sorcery
// Tap target creature. Put twice X stun counters on it. (If a permanent with a stun counter would become untapped, remove one from it instead.)
// XXX: Stun counters are not implemented in the engine (no CounterType for Stun, no replacement effect to remove stun counters on untap). Implements tap-only.
	Register("Procrastinate", func() Card {
		return NewSorcery("Procrastinate", "{X}{U}",
			NewTargetedSpell(
				TargetCreature(),
				TapTarget(),
				// XXX: stun counters not implemented
			),
		)
	})


// Proctor's Gaze {2}{G}{U}
// Instant
// Return up to one target nonland permanent to its owner's hand. Search your library for a basic land card, put it onto the battlefield tapped, then shuffle.
// XXX: No TargetUpToOnePermanent exists; uses TargetUpToNCreatures(1) as approximation (only targets creatures).
	Register("Proctor's Gaze", func() Card {
		return NewInstant("Proctor's Gaze", "{2}{G}{U}",
			NewMultiTargetSpell(
				[]Target{TargetUpToNCreatures(1)},
				FuncEffect(
					"return up to one target nonland permanent to owner's hand; search for basic land, put it onto battlefield tapped",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						for _, tid := range targets {
							if tid == uuid.Nil {
								continue
							}
							perm := g.FindPermanent(tid)
							if perm == nil || perm.HasType(TypeLand) {
								continue
							}
							g.BouncePermanentToHand(perm)
						}
						searchBasicLandToBattlefieldTappedSOS(g, controller)
						return nil
					},
				),
			),
		)
	})


// Professor Dellian Fel {2}{B}{G}
// Legendary Planeswalker — Dellian
// +2: You gain 3 life.
// 0: You draw a card and lose 1 life.
// −3: Destroy target creature.
// −6: You get an emblem with "Whenever you gain life, target opponent loses that much life."
// XXX: Loyalty-activated abilities (+2/0/−3/−6) are not implemented (engine lacks loyalty ability support, CR 606).
// XXX: −6 emblem "Whenever you gain life, target opponent loses that much life" requires emblem support.
	Register("Professor Dellian Fel", func() Card {
		return NewPlaneswalker("Professor Dellian Fel", "{2}{B}{G}", 5,
			WithSuperTypes(SuperLegendary),
			WithSubTypes("Dellian"),
		)
	})


// Pull from the Grave {2}{B}
// Sorcery
// Return up to two target creature cards from your graveyard to your hand. You gain 2 life.
	Register("Pull from the Grave", func() Card {
		return NewSorcery("Pull from the Grave", "{2}{B}",
			NewMultiTargetSpell(
				[]Target{TargetUpToNCardsInYourGraveyard(2, IsCreatureCard)},
				ReturnFromGraveyardToHandTarget(),
				GainLife(2),
			),
		)
	})


// Pursue the Past {R}{W}
// Sorcery
// You gain 2 life. You may discard a card. If you do, draw two cards.
// Flashback {2}{R}{W} (You may cast this card from your graveyard for its flashback cost. Then exile it.)
// XXX: Flashback does not exile the card after resolution (engine feature needed).
	Register("Pursue the Past", func() Card {
		return NewSorcery("Pursue the Past", "{R}{W}",
			NewSpellAbility(
				GainLife(2),
				FuncEffect(
					"you may discard a card; if you do, draw two cards",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, _, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil || len(p.Hand()) == 0 {
							return nil
						}
						if !p.ChooseMayAbility("discard a card to draw two cards") {
							return nil
						}
						chosen := p.ChooseCardsFromHand(1, "discard a card", g)
						for _, c := range chosen {
							g.PlayerDiscard(p, c.ID())
						}
						g.PlayerDrawCard(p)
						g.PlayerDrawCard(p)
						return nil
					},
				),
			),
			WithAlternateCost(ZoneGraveyard, ParseManaCost("{2}{R}{W}")),
		)
	})


// Quandrix Charm {G}{U}
// Instant
// Choose one —
// • Counter target spell unless its controller pays {2}.
// • Destroy target enchantment.
// • Target creature has base power and toughness 5/5 until end of turn.
	Register("Quandrix Charm", func() Card {
		c := NewInstant("Quandrix Charm", "{G}{U}", nil)
		c.AddAbility(NewModalSpell([]Mode{
			{
				Label:   "Counter target spell unless its controller pays {2}",
				Targets: []Target{TargetSpellOnStack()},
				Effects: []Effect{CounterUnlessPay("{2}")},
			},
			{
				Label:   "Destroy target enchantment",
				Targets: []Target{TargetPermanent(IsEnchantment)},
				Effects: []Effect{DestroyTarget()},
			},
			{
				Label:   "Target creature has base power and toughness 5/5 until end of turn",
				Targets: []Target{TargetCreature()},
				Effects: []Effect{SetPTUntilEndOfTurn(5, 5, SelectTarget)},
			},
		}))
		return c
	})


// Quick Study {2}{U}
// Instant
// Draw two cards.
	Register("Quick Study", func() Card {
		return NewInstant("Quick Study", "{2}{U}",
			NewSpellAbility(DrawCards(Fixed(2))),
		)
	})


// Rabid Attack {1}{B}
// Instant
// Until end of turn, any number of target creatures you control each get +1/+0 and gain "When this creature dies, draw a card."
	Register("Rabid Attack", func() Card {
		return NewInstant("Rabid Attack", "{1}{B}",
			NewMultiTargetSpell(
				[]Target{TargetUpToNCreatures(100, IsCreature)},
				FuncEffect(
					"target creatures you control get +1/+0 and gain 'when this creature dies, draw a card' until end of turn",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						for _, tid := range targets {
							if tid == uuid.Nil {
								continue
							}
							perm := g.FindPermanent(tid)
							if perm == nil || perm.Controller != controller {
								continue
							}
							g.AddContinuousEffect(TemporaryBoost(perm.ID(), 1, 0))
							// Grant "when this creature dies, draw a card" until end of turn.
							permID := perm.ID()
							ctrl := perm.Controller
							deathDraw := NewTriggered(EvtZoneChange, false,
								FuncEffect("draw a card when this creature dies",
									EffectProperties{Outcome: OutcomeBenefit, DrawCount: 1},
									func(g *Game, _, ctrl uuid.UUID, _ []uuid.UUID) error {
										p := g.GetPlayer(ctrl)
										if p != nil {
											g.PlayerDrawCard(p)
										}
										return nil
									}),
							)
							deathDraw.SetCondition(func(evt *GameEvent, _ GameReader, _, _ uuid.UUID) bool {
								return evt.SourceID == permID &&
									evt.FromZone == ZoneBattlefield &&
									evt.ToZone == ZoneGraveyard &&
									evt.Flag // was a creature
							})
							deathDraw.SetSource(permID)
							deathDraw.SetController(ctrl)
							g.AddContinuousEffect(FuncContinuousEffect(
								LayerAbility, EndOfTurn,
								func(g *Game, _ uuid.UUID) error {
									p := g.FindPermanent(permID)
									if p != nil {
										p.RuntimeAbilities = append(p.RuntimeAbilities, WrapGrantedAbility(deathDraw))
									}
									return nil
								},
							))
						}
						return nil
					},
				),
			),
		)
	})


// Ral Zarek, Guest Lecturer {1}{B}{B}
// Legendary Planeswalker — Ral
// +1: Surveil 2.
// −1: Any number of target players each discard a card.
// −2: Return target creature card with mana value 3 or less from your graveyard to the battlefield.
// −7: Flip five coins. Target opponent skips their next X turns, where X is the number of coins that came up heads.
// XXX: Loyalty-activated abilities (+1/−1/−2/−7) are not implemented (engine lacks loyalty ability support, CR 606).
// XXX: −7 "skip turns" requires skip-turn engine support.
	Register("Ral Zarek, Guest Lecturer", func() Card {
		return NewPlaneswalker("Ral Zarek, Guest Lecturer", "{1}{B}{B}", 4,
			WithSuperTypes(SuperLegendary),
			WithSubTypes("Ral"),
		)
	})


// Rapier Wit {1}{W}
// Instant
// Tap target creature. If it's your turn, put a stun counter on it. (If a permanent with a stun counter would become untapped, remove one from it instead.)
// Draw a card.
// XXX: Stun counters are not implemented in the engine (no CounterType for Stun, no replacement effect for untap). Tap and draw are implemented.
	Register("Rapier Wit", func() Card {
		return NewInstant("Rapier Wit", "{1}{W}",
			NewTargetedSpell(
				TargetCreature(),
				TapTarget(),
				// XXX: "if it's your turn, put a stun counter on it" — stun counters not implemented
				FuncEffect("draw a card",
					EffectProperties{Outcome: OutcomeBenefit, DrawCount: 1},
					func(g *Game, _, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p != nil {
							g.PlayerDrawCard(p)
						}
						return nil
					},
				),
			),
		)
	})


// Rapturous Moment {4}{U}{R}
// Sorcery
// Draw three cards, then discard two cards. Add {U}{U}{R}{R}{R}.
	Register("Rapturous Moment", func() Card {
		return NewSorcery("Rapturous Moment", "{4}{U}{R}",
			NewSpellAbility(
				FuncEffect(
					"draw three cards, then discard two cards, add {U}{U}{R}{R}{R}",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, _, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						g.PlayerDrawCard(p)
						g.PlayerDrawCard(p)
						g.PlayerDrawCard(p)
						// Discard two cards.
						hand := p.Hand()
						if len(hand) > 0 {
							count := 2
							if len(hand) < count {
								count = len(hand)
							}
							chosen := p.ChooseCardsFromHand(count, "discard two cards", g)
							for _, c := range chosen {
								g.PlayerDiscard(p, c.ID())
							}
						}
						// Add {U}{U}{R}{R}{R}.
						p.ManaPool().Add(Blue, 2)
						p.ManaPool().Add(Red, 3)
						return nil
					},
				),
			),
		)
	})


// Render Speechless {2}{W}{B}
// Sorcery
// Target opponent reveals their hand. You choose a nonland card from it. That player discards that card.
// Put two +1/+1 counters on up to one target creature.
	Register("Render Speechless", func() Card {
		return NewSorcery("Render Speechless", "{2}{W}{B}",
			NewMultiTargetSpell(
				[]Target{
					TargetOpponent(),
					TargetUpToOneCreature(),
				},
				FuncEffect(
					"reveal opponent's hand; choose nonland card; they discard it; put two +1/+1 counters on up to one creature",
					EffectProperties{Outcome: OutcomeUnknown},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						oppID := targets[0]
						opp := g.GetPlayer(oppID)
						if opp != nil {
							nonlandFilter := NewCardFilter("nonland card", func(c Card) bool {
								return !c.HasType(TypeLand)
							})
							chosen := g.PickFromHand(g.GetPlayer(controller), opp, nonlandFilter, true, "choose a nonland card to discard")
							if chosen != nil {
								g.PlayerDiscard(opp, chosen.ID())
							}
						}
						if len(targets) >= 2 && targets[1] != uuid.Nil {
							perm := g.FindPermanent(targets[1])
							if perm != nil {
								g.AddCountersWithReplacement(perm, P1P1, 1, sourceID, false)
								g.AddCountersWithReplacement(perm, P1P1, 1, sourceID, false)
							}
						}
						return nil
					},
				),
			),
		)
	})


// Restoration Seminar {5}{W}{W}
// Sorcery — Lesson
// Return target nonland permanent card from your graveyard to the battlefield.
// Paradigm (Then exile this spell. After you first resolve a spell with this name, you may cast a copy of it from exile without paying its mana cost at the beginning of each of your first main phases.)
// TODO: implement
	Register("Restoration Seminar", func() Card {
		return NewSorcery("Restoration Seminar", "{5}{W}{W}",
			NewSpellAbility(),
		)
	})


// Root Manipulation {3}{B}{G}
// Sorcery
// Until end of turn, creatures you control get +2/+2 and gain menace and "Whenever this creature attacks, you gain 1 life." (A creature with menace can't be blocked except by two or more creatures.)
	Register("Root Manipulation", func() Card {
		return NewSorcery("Root Manipulation", "{3}{B}{G}",
			NewSpellAbility(
				FuncEffect(
					"until end of turn, creatures you control get +2/+2, gain menace, and 'whenever this creature attacks, you gain 1 life'",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						creatures := g.FilterBattlefield(And(ControlledBy(controller), IsCreature))
						for _, perm := range creatures {
							g.AddContinuousEffect(TemporaryBoost(perm.ID(), 2, 2))
							g.AddContinuousEffect(TemporaryKeyword(perm.ID(), Menace))
							// Grant "whenever this creature attacks, you gain 1 life" until end of turn.
							permID := perm.ID()
							ctrl := controller
							attackTrigger := NewTriggered(EvtDeclaredAttacker, false,
								FuncEffect("gain 1 life when this creature attacks",
									EffectProperties{Outcome: OutcomeBenefit},
									func(g *Game, _, ctrl uuid.UUID, _ []uuid.UUID) error {
										p := g.GetPlayer(ctrl)
										if p != nil {
											g.PlayerGainLife(p, 1)
										}
										return nil
									},
								),
							)
							attackTrigger.SetCondition(func(evt *GameEvent, _ GameReader, _, _ uuid.UUID) bool {
								return evt.SourceID == permID
							})
							attackTrigger.SetSource(permID)
							attackTrigger.SetController(ctrl)
							g.AddContinuousEffect(FuncContinuousEffect(
								LayerAbility, EndOfTurn,
								func(g *Game, _ uuid.UUID) error {
									p := g.FindPermanent(permID)
									if p != nil {
										p.RuntimeAbilities = append(p.RuntimeAbilities, WrapGrantedAbility(attackTrigger))
									}
									return nil
								},
							))
						}
						return nil
					},
				),
			),
		)
	})


// Run Behind {3}{U}
// Instant
// This spell costs {1} less to cast if it targets an attacking creature.
// Target creature's owner puts it on their choice of the top or bottom of their library.
	Register("Run Behind", func() Card {
		return NewInstant("Run Behind", "{3}{U}",
			NewTargetedSpell(
				TargetCreature(),
				FuncEffect(
					"target creature's owner puts it on their choice of the top or bottom of their library",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						perm := g.FindPermanent(targets[0])
						if perm == nil {
							return nil
						}
						card := perm.Card
						ownerID := perm.Card.Owner()
						if ownerID == uuid.Nil {
							ownerID = perm.Controller
						}
						ownerPlayer := g.GetPlayer(ownerID)
						if ownerPlayer == nil {
							return nil
						}
						permID := perm.ID()
						permController := perm.Controller
						choice := ownerPlayer.ChooseMode([]string{"top", "bottom"}, "put "+card.Name()+" on top or bottom of your library")
						g.RemoveFromBattlefield(perm)
						if choice == 0 {
							// top of library
							ownerPlayer.SetLibrary(append([]Card{card}, ownerPlayer.Library()...))
						} else {
							// bottom of library
							ownerPlayer.SetLibrary(append(ownerPlayer.Library(), card))
						}
						g.FireEvent(GameEvent{
							Type:     EvtZoneChange,
							SourceID: permID,
							PlayerID: permController,
							FromZone: ZoneBattlefield,
							ToZone:   ZoneLibrary,
						})
						return nil
					},
				),
			),
			WithTargetConditionalCostReduction(1, func(g *Game, controller uuid.UUID, _ Card, targets []uuid.UUID) bool {
				for _, id := range targets {
					if g.IsAttackingInCombat(id) {
						return true
					}
				}
				return false
			}),
		)
	})


// Seize the Spoils {2}{R}
// Sorcery
// As an additional cost to cast this spell, discard a card.
// Draw two cards and create a Treasure token. (It's an artifact with "{T}, Sacrifice this token: Add one mana of any color.")
	Register("Seize the Spoils", func() Card {
		return NewSorcery("Seize the Spoils", "{2}{R}",
			NewSpellAbility(
				DrawCards(Fixed(2)),
				CreateTreasureToken(),
			),
			WithAdditionalCost(DiscardCost(1)),
		)
	})


// Send in the Pest {1}{B}
// Sorcery
// Each opponent discards a card. You create a 1/1 black and green Pest creature token with "Whenever this token attacks, you gain 1 life."
	Register("Send in the Pest", func() Card {
		pestToken := TokenWithAbilities(
			CreateColoredToken("Pest Token", 1, 1,
				[]Color{Black, Green},
				[]CardType{TypeCreature},
				[]string{"Pest"},
			),
			AttacksTrigger(GainLife(1), false),
		)
		return NewSorcery("Send in the Pest", "{1}{B}",
			NewSpellAbility(
				FuncEffect(
					"each opponent discards a card",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						opp := g.GetOpponent(controller)
						if opp == nil {
							return nil
						}
						hand := opp.Hand()
						if len(hand) == 0 {
							return nil
						}
						chosen := opp.ChooseCardFromHand(hand, "discard a card", g)
						if chosen == nil {
							return nil
						}
						g.PlayerDiscard(opp, chosen.ID())
						return nil
					},
				),
				pestToken,
			),
		)
	})


// Silverquill Charm {W}{B}
// Instant
// Choose one —
// • Put two +1/+1 counters on target creature.
// • Exile target creature with power 2 or less.
// • Each opponent loses 3 life and you gain 3 life.
	Register("Silverquill Charm", func() Card {
		c := NewInstant("Silverquill Charm", "{W}{B}", nil)
		c.AddAbility(NewModalSpell([]Mode{
			{
				Label:   "Put two +1/+1 counters on target creature",
				Targets: []Target{TargetCreature()},
				Effects: []Effect{FuncEffect(
					"put two +1/+1 counters on target creature",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						perm := g.FindPermanent(targets[0])
						if perm == nil {
							return nil
						}
						g.AddCountersWithReplacement(perm, P1P1, 1, sourceID, false)
						g.AddCountersWithReplacement(perm, P1P1, 1, sourceID, false)
						return nil
					},
				)},
			},
			{
				Label:   "Exile target creature with power 2 or less",
				Targets: []Target{TargetCreature(HasPowerLTE(2))},
				Effects: []Effect{ExileTarget()},
			},
			{
				Label: "Each opponent loses 3 life and you gain 3 life",
				Effects: []Effect{FuncEffect(
					"each opponent loses 3 life and you gain 3 life",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						opp := g.GetOpponent(controller)
						if opp != nil {
							g.PlayerLoseLife(opp, 3)
						}
						p := g.GetPlayer(controller)
						if p != nil {
							g.PlayerGainLife(p, 3)
						}
						return nil
					},
				)},
			},
		}))
		return c
	})


// Snarl Song {5}{G}
// Sorcery
// Converge — Create two 0/0 green and blue Fractal creature tokens. Put X +1/+1 counters on each of them and you gain X life, where X is the number of colors of mana spent to cast this spell.
	Register("Snarl Song", func() Card {
		return NewSorcery("Snarl Song", "{5}{G}",
			NewSpellAbility(
				FuncEffect(
					"converge: create two 0/0 Fractal tokens with X counters each and gain X life",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						ctx := g.ResolvingCastContext()
						x := 0
						if ctx != nil {
							x = ctx.DistinctColorsSpent()
						}
						for i := 0; i < 2; i++ {
							token := NewToken("Fractal Token", 0, 0,
								[]CardType{TypeCreature},
								[]string{"Fractal"},
							)
							token.SetColorOverride([]Color{Green, Blue})
							token.SetOwner(controller)
							perm := g.PutOnBattlefield(token, controller)
							if perm != nil && x > 0 {
								perm.AddCounter(P1P1, x)
								g.ApplyContinuousEffects()
							}
						}
						if x > 0 {
							p := g.GetPlayer(controller)
							if p != nil {
								g.PlayerGainLife(p, x)
							}
						}
						return nil
					},
				),
			),
		)
	})


// Social Snub {1}{W}{B}
// Sorcery
// When you cast this spell while you control a creature, you may copy this spell.
// Each player sacrifices a creature of their choice. Each opponent loses 1 life and you gain 1 life.
	Register("Social Snub", func() Card {
		socialSnubEffect := FuncEffect(
			"each player sacrifices a creature; opponent loses 1, you gain 1",
			EffectProperties{Outcome: OutcomeUnknown},
			func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
				// Each player sacrifices a creature of their choice (APNAP order).
				for _, pid := range []uuid.UUID{controller, g.GetOpponent(controller).PlayerID()} {
					p := g.GetPlayer(pid)
					if p == nil {
						continue
					}
					candidates := g.FilterBattlefield(And(ControlledBy(pid), IsCreature))
					if len(candidates) == 0 {
						continue
					}
					chosen := p.ChoosePermanent(candidates, "sacrifice a creature", g)
					if chosen != nil {
						g.Sacrifice(chosen)
					}
				}
				// Each opponent loses 1 life and you gain 1 life.
				opp := g.GetOpponent(controller)
				if opp != nil {
					opp.LoseLife(1)
				}
				p := g.GetPlayer(controller)
				if p != nil {
					g.PlayerGainLife(p, 1)
				}
				return nil
			},
		)
		c := NewSorcery("Social Snub", "{1}{W}{B}",
			NewSpellAbility(socialSnubEffect),
		)
		// When you cast this spell while you control a creature, you may copy this spell.
		castTrigger := WheneverYouCastSpellTrigger(
			FuncEffect(
				"if you control a creature, you may copy Social Snub",
				EffectProperties{Outcome: OutcomeBenefit},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if !g.AnyBattlefield(And(ControlledBy(controller), IsCreature)) {
						return nil
					}
					p := g.GetPlayer(controller)
					if p == nil {
						return nil
					}
					if !p.ChooseMayAbility("copy Social Snub") {
						return nil
					}
					g.CopySpellOnStack(sourceID, controller, false)
					return nil
				},
			),
			true,
			NewCardFilter("Social Snub", func(c Card) bool {
				return c.Name() == "Social Snub"
			}),
		)
		c.AddAbility(castTrigger)
		return c
	})


// Splatter Technique {1}{U}{U}{R}{R}
// Sorcery
// Choose one —
// • Draw four cards.
// • Splatter Technique deals 4 damage to each creature and planeswalker.
	Register("Splatter Technique", func() Card {
		c := NewSorcery("Splatter Technique", "{1}{U}{U}{R}{R}", nil)
		c.AddAbility(NewModalSpell([]Mode{
			{
				Label:   "Draw four cards",
				Targets: []Target{},
				Effects: []Effect{
					FuncEffect("draw four cards",
						EffectProperties{Outcome: OutcomeBenefit, DrawCount: 4},
						func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
							p := g.GetPlayer(controller)
							if p == nil {
								return nil
							}
							for i := 0; i < 4; i++ {
								g.PlayerDrawCard(p)
							}
							return nil
						},
					),
				},
			},
			{
				Label:   "Splatter Technique deals 4 damage to each creature and planeswalker",
				Targets: []Target{},
				Effects: []Effect{
					FuncEffect(
						"deal 4 damage to each creature and planeswalker",
						EffectProperties{Outcome: OutcomeDetriment, Mass: true},
						func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
							// XXX: planeswalkers not fully supported; hits all creatures and planeswalkers
							for _, perm := range g.AllBattlefield() {
								if perm.HasType(TypeCreature) || perm.HasType(TypePlaneswalker) {
									g.DealDamageToPermanent(perm, 4, sourceID)
								}
							}
							return nil
						},
					),
				},
			},
		}))
		return c
	})


// Stand Up for Yourself {2}{W}
// Instant
// Destroy target creature with power 3 or greater.
	Register("Stand Up for Yourself", func() Card {
		return NewInstant("Stand Up for Yourself", "{2}{W}",
			NewTargetedSpell(
				TargetCreature(HasPowerGTE(3)),
				DestroyTarget(),
			),
		)
	})


// Steal the Show {2}{R}
// Sorcery
// Choose one or both —
// • Target player discards any number of cards, then draws that many cards.
// • Steal the Show deals damage equal to the number of instant and sorcery cards in your graveyard to target creature or planeswalker.
// XXX: "Choose one or both" is not supported by the engine (only "choose one"). Implements as "choose one" modal.
	Register("Steal the Show", func() Card {
		instantOrSorceryGY := NewCardFilter("instant or sorcery in graveyard", func(c Card) bool {
			return c.HasType(TypeInstant) || c.HasType(TypeSorcery)
		})
		c := NewSorcery("Steal the Show", "{2}{R}", nil)
		c.AddAbility(NewModalSpell([]Mode{
			{
				Label:   "Target player discards any number of cards, then draws that many cards",
				Targets: []Target{TargetPlayer()},
				Effects: []Effect{
					FuncEffect(
						"target player discards any number of cards then draws that many",
						EffectProperties{Outcome: OutcomeUnknown},
						func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
							if len(targets) == 0 {
								return nil
							}
							target := g.GetPlayer(targets[0])
							if target == nil {
								return nil
							}
							hand := target.Hand()
							n := len(hand)
							if n == 0 {
								return nil
							}
							chosen := target.ChooseCardsFromHand(n, "discard any number", g)
							discarded := 0
							for _, card := range chosen {
								if _, ok := g.PlayerDiscard(target, card.ID()); ok {
									discarded++
								}
							}
							for i := 0; i < discarded; i++ {
								g.PlayerDrawCard(target)
							}
							return nil
						},
					),
				},
			},
			{
				Label:   "Steal the Show deals damage equal to instant and sorcery cards in your graveyard to target creature or planeswalker",
				Targets: []Target{TargetAnyTarget()},
				Effects: []Effect{
					FuncEffect(
						"deal damage equal to instants/sorceries in your graveyard",
						EffectProperties{Outcome: OutcomeDetriment},
						func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
							if len(targets) == 0 {
								return nil
							}
							p := g.GetPlayer(controller)
							if p == nil {
								return nil
							}
							count := 0
							for _, card := range p.Graveyard() {
								if instantOrSorceryGY.Match(card) {
									count++
								}
							}
							if count == 0 {
								return nil
							}
							perm := g.FindPermanent(targets[0])
							if perm != nil {
								g.DealDamageToPermanent(perm, count, sourceID)
							} else {
								player := g.GetPlayer(targets[0])
								if player != nil {
									g.DealDamageToPlayer(player, count, sourceID)
								}
							}
							return nil
						},
					),
				},
			},
		}))
		return c
	})


// Stress Dream {3}{U}{R}
// Instant
// Stress Dream deals 5 damage to up to one target creature. Look at the top two cards of your library. Put one of those cards into your hand and the other on the bottom of your library.
	Register("Stress Dream", func() Card {
		return NewInstant("Stress Dream", "{3}{U}{R}",
			NewMultiTargetSpell(
				[]Target{TargetUpToOneCreature()},
				FuncEffect(
					"deal 5 damage to up to one target creature; look at top 2, keep one, put other on bottom",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) > 0 && targets[0] != uuid.Nil {
							perm := g.FindPermanent(targets[0])
							if perm != nil {
								g.DealDamageToPermanent(perm, 5, sourceID)
							}
						}
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						top2 := g.RemoveTopN(p, 2)
						if len(top2) == 0 {
							return nil
						}
						if len(top2) == 1 {
							p.AddToHand(top2[0])
							return nil
						}
						chosen, _ := g.RevealAndPickFromTop(g.GetPlayer(controller), g.GetPlayer(controller), 2,
							NewCardFilter("any card", func(c Card) bool { return true }),
							false, "put into hand")
						if chosen != nil {
							p.AddToHand(chosen)
							for _, c := range top2 {
								if c.ID() != chosen.ID() {
									g.PutOnBottomInRandomOrder(p, []Card{c})
								}
							}
						} else {
							p.AddToHand(top2[0])
							g.PutOnBottomInRandomOrder(p, top2[1:])
						}
						return nil
					},
				),
			),
		)
	})


// Suspend Aggression {1}{R}{W}
// Instant
// Exile target nonland permanent and the top card of your library. For each of those cards, its owner may play it until the end of their next turn.
// XXX: "owner may play until end of their next turn" — per-card exile play permissions expiring at end of next turn are not implemented.
	Register("Suspend Aggression", func() Card {
		return NewInstant("Suspend Aggression", "{1}{R}{W}",
			NewTargetedSpell(
				TargetPermanent(Not(IsLand)),
				FuncEffect(
					"exile target nonland permanent and top card of library; owners may play until end of their next turn",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) > 0 {
							perm := g.FindPermanent(targets[0])
							if perm != nil {
								g.ExilePermanent(perm)
								// XXX: owner may play until end of their next turn not implemented
							}
						}
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						top := g.RemoveTopN(p, 1)
						for _, card := range top {
							g.ExileCardFaceDown(card, sourceID, controller)
							// XXX: owner may play until end of their next turn not implemented
						}
						return nil
					},
				),
			),
		)
	})


// Together as One {6}
// Sorcery
// Converge — Target player draws X cards, Together as One deals X damage to any target, and you gain X life, where X is the number of colors of mana spent to cast this spell.
	Register("Together as One", func() Card {
		return NewSorcery("Together as One", "{6}",
			NewMultiTargetSpell(
				[]Target{TargetPlayer(), TargetAnyTarget()},
				FuncEffect(
					"converge: target player draws X, deal X damage to any target, gain X life",
					EffectProperties{Outcome: OutcomeUnknown},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						ctx := g.ResolvingCastContext()
						x := 0
						if ctx != nil {
							x = ctx.DistinctColorsSpent()
						}
						if len(targets) > 0 {
							tp := g.GetPlayer(targets[0])
							if tp != nil {
								for i := 0; i < x; i++ {
									g.PlayerDrawCard(tp)
								}
							}
						}
						if len(targets) > 1 && targets[1] != uuid.Nil && x > 0 {
							perm := g.FindPermanent(targets[1])
							if perm != nil {
								g.DealDamageToPermanent(perm, x, sourceID)
							} else {
								player := g.GetPlayer(targets[1])
								if player != nil {
									g.DealDamageToPlayer(player, x, sourceID)
								}
							}
						}
						if x > 0 {
							p := g.GetPlayer(controller)
							if p != nil {
								g.PlayerGainLife(p, x)
							}
						}
						return nil
					},
				),
			),
		)
	})


// Tome Blast {1}{R}
// Sorcery
// Tome Blast deals 2 damage to any target.
// Flashback {4}{R} (You may cast this card from your graveyard for its flashback cost. Then exile it.)
// XXX: Flashback does not exile the card after resolution (engine feature needed).
	Register("Tome Blast", func() Card {
		return NewSorcery("Tome Blast", "{1}{R}",
			NewTargetedSpell(TargetAnyTarget(), DealDamage(Fixed(2))),
			WithAlternateCost(ZoneGraveyard, ParseManaCost("{4}{R}")),
		)
	})


// Traumatic Critique {X}{U}{R}
// Instant
// Traumatic Critique deals X damage to any target. Draw two cards, then discard a card.
	Register("Traumatic Critique", func() Card {
		return NewInstant("Traumatic Critique", "{X}{U}{R}",
			NewTargetedSpell(
				TargetAnyTarget(),
				CompositeEffects(
					"deal X damage; draw 2 then discard 1",
					DealDamage(XValue()),
					FuncEffect(
						"draw two cards then discard a card",
						EffectProperties{Outcome: OutcomeBenefit, DrawCount: 2},
						func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
							p := g.GetPlayer(controller)
							if p == nil {
								return nil
							}
							g.PlayerDrawCard(p)
							g.PlayerDrawCard(p)
							chosen := p.ChooseCardsFromHand(1, "discard", g)
							for _, card := range chosen {
								g.PlayerDiscard(p, card.ID())
							}
							return nil
						},
					),
				),
			),
		)
	})


// Unsubtle Mockery {2}{R}
// Instant
// Unsubtle Mockery deals 4 damage to target creature. Surveil 1. (Look at the top card of your library. You may put it into your graveyard.)
// XXX: Surveil 1 not implemented (engine lacks Surveil; differs from Scry in that cards go to graveyard).
	Register("Unsubtle Mockery", func() Card {
		return NewInstant("Unsubtle Mockery", "{2}{R}",
			NewTargetedSpell(
				TargetCreature(),
				DealDamage(Fixed(4)),
				// XXX: Surveil 1 not implemented
			),
		)
	})


// Vibrant Outburst {U}{R}
// Instant
// Vibrant Outburst deals 3 damage to any target. Tap up to one target creature.
	Register("Vibrant Outburst", func() Card {
		return NewInstant("Vibrant Outburst", "{U}{R}",
			NewMultiTargetSpell(
				[]Target{TargetAnyTarget(), TargetUpToOneCreature()},
				FuncEffect(
					"deal 3 damage to any target; tap up to one target creature",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) > 0 && targets[0] != uuid.Nil {
							perm := g.FindPermanent(targets[0])
							if perm != nil {
								g.DealDamageToPermanent(perm, 3, sourceID)
							} else {
								player := g.GetPlayer(targets[0])
								if player != nil {
									g.DealDamageToPlayer(player, 3, sourceID)
								}
							}
						}
						if len(targets) > 1 && targets[1] != uuid.Nil {
							perm := g.FindPermanent(targets[1])
							if perm != nil {
								g.TapPermanent(perm)
							}
						}
						return nil
					},
				),
			),
		)
	})


// Vicious Rivalry {2}{B}{G}
// Sorcery
// As an additional cost to cast this spell, pay X life.
// Destroy all artifacts and creatures with mana value X or less.
// XXX: "pay X life" as additional cost — engine lacks a variable life-payment cost attached to the spell.
// Implemented by prompting for X at resolution time, then paying life and executing the board sweep.
	Register("Vicious Rivalry", func() Card {
		return NewSorcery("Vicious Rivalry", "{2}{B}{G}",
			NewSpellAbility(
				FuncEffect(
					"pay X life then destroy all artifacts and creatures with mana value X or less",
					EffectProperties{Outcome: OutcomeDetriment, Mass: true},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						maxLife := p.Life() - 1
						if maxLife < 0 {
							maxLife = 0
						}
						x := p.ChooseNumber(0, maxLife, "pay X life — choose X")
						if x > 0 {
							g.PlayerLoseLife(p, x)
						}
						g.SetXValue(x)
						var toDestroy []*Permanent
						for _, perm := range g.AllBattlefield() {
							if !perm.HasType(TypeArtifact) && !perm.HasType(TypeCreature) {
								continue
							}
							if perm.Card.ManaCost().CMC() <= x {
								toDestroy = append(toDestroy, perm)
							}
						}
						for _, perm := range toDestroy {
							g.DestroyPermanent(perm)
						}
						return nil
					},
				),
			),
		)
	})


// Visionary's Dance {5}{U}{R}
// Sorcery
// Create two 3/3 blue and red Elemental creature tokens with flying.
// {2}, Discard this card: Look at the top two cards of your library. Put one of them into your hand and the other into your graveyard.
	Register("Visionary's Dance", func() Card {
		c := NewSorcery("Visionary's Dance", "{5}{U}{R}",
			NewSpellAbility(
				CreateColoredToken("Elemental Token", 3, 3, []Color{Blue, Red}, []CardType{TypeCreature}, []string{"Elemental"}, Flying),
				CreateColoredToken("Elemental Token", 3, 3, []Color{Blue, Red}, []CardType{TypeCreature}, []string{"Elemental"}, Flying),
			),
		)
		// {2}, Discard this card: Look at the top two cards of your library. Put one in hand, other in graveyard.
		c.AddAbility(NewActivatedAbility(
			FuncEffect(
				"look at top 2, put one in hand and other in graveyard",
				EffectProperties{Outcome: OutcomeBenefit},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					p := g.GetPlayer(controller)
					if p == nil {
						return nil
					}
					top2 := g.RemoveTopN(p, 2)
					if len(top2) == 0 {
						return nil
					}
					if len(top2) == 1 {
						p.AddToHand(top2[0])
						return nil
					}
					chosen, _ := g.RevealAndPickFromTop(g.GetPlayer(controller), g.GetPlayer(controller), 2,
						NewCardFilter("any card", func(c Card) bool { return true }),
						false, "put into hand")
					if chosen != nil {
						p.AddToHand(chosen)
						for _, c := range top2 {
							if c.ID() != chosen.ID() {
								p.AddToGraveyard(c)
							}
						}
					} else {
						p.AddToHand(top2[0])
						p.AddToGraveyard(top2[1])
					}
					return nil
				},
			),
			ManaCostOf("{2}"),
			WithCost(DiscardCost(1)),
		))
		return c
	})


// Wander Off {3}{B}
// Instant
// Exile target creature.
	Register("Wander Off", func() Card {
		return NewInstant("Wander Off", "{3}{B}",
			NewTargetedSpell(TargetCreature(), ExileTarget()),
		)
	})


// Wild Hypothesis {X}{G}
// Sorcery
// Create a 0/0 green and blue Fractal creature token. Put X +1/+1 counters on it.
// Surveil 2. (Look at the top two cards of your library, then put any number of them into your graveyard and the rest on top of your library in any order.)
// XXX: Surveil 2 not implemented (engine lacks Surveil; differs from Scry in that cards go to graveyard).
	Register("Wild Hypothesis", func() Card {
		return NewSorcery("Wild Hypothesis", "{X}{G}",
			NewSpellAbility(
				FuncEffect(
					"create 0/0 Fractal token with X counters",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						x := g.XValue()
						token := NewToken("Fractal Token", 0, 0,
							[]CardType{TypeCreature},
							[]string{"Fractal"},
						)
						token.SetColorOverride([]Color{Green, Blue})
						token.SetOwner(controller)
						perm := g.PutOnBattlefield(token, controller)
						if perm != nil && x > 0 {
							perm.AddCounter(P1P1, x)
							g.ApplyContinuousEffects()
						}
						// XXX: Surveil 2 not implemented
						return nil
					},
				),
			),
		)
	})


// Wilt in the Heat {2}{R}{W}
// Instant
// This spell costs {2} less to cast if one or more cards left your graveyard this turn.
// Wilt in the Heat deals 5 damage to target creature. If that creature would die this turn, exile it instead.
// XXX: "costs {2} less if one or more cards left your graveyard this turn" — engine has no tracking for
// cards leaving the graveyard this turn; cost reduction not implemented.
	Register("Wilt in the Heat", func() Card {
		return NewInstant("Wilt in the Heat", "{2}{R}{W}",
			NewTargetedSpell(
				TargetCreature(),
				FuncEffect(
					"deal 5 damage to target creature; if it would die this turn, exile it instead",
					EffectProperties{Outcome: OutcomeDetriment, DamageValue: Fixed(5)},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						perm := g.FindPermanent(targets[0])
						if perm == nil {
							return nil
						}
						g.AddExileIfWouldGoToGraveyardThisTurn(perm.ID(), sourceID)
						g.DealDamageToPermanent(perm, 5, sourceID)
						return nil
					},
				),
			),
			// XXX: {2} cost reduction if cards left graveyard this turn not implemented
		)
	})


// Wisdom of Ages {4}{U}{U}{U}
// Sorcery
// Return all instant and sorcery cards from your graveyard to your hand. You have no maximum hand size for the rest of the game.
// Exile Wisdom of Ages.
	Register("Wisdom of Ages", func() Card {
		return NewSorcery("Wisdom of Ages", "{4}{U}{U}{U}",
			NewSpellAbility(
				FuncEffect(
					"return all instants and sorceries from graveyard to hand; no max hand size; exile self",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						// Return all instant and sorcery cards from graveyard to hand.
						gy := append([]Card(nil), p.Graveyard()...)
						for _, card := range gy {
							if card.HasType(TypeInstant) || card.HasType(TypeSorcery) {
								p.RemoveFromGraveyard(card.ID())
								p.AddToHand(card)
							}
						}
						// No maximum hand size for the rest of the game.
						g.SetMaxHandSize(controller, 999)
						// Exile Wisdom of Ages: mark so resolver exiles instead of putting in graveyard.
						g.AddExileIfWouldGoToGraveyardThisTurn(sourceID, sourceID)
						return nil
					},
				),
			),
		)
	})


// Witherbloom Charm {B}{G}
// Instant
// Choose one —
// • You may sacrifice a permanent. If you do, draw two cards.
// • You gain 5 life.
// • Destroy target nonland permanent with mana value 2 or less.
	Register("Witherbloom Charm", func() Card {
		c := NewInstant("Witherbloom Charm", "{B}{G}", nil)
		c.AddAbility(NewModalSpell([]Mode{
			{
				Label:   "You may sacrifice a permanent. If you do, draw two cards",
				Targets: []Target{},
				Effects: []Effect{
					FuncEffect(
						"may sacrifice a permanent to draw two cards",
						EffectProperties{Outcome: OutcomeBenefit, DrawCount: 2},
						func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
							p := g.GetPlayer(controller)
							if p == nil {
								return nil
							}
							candidates := g.FilterBattlefield(ControlledBy(controller))
							if len(candidates) == 0 {
								return nil
							}
							if !p.ChooseMayAbility("sacrifice a permanent to draw two cards") {
								return nil
							}
							chosen := p.ChoosePermanent(candidates, "sacrifice a permanent", g)
							if chosen == nil {
								return nil
							}
							g.Sacrifice(chosen)
							g.PlayerDrawCard(p)
							g.PlayerDrawCard(p)
							return nil
						},
					),
				},
			},
			{
				Label:   "You gain 5 life",
				Targets: []Target{},
				Effects: []Effect{
					FuncEffect(
						"gain 5 life",
						EffectProperties{Outcome: OutcomeBenefit},
						func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
							p := g.GetPlayer(controller)
							if p != nil {
								g.PlayerGainLife(p, 5)
							}
							return nil
						},
					),
				},
			},
			{
				Label: "Destroy target nonland permanent with mana value 2 or less",
				Targets: []Target{
					TargetPermanent(And(
						Not(IsLand),
						NewPermanentFilter("mana value 2 or less", func(perm *Permanent, gr *Game) bool {
							return perm.Card.ManaCost().CMC() <= 2
						}),
					)),
				},
				Effects: []Effect{DestroyTarget()},
			},
		}))
		return c
	})


// Withering Curse {1}{B}{B}
// Sorcery
// All creatures get -2/-2 until end of turn.
// Infusion — If you gained life this turn, destroy all creatures instead.
// TODO: implement
	Register("Withering Curse", func() Card {
		return NewSorcery("Withering Curse", "{1}{B}{B}",
			NewSpellAbility(),
		)
	})


// Zimone's Experiment {3}{G}
// Sorcery
// Look at the top five cards of your library. You may reveal up to two creature and/or land cards from among them, then put the rest on the bottom of your library in a random order. Put all land cards revealed this way onto the battlefield tapped and put all creature cards revealed this way into your hand.
	Register("Zimone's Experiment", func() Card {
		return NewSorcery("Zimone's Experiment", "{3}{G}",
			NewSpellAbility(
				FuncEffect(
					"look at top 5; reveal up to 2 creature/land; lands to battlefield tapped, creatures to hand; rest on bottom random",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						top5 := g.RemoveTopN(p, 5)
						if len(top5) == 0 {
							return nil
						}
						// Temporarily put cards back to use RevealAndPickFromTop.
						g.PutOnTopInChosenOrder(p, top5)
						revealed := make([]Card, 0, 2)
						for i := 0; i < 2; i++ {
							revealedIDs := make(map[uuid.UUID]bool, len(revealed))
							for _, r := range revealed {
								revealedIDs[r.ID()] = true
							}
							creatOrLandFilter := NewCardFilter("creature or land (not yet revealed)", func(c Card) bool {
								if revealedIDs[c.ID()] {
									return false
								}
								return c.HasType(TypeCreature) || c.HasType(TypeLand)
							})
							chosen, _ := g.RevealAndPickFromTop(g.GetPlayer(controller), g.GetPlayer(controller), 5,
								creatOrLandFilter,
								true, "reveal a creature or land card")
							if chosen == nil {
								break
							}
							revealed = append(revealed, chosen)
						}
						// Remove all top5 from library; separate revealed from rest.
						top5 = g.RemoveTopN(p, 5)
						revealedSet := make(map[uuid.UUID]bool, len(revealed))
						for _, c := range revealed {
							revealedSet[c.ID()] = true
						}
						var rest []Card
						for _, c := range top5 {
							if !revealedSet[c.ID()] {
								rest = append(rest, c)
							}
						}
						if len(rest) > 0 {
							g.PutOnBottomInRandomOrder(p, rest)
						}
						for _, card := range revealed {
							if card.HasType(TypeLand) {
								perm := g.PutOnBattlefield(card, controller)
								if perm != nil {
									g.TapPermanent(perm)
								}
							} else if card.HasType(TypeCreature) {
								p.AddToHand(card)
							}
						}
						return nil
					},
				),
			),
		)
	})

}
