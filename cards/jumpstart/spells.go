package jumpstart

import (
	"github.com/google/uuid"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage"
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

func init() {
	registerSpells()
}

// discardCardOfTypeCost is a Cost that discards one card from the controller's
// hand matching the given filter. Used by Thirst for Knowledge ("discard an
// artifact card"). The controller chooses which matching card to discard.
type discardCardOfTypeCost struct {
	filter CardFilter
	label  string
}

func (c *discardCardOfTypeCost) candidates(p Player) []Card {
	var out []Card
	for _, card := range p.Hand() {
		if c.filter.IsZero() || c.filter.Match(card) {
			out = append(out, card)
		}
	}
	return out
}

func (c *discardCardOfTypeCost) CanPay(_, controller uuid.UUID, g *Game) bool {
	p := g.GetPlayer(controller)
	if p == nil {
		return false
	}
	return len(c.candidates(p)) > 0
}

func (c *discardCardOfTypeCost) Pay(_, controller uuid.UUID, g *Game) error {
	p := g.GetPlayer(controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	cands := c.candidates(p)
	if len(cands) == 0 {
		return ErrPlayerNotFound
	}
	chosen := p.ChooseCardFromHand(cands, c.label, g)
	if chosen == nil {
		chosen = cands[0]
	}
	g.PlayerDiscard(p, chosen.ID())
	return nil
}

func (c *discardCardOfTypeCost) Text() string { return c.label }

// fightBetween implements CR 701.13 between two arbitrary permanents (neither
// of which is the spell source). Both must be creatures on the battlefield;
// damage is dealt simultaneously, with protection respected per direction.
// Used by spells like Savage Stomp and Time to Feed where the spell source is
// the spell card itself rather than a fighting creature.
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

func registerSpells() {

	// Act of Treason {2}{R}
	// Sorcery
	// Gain control of target creature until end of turn. Untap that creature. It gains haste until end of turn.
	Register("Act of Treason", func() Card {
		return NewSorcery("Act of Treason", "{2}{R}",
			NewTargetedSpell(TargetCreature(), CompositeEffects(
				"gain control until end of turn, untap, gains haste",
				gainControlUntilEOT(),
				UntapTarget(),
				GrantKeyword(Haste),
			)),
		)
	})

	// Aegis of the Heavens {1}{W}
	// Instant
	// Target creature gets +1/+7 until end of turn.
	Register("Aegis of the Heavens", func() Card {
		return NewInstant("Aegis of the Heavens", "{1}{W}",
			NewTargetedSpell(TargetCreature(), Boost(Fixed(1), Fixed(7))),
		)
	})

	// Aerial Assault {2}{W}
	// Sorcery
	// Destroy target tapped creature. You gain 1 life for each creature you control with flying.
	Register("Aerial Assault", func() Card {
		return NewSorcery("Aerial Assault", "{2}{W}",
			NewTargetedSpell(TargetCreature(IsTapped), CompositeEffects(
				"destroy tapped creature; gain 1 life per flying creature you control",
				DestroyTarget(),
				GainLifeAmount(CountBattlefield(SelectController(), And(IsCreature, HasKeywordFilter(Flying)))),
			)),
		)
	})

	// Aggressive Urge {1}{G}
	// Instant
	// Target creature gets +1/+1 until end of turn.
	// Draw a card.
	Register("Aggressive Urge", func() Card {
		return NewInstant("Aggressive Urge", "{1}{G}",
			NewTargetedSpell(TargetCreature(), CompositeEffects(
				"+1/+1 and draw a card",
				Boost(Fixed(1), Fixed(1)),
				drawSelfCard(1),
			)),
		)
	})

	// Agonizing Syphon {3}{B}
	// Sorcery
	// Agonizing Syphon deals 3 damage to any target and you gain 3 life.
	Register("Agonizing Syphon", func() Card {
		return NewSorcery("Agonizing Syphon", "{3}{B}",
			NewTargetedSpell(TargetAnyTarget(), CompositeEffects(
				"deal 3 damage and gain 3 life",
				DealDamage(Fixed(3)),
				GainLife(3),
			)),
		)
	})

	// Angelic Edict {4}{W}
	// Sorcery
	// Exile target creature or enchantment.
	Register("Angelic Edict", func() Card {
		return NewSorcery("Angelic Edict", "{4}{W}",
			NewTargetedSpell(TargetPermanent(Or(IsCreature, IsEnchantment)), ExileTarget()),
		)
	})

	// Arbor Armament {G}
	// Instant
	// Put a +1/+1 counter on target creature. That creature gains reach until end of turn.
	Register("Arbor Armament", func() Card {
		return NewInstant("Arbor Armament", "{G}",
			NewTargetedSpell(TargetCreature(), CompositeEffects(
				"+1/+1 counter and reach",
				AddCounters(P1P1, Fixed(1)),
				GrantKeyword(Reach),
			)),
		)
	})

	// Assassin's Strike {4}{B}{B}
	// Sorcery
	// Destroy target creature. Its controller discards a card.
	Register("Assassin's Strike", func() Card {
		return NewSorcery("Assassin's Strike", "{4}{B}{B}",
			NewTargetedSpell(TargetCreature(), FuncEffect(
				"destroy target creature; its controller discards a card",
				EffectProperties{Outcome: OutcomeDetriment},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					perm := g.FindPermanent(targets[0])
					if perm == nil {
						return nil
					}
					ownerID := perm.Controller
					g.DestroyPermanent(perm)
					if owner := g.GetPlayer(ownerID); owner != nil {
						if len(owner.Hand()) > 0 {
							pick := owner.ChooseCardsFromHand(1, "discard a card", g)
							for _, c := range pick {
								owner.DiscardCard(c.ID())
							}
						}
					}
					return nil
				},
			)),
		)
	})

	// Auger Spree {1}{B}{R}
	// Instant
	// Target creature gets +4/-4 until end of turn.
	Register("Auger Spree", func() Card {
		return NewInstant("Auger Spree", "{1}{B}{R}",
			NewTargetedSpell(TargetCreature(), Boost(Fixed(4), Fixed(-4))),
		)
	})

	// Bake into a Pie {2}{B}{B}
	// Instant
	// Destroy target creature. Create a Food token.
	Register("Bake into a Pie", func() Card {
		return NewInstant("Bake into a Pie", "{2}{B}{B}",
			NewTargetedSpell(TargetCreature(), DestroyTarget(), CreateFoodToken()),
		)
	})

	// Barter in Blood {2}{B}{B}
	// Sorcery
	// Each player sacrifices two creatures of their choice.
	Register("Barter in Blood", func() Card {
		return NewSorcery("Barter in Blood", "{2}{B}{B}",
			NewSpellAbility(eachPlayerSacrificesNCreatures(2)),
		)
	})

	// Bathe in Dragonfire {2}{R}
	// Sorcery
	// Bathe in Dragonfire deals 4 damage to target creature.
	Register("Bathe in Dragonfire", func() Card {
		return NewSorcery("Bathe in Dragonfire", "{2}{R}",
			NewTargetedSpell(TargetCreature(), DealDamage(Fixed(4))),
		)
	})

	// Battlefield Promotion {1}{W}
	// Instant
	// Put a +1/+1 counter on target creature. That creature gains first strike until end of turn. You gain 2 life.
	Register("Battlefield Promotion", func() Card {
		return NewInstant("Battlefield Promotion", "{1}{W}",
			NewTargetedSpell(TargetCreature(), CompositeEffects(
				"+1/+1 counter, first strike, gain 2 life",
				AddCounters(P1P1, Fixed(1)),
				GrantKeyword(FirstStrike),
				GainLife(2),
			)),
		)
	})

	// Befuddle {2}{U}
	// Instant
	// Target creature gets -4/-0 until end of turn.
	// Draw a card.
	Register("Befuddle", func() Card {
		return NewInstant("Befuddle", "{2}{U}",
			NewTargetedSpell(TargetCreature(), CompositeEffects(
				"-4/-0 and draw a card",
				Boost(Fixed(-4), Fixed(0)),
				drawSelfCard(1),
			)),
		)
	})

	// Blindblast {2}{R}
	// Instant
	// Blindblast deals 1 damage to target creature. That creature can't block this turn.
	// Draw a card.
	Register("Blindblast", func() Card {
		return NewInstant("Blindblast", "{2}{R}",
			NewTargetedSpell(TargetCreature(), CompositeEffects(
				"1 damage, can't block, draw a card",
				DealDamage(Fixed(1)),
				cantBlockThisTurn(),
				drawSelfCard(1),
			)),
		)
	})

	// Blood Divination {3}{B}
	// Sorcery
	// As an additional cost to cast this spell, sacrifice a creature.
	// Draw three cards.
	Register("Blood Divination", func() Card {
		return NewSorcery("Blood Divination", "{3}{B}",
			NewSpellAbility(drawSelfCard(3)),
			WithAdditionalCost(SacrificeCreatureCost()),
		)
	})

	// Bone Splinters {B}
	// Sorcery
	// As an additional cost to cast this spell, sacrifice a creature.
	// Destroy target creature.
	Register("Bone Splinters", func() Card {
		return NewSorcery("Bone Splinters", "{B}",
			NewTargetedSpell(TargetCreature(), DestroyTarget()),
			WithAdditionalCost(SacrificeCreatureCost()),
		)
	})

	// Cemetery Recruitment {1}{B}
	// Sorcery
	// Return target creature card from your graveyard to your hand. If it's a Zombie card, draw a card.
	Register("Cemetery Recruitment", func() Card {
		return NewSorcery("Cemetery Recruitment", "{1}{B}",
			NewTargetedSpell(TargetCardInYourGraveyard(IsCreatureCard), FuncEffect(
				"return creature card; if Zombie, draw a card",
				EffectProperties{Outcome: OutcomeBenefit},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					p := g.GetPlayer(controller)
					if p == nil {
						return nil
					}
					picked, ok := p.RemoveFromGraveyard(targets[0])
					if !ok {
						return nil
					}
					p.AddToHand(picked)
					if hasSubType(picked, "Zombie") {
						g.PlayerDrawCard(p)
					}
					return nil
				},
			)),
		)
	})

	// Chart a Course {1}{U}
	// Sorcery
	// Draw two cards. Then discard a card unless you attacked this turn.
	Register("Chart a Course", func() Card {
		return NewSorcery("Chart a Course", "{1}{U}",
			NewSpellAbility(FuncEffect(
				"draw two; discard one unless attacked this turn",
				EffectProperties{Outcome: OutcomeBenefit, DrawCount: 2},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					p := g.GetPlayer(controller)
					if p == nil {
						return nil
					}
					g.PlayerDrawCard(p)
					g.PlayerDrawCard(p)
					attacked := false
					for _, perm := range g.FilterBattlefield(And(ControlledBy(controller), IsCreature)) {
						if g.HasAttackedThisTurn(perm.ID()) {
							attacked = true
							break
						}
					}
					if attacked {
						return nil
					}
					if len(p.Hand()) == 0 {
						return nil
					}
					pick := p.ChooseCardsFromHand(1, "discard a card", g)
					for _, c := range pick {
						p.DiscardCard(c.ID())
					}
					return nil
				},
			)),
		)
	})

	// Cloudshift {W}
	// Instant
	// Exile target creature you control, then return that card to the battlefield under your control.
	Register("Cloudshift", func() Card {
		return NewInstant("Cloudshift", "{W}",
			NewTargetedSpell(TargetControlledCreature(), flickerEffect(false)),
		)
	})

	// Collateral Damage {R}
	// Instant
	// As an additional cost to cast this spell, sacrifice a creature.
	// Collateral Damage deals 3 damage to any target.
	Register("Collateral Damage", func() Card {
		return NewInstant("Collateral Damage", "{R}",
			NewTargetedSpell(TargetAnyTarget(), DealDamage(Fixed(3))),
			WithAdditionalCost(SacrificeCreatureCost()),
		)
	})

	// Commune with Dinosaurs {G}
	// Sorcery
	// Look at the top five cards of your library. You may reveal a Dinosaur or land card from among them and put it into your hand. Put the rest on the bottom of your library in any order.
	Register("Commune with Dinosaurs", func() Card {
		dinoOrLandCard := NewCardFilter("Dinosaur or land card", func(c Card) bool {
			if c.HasType(TypeLand) {
				return true
			}
			if !c.HasType(TypeCreature) {
				return false
			}
			for _, st := range c.SubTypes() {
				if st == "Dinosaur" {
					return true
				}
			}
			return false
		})
		return NewSorcery("Commune with Dinosaurs", "{G}",
			NewSpellAbility(FuncEffect(
				"look at top 5; may reveal a Dinosaur or land and put in hand; rest to bottom",
				EffectProperties{Outcome: OutcomeBenefit},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					p := g.GetPlayer(controller)
					if p == nil {
						return nil
					}
					chosen, _ := g.RevealAndPickFromTop(p, p, 5, dinoOrLandCard, true,
						"reveal a Dinosaur or land card and put it into your hand")
					taken := g.RemoveTopN(p, 5)
					rest := taken
					if chosen != nil {
						rest = make([]Card, 0, len(taken))
						for _, c := range taken {
							if c.ID() == chosen.ID() {
								p.AddToHand(c)
								continue
							}
							rest = append(rest, c)
						}
					}
					g.PutOnBottomInChosenOrder(p, rest)
					return nil
				}),
			),
		)
	})

	// Crushing Canopy {2}{G}
	// Instant
	// Choose one —
	// • Destroy target creature with flying.
	// • Destroy target enchantment.
	Register("Crushing Canopy", func() Card {
		c := NewInstant("Crushing Canopy", "{2}{G}", nil)
		c.AddAbility(NewModalSpell([]Mode{
			{
				Label:   "Destroy target creature with flying",
				Targets: []Target{TargetCreature(HasKeywordFilter(Flying))},
				Effects: []Effect{DestroyTarget()},
			},
			{
				Label:   "Destroy target enchantment",
				Targets: []Target{TargetPermanent(IsEnchantment)},
				Effects: []Effect{DestroyTargetPermanent()},
			},
		}))
		return c
	})

	// Dance with Devils {3}{R}
	// Instant
	// Create two 1/1 red Devil creature tokens. They have "When this token dies, it deals 1 damage to any target."
	Register("Dance with Devils", func() Card {
		return NewInstant("Dance with Devils", "{3}{R}",
			NewSpellAbility(TokenWithAbilities(
				CreateTokens(2, "Devil", 1, 1,
					[]CardType{TypeCreature},
					[]string{"Devil"}),
				PutIntoGraveyardFromBattlefieldTrigger(
					DealDamage(Fixed(1)), false,
				).AddTarget(TargetAnyTarget()),
			)),
		)
	})

	// Dauntless Onslaught {2}{W}
	// Instant
	// Up to two target creatures each get +2/+2 until end of turn.
	Register("Dauntless Onslaught", func() Card {
		return NewInstant("Dauntless Onslaught", "{2}{W}",
			NewMultiTargetSpell(
				[]Target{TargetUpToNCreatures(2)},
				Boost(Fixed(2), Fixed(2)).Targeting(ToAllTargets()),
			),
		)
	})

	// Divine Arrow {1}{W}
	// Instant
	// Divine Arrow deals 4 damage to target attacking or blocking creature.
	Register("Divine Arrow", func() Card {
		return NewInstant("Divine Arrow", "{1}{W}",
			NewTargetedSpell(TargetCreature(Or(IsAttacking, IsBlocking)), DealDamage(Fixed(4))),
		)
	})

	// Doublecast {R}{R}
	// Sorcery
	// When you next cast an instant or sorcery spell this turn, copy that spell. You may choose new targets for the copy.
	Register("Doublecast", func() Card {
		return NewSorcery("Doublecast", "{R}{R}",
			NewSpellAbility(FuncEffect(
				"register delayed trigger: copy next instant/sorcery spell you cast this turn",
				EffectProperties{Outcome: OutcomeBenefit},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					registerDoublecastDelayedTrigger(g, sourceID, controller)
					return nil
				},
			)),
		)
	})

	// Douse in Gloom {2}{B}
	// Instant
	// Douse in Gloom deals 2 damage to target creature and you gain 2 life.
	Register("Douse in Gloom", func() Card {
		return NewInstant("Douse in Gloom", "{2}{B}",
			NewTargetedSpell(TargetCreature(), CompositeEffects(
				"2 damage, gain 2 life",
				DealDamage(Fixed(2)),
				GainLife(2),
			)),
		)
	})

	// Draconic Roar {1}{R}
	// Instant
	// As an additional cost to cast this spell, you may reveal a Dragon card from your hand.
	// Draconic Roar deals 3 damage to target creature. If you revealed a Dragon card or controlled a Dragon as you cast this spell, Draconic Roar deals 3 damage to that creature's controller.
	// CR 608.2g: both halves of the bonus condition are evaluated against
	// the cast-time snapshot (CastContext), not live state at resolution.
	Register("Draconic Roar", func() Card {
		dragonCard := NewCardFilter("Dragon card", func(c Card) bool {
			return c.HasSubType("Dragon")
		})
		return NewInstant("Draconic Roar", "{1}{R}",
			NewTargetedSpell(TargetCreature(), FuncEffect(
				"3 damage to target creature; +3 to its controller if revealed/controlled a Dragon",
				EffectProperties{Outcome: OutcomeDetriment, DamageValue: Fixed(3)},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						g.ClearOptionalCostPaid(sourceID)
						return nil
					}
					ctx := g.ResolvingCastContext()
					revealedDragon := ctx.RevealedSubtypeAtCast("Dragon")
					controlledDragon := ctx.HasControlledSubtypeAtCast("Dragon")
					perm := g.FindPermanent(targets[0])
					if perm == nil {
						g.ClearOptionalCostPaid(sourceID)
						return nil
					}
					g.DealDamageToPermanent(perm, 3, sourceID)
					if revealedDragon || controlledDragon {
						if owner := g.GetPlayer(perm.Controller); owner != nil {
							g.DealDamageToPlayer(owner, 3, sourceID)
						}
					}
					g.ClearOptionalCostPaid(sourceID)
					return nil
				},
			)),
			WithAdditionalCost(OptionalCost(
				RevealFromHandCost(dragonCard, "Reveal a Dragon card"),
				"Reveal a Dragon card from your hand?",
			)),
		)
	})

	// Dragon Fodder {1}{R}
	// Sorcery
	// Create two 1/1 red Goblin creature tokens.
	Register("Dragon Fodder", func() Card {
		return NewSorcery("Dragon Fodder", "{1}{R}",
			NewSpellAbility(CreateTokens(2, "Goblin", 1, 1, []CardType{TypeCreature}, []string{"Goblin"})),
		)
	})

	// Elemental Uprising {1}{G}
	// Instant
	// Target land you control becomes a 4/4 Elemental creature with haste until end of turn. It's still a land. It must be blocked this turn if able.
	Register("Elemental Uprising", func() Card {
		return NewInstant("Elemental Uprising", "{1}{G}",
			NewTargetedSpell(
				TargetPermanent(IsLand),
				FuncEffect(
					"target land you control becomes a 4/4 Elemental with haste; must be blocked",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						land := g.FindPermanent(targets[0])
						if land == nil {
							return nil
						}
						anim := AnimateTargetLand(land.ID(), AnimateLandOptions{
							Power:     4,
							Toughness: 4,
							SubTypes:  []string{"Elemental"},
							Colors:    []Color{Green},
							Keywords:  []Attr{Haste},
						}, EndOfTurn)
						anim.SetSourceID(sourceID)
						g.AddContinuousEffect(anim)
						mb := TargetMustBeBlockedIfAble(land.ID(), EndOfTurn)
						mb.SetSourceID(sourceID)
						g.AddContinuousEffect(mb)
						return nil
					},
				),
			),
		)
	})

	// Enlarge {3}{G}{G}
	// Sorcery
	// Target creature gets +7/+7 and gains trample until end of turn. It must be blocked this turn if able.
	Register("Enlarge", func() Card {
		return NewSorcery("Enlarge", "{3}{G}{G}",
			NewTargetedSpell(TargetCreature(), CompositeEffects(
				"+7/+7, trample, must be blocked if able",
				Boost(Fixed(7), Fixed(7)),
				GrantKeyword(Trample),
				FuncEffect(
					"target creature must be blocked this turn if able",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						eff := TargetMustBeBlockedIfAble(targets[0], EndOfTurn)
						eff.SetSourceID(sourceID)
						g.AddContinuousEffect(eff)
						return nil
					},
				),
			)),
		)
	})

	// Essence Flux {U}
	// Instant
	// Exile target creature you control, then return that card to the battlefield under its owner's control. If it's a Spirit, put a +1/+1 counter on it.
	Register("Essence Flux", func() Card {
		return NewInstant("Essence Flux", "{U}",
			NewTargetedSpell(TargetControlledCreature(), flickerEffect(true)),
		)
	})

	// Exclude {2}{U}
	// Instant
	// Counter target creature spell.
	// Draw a card.
	Register("Exclude", func() Card {
		return NewInstant("Exclude", "{2}{U}",
			NewTargetedSpell(TargetSpellOnStack(creatureSpellFilter), CompositeEffects(
				"counter creature spell, draw a card",
				CounterSpell(),
				drawSelfCard(1),
			)),
		)
	})

	// Exhume {1}{B}
	// Sorcery
	// Each player puts a creature card from their graveyard onto the battlefield.
	Register("Exhume", func() Card {
		return NewSorcery("Exhume", "{1}{B}",
			NewSpellAbility(FuncEffect(
				"each player puts a creature card from their graveyard onto the battlefield",
				EffectProperties{Outcome: OutcomeBenefit},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					order := []Player{g.ActivePlayerObj(), g.NonActivePlayerObj()}
					for _, p := range order {
						if p == nil {
							continue
						}
						var candidates []Card
						for _, c := range p.Graveyard() {
							if c.HasType(TypeCreature) {
								candidates = append(candidates, c)
							}
						}
						if len(candidates) == 0 {
							continue
						}
						chosen := p.ChooseCardFromLibrary(candidates, "Exhume: put a creature card from your graveyard onto the battlefield", g)
						if chosen == nil {
							continue
						}
						if c, ok := p.RemoveFromGraveyard(chosen.ID()); ok {
							g.PutOnBattlefield(c, p.PlayerID())
						}
					}
					return nil
				},
			)),
		)
	})

	// Explore {1}{G}
	// Sorcery
	// You may play an additional land this turn.
	// Draw a card.
	Register("Explore", func() Card {
		return NewSorcery("Explore", "{1}{G}",
			NewSpellAbility(
				FuncEffect(
					"you may play an additional land this turn",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						g.GrantExtraLandPlay(controller, 1)
						return nil
					},
				),
				drawSelfCard(1),
			),
		)
	})

	// Flame Lash {3}{R}
	// Instant
	// Flame Lash deals 4 damage to any target.
	Register("Flame Lash", func() Card {
		return NewInstant("Flame Lash", "{3}{R}",
			NewTargetedSpell(TargetAnyTarget(), DealDamage(Fixed(4))),
		)
	})

	// Flames of the Firebrand {2}{R}
	// Sorcery
	// Flames of the Firebrand deals 3 damage divided as you choose among one, two, or three targets.
	Register("Flames of the Firebrand", func() Card {
		return NewSorcery("Flames of the Firebrand", "{2}{R}",
			NewMultiTargetSpell(
				[]Target{TargetUpToNCreaturesOrPlayers(3)},
				DealDividedDamage(Fixed(3)),
			),
		)
	})

	// Flames of the Raze-Boar {5}{R}
	// Instant
	// Flames of the Raze-Boar deals 4 damage to target creature an opponent controls. Then Flames of the Raze-Boar deals 2 damage to each other creature that player controls if you control a creature with power 4 or greater.
	Register("Flames of the Raze-Boar", func() Card {
		return NewInstant("Flames of the Raze-Boar", "{5}{R}",
			NewTargetedSpell(TargetPermanentOpponentControls(IsCreature), FuncEffect(
				"4 damage; then 2 to each other of that player's creatures if you control power-4+",
				EffectProperties{Outcome: OutcomeDetriment, DamageValue: Fixed(4)},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					perm := g.FindPermanent(targets[0])
					if perm == nil {
						return nil
					}
					victimController := perm.Controller
					g.DealDamageToPermanent(perm, 4, sourceID)
					hasBig := false
					for _, p := range g.FilterBattlefield(And(ControlledBy(controller), IsCreature)) {
						if p.CurrentPower(g) >= 4 {
							hasBig = true
							break
						}
					}
					if !hasBig {
						return nil
					}
					for _, other := range g.FilterBattlefield(And(ControlledBy(victimController), IsCreature, NotID(perm.ID()))) {
						g.DealDamageToPermanent(other, 2, sourceID)
					}
					return nil
				},
			)),
		)
	})

	// Fling {1}{R}
	// Instant
	// As an additional cost to cast this spell, sacrifice a creature.
	// Fling deals damage equal to the sacrificed creature's power to any target.
	Register("Fling", func() Card {
		return NewInstant("Fling", "{1}{R}",
			NewTargetedSpell(TargetAnyTarget(),
				FuncEffect("Fling deals damage equal to the sacrificed creature's power to any target",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						snap := g.LastSacrificed()
						if snap == nil || len(targets) == 0 {
							return nil
						}
						amt := snap.Power
						if amt < 0 {
							amt = 0
						}
						if pl := g.GetPlayer(targets[0]); pl != nil {
							g.DealDamageToPlayer(pl, amt, sourceID)
							return nil
						}
						if perm := g.FindPermanent(targets[0]); perm != nil {
							g.DealDamageToPermanent(perm, amt, sourceID)
						}
						return nil
					}),
			),
			WithAdditionalCost(SacrificeCreatureCost()),
		)
	})

	// Flurry of Horns {4}{R}
	// Sorcery
	// Create two 2/3 red Minotaur creature tokens with haste.
	Register("Flurry of Horns", func() Card {
		return NewSorcery("Flurry of Horns", "{4}{R}",
			NewSpellAbility(CreateTokens(2, "Minotaur", 2, 3, []CardType{TypeCreature}, []string{"Minotaur"}, Haste)),
		)
	})

	// Fortify {2}{W}
	// Instant
	// Choose one —
	// • Creatures you control get +2/+0 until end of turn.
	// • Creatures you control get +0/+2 until end of turn.
	Register("Fortify", func() Card {
		c := NewInstant("Fortify", "{2}{W}", nil)
		c.AddAbility(NewModalSpell([]Mode{
			{
				Label:   "Creatures you control get +2/+0 until end of turn",
				Effects: []Effect{BoostMatchingUntilEndOfTurn(Fixed(2), Fixed(0), AnyPermanent)},
			},
			{
				Label:   "Creatures you control get +0/+2 until end of turn",
				Effects: []Effect{BoostMatchingUntilEndOfTurn(Fixed(0), Fixed(2), AnyPermanent)},
			},
		}))
		return c
	})

	// Funeral Rites {2}{B}
	// Sorcery
	// You draw two cards, lose 2 life, then mill two cards.
	Register("Funeral Rites", func() Card {
		return NewSorcery("Funeral Rites", "{2}{B}",
			NewSpellAbility(FuncEffect(
				"draw two, lose 2, mill two",
				EffectProperties{DrawCount: 2},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					p := g.GetPlayer(controller)
					if p == nil {
						return nil
					}
					g.PlayerDrawCard(p)
					g.PlayerDrawCard(p)
					p.LoseLife(2)
					millSelf(g, p, 2)
					return nil
				},
			)),
		)
	})

	// Gird for Battle {W}
	// Sorcery
	// Put a +1/+1 counter on each of up to two target creatures.
	Register("Gird for Battle", func() Card {
		return NewSorcery("Gird for Battle", "{W}",
			NewMultiTargetSpell(
				[]Target{TargetUpToNCreatures(2)},
				AddCounters(P1P1, Fixed(1)).Targeting(ToAllTargets()),
			),
		)
	})

	// Goblin Lore {1}{R}
	// Sorcery
	// Draw four cards, then discard three cards at random.
	Register("Goblin Lore", func() Card {
		return NewSorcery("Goblin Lore", "{1}{R}",
			NewSpellAbility(
				drawSelfCard(4),
				FuncEffect("discard three cards at random",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						g.DiscardAtRandom(p, 3)
						return nil
					}),
			),
		)
	})

	// Goblin Rally {3}{R}{R}
	// Sorcery
	// Create four 1/1 red Goblin creature tokens.
	Register("Goblin Rally", func() Card {
		return NewSorcery("Goblin Rally", "{3}{R}{R}",
			NewSpellAbility(CreateTokens(4, "Goblin", 1, 1, []CardType{TypeCreature}, []string{"Goblin"})),
		)
	})

	// Heartfire {1}{R}
	// Instant
	// As an additional cost to cast this spell, sacrifice a creature or planeswalker.
	// Heartfire deals 4 damage to any target.
	Register("Heartfire", func() Card {
		return NewInstant("Heartfire", "{1}{R}",
			NewTargetedSpell(TargetAnyTarget(), DealDamage(Fixed(4))),
			WithAdditionalCost(SacrificeCreatureCost()),
		)
	})

	// Homing Lightning {2}{R}{R}
	// Instant
	// Homing Lightning deals 4 damage to target creature and each other creature with the same name as that creature.
	Register("Homing Lightning", func() Card {
		return NewInstant("Homing Lightning", "{2}{R}{R}",
			NewTargetedSpell(TargetCreature(), FuncEffect(
				"4 damage to target and each other same-named creature",
				EffectProperties{Outcome: OutcomeDetriment, DamageValue: Fixed(4)},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					perm := g.FindPermanent(targets[0])
					if perm == nil {
						return nil
					}
					name := perm.Card.Name()
					for _, p := range g.FilterBattlefield(And(IsCreature, Named(name))) {
						g.DealDamageToPermanent(p, 4, sourceID)
					}
					return nil
				},
			)),
		)
	})

	// Hungry Flames {2}{R}
	// Instant
	// Hungry Flames deals 3 damage to target creature and 2 damage to target player or planeswalker.
	Register("Hungry Flames", func() Card {
		return NewInstant("Hungry Flames", "{2}{R}",
			NewMultiTargetSpell(
				[]Target{TargetCreature(), TargetPlayer()},
				FuncEffect(
					"3 damage to target creature, 2 damage to target player",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) >= 1 && targets[0] != uuid.Nil {
							if perm := g.FindPermanent(targets[0]); perm != nil {
								g.DealDamageToPermanent(perm, 3, sourceID)
							}
						}
						if len(targets) >= 2 && targets[1] != uuid.Nil {
							for _, pl := range g.AllPlayers() {
								if pl.PlayerID() == targets[1] {
									g.DealDamageToPlayer(pl, 2, sourceID)
									break
								}
							}
						}
						return nil
					},
				),
			),
		)
	})

	// Hunter's Insight {2}{G}
	// Instant
	// Choose target creature you control. Whenever that creature deals combat damage to a player or planeswalker this turn, draw that many cards.
	Register("Hunter's Insight", func() Card {
		return NewInstant("Hunter's Insight", "{2}{G}",
			NewTargetedSpell(TargetCreatureYouControl(),
				FuncEffect("register a delayed trigger this turn",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						g.RegisterDelayedTrigger(&DelayedTrigger{
							EventType:    EvtDamageDealt,
							Effects:      []Effect{DrawCards(EventAmountValue())},
							SourceID:     sourceID,
							Controller:   controller,
							MatchEventID: targets[0],
							MatchFlag:    true,
							Persistent:   true,
						})
						return nil
					},
				),
			),
		)
	})

	// Immolating Gyre {4}{R}{R}
	// Sorcery
	// Immolating Gyre deals X damage to each creature and planeswalker you don't control, where X is the number of instant and sorcery cards in your graveyard.
	Register("Immolating Gyre", func() Card {
		return NewSorcery("Immolating Gyre", "{4}{R}{R}",
			NewSpellAbility(FuncEffect(
				"X damage to each creature you don't control",
				EffectProperties{Outcome: OutcomeDetriment, Mass: true},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					p := g.GetPlayer(controller)
					if p == nil {
						return nil
					}
					x := 0
					for _, c := range p.Graveyard() {
						if c.HasType(TypeInstant) || c.HasType(TypeSorcery) {
							x++
						}
					}
					if x <= 0 {
						return nil
					}
					for _, perm := range g.FilterBattlefield(And(IsCreature, NotControlledBy(controller))) {
						g.DealDamageToPermanent(perm, x, sourceID)
					}
					return nil
				},
			)),
		)
	})

	// Innocent Blood {B}
	// Sorcery
	// Each player sacrifices a creature of their choice.
	Register("Innocent Blood", func() Card {
		return NewSorcery("Innocent Blood", "{B}",
			NewSpellAbility(eachPlayerSacrificesNCreatures(1)),
		)
	})

	// Inspired Charge {2}{W}{W}
	// Instant
	// Creatures you control get +2/+1 until end of turn.
	Register("Inspired Charge", func() Card {
		return NewInstant("Inspired Charge", "{2}{W}{W}",
			NewSpellAbility(BoostMatchingUntilEndOfTurn(Fixed(2), Fixed(1), AnyPermanent)),
		)
	})

	// Inspiring Call {2}{G}
	// Instant
	// Draw a card for each creature you control with a +1/+1 counter on it. Those creatures gain indestructible until end of turn.
	Register("Inspiring Call", func() Card {
		return NewInstant("Inspiring Call", "{2}{G}",
			NewSpellAbility(FuncEffect(
				"draw card per +1/+1 creature; those gain indestructible EOT",
				EffectProperties{Outcome: OutcomeBenefit},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					p := g.GetPlayer(controller)
					if p == nil {
						return nil
					}
					var matched []*Permanent
					for _, perm := range g.FilterBattlefield(And(ControlledBy(controller), IsCreature)) {
						if int(perm.Counters[P1P1]) > 0 {
							matched = append(matched, perm)
						}
					}
					for range matched {
						g.PlayerDrawCard(p)
					}
					for _, perm := range matched {
						eff := TargetEffect(LayerAbility, EndOfTurn, perm.ID(), func(g2 *Game, target *Permanent) error {
							g2.GrantAttr(target.ID(), Indestructible)
							return nil
						})
						eff.SetSourceID(sourceID)
						g.AddContinuousEffect(eff)
					}
					if len(matched) > 0 {
						g.ApplyContinuousEffects()
					}
					return nil
				},
			)),
		)
	})

	// Irresistible Prey {G}
	// Sorcery
	// Target creature must be blocked this turn if able.
	// Draw a card.
	Register("Irresistible Prey", func() Card {
		return NewSorcery("Irresistible Prey", "{G}",
			NewTargetedSpell(TargetCreature(), CompositeEffects(
				"must be blocked if able, draw a card",
				FuncEffect(
					"target creature must be blocked this turn if able",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						eff := TargetMustBeBlockedIfAble(targets[0], EndOfTurn)
						eff.SetSourceID(sourceID)
						g.AddContinuousEffect(eff)
						return nil
					},
				),
				drawSelfCard(1),
			)),
		)
	})

	// Languish {2}{B}{B}
	// Sorcery
	// All creatures get -4/-4 until end of turn.
	Register("Languish", func() Card {
		return NewSorcery("Languish", "{2}{B}{B}",
			NewSpellAbility(Boost(Fixed(-4), Fixed(-4)).Targeting(ToAllMatching(AnyPermanent)).Until(EndOfTurn)),
		)
	})

	// Last Gasp {1}{B}
	// Instant
	// Target creature gets -3/-3 until end of turn.
	Register("Last Gasp", func() Card {
		return NewInstant("Last Gasp", "{1}{B}",
			NewTargetedSpell(TargetCreature(), Boost(Fixed(-3), Fixed(-3))),
		)
	})

	// Launch Party {3}{B}
	// Instant
	// As an additional cost to cast this spell, sacrifice a creature.
	// Destroy target creature. Its controller loses 2 life.
	Register("Launch Party", func() Card {
		return NewInstant("Launch Party", "{3}{B}",
			NewTargetedSpell(TargetCreature(), FuncEffect(
				"destroy target; its controller loses 2 life",
				EffectProperties{Outcome: OutcomeDetriment},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					perm := g.FindPermanent(targets[0])
					if perm == nil {
						return nil
					}
					ownerID := perm.Controller
					g.DestroyPermanent(perm)
					if owner := g.GetPlayer(ownerID); owner != nil {
						owner.LoseLife(2)
					}
					return nil
				},
			)),
			WithAdditionalCost(SacrificeCreatureCost()),
		)
	})

	// Leave in the Dust {3}{U}
	// Instant
	// Return target nonland permanent to its owner's hand.
	// Draw a card.
	Register("Leave in the Dust", func() Card {
		return NewInstant("Leave in the Dust", "{3}{U}",
			NewTargetedSpell(TargetPermanent(Not(IsLand)), CompositeEffects(
				"bounce nonland permanent and draw a card",
				ReturnToHandTarget(),
				drawSelfCard(1),
			)),
		)
	})

	// Lifecrafter's Gift {3}{G}
	// Instant
	// Put a +1/+1 counter on target creature, then put a +1/+1 counter on each creature you control with a +1/+1 counter on it.
	Register("Lifecrafter's Gift", func() Card {
		return NewInstant("Lifecrafter's Gift", "{3}{G}",
			NewTargetedSpell(TargetCreature(), FuncEffect(
				"+1/+1 on target then +1/+1 on each of yours with a counter",
				EffectProperties{Outcome: OutcomeBenefit},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					if perm := g.FindPermanent(targets[0]); perm != nil {
						perm.AddCounter(P1P1, 1)
					}
					for _, perm := range g.FilterBattlefield(And(ControlledBy(controller), IsCreature)) {
						if int(perm.Counters[P1P1]) > 0 {
							perm.AddCounter(P1P1, 1)
						}
					}
					g.ApplyContinuousEffects()
					return nil
				},
			)),
		)
	})

	// Lightning Axe {R}
	// Instant
	// As an additional cost to cast this spell, discard a card or pay {5}.
	// Lightning Axe deals 5 damage to target creature.
	Register("Lightning Axe", func() Card {
		return NewInstant("Lightning Axe", "{R}",
			NewTargetedSpell(TargetCreature(), DealDamage(Fixed(5))),
			WithAdditionalCost(EitherCost(DiscardCost(1), ManaCostOf("{5}"))),
		)
	})

	// Long Road Home {1}{W}
	// Instant
	// Exile target creature. At the beginning of the next end step, return that card to the battlefield under its owner's control with a +1/+1 counter on it.
	Register("Long Road Home", func() Card {
		return NewInstant("Long Road Home", "{1}{W}",
			NewTargetedSpell(TargetCreature(),
				ExileTargetReturnAtEndStep(ReturnWithCounter(P1P1, 1)),
			),
		)
	})

	// Macabre Waltz {1}{B}
	// Sorcery
	// Return up to two target creature cards from your graveyard to your hand, then discard a card.
	Register("Macabre Waltz", func() Card {
		return NewSorcery("Macabre Waltz", "{1}{B}",
			NewMultiTargetSpell(
				[]Target{TargetUpToNCardsInYourGraveyard(2, IsCreatureCard)},
				ReturnFromGraveyardToHandTarget(),
				FuncEffect("then discard a card",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, _, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil || len(p.Hand()) == 0 {
							return nil
						}
						pick := p.ChooseCardsFromHand(1, "discard a card", g)
						for _, c := range pick {
							p.DiscardCard(c.ID())
						}
						return nil
					}),
			),
		)
	})

	// Magma Jet {1}{R}
	// Instant
	// Magma Jet deals 2 damage to any target. Scry 2.
	Register("Magma Jet", func() Card {
		return NewInstant("Magma Jet", "{1}{R}",
			NewTargetedSpell(TargetAnyTarget(), DealDamage(Fixed(2)), Scry(Fixed(2))),
		)
	})

	// Magmaquake {X}{R}{R}
	// Instant
	// Magmaquake deals X damage to each creature without flying and each planeswalker.
	Register("Magmaquake", func() Card {
		return NewInstant("Magmaquake", "{X}{R}{R}",
			NewSpellAbility(DealDamageToAllCreatures(XValue(), NotHasKeywordFilter(Flying))),
		)
	})

	// Moment of Heroism {1}{W}
	// Instant
	// Target creature gets +2/+2 and gains lifelink until end of turn.
	Register("Moment of Heroism", func() Card {
		return NewInstant("Moment of Heroism", "{1}{W}",
			NewTargetedSpell(TargetCreature(), CompositeEffects(
				"+2/+2 and lifelink",
				Boost(Fixed(2), Fixed(2)),
				GrantKeyword(Lifelink),
			)),
		)
	})

	// Momentous Fall {2}{G}{G}
	// Instant
	// As an additional cost to cast this spell, sacrifice a creature.
	// You draw cards equal to the sacrificed creature's power, then you gain life equal to its toughness.
	Register("Momentous Fall", func() Card {
		return NewInstant("Momentous Fall", "{2}{G}{G}",
			NewSpellAbility(
				FuncEffect("draw cards equal to the sacrificed creature's power, then you gain life equal to its toughness",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						snap := g.LastSacrificed()
						if snap == nil {
							return nil
						}
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						for i := 0; i < snap.Power; i++ {
							p.DrawCard()
						}
						if snap.Toughness > 0 {
							g.PlayerGainLife(p, snap.Toughness)
						}
						return nil
					}),
			),
			WithAdditionalCost(SacrificeCreatureCost()),
		)
	})

	// Mugging {R}
	// Sorcery
	// Mugging deals 2 damage to target creature. That creature can't block this turn.
	Register("Mugging", func() Card {
		return NewSorcery("Mugging", "{R}",
			NewTargetedSpell(TargetCreature(), CompositeEffects(
				"2 damage and can't block",
				DealDamage(Fixed(2)),
				cantBlockThisTurn(),
			)),
		)
	})

	// Nature's Way {1}{G}
	// Sorcery
	// Target creature you control gains vigilance and trample until end of turn. It deals damage equal to its power to target creature you don't control.
	Register("Nature's Way", func() Card {
		return NewSorcery("Nature's Way", "{1}{G}",
			NewMultiTargetSpell(
				[]Target{TargetCreatureYouControl(), TargetCreatureOpponentControls()},
				GrantKeyword(Vigilance),
				GrantKeyword(Trample),
				FuncEffect(
					"target you-control deals damage equal to its power to target opponent's creature",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) < 2 || targets[0] == uuid.Nil || targets[1] == uuid.Nil {
							return nil
						}
						mine := g.FindPermanent(targets[0])
						foe := g.FindPermanent(targets[1])
						if mine == nil || foe == nil {
							return nil
						}
						power := mine.CurrentPower(g)
						if power > 0 {
							g.DealDamageToPermanent(foe, power, mine.ID())
						}
						return nil
					},
				),
			),
		)
	})

	// Outnumber {R}
	// Instant
	// Outnumber deals damage to target creature equal to the number of creatures you control.
	Register("Outnumber", func() Card {
		return NewInstant("Outnumber", "{R}",
			NewTargetedSpell(TargetCreature(), DealDamage(CountBattlefield(SelectController(), IsCreature))),
		)
	})

	// Path to Exile {W}
	// Instant
	// Exile target creature. Its controller may search their library for a basic land card, put that card onto the battlefield tapped, then shuffle.
	Register("Path to Exile", func() Card {
		return NewInstant("Path to Exile", "{W}",
			NewTargetedSpell(TargetCreature(),
				FuncEffect("exile target creature; its controller may search their library for a basic land",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 || targets[0] == uuid.Nil {
							return nil
						}
						perm := g.FindPermanent(targets[0])
						if perm == nil {
							return nil
						}
						creatureCtrl := perm.Controller
						g.ExilePermanent(perm)
						p := g.GetPlayer(creatureCtrl)
						if p == nil {
							return nil
						}
						mode := p.ChooseMode([]string{"yes", "no"}, "search your library for a basic land?")
						if mode != 0 {
							return nil
						}
						var basics []Card
						for _, c := range p.Library() {
							if c.HasType(TypeLand) && c.HasSuperType(SuperBasic) {
								basics = append(basics, c)
							}
						}
						if len(basics) > 0 {
							chosen := p.ChooseCardFromLibrary(basics, "basic land", g)
							if chosen != nil {
								newLib := make([]Card, 0, len(p.Library())-1)
								for _, c := range p.Library() {
									if c.ID() != chosen.ID() {
										newLib = append(newLib, c)
									}
								}
								p.SetLibrary(newLib)
								newPerm := g.PutOnBattlefield(chosen, creatureCtrl)
								if newPerm != nil {
									newPerm.Tapped = true
								}
							}
						}
						p.ShuffleLibrary()
						return nil
					},
				),
			),
		)
	})

	// Peel from Reality {1}{U}
	// Instant
	// Return target creature you control and target creature you don't control to their owners' hands.
	Register("Peel from Reality", func() Card {
		return NewInstant("Peel from Reality", "{1}{U}",
			NewMultiTargetSpell(
				[]Target{TargetCreatureYouControl(), TargetCreatureOpponentControls()},
				FuncEffect(
					"return both creatures to their owners' hands",
					EffectProperties{Outcome: OutcomeUnknown, IsBounce: true},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						for _, tid := range targets {
							if tid == uuid.Nil {
								continue
							}
							perm := g.FindPermanent(tid)
							if perm == nil {
								continue
							}
							card := perm.Card
							owner := card.Owner()
							if owner == uuid.Nil {
								owner = perm.Controller
							}
							g.RemoveFromBattlefield(perm)
							if pl := g.GetPlayer(owner); pl != nil {
								pl.AddToHand(card)
							}
						}
						return nil
					},
				),
			),
		)
	})

	// Pillar of Flame {R}
	// Sorcery
	// Pillar of Flame deals 2 damage to any target. If a creature dealt damage this way would die this turn, exile it instead.
	Register("Pillar of Flame", func() Card {
		return NewSorcery("Pillar of Flame", "{R}",
			NewTargetedSpell(TargetAnyTarget(), CompositeEffects(
				"deal 2 damage; if a creature, exile it if it would die this turn",
				FuncEffect(
					"exile-if-would-die marker",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						perm := g.FindPermanent(targets[0])
						if perm == nil || perm.Card == nil {
							return nil
						}
						if !perm.Card.HasType(TypeCreature) {
							return nil
						}
						g.AddExileIfWouldGoToGraveyardThisTurn(perm.Card.ID(), sourceID)
						return nil
					},
				),
				DealDamage(Fixed(2)),
			)),
		)
	})

	// Raise the Alarm {1}{W}
	// Instant
	// Create two 1/1 white Soldier creature tokens.
	Register("Raise the Alarm", func() Card {
		return NewInstant("Raise the Alarm", "{1}{W}",
			NewSpellAbility(CreateTokens(2, "Soldier", 1, 1, []CardType{TypeCreature}, []string{"Soldier"})),
		)
	})

	// Read the Runes {X}{U}
	// Instant
	// Draw X cards. For each card drawn this way, discard a card unless you sacrifice a permanent.
	Register("Read the Runes", func() Card {
		anyPermanent := NewPermanentFilter("a permanent", func(p *Permanent, _ *Game) bool { return true })
		return NewInstant("Read the Runes", "{X}{U}",
			NewSpellAbility(
				DrawCards(XValue()),
				FuncEffect(
					"for each card drawn this way, discard a card unless you sacrifice a permanent",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						x := g.XValue()
						discardOne := FuncEffect(
							"discard a card",
							EffectProperties{Outcome: OutcomeDetriment},
							func(g *Game, _, c uuid.UUID, _ []uuid.UUID) error {
								p := g.GetPlayer(c)
								if p == nil {
									return nil
								}
								chosen := p.ChooseCardsFromHand(1, "discard", g)
								for _, cd := range chosen {
									p.RemoveFromHand(cd.ID())
									p.AddToGraveyard(cd)
								}
								return nil
							},
						)
						unless := UnlessTargetPays(
							SelectController(),
							SacrificeMatchingCost(anyPermanent, "Sacrifice a permanent"),
							"Sacrifice a permanent to avoid discarding a card?",
							discardOne,
						)
						for i := 0; i < x; i++ {
							if err := unless.Apply(g, sourceID, controller, nil); err != nil {
								return err
							}
						}
						return nil
					},
				),
			),
		)
	})

	// Reanimate {B}
	// Sorcery
	// Put target creature card from a graveyard onto the battlefield under your control. You lose life equal to that card's mana value.
	Register("Reanimate", func() Card {
		return NewSorcery("Reanimate", "{B}",
			NewTargetedSpell(TargetCreatureInYourGraveyard(), FuncEffect(
				"reanimate; lose life = mana value",
				EffectProperties{Outcome: OutcomeBenefit},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					p := g.GetPlayer(controller)
					if p == nil {
						return nil
					}
					picked, ok := p.RemoveFromGraveyard(targets[0])
					if !ok {
						return nil
					}
					g.PutOnBattlefield(picked, controller)
					p.LoseLife(picked.ManaCost().CMC())
					return nil
				},
			)),
		)
	})

	// Release the Dogs {3}{W}
	// Sorcery
	// Create four 1/1 white Dog creature tokens.
	Register("Release the Dogs", func() Card {
		return NewSorcery("Release the Dogs", "{3}{W}",
			NewSpellAbility(CreateTokens(4, "Dog", 1, 1, []CardType{TypeCreature}, []string{"Dog"})),
		)
	})

	// Riddle of Lightning {3}{R}{R}
	// Instant
	// Choose any target. Scry 3, then reveal the top card of your library. Riddle of Lightning deals damage equal to that card's mana value to that permanent or player.
	Register("Riddle of Lightning", func() Card {
		return NewInstant("Riddle of Lightning", "{3}{R}{R}",
			NewTargetedSpell(TargetAnyTarget(), CompositeEffects(
				"scry 3, reveal top, deal damage equal to its mana value",
				Scry(Fixed(3)),
				DealDamage(TopOfLibraryManaValue(SelectController())),
			)),
		)
	})

	// Rise of the Dark Realms {7}{B}{B}
	// Sorcery
	// Put all creature cards from all graveyards onto the battlefield under your control.
	Register("Rise of the Dark Realms", func() Card {
		return NewSorcery("Rise of the Dark Realms", "{7}{B}{B}",
			NewSpellAbility(FuncEffect(
				"reanimate all creature cards from all graveyards under your control",
				EffectProperties{Outcome: OutcomeBenefit, Mass: true},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					for _, p := range g.AllPlayers() {
						var creatureIDs []uuid.UUID
						for _, c := range p.Graveyard() {
							if c.HasType(TypeCreature) {
								creatureIDs = append(creatureIDs, c.ID())
							}
						}
						for _, id := range creatureIDs {
							if c, ok := p.RemoveFromGraveyard(id); ok {
								g.PutOnBattlefield(c, controller)
							}
						}
					}
					return nil
				},
			)),
		)
	})

	// Sarkhan's Rage {4}{R}
	// Instant
	// Sarkhan's Rage deals 5 damage to any target. If you control no Dragons, Sarkhan's Rage deals 2 damage to you.
	Register("Sarkhan's Rage", func() Card {
		return NewInstant("Sarkhan's Rage", "{4}{R}",
			NewTargetedSpell(TargetAnyTarget(), FuncEffect(
				"5 damage; if no Dragons, 2 to you",
				EffectProperties{Outcome: OutcomeDetriment, DamageValue: Fixed(5)},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					id := targets[0]
					if perm := g.FindPermanent(id); perm != nil {
						g.DealDamageToPermanent(perm, 5, sourceID)
					} else if pl := g.GetPlayer(id); pl != nil {
						g.DealDamageToPlayer(pl, 5, sourceID)
					}
					if !g.AnyBattlefield(And(ControlledBy(controller), IsCreature, HasSubType("Dragon"))) {
						if me := g.GetPlayer(controller); me != nil {
							g.DealDamageToPlayer(me, 2, sourceID)
						}
					}
					return nil
				},
			)),
		)
	})

	// Savage Stomp {2}{G}
	// Sorcery
	// This spell costs {2} less to cast if it targets a Dinosaur you control.
	// Put a +1/+1 counter on target creature you control. Then that creature fights target creature you don't control.
	Register("Savage Stomp", func() Card {
		return NewSorcery("Savage Stomp", "{2}{G}",
			NewMultiTargetSpell(
				[]Target{TargetCreatureYouControl(), TargetCreatureOpponentControls()},
				AddCounters(P1P1, Fixed(1)).Targeting(ToTarget()),
				FuncEffect(
					"target creature you control fights target creature you don't control",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) < 2 {
							return nil
						}
						fightBetween(g, targets[0], targets[1])
						return nil
					},
				),
			),
			WithTargetConditionalCostReduction(2, func(g *Game, controller uuid.UUID, c Card, targets []uuid.UUID) bool {
				for _, id := range targets {
					perm := g.FindPermanent(id)
					if perm == nil {
						continue
					}
					if perm.Controller != controller {
						continue
					}
					if perm.Card.HasSubType("Dinosaur") {
						return true
					}
				}
				return false
			}),
		)
	})

	// Settle the Score {2}{B}{B}
	// Sorcery
	// Exile target creature. Put two loyalty counters on a planeswalker you control.
	// XXX: requires planeswalker / loyalty-counter primitive; implement only exile portion
	Register("Settle the Score", func() Card {
		return NewSorcery("Settle the Score", "{2}{B}{B}",
			NewTargetedSpell(TargetCreature(), ExileTarget()),
		)
	})

	// Soul Salvage {2}{B}
	// Sorcery
	// Return up to two target creature cards from your graveyard to your hand.
	Register("Soul Salvage", func() Card {
		return NewSorcery("Soul Salvage", "{2}{B}",
			NewMultiTargetSpell(
				[]Target{TargetUpToNCardsInYourGraveyard(2, IsCreatureCard)},
				ReturnFromGraveyardToHandTarget(),
			),
		)
	})

	// Spitting Earth {1}{R}
	// Sorcery
	// Spitting Earth deals damage to target creature equal to the number of Mountains you control.
	Register("Spitting Earth", func() Card {
		return NewSorcery("Spitting Earth", "{1}{R}",
			NewTargetedSpell(TargetCreature(), DealDamage(CountBattlefield(SelectController(), And(IsLand, HasSubType("Mountain"))))),
		)
	})

	// Sweep Away {2}{U}
	// Instant
	// Return target creature to its owner's hand. If that creature is attacking, you may put it on top of its owner's library instead.
	Register("Sweep Away", func() Card {
		return NewInstant("Sweep Away", "{2}{U}",
			NewTargetedSpell(TargetCreature(), FuncEffect(
				"return to owner's hand; if attacking, may put on top of library instead",
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
					ownerPlayer := g.GetPlayer(owner)
					attacking := g.IsAttackingInCombat(perm.ID())
					controllerPlayer := g.GetPlayer(controller)
					if attacking && controllerPlayer != nil && ownerPlayer != nil &&
						controllerPlayer.ChooseMayAbility("put "+card.Name()+" on top of "+ownerPlayer.Name()+"'s library instead of returning to hand") {
						permID := perm.ID()
						permController := perm.Controller
						g.RemoveFromBattlefield(perm)
						ownerPlayer.SetLibrary(append([]Card{card}, ownerPlayer.Library()...))
						g.FireEvent(GameEvent{
							Type:     EvtZoneChange,
							SourceID: permID,
							PlayerID: permController,
							FromZone: ZoneBattlefield,
							ToZone:   ZoneLibrary,
						})
						return nil
					}
					g.BouncePermanentToHand(perm)
					return nil
				},
			)),
		)
	})

	// Take Heart {W}
	// Instant
	// Target creature gets +2/+2 until end of turn. You gain 1 life for each attacking creature you control.
	Register("Take Heart", func() Card {
		return NewInstant("Take Heart", "{W}",
			NewTargetedSpell(TargetCreature(), CompositeEffects(
				"+2/+2 and gain 1 life per attacking creature you control",
				Boost(Fixed(2), Fixed(2)),
				GainLifeAmount(CountBattlefield(SelectController(), And(IsCreature, IsAttacking))),
			)),
		)
	})

	// Talrand's Invocation {2}{U}{U}
	// Sorcery
	// Create two 2/2 blue Drake creature tokens with flying.
	Register("Talrand's Invocation", func() Card {
		return NewSorcery("Talrand's Invocation", "{2}{U}{U}",
			NewSpellAbility(CreateTokens(2, "Drake", 2, 2, []CardType{TypeCreature}, []string{"Drake"}, Flying)),
		)
	})

	// Tandem Tactics {1}{W}
	// Instant
	// Up to two target creatures each get +1/+2 until end of turn. You gain 2 life.
	Register("Tandem Tactics", func() Card {
		return NewInstant("Tandem Tactics", "{1}{W}",
			NewMultiTargetSpell(
				[]Target{TargetUpToNCreatures(2)},
				Boost(Fixed(1), Fixed(2)).Targeting(ToAllTargets()),
				GainLife(2),
			),
		)
	})

	// Thirst for Knowledge {2}{U}
	// Instant
	// Draw three cards. Then discard two cards unless you discard an artifact card.
	Register("Thirst for Knowledge", func() Card {
		artifactCard := NewCardFilter("artifact card", func(c Card) bool {
			return c.HasType(TypeArtifact)
		})
		return NewInstant("Thirst for Knowledge", "{2}{U}",
			NewSpellAbility(
				drawSelfCard(3),
				UnlessTargetPays(
					SelectController(),
					&discardCardOfTypeCost{filter: artifactCard, label: "Discard an artifact card"},
					"Discard an artifact card to avoid discarding two cards?",
					FuncEffect(
						"discard two cards",
						EffectProperties{Outcome: OutcomeDetriment},
						func(g *Game, _, c uuid.UUID, _ []uuid.UUID) error {
							p := g.GetPlayer(c)
							if p == nil {
								return nil
							}
							n := 2
							if len(p.Hand()) < n {
								n = len(p.Hand())
							}
							chosen := p.ChooseCardsFromHand(n, "discard", g)
							for _, cd := range chosen {
								g.PlayerDiscard(p, cd.ID())
							}
							return nil
						},
					),
				),
			),
		)
	})

	// Thought Collapse {1}{U}{U}
	// Instant
	// Counter target spell. Its controller mills three cards.
	Register("Thought Collapse", func() Card {
		return NewInstant("Thought Collapse", "{1}{U}{U}",
			NewTargetedSpell(TargetSpellOnStack(), FuncEffect(
				"counter spell; its controller mills 3",
				EffectProperties{Outcome: OutcomeDetriment},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					id := targets[0]
					var ownerID uuid.UUID
					if so := g.FindStackObject(id); so != nil {
						ownerID = so.Controller
					}
					g.CounterSpellOnStack(id)
					if owner := g.GetPlayer(ownerID); owner != nil {
						millSelf(g, owner, 3)
					}
					return nil
				},
			)),
		)
	})

	// Thought Scour {U}
	// Instant
	// Target player mills two cards.
	// Draw a card.
	Register("Thought Scour", func() Card {
		return NewInstant("Thought Scour", "{U}",
			NewTargetedSpell(TargetPlayer(), CompositeEffects(
				"target mills two; draw one",
				MillTargetPlayer(Fixed(2)),
				drawSelfCard(1),
			)),
		)
	})

	// Time to Feed {2}{G}
	// Sorcery
	// Choose target creature an opponent controls. When that creature dies this turn, you gain 3 life. Target creature you control fights that creature.
	Register("Time to Feed", func() Card {
		return NewSorcery("Time to Feed", "{2}{G}",
			NewMultiTargetSpell(
				[]Target{TargetCreatureOpponentControls(), TargetCreatureYouControl()},
				FuncEffect(
					"register on-death gain-3 trigger; your creature fights opponent's",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) < 2 {
							return nil
						}
						oppCreature := targets[0]
						yourCreature := targets[1]
						if err := OnPermanentDiesThisTurn(oppCreature, GainLife(3)).Apply(g, sourceID, controller, nil); err != nil {
							return err
						}
						fightBetween(g, yourCreature, oppCreature)
						return nil
					},
				),
			),
		)
	})

	// Valorous Stance {1}{W}
	// Instant
	// Choose one —
	// • Target creature gains indestructible until end of turn.
	// • Destroy target creature with toughness 4 or greater.
	Register("Valorous Stance", func() Card {
		c := NewInstant("Valorous Stance", "{1}{W}", nil)
		c.AddAbility(NewModalSpell([]Mode{
			{
				Label:   "Target creature gains indestructible until end of turn",
				Targets: []Target{TargetCreature()},
				Effects: []Effect{GrantKeyword(Indestructible)},
			},
			{
				Label:   "Destroy target creature with toughness 4 or greater",
				Targets: []Target{TargetCreature(toughness4OrGreater)},
				Effects: []Effect{DestroyTarget()},
			},
		}))
		return c
	})

	// Volcanic Fallout {1}{R}{R}
	// Instant
	// This spell can't be countered.
	// Volcanic Fallout deals 2 damage to each creature and each player.
	Register("Volcanic Fallout", func() Card {
		return NewInstant("Volcanic Fallout", "{1}{R}{R}",
			NewSpellAbility(CompositeEffects(
				"2 damage to each creature and each player",
				DealDamageToAllCreatures(Fixed(2), AnyPermanent),
				DealDamageToPlayers(Fixed(2), SelectEachPlayer()),
			)),
			WithUncounterable(),
		)
	})

	// Voyage's End {1}{U}
	// Instant
	// Return target creature to its owner's hand. Scry 1.
	Register("Voyage's End", func() Card {
		return NewInstant("Voyage's End", "{1}{U}",
			NewTargetedSpell(TargetCreature(), ReturnToHandTarget(), Scry(Fixed(1))),
		)
	})

	// Whelming Wave {2}{U}{U}
	// Sorcery
	// Return all creatures to their owners' hands except for Krakens, Leviathans, Octopuses, and Serpents.
	Register("Whelming Wave", func() Card {
		return NewSorcery("Whelming Wave", "{2}{U}{U}",
			NewSpellAbility(FuncEffect(
				"bounce all creatures except Krakens, Leviathans, Octopuses, Serpents",
				EffectProperties{Outcome: OutcomeUnknown, Mass: true},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					exempt := func(p *Permanent) bool {
						for _, st := range []string{"Kraken", "Leviathan", "Octopus", "Serpent"} {
							if hasSubType(p.Card, st) {
								return true
							}
						}
						return false
					}
					var toBounce []*Permanent
					for _, perm := range g.FilterBattlefield(IsCreature) {
						if !exempt(perm) {
							toBounce = append(toBounce, perm)
						}
					}
					for _, perm := range toBounce {
						card := perm.Card
						ownerID := card.Owner()
						if ownerID == uuid.Nil {
							ownerID = perm.Controller
						}
						g.RemoveFromBattlefield(perm)
						if owner := g.GetPlayer(ownerID); owner != nil {
							owner.AddToHand(card)
						}
					}
					return nil
				},
			)),
		)
	})

	// Wildsize {2}{G}
	// Instant
	// Target creature gets +2/+2 and gains trample until end of turn.
	// Draw a card.
	Register("Wildsize", func() Card {
		return NewInstant("Wildsize", "{2}{G}",
			NewTargetedSpell(TargetCreature(), CompositeEffects(
				"+2/+2 and trample, draw a card",
				Boost(Fixed(2), Fixed(2)),
				GrantKeyword(Trample),
				drawSelfCard(1),
			)),
		)
	})

	// Winged Words {2}{U}
	// Sorcery
	// This spell costs {1} less to cast if you control a creature with flying.
	// Draw two cards.
	Register("Winged Words", func() Card {
		return NewSorcery("Winged Words", "{2}{U}",
			NewSpellAbility(drawSelfCard(2)),
			WithSelfCostReduction(
				FixedAmount(1),
				CondControlsMatching(HasKeywordFilter(Flying)),
			),
		)
	})

	// Wizard's Retort {1}{U}{U}
	// Instant
	// This spell costs {1} less to cast if you control a Wizard.
	// Counter target spell.
	Register("Wizard's Retort", func() Card {
		return NewInstant("Wizard's Retort", "{1}{U}{U}",
			NewTargetedSpell(TargetSpellOnStack(), CounterSpell()),
			WithSelfCostReduction(
				FixedAmount(1),
				CondControlsMatching(HasSubType("Wizard")),
			),
		)
	})

}

// gainControlUntilEOT registers a LayerControl continuous effect that switches
// the target's controller until end of turn.
func gainControlUntilEOT() Effect {
	return FuncEffect(
		"gain control until end of turn",
		EffectProperties{Outcome: OutcomeBenefit},
		func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
			if len(targets) == 0 {
				return nil
			}
			eff := TargetEffect(LayerControl, EndOfTurn, targets[0], func(g2 *Game, target *Permanent) error {
				target.Controller = controller
				return nil
			})
			eff.SetSourceID(sourceID)
			g.AddContinuousEffect(eff)
			g.ApplyContinuousEffects()
			return nil
		},
	)
}

// drawSelfCard returns an effect that has the spell controller draw N cards.
func drawSelfCard(n int) Effect {
	return FuncEffect(
		"draw cards",
		EffectProperties{Outcome: OutcomeBenefit, DrawCount: n},
		func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
			p := g.GetPlayer(controller)
			if p == nil {
				return nil
			}
			for i := 0; i < n; i++ {
				g.PlayerDrawCard(p)
			}
			return nil
		},
	)
}

// cantBlockThisTurn revokes CanBlock from targets[0] until end of turn.
func cantBlockThisTurn() Effect {
	return FuncEffect(
		"target creature can't block this turn",
		EffectProperties{Outcome: OutcomeDetriment},
		func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
			if len(targets) == 0 {
				return nil
			}
			eff := TargetEffect(LayerAbility, EndOfTurn, targets[0], func(g2 *Game, target *Permanent) error {
				g2.RevokeAttr(target.ID(), AttrCanBlock)
				return nil
			})
			eff.SetSourceID(sourceID)
			g.AddContinuousEffect(eff)
			g.ApplyContinuousEffects()
			return nil
		},
	)
}

// flickerEffect exiles target you-control creature and returns it; if spirit and
// withSpiritCounter is true, also adds a +1/+1 counter on return.
func flickerEffect(withSpiritCounter bool) Effect {
	return FuncEffect(
		"flicker target creature you control",
		EffectProperties{Outcome: OutcomeBenefit},
		func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
			if len(targets) == 0 {
				return nil
			}
			perm := g.FindPermanent(targets[0])
			if perm == nil {
				return nil
			}
			isSpirit := hasSubType(perm.Card, "Spirit")
			cardID := perm.Card.ID()
			g.ExilePermanent(perm)
			picked, ok := g.RemoveFromExile(cardID)
			if !ok {
				return nil
			}
			newPerm := g.PutOnBattlefield(picked, controller)
			if newPerm != nil && withSpiritCounter && isSpirit {
				newPerm.AddCounter(P1P1, 1)
				g.ApplyContinuousEffects()
			}
			return nil
		},
	)
}

// eachPlayerSacrificesNCreatures: each player sacrifices N creatures of their choice.
// Active player chooses first (CR 800.4 / APNAP).
func eachPlayerSacrificesNCreatures(n int) Effect {
	return FuncEffect(
		"each player sacrifices N creatures of their choice",
		EffectProperties{Outcome: OutcomeUnknown, Mass: true},
		func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
			ordered := apnapOrder(g)
			for _, pid := range ordered {
				p := g.GetPlayer(pid)
				if p == nil {
					continue
				}
				for i := 0; i < n; i++ {
					candidates := g.FilterBattlefield(And(ControlledBy(pid), IsCreature))
					if len(candidates) == 0 {
						break
					}
					chosen := p.ChoosePermanent(candidates, "sacrifice a creature", g)
					if chosen == nil {
						break
					}
					g.Sacrifice(chosen)
				}
			}
			return nil
		},
	)
}

// apnapOrder returns player IDs in active-player-then-non-active order.
func apnapOrder(g *Game) []uuid.UUID {
	all := g.AllPlayers()
	out := make([]uuid.UUID, 0, len(all))
	if ap := g.ActivePlayerObj(); ap != nil {
		out = append(out, ap.PlayerID())
		for _, p := range all {
			if p.PlayerID() != ap.PlayerID() {
				out = append(out, p.PlayerID())
			}
		}
		return out
	}
	for _, p := range all {
		out = append(out, p.PlayerID())
	}
	return out
}

// hasSubType reports whether the card has the given subtype.
func hasSubType(c Card, st string) bool {
	for _, s := range c.SubTypes() {
		if s == st {
			return true
		}
	}
	return false
}

// millSelf mills n cards from the player's library to their graveyard.
func millSelf(g *Game, p Player, n int) {
	lib := p.Library()
	for i := 0; i < n && len(lib) > 0; i++ {
		top := lib[0]
		lib = lib[1:]
		p.AddToGraveyard(top)
	}
	p.SetLibrary(lib)
}

// creatureSpellFilter matches creature spells on the stack.
var creatureSpellFilter = NewCardFilter("creature spell", func(c Card) bool {
	return c.HasType(TypeCreature)
})

// registerDoublecastDelayedTrigger sets up a one-shot delayed trigger that
// fires on the controller's next spell cast. If the cast spell is an instant
// or sorcery, it is copied with new targets allowed (CR 706.10c). Otherwise
// the trigger re-arms so the next-cast hook still applies to the next
// instant/sorcery cast this turn (per Oracle "next ... instant or sorcery").
func registerDoublecastDelayedTrigger(g *Game, sourceID, controller uuid.UUID) {
	dt := &DelayedTrigger{
		EventType:     EvtSpellCast,
		MatchPlayerID: controller,
		SourceID:      sourceID,
		Controller:    controller,
		Effects: []Effect{FuncEffect(
			"copy next instant/sorcery spell you cast this turn (or re-arm)",
			EffectProperties{Outcome: OutcomeBenefit},
			func(g2 *Game, src, ctrl uuid.UUID, _ []uuid.UUID) error {
				top := g2.Stack().Peek()
				if top != nil && !top.IsAbility && top.Card != nil && top.Controller == ctrl &&
					(top.Card.HasType(TypeInstant) || top.Card.HasType(TypeSorcery)) {
					g2.CopySpellOnStack(top.SourceID, ctrl, true)
					return nil
				}
				registerDoublecastDelayedTrigger(g2, src, ctrl)
				return nil
			},
		)},
	}
	g.RegisterDelayedTrigger(dt)
}

// instantOrSorcerySpellFilter matches instant or sorcery spells on the stack.
var instantOrSorcerySpellFilter = NewCardFilter("instant or sorcery spell", func(c Card) bool {
	return c.HasType(TypeInstant) || c.HasType(TypeSorcery)
})

// toughness4OrGreater matches creatures with current toughness >= 4.
var toughness4OrGreater = NewPermanentFilter("toughness 4 or greater", func(p *Permanent, g *Game) bool {
	return p.CurrentToughness(g) >= 4
})
