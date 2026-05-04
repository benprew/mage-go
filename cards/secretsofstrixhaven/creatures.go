package secretsofstrixhaven

import (
	"github.com/google/uuid"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage"
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// basicLandNamesSOS is the set of basic land card names used for search filters.
var basicLandNamesSOS = map[string]bool{
	"Plains": true, "Island": true, "Swamp": true, "Mountain": true, "Forest": true,
}

func isBasicLandCardSOS(c Card) bool { return basicLandNamesSOS[c.Name()] }

// searchBasicLandToBattlefieldTappedSOS searches the controller's library for
// a basic land card, puts it onto the battlefield tapped, then shuffles.
func searchBasicLandToBattlefieldTappedSOS(g *Game, controller uuid.UUID) {
	p := g.GetPlayer(controller)
	if p == nil {
		return
	}
	lib := p.Library()
	var candidates []Card
	for _, c := range lib {
		if isBasicLandCardSOS(c) {
			candidates = append(candidates, c)
		}
	}
	if len(candidates) == 0 {
		p.ShuffleLibrary()
		return
	}
	chosen := p.ChooseCardFromLibrary(candidates, "search for basic land", g)
	if chosen == nil {
		p.ShuffleLibrary()
		return
	}
	newLib := make([]Card, 0, len(lib)-1)
	for _, c := range lib {
		if c.ID() != chosen.ID() {
			newLib = append(newLib, c)
		}
	}
	p.SetLibrary(newLib)
	p.ShuffleLibrary()
	perm := g.PutOnBattlefield(chosen, controller)
	if perm != nil {
		g.TapPermanent(perm)
	}
}

// searchBasicLandToHandSOS searches the controller's library for a basic land
// card and puts it into hand, then shuffles.
func searchBasicLandToHandSOS(g *Game, controller uuid.UUID) {
	p := g.GetPlayer(controller)
	if p == nil {
		return
	}
	lib := p.Library()
	var candidates []Card
	for _, c := range lib {
		if isBasicLandCardSOS(c) {
			candidates = append(candidates, c)
		}
	}
	if len(candidates) == 0 {
		p.ShuffleLibrary()
		return
	}
	chosen := p.ChooseCardFromLibrary(candidates, "search for basic land", g)
	if chosen == nil {
		p.ShuffleLibrary()
		return
	}
	newLib := make([]Card, 0, len(lib)-1)
	for _, c := range lib {
		if c.ID() != chosen.ID() {
			newLib = append(newLib, c)
		}
	}
	p.SetLibrary(newLib)
	p.ShuffleLibrary()
	p.AddToHand(chosen)
}

func init() {
	registerCreatures()
}

// Used for the "Repartee" ability word (SOS): "Whenever you cast an instant or
// sorcery spell that targets a creature, …"
func reparteeCondition(evt *GameEvent, g GameReader, _, controllerID uuid.UUID) bool {
	if evt.PlayerID != controllerID {
		return false
	}
	card := g.FindCardAnywhere(evt.SourceID)
	if card == nil {
		return false
	}
	if !card.HasType(TypeInstant) && !card.HasType(TypeSorcery) {
		return false
	}
	obj := g.FindStackObject(evt.SourceID)
	if obj == nil {
		return false
	}
	for _, tid := range obj.Targets {
		if perm := g.FindPermanent(tid); perm != nil && perm.HasAttr(AttrIsCreature) {
			return true
		}
	}
	return false
}

func registerCreatures() {

	// ===== WHITE CREATURES =====

	// Ascendant Dustspeaker {4}{W}
	// Creature — Orc Cleric
	// 3/4
	// Flying
	// When this creature enters, put a +1/+1 counter on another target creature you control.
	// At the beginning of combat on your turn, exile up to one target card from a graveyard.
	Register("Ascendant Dustspeaker", func() Card {
		return NewCreature("Ascendant Dustspeaker", "{4}{W}", 3, 4,
			WithSubTypes("Orc", "Cleric"),
			WithKeyword(Flying),
			// When this creature enters, put a +1/+1 counter on another target creature you control.
			WithAbility(EntersBattlefieldTrigger(
				AddCounters(P1P1, Fixed(1)),
				false,
			).AddTarget(TargetAnotherCreatureYouControl())),
			// At the beginning of combat on your turn, exile up to one target card from a graveyard.
			WithAbility(NewTriggered(EvtBeginCombat, true,
				FuncEffect("exile up to one target card from a graveyard",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						// Collect all graveyard cards
						var candidates []Card
						for _, pl := range g.AllPlayers() {
							candidates = append(candidates, pl.Graveyard()...)
						}
						if len(candidates) == 0 {
							return nil
						}
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						chosen := p.ChooseCardFromLibrary(candidates, "exile up to one card from a graveyard", g)
						if chosen == nil {
							return nil
						}
						// Remove from graveyard and exile
						for _, pl := range g.AllPlayers() {
							if removed, ok := pl.RemoveFromGraveyard(chosen.ID()); ok {
								g.ExileCard(removed, sourceID)
								break
							}
						}
						return nil
					},
				)).SetConditionData(EventPlayerIsController{})),
		)
	})

	// Eager Glyphmage {3}{W}
	// Creature — Cat Cleric
	// 3/3
	// When this creature enters, create a 1/1 white and black Inkling creature token with flying.
	Register("Eager Glyphmage", func() Card {
		return NewCreature("Eager Glyphmage", "{3}{W}", 3, 3,
			WithSubTypes("Cat", "Cleric"),
			WithAbility(EntersBattlefieldTrigger(
				CreateColoredToken("Inkling Token", 1, 1,
					[]Color{White, Black},
					[]CardType{TypeCreature},
					[]string{"Inkling"},
					Flying,
				),
				false,
			)),
		)
	})

	// Elite Interceptor // Rejoinder {W} // {1}{W}
	// Creature — Human Wizard // Sorcery
	// 1/2
	// TODO: implement
	Register("Elite Interceptor // Rejoinder", func() Card {
		return NewCreature("Elite Interceptor // Rejoinder", "{W} // {1}{W}", 1, 2,
			WithSubTypes("Human", "Wizard", "//", "Sorcery"),
		)
	})

	// Emeritus of Truce // Swords to Plowshares {1}{W}{W} // {W}
	// Creature — Cat Cleric // Instant
	// 3/3
	// TODO: implement
	Register("Emeritus of Truce // Swords to Plowshares", func() Card {
		return NewCreature("Emeritus of Truce // Swords to Plowshares", "{1}{W}{W} // {W}", 3, 3,
			WithSubTypes("Cat", "Cleric", "//", "Instant"),
		)
	})

	// Ennis, Debate Moderator {1}{W}
	// Legendary Creature — Human Cleric
	// 1/1
	// When Ennis enters, exile up to one other target creature you control. Return that card to the battlefield under its owner's control at the beginning of the next end step.
	// At the beginning of your end step, if one or more cards were put into exile this turn, put a +1/+1 counter on Ennis.
	Register("Ennis, Debate Moderator", func() Card {
		return NewCreature("Ennis, Debate Moderator", "{1}{W}", 1, 1,
			WithSubTypes("Human", "Cleric"),
			WithSuperTypes(SuperLegendary),
			// When Ennis enters, exile up to one other target creature you control. Return at next end step.
			WithAbility(EntersBattlefieldTrigger(
				ExileTargetReturnAtEndStep(nil),
				false,
			).AddTarget(TargetAnotherCreatureYouControl())),
			// At the beginning of your end step, if one or more cards were put into exile this turn,
			// put a +1/+1 counter on Ennis.
			// XXX: no per-turn exile tracking in engine; this ability is not implemented.
		)
	})

	// Honorbound Page // Forum's Favor {3}{W} // {W}
	// Creature — Cat Cleric // Sorcery
	// 3/3
	// TODO: implement
	Register("Honorbound Page // Forum's Favor", func() Card {
		return NewCreature("Honorbound Page // Forum's Favor", "{3}{W} // {W}", 3, 3,
			WithSubTypes("Cat", "Cleric", "//", "Sorcery"),
		)
	})

	// Informed Inkwright {1}{W}
	// Creature — Human Wizard
	// 2/2
	// Vigilance
	// Repartee — Whenever you cast an instant or sorcery spell that targets a creature,
	// create a 1/1 white and black Inkling creature token with flying.
	Register("Informed Inkwright", func() Card {
		return NewCreature("Informed Inkwright", "{1}{W}", 2, 2,
			WithSubTypes("Human", "Wizard"),
			WithKeyword(Vigilance),
			WithAbility(NewTriggered(EvtSpellCast, false,
				CreateColoredToken("Inkling", 1, 1, []Color{White, Black}, []CardType{TypeCreature}, []string{"Inkling"}, Flying),
			).SetCondition(reparteeCondition)),
		)
	})
	// Inkshape Demonstrator {3}{W}
	// Creature — Elephant Cleric
	// 3/4
	// Ward {2} (Whenever this creature becomes the target of a spell or ability an
	// opponent controls, counter it unless that player pays {2}.)
	// Repartee — Whenever you cast an instant or sorcery spell that targets a creature,
	// this creature gets +1/+0 and gains lifelink until end of turn.
	Register("Inkshape Demonstrator", func() Card {
		wardEffect := FuncEffect(
			"counter that spell or ability unless its controller pays {2}",
			EffectProperties{Outcome: OutcomeBenefit},
			func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
				// targets[0] = the targeted object (this creature, auto-bound by EvtBecomesTarget)
				// targets[1] = the spell/ability source
				if len(targets) < 2 || targets[1] == uuid.Nil {
					return nil
				}
				// The controller of the targeting spell/ability is the opponent.
				obj := g.FindStackObject(targets[1])
				var opponentID uuid.UUID
				if obj != nil {
					opponentID = obj.Controller
				} else {
					// Activated ability: find permanent and get its controller.
					perm := g.FindPermanent(targets[1])
					if perm != nil {
						opponentID = perm.Controller
					}
				}
				if opponentID == uuid.Nil {
					return nil
				}
				opponent := g.GetPlayer(opponentID)
				if opponent == nil {
					return nil
				}
				// Give the opponent the option to pay {2}; if they decline, counter.
				paid := false
				if ManaCostOf("{2}").CanPay(sourceID, opponentID, g) {
					if opponent.ChooseMayAbility("pay {2} to prevent Ward from countering") {
						if err := ManaCostOf("{2}").Pay(sourceID, opponentID, g); err == nil {
							paid = true
						}
					}
				}
				if !paid {
					g.CounterSpellOnStack(targets[1])
				}
				return nil
			},
		)
		return NewCreature("Inkshape Demonstrator", "{3}{W}", 3, 4,
			WithSubTypes("Elephant", "Cleric"),
			WithAbility(NewTriggered(EvtBecomesTarget, false, wardEffect).
				SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
					EventTargetIsSelf{},
					EventPlayerIsNotController{},
				}})),
			WithAbility(NewTriggered(EvtSpellCast, false,
				CompositeEffects("gets +1/+0 and gains lifelink until end of turn",
					Boost(Fixed(1), Fixed(0)).Targeting(ToSource()).Until(EndOfTurn),
					GrantKeyword(Lifelink).Targeting(ToSource()).Until(EndOfTurn),
				),
			).SetCondition(reparteeCondition)),
		)
	})
	// Joined Researchers // Secret Rendezvous {1}{W} // {1}{W}{W}
	// Creature — Human Cleric Wizard // Sorcery
	// 2/2
	// TODO: implement
	Register("Joined Researchers // Secret Rendezvous", func() Card {
		return NewCreature("Joined Researchers // Secret Rendezvous", "{1}{W} // {1}{W}{W}", 2, 2,
			WithSubTypes("Human", "Cleric", "Wizard", "//", "Sorcery"),
		)
	})

	// Owlin Historian {2}{W}
	// Creature — Bird Cleric
	// 2/3
	// Flying
	// When this creature enters, surveil 1. (Look at the top card of your library. You may put it into your graveyard.)
	// Whenever one or more cards leave your graveyard, this creature gets +1/+1 until end of turn.
	Register("Owlin Historian", func() Card {
		return NewCreature("Owlin Historian", "{2}{W}", 2, 3,
			WithSubTypes("Bird", "Cleric"),
			WithKeyword(Flying),
			// When this creature enters, surveil 1.
			WithAbility(EntersBattlefieldTrigger(surveilEffect(1), false)),
			// XXX: "Whenever one or more cards leave your graveyard, this creature gets +1/+1 until end of turn."
			// The engine has no EvtLeaveGraveyard event; this trigger cannot be implemented.
		)
	})

	// Quill-Blade Laureate // Twofold Intent {1}{W} // {1}{W}
	// Creature — Human Cleric // Sorcery
	// 1/1
	// TODO: implement
	Register("Quill-Blade Laureate // Twofold Intent", func() Card {
		return NewCreature("Quill-Blade Laureate // Twofold Intent", "{1}{W} // {1}{W}", 1, 1,
			WithSubTypes("Human", "Cleric", "//", "Sorcery"),
		)
	})

	// Rehearsed Debater {2}{W}
	// Creature — Djinn Bard
	// 3/3
	// Vigilance
	// Repartee — Whenever you cast an instant or sorcery spell that targets a creature, this creature gets +1/+1 until end of turn.
	Register("Rehearsed Debater", func() Card {
		return NewCreature("Rehearsed Debater", "{2}{W}", 3, 3,
			WithSubTypes("Djinn", "Bard"),
			WithKeyword(Vigilance),
			WithAbility(NewTriggered(EvtSpellCast, false,
				Boost(Fixed(1), Fixed(1)).Targeting(ToSource()).Until(EndOfTurn),
			).SetCondition(reparteeCondition)),
		)
	})

	// Shattered Acolyte {1}{W}
	// Creature — Dwarf Warlock
	// 2/2
	// Lifelink
	// {1}, Sacrifice this creature: Destroy target artifact or enchantment.
	Register("Shattered Acolyte", func() Card {
		return NewCreature("Shattered Acolyte", "{1}{W}", 2, 2,
			WithSubTypes("Dwarf", "Warlock"),
			WithKeyword(Lifelink),
			WithActivatedAbility(
				DestroyTarget(),
				ManaCostOf("{1}"),
				WithCost(SacrificeSourceCost()),
				WithTarget(TargetArtifactOrEnchantment()),
			),
		)
	})

	// Soaring Stoneglider {2}{W}
	// Creature — Elephant Cleric
	// 4/3
	// As an additional cost to cast this spell, exile two cards from your graveyard or pay {1}{W}.
	// Flying, vigilance
	Register("Soaring Stoneglider", func() Card {
		return NewCreature("Soaring Stoneglider", "{2}{W}", 4, 3,
			WithSubTypes("Elephant", "Cleric"),
			WithAdditionalCost(EitherCost(ExileFromGraveyardCost(2), ManaCostOf("{1}{W}"))),
			WithKeyword(Flying),
			WithKeyword(Vigilance),
		)
	})

	// Spiritcall Enthusiast // Scrollboost {2}{W} // {1}{W}
	// Creature — Cat Cleric // Sorcery
	// 3/3
	// TODO: implement
	Register("Spiritcall Enthusiast // Scrollboost", func() Card {
		return NewCreature("Spiritcall Enthusiast // Scrollboost", "{2}{W} // {1}{W}", 3, 3,
			WithSubTypes("Cat", "Cleric", "//", "Sorcery"),
		)
	})

	// Stirring Hopesinger {2}{W}
	// Creature — Bird Bard
	// 1/3
	// Flying, lifelink
	// Repartee — Whenever you cast an instant or sorcery spell that targets a creature, put a +1/+1 counter on each creature you control.
	// TODO: implement
	Register("Stirring Hopesinger", func() Card {
		return NewCreature("Stirring Hopesinger", "{2}{W}", 1, 3,
			WithSubTypes("Bird", "Bard"),
		)
	})

	// Stone Docent {1}{W}
	// Creature — Spirit Chimera
	// 3/1
	// {W}, Exile this card from your graveyard: You gain 2 life. Surveil 1. Activate only as a sorcery. (Look at the top card of your library. You may put it into your graveyard.)
	Register("Stone Docent", func() Card {
		return NewCreature("Stone Docent", "{1}{W}", 3, 1,
			WithSubTypes("Spirit", "Chimera"),
			// {W}, Exile this card from your graveyard: You gain 2 life. Surveil 1. Activate only as a sorcery.
			WithGraveyardActivatedAbility(
				CompositeEffects("gain 2 life and surveil 1",
					GainLife(2),
					surveilEffect(1),
				),
				ManaCostOf("{W}"),
				WithCost(ExileSelfFromGraveyardCost()),
			),
		)
	})

	// Summoned Dromedary {3}{W}
	// Creature — Spirit Camel
	// 4/3
	// Vigilance
	// {1}{W}: Return this card from your graveyard to your hand. Activate only as a sorcery.
	Register("Summoned Dromedary", func() Card {
		return NewCreature("Summoned Dromedary", "{3}{W}", 4, 3,
			WithSubTypes("Spirit", "Camel"),
			WithKeyword(Vigilance),
			// {1}{W}: Return this card from your graveyard to your hand. Activate only as a sorcery.
			WithGraveyardActivatedAbility(
				ReturnSourceToHand(),
				ManaCostOf("{1}{W}"),
			),
		)
	})

	// ===== BLUE CREATURES =====

	// Campus Composer // Aqueous Aria {3}{U} // {4}{U}
	// Creature — Merfolk Bard // Sorcery
	// 3/4
	// TODO: implement
	Register("Campus Composer // Aqueous Aria", func() Card {
		return NewCreature("Campus Composer // Aqueous Aria", "{3}{U} // {4}{U}", 3, 4,
			WithSubTypes("Merfolk", "Bard", "//", "Sorcery"),
		)
	})

	// Deluge Virtuoso {2}{U}
	// Creature — Human Wizard
	// 2/2
	// When this creature enters, tap target creature an opponent controls and put a stun counter on it. (If a permanent with a stun counter would become untapped, remove one from it instead.)
	// Opus — Whenever you cast an instant or sorcery spell, this creature gets +1/+1 until end of turn. If five or more mana was spent to cast that spell, this creature gets +2/+2 until end of turn instead.
	Register("Deluge Virtuoso", func() Card {
		isInstantOrSorcery := NewCardFilter("instant or sorcery", func(c Card) bool {
			return c.HasType(TypeInstant) || c.HasType(TypeSorcery)
		})
		return NewCreature("Deluge Virtuoso", "{2}{U}", 2, 2,
			WithSubTypes("Human", "Wizard"),
			// ETB: tap target creature an opponent controls and put a stun counter on it.
			// XXX: Stun counter (untap-prevention counter type) not in engine; tap is implemented but stun counter is not.
			WithAbility(EntersBattlefieldTrigger(
				FuncEffect(
					"tap target creature an opponent controls",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						opp := g.GetOpponent(controller)
						if opp == nil {
							return nil
						}
						candidates := g.FilterBattlefield(And(
							ControlledBy(opp.PlayerID()),
							IsCreature,
						))
						if len(candidates) == 0 {
							return nil
						}
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						chosen := p.ChoosePermanent(candidates, "tap target creature an opponent controls", g)
						if chosen == nil {
							return nil
						}
						g.TapPermanent(chosen)
						// XXX: put a stun counter on it — engine lacks Stun CounterType
						return nil
					},
				),
				false,
			)),
			// Opus — Whenever you cast an instant or sorcery spell, this creature gets +1/+1 until end
			// of turn. If five or more mana was spent to cast that spell, this creature gets +2/+2
			// until end of turn instead.
			WithAbility(WheneverYouCastSpellTrigger(
				OpusEffect(
					"this creature gets +1/+1 (or +2/+2 if 5+ mana spent) until end of turn",
					func(g *Game, sourceID, controller uuid.UUID, manaSpent int) error {
						p, t := 1, 1
						if manaSpent >= 5 {
							p, t = 2, 2
						}
						ce := TemporaryBoost(sourceID, p, t)
						ce.SetSourceID(sourceID)
						g.AddContinuousEffect(ce)
						return nil
					},
				),
				false,
				isInstantOrSorcery,
			)),
		)
	})

	// Emeritus of Ideation // Ancestral Recall {3}{U}{U} // {U}
	// Creature — Human Wizard // Instant
	// 5/5
	// TODO: implement
	Register("Emeritus of Ideation // Ancestral Recall", func() Card {
		return NewCreature("Emeritus of Ideation // Ancestral Recall", "{3}{U}{U} // {U}", 5, 5,
			WithSubTypes("Human", "Wizard", "//", "Instant"),
		)
	})

	// Encouraging Aviator // Jump {2}{U} // {U}
	// Creature — Bird Wizard // Instant
	// 2/3
	// TODO: implement
	Register("Encouraging Aviator // Jump", func() Card {
		return NewCreature("Encouraging Aviator // Jump", "{2}{U} // {U}", 2, 3,
			WithSubTypes("Bird", "Wizard", "//", "Instant"),
		)
	})

	// Exhibition Tidecaller {U}
	// Creature — Djinn Wizard
	// 0/2
	// Opus — Whenever you cast an instant or sorcery spell, target player mills three cards. If five or more mana was spent to cast that spell, that player mills ten cards instead.
	Register("Exhibition Tidecaller", func() Card {
		isInstantOrSorcery := NewCardFilter("instant or sorcery", func(c Card) bool {
			return c.HasType(TypeInstant) || c.HasType(TypeSorcery)
		})
		return NewCreature("Exhibition Tidecaller", "{U}", 0, 2,
			WithSubTypes("Djinn", "Wizard"),
			// Opus — Whenever you cast an instant or sorcery spell, target player mills
			// three cards. If five or more mana was spent to cast that spell, that player
			// mills ten cards instead.
			WithAbility(WheneverYouCastSpellTrigger(
				FuncEffect(
					"target player mills 3; if 5+ mana spent, mills 10 instead",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						targetPlayerID := targets[0]
						tp := g.GetPlayer(targetPlayerID)
						if tp == nil {
							return nil
						}
						var manaSpent int
						objs := g.GetStack().Objects()
						for i := len(objs) - 1; i >= 0; i-- {
							if !objs[i].IsAbility {
								manaSpent = ManaSpentToCast(objs[i])
								break
							}
						}
						amount := 3
						if manaSpent >= 5 {
							amount = 10
						}
						amount = g.ApplyMillModifiers(targetPlayerID, amount)
						lib := tp.Library()
						for i := 0; i < amount && len(lib) > 0; i++ {
							card := lib[len(lib)-1]
							lib = lib[:len(lib)-1]
							tp.AddToGraveyard(card)
						}
						tp.SetLibrary(lib)
						return nil
					},
				),
				false,
				isInstantOrSorcery,
			).AddTarget(TargetPlayer())),
		)
	})

	// Harmonized Trio // Brainstorm {U} // {U}
	// Creature — Merfolk Bard Wizard // Instant
	// 1/1
	// TODO: implement
	Register("Harmonized Trio // Brainstorm", func() Card {
		return NewCreature("Harmonized Trio // Brainstorm", "{U} // {U}", 1, 1,
			WithSubTypes("Merfolk", "Bard", "Wizard", "//", "Instant"),
		)
	})

	// Hydro-Channeler {1}{U}
	// Creature — Merfolk Wizard
	// 1/3
	// {T}: Add {U}. Spend this mana only to cast an instant or sorcery spell.
	// {1}, {T}: Add one mana of any color. Spend this mana only to cast an instant or sorcery spell.
	Register("Hydro-Channeler", func() Card {
		return NewCreature("Hydro-Channeler", "{1}{U}", 1, 3,
			WithSubTypes("Merfolk", "Wizard"),
			// {T}: Add {U}.
			// XXX: "Spend this mana only to cast an instant or sorcery spell" restriction not enforced (engine has no mana restriction tagging).
			WithManaAbility(Blue),
			// {1}, {T}: Add one mana of any color.
			// XXX: "Spend this mana only to cast an instant or sorcery spell" restriction not enforced.
			WithActivatedAbility(
				AddAnyMana(1, Colorless),
				ManaCostOf("{1}"),
				WithCost(TapSourceCost()),
			),
		)
	})

	// Jadzi, Steward of Fate // Oracle's Gift {2}{U} // {X}{X}{U}
	// Legendary Creature — Human Wizard // Sorcery
	// 2/4
	// TODO: implement
	Register("Jadzi, Steward of Fate // Oracle's Gift", func() Card {
		return NewCreature("Jadzi, Steward of Fate // Oracle's Gift", "{2}{U} // {X}{X}{U}", 2, 4,
			WithSubTypes("Human", "Wizard", "//", "Sorcery"),
			WithSuperTypes(SuperLegendary),
		)
	})

	// Landscape Painter // Vibrant Idea {1}{U} // {4}{U}
	// Creature — Merfolk Wizard // Sorcery
	// 2/1
	// TODO: implement
	Register("Landscape Painter // Vibrant Idea", func() Card {
		return NewCreature("Landscape Painter // Vibrant Idea", "{1}{U} // {4}{U}", 2, 1,
			WithSubTypes("Merfolk", "Wizard", "//", "Sorcery"),
		)
	})

	// Matterbending Mage {2}{U}
	// Creature — Human Wizard
	// 2/2
	// When this creature enters, return up to one other target creature to its owner's hand.
	// Whenever you cast a spell with {X} in its mana cost, this creature can't be blocked this turn.
	Register("Matterbending Mage", func() Card {
		hasXFilter := NewCardFilter("spell with {X} in mana cost", func(c Card) bool {
			return c.ManaCost().HasX
		})
		return NewCreature("Matterbending Mage", "{2}{U}", 2, 2,
			WithSubTypes("Human", "Wizard"),
			// When this creature enters, return up to one other target creature to its owner's hand.
			// "Up to one" is optional (0 or 1). The trigger is mandatory but picking 0 is legal.
			// Uses ChoosePermanent so the test harness can script the choice via g.ChoosePermanent.
			// Offering all creatures (including self) as candidates; choosing self skips the bounce.
			WithAbility(EntersBattlefieldTrigger(
				FuncEffect("return up to one other target creature to its owner's hand",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						// All creatures on the battlefield. Choosing self = "skip bounce" (up to one).
						candidates := g.FilterBattlefield(IsCreature)
						if len(candidates) == 0 {
							return nil
						}
						chosen := p.ChoosePermanent(candidates, "return up to one other target creature to its owner's hand", g)
						if chosen == nil || chosen.ID() == sourceID {
							return nil
						}
						owner := g.GetPlayer(chosen.Card.Owner())
						if owner == nil {
							return nil
						}
						g.RemoveFromBattlefield(chosen)
						owner.AddToHand(chosen.Card)
						return nil
					},
				),
				false,
			)),
			// Whenever you cast a spell with {X} in its mana cost, this creature can't be blocked this turn.
			WithAbility(WheneverYouCastSpellTrigger(
				FuncEffect("this creature can't be blocked this turn",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						src := g.FindPermanent(sourceID)
						if src == nil {
							return nil
						}
						ce := TemporaryKeyword(sourceID, UnblockableKW)
						ce.SetSourceID(sourceID)
						g.AddContinuousEffect(ce)
						g.ApplyContinuousEffects()
						return nil
					},
				),
				false,
				hasXFilter,
			)),
		)
	})

	// Muse Seeker {1}{U}
	// Creature — Elf Wizard
	// 1/2
	// Opus — Whenever you cast an instant or sorcery spell, draw a card. Then discard a card unless five or more mana was spent to cast that spell.
	Register("Muse Seeker", func() Card {
		isInstantOrSorcery := NewCardFilter("instant or sorcery", func(c Card) bool {
			return c.HasType(TypeInstant) || c.HasType(TypeSorcery)
		})
		return NewCreature("Muse Seeker", "{1}{U}", 1, 2,
			WithSubTypes("Elf", "Wizard"),
			// Opus — Whenever you cast an instant or sorcery spell, draw a card. Then discard
			// a card unless five or more mana was spent to cast that spell.
			WithAbility(WheneverYouCastSpellTrigger(
				OpusEffect("draw a card, then discard a card unless 5+ mana spent",
					func(g *Game, sourceID, controller uuid.UUID, manaSpent int) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						g.PlayerDrawCard(p)
						if manaSpent >= 5 {
							return nil
						}
						hand := p.Hand()
						if len(hand) == 0 {
							return nil
						}
						chosen := p.ChooseCardsFromHand(1, "discard a card", g)
						if len(chosen) == 0 {
							return nil
						}
						g.PlayerDiscard(p, chosen[0].ID())
						return nil
					},
				),
				false,
				isInstantOrSorcery,
			)),
		)
	})
	// Orysa, Tide Choreographer {4}{U}
	// Legendary Creature — Merfolk Bard
	// 2/2
	// This spell costs {3} less to cast if creatures you control have total toughness 10 or greater.
	// When Orysa enters, draw two cards.
	Register("Orysa, Tide Choreographer", func() Card {
		totalToughnessGTE10 := func(g *Game, controller uuid.UUID, _ Card, _ uuid.UUID) bool {
			total := 0
			for _, perm := range g.AllBattlefield() {
				if perm.Controller == controller && perm.HasAttr(AttrIsCreature) {
					total += perm.CurrentToughness(g)
				}
			}
			return total >= 10
		}
		return NewCreature("Orysa, Tide Choreographer", "{4}{U}", 2, 2,
			WithSubTypes("Merfolk", "Bard"),
			WithSuperTypes(SuperLegendary),
			// This spell costs {3} less to cast if creatures you control have total toughness 10 or greater.
			WithSelfCostReduction(FixedAmount(3), totalToughnessGTE10),
			// When Orysa enters, draw two cards.
			WithAbility(EntersBattlefieldTrigger(DrawCards(Fixed(2)), false)),
		)
	})

	// Pensive Professor {1}{U}{U}
	// Creature — Human Wizard
	// 0/2
	// Increment (Whenever you cast a spell, if the amount of mana you spent is greater than this creature's power or toughness, put a +1/+1 counter on this creature.)
	// Whenever one or more +1/+1 counters are put on this creature, draw a card.
	// TODO: implement
	Register("Pensive Professor", func() Card {
		return NewCreature("Pensive Professor", "{1}{U}{U}", 0, 2,
			WithSubTypes("Human", "Wizard"),
		)
	})

	// Skycoach Conductor // All Aboard {2}{U} // {U}
	// Creature — Bird Pilot // Instant
	// 2/3
	// TODO: implement
	Register("Skycoach Conductor // All Aboard", func() Card {
		return NewCreature("Skycoach Conductor // All Aboard", "{2}{U} // {U}", 2, 3,
			WithSubTypes("Bird", "Pilot", "//", "Instant"),
		)
	})

	// Spellbook Seeker // Careful Study {3}{U} // {U}
	// Creature — Bird Wizard // Sorcery
	// 3/3
	// TODO: implement
	Register("Spellbook Seeker // Careful Study", func() Card {
		return NewCreature("Spellbook Seeker // Careful Study", "{3}{U} // {U}", 3, 3,
			WithSubTypes("Bird", "Wizard", "//", "Sorcery"),
		)
	})

	// Tester of the Tangential {1}{U}
	// Creature — Djinn Wizard
	// 1/1
	// Increment (Whenever you cast a spell, if the amount of mana you spent is greater than this creature's power or toughness, put a +1/+1 counter on this creature.)
	// At the beginning of combat on your turn, you may pay {X}. When you do, move X +1/+1 counters from this creature onto another target creature.
	// TODO: implement
	Register("Tester of the Tangential", func() Card {
		return NewCreature("Tester of the Tangential", "{1}{U}", 1, 1,
			WithSubTypes("Djinn", "Wizard"),
		)
	})

	// Textbook Tabulator {2}{U}
	// Creature — Frog Wizard
	// 0/3
	// Increment (Whenever you cast a spell, if the amount of mana you spent is greater than this creature's power or toughness, put a +1/+1 counter on this creature.)
	// When this creature enters, surveil 2. (Look at the top two cards of your library, then put any number of them into your graveyard and the rest on top of your library in any order.)
	// TODO: implement
	Register("Textbook Tabulator", func() Card {
		return NewCreature("Textbook Tabulator", "{2}{U}", 0, 3,
			WithSubTypes("Frog", "Wizard"),
		)
	})

	// ===== BLACK CREATURES =====

	// Adventurous Eater // Have a Bite {2}{B} // {B}
	// Creature — Human Warlock // Sorcery
	// 3/2
	// TODO: implement
	Register("Adventurous Eater // Have a Bite", func() Card {
		return NewCreature("Adventurous Eater // Have a Bite", "{2}{B} // {B}", 3, 2,
			WithSubTypes("Human", "Warlock", "//", "Sorcery"),
		)
	})

	// Arnyn, Deathbloom Botanist {2}{B}
	// Legendary Creature — Vampire Druid
	// 2/2
	// Deathtouch
	// Whenever a creature you control with power or toughness 1 or less dies, target opponent loses 2 life and you gain 2 life.
	Register("Arnyn, Deathbloom Botanist", func() Card {
		return NewCreature("Arnyn, Deathbloom Botanist", "{2}{B}", 2, 2,
			WithSubTypes("Vampire", "Druid"),
			WithSuperTypes(SuperLegendary),
			WithKeyword(Deathtouch),
			// Whenever a creature you control with power or toughness 1 or less dies,
			// target opponent loses 2 life and you gain 2 life.
			WithAbility(DiesCreatureTrigger(
				FuncEffect("opponent loses 2, you gain 2",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						view := g.LookupObject(targets[0])
						if view == nil {
							return nil
						}
						// Only fire if dead creature had power or toughness 1 or less.
						if view.ViewPower() > 1 && view.ViewToughness() > 1 {
							return nil
						}
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						for _, opp := range g.AllPlayers() {
							if opp.PlayerID() != controller {
								g.PlayerLoseLife(opp, 2)
							}
						}
						g.PlayerGainLife(p, 2)
						return nil
					},
				),
				false,
				AnyPermanent,
			)),
		)
	})

	// Burrog Banemaker {B}
	// Creature — Frog Warlock
	// 1/1
	// Deathtouch
	// {1}{B}: This creature gets +1/+1 until end of turn.
	Register("Burrog Banemaker", func() Card {
		return NewCreature("Burrog Banemaker", "{B}", 1, 1,
			WithSubTypes("Frog", "Warlock"),
			WithKeyword(Deathtouch),
			// {1}{B}: This creature gets +1/+1 until end of turn.
			WithActivatedAbility(
				Boost(Fixed(1), Fixed(1)).Targeting(ToSource()),
				ManaCostOf("{1}{B}"),
			),
		)
	})

	// Cheerful Osteomancer // Raise Dead {3}{B} // {B}
	// Creature — Orc Warlock // Sorcery
	// 4/2
	// TODO: implement
	Register("Cheerful Osteomancer // Raise Dead", func() Card {
		return NewCreature("Cheerful Osteomancer // Raise Dead", "{3}{B} // {B}", 4, 2,
			WithSubTypes("Orc", "Warlock", "//", "Sorcery"),
		)
	})

	// Emeritus of Woe // Demonic Tutor {3}{B} // {1}{B}
	// Creature — Vampire Warlock // Sorcery
	// 5/4
	// TODO: implement
	Register("Emeritus of Woe // Demonic Tutor", func() Card {
		return NewCreature("Emeritus of Woe // Demonic Tutor", "{3}{B} // {1}{B}", 5, 4,
			WithSubTypes("Vampire", "Warlock", "//", "Sorcery"),
		)
	})

	// Eternal Student {3}{B}
	// Creature — Zombie Warlock
	// 4/2
	// {1}{B}, Exile this card from your graveyard: Create two 1/1 white and black Inkling creature tokens with flying.
	Register("Eternal Student", func() Card {
		inklingTokenEffect := CreateColoredToken("Inkling Token", 1, 1,
			[]Color{White, Black},
			[]CardType{TypeCreature},
			[]string{"Inkling"},
			Flying,
		)
		return NewCreature("Eternal Student", "{3}{B}", 4, 2,
			WithSubTypes("Zombie", "Warlock"),
			// {1}{B}, Exile this card from your graveyard: Create two 1/1 white and black Inkling creature tokens with flying.
			WithGraveyardActivatedAbility(
				CompositeEffects("create two Inkling tokens with flying",
					inklingTokenEffect,
					inklingTokenEffect,
				),
				ManaCostOf("{1}{B}"),
				WithCost(ExileSelfFromGraveyardCost()),
			),
		)
	})

	// Forum Necroscribe {5}{B}
	// Creature — Troll Warlock
	// 5/4
	// Ward—Discard a card.
	// Repartee — Whenever you cast an instant or sorcery spell that targets a creature, return target creature card from your graveyard to the battlefield.
	Register("Forum Necroscribe", func() Card {
		return NewCreature("Forum Necroscribe", "{5}{B}", 5, 4,
			WithSubTypes("Troll", "Warlock"),
			// XXX: Ward—Discard a card. The engine has no Ward mechanic implementation.
			WithAbility(WheneverYouCastInstantOrSorceryTargetingCreatureTrigger(
				ReturnFromGraveyardToBattlefield(),
				false,
			).AddTarget(TargetCardInYourGraveyard(IsCreatureCard))),
		)
	})

	// Grave Researcher // Reanimate {2}{B} // {B}
	// Creature — Troll Warlock // Sorcery
	// 3/3
	// TODO: implement
	Register("Grave Researcher // Reanimate", func() Card {
		return NewCreature("Grave Researcher // Reanimate", "{2}{B} // {B}", 3, 3,
			WithSubTypes("Troll", "Warlock", "//", "Sorcery"),
		)
	})

	// Lecturing Scornmage {B}
	// Creature — Human Warlock
	// 1/1
	// Repartee — Whenever you cast an instant or sorcery spell that targets a creature,
	// put a +1/+1 counter on this creature.
	Register("Lecturing Scornmage", func() Card {
		return NewCreature("Lecturing Scornmage", "{B}", 1, 1,
			WithSubTypes("Human", "Warlock"),
			WithAbility(NewTriggered(EvtSpellCast, false,
				AddCounters(P1P1, Fixed(1)).Targeting(ToSource()),
			).SetCondition(reparteeCondition)),
		)
	})
	// Leech Collector // Bloodletting {1}{B} // {B}
	// Creature — Human Warlock // Sorcery
	// 2/2
	// TODO: implement
	Register("Leech Collector // Bloodletting", func() Card {
		return NewCreature("Leech Collector // Bloodletting", "{1}{B} // {B}", 2, 2,
			WithSubTypes("Human", "Warlock", "//", "Sorcery"),
		)
	})

	// Melancholic Poet {1}{B}
	// Creature — Elf Bard
	// 2/2
	// Repartee — Whenever you cast an instant or sorcery spell that targets a creature, each opponent loses 1 life and you gain 1 life.
	Register("Melancholic Poet", func() Card {
		return NewCreature("Melancholic Poet", "{1}{B}", 2, 2,
			WithSubTypes("Elf", "Bard"),
			// Repartee — Whenever you cast an instant or sorcery spell that targets a creature,
			// each opponent loses 1 life and you gain 1 life.
			WithAbility(WheneverYouCastInstantOrSorceryTargetingCreatureTrigger(
				FuncEffect("each opponent loses 1 life and you gain 1 life",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						for _, pl := range g.AllPlayers() {
							if pl.PlayerID() != controller {
								pl.LoseLife(1)
							}
						}
						if you := g.GetPlayer(controller); you != nil {
							g.PlayerGainLife(you, 1)
						}
						return nil
					},
				),
				false,
			)),
		)
	})
	// Moseo, Vein's New Dean {2}{B}
	// Legendary Creature — Bird Skeleton Warlock
	// 2/1
	// Flying
	// When Moseo enters, create a 1/1 black and green Pest creature token with "Whenever this token attacks, you gain 1 life."
	// Infusion — At the beginning of your end step, if you gained life this turn, return up to one target creature card with mana value X or less from your graveyard to the battlefield, where X is the amount of life you gained this turn.
	Register("Moseo, Vein's New Dean", func() Card {
		pestTokenEffect := TokenWithAbilities(
			CreateColoredToken("Pest Token", 1, 1,
				[]Color{Black, Green},
				[]CardType{TypeCreature},
				[]string{"Pest"},
			),
			AttacksTrigger(GainLife(1), false),
		)
		return NewCreature("Moseo, Vein's New Dean", "{2}{B}", 2, 1,
			WithSubTypes("Bird", "Skeleton", "Warlock"),
			WithSuperTypes(SuperLegendary),
			WithKeyword(Flying),
			// When Moseo enters, create a 1/1 black and green Pest creature token with
			// "Whenever this token attacks, you gain 1 life."
			WithAbility(EntersBattlefieldTrigger(pestTokenEffect, false)),
			// Infusion — At the beginning of your end step, if you gained life this turn,
			// return up to one target creature card with mana value X or less from your
			// graveyard to the battlefield, where X is the amount of life you gained this turn.
			WithAbility(NewTriggered(EvtEndStep, false,
				InfusionEffect(
					"return up to one creature card with MV <= life gained from graveyard",
					FuncEffect(
						"return up to one creature card with MV <= life gained from graveyard",
						EffectProperties{Outcome: OutcomeBenefit},
						func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							p := g.GetPlayer(controller)
							if p == nil {
								return nil
							}
							lifeGained := LifeGainedThisTurnFor(g, controller)
							if lifeGained <= 0 {
								return nil
							}
							var eligible []Card
							for _, c := range p.Graveyard() {
								if !c.HasType(TypeCreature) {
									continue
								}
								if c.ManaCost().CMC() <= lifeGained {
									eligible = append(eligible, c)
								}
							}
							if len(eligible) == 0 {
								return nil
							}
							chosen := p.ChooseCardFromLibrary(eligible, "return creature from graveyard to battlefield (MV <= life gained)", g)
							if chosen == nil {
								return nil
							}
							if card, ok := p.RemoveFromGraveyard(chosen.ID()); ok {
								g.PutOnBattlefield(card, controller)
							}
							return nil
						},
					),
				),
			).SetConditionData(EventPlayerIsController{})),
		)
	})
	// Poisoner's Apprentice {2}{B}
	// Creature — Orc Warlock
	// 2/2
	// Infusion — When this creature enters, target creature an opponent controls gets -4/-4 until end of turn if you gained life this turn.
	// TODO: implement
	Register("Poisoner's Apprentice", func() Card {
		return NewCreature("Poisoner's Apprentice", "{2}{B}", 2, 2,
			WithSubTypes("Orc", "Warlock"),
		)
	})

	// Postmortem Professor {1}{B}
	// Creature — Zombie Warlock
	// 2/2
	// This creature can't block.
	// Whenever this creature attacks, each opponent loses 1 life and you gain 1 life.
	// {1}{B}, Exile an instant or sorcery card from your graveyard: Return this card from your graveyard to the battlefield.
	Register("Postmortem Professor", func() Card {
		return NewCreature("Postmortem Professor", "{1}{B}", 2, 2,
			WithSubTypes("Zombie", "Warlock"),
			// This creature can't block.
			WithStaticAbility(FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
				g.RevokeAttr(sourceID, AttrCanBlock)
				return nil
			})),
			// Whenever this creature attacks, each opponent loses 1 life and you gain 1 life.
			WithAbility(AttacksTrigger(
				FuncEffect("each opponent loses 1 life and you gain 1 life",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						for _, pl := range g.AllPlayers() {
							if pl.PlayerID() != controller {
								pl.LoseLife(1)
							}
						}
						if you := g.GetPlayer(controller); you != nil {
							g.PlayerGainLife(you, 1)
						}
						return nil
					},
				),
				false,
			)),
			// {1}{B}, Exile an instant or sorcery card from your graveyard: Return this card from your graveyard to the battlefield.
			// XXX: "Exile an instant or sorcery card from your graveyard" should be an additional cost, but there is
			// no filtered graveyard exile cost API; the exile is performed inside the effect instead.
			WithGraveyardActivatedAbility(
				FuncEffect("exile an instant or sorcery card from your graveyard, then return this card to the battlefield",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						// Exile an instant or sorcery card from graveyard (not self).
						for _, c := range p.Graveyard() {
							if c.ID() == sourceID {
								continue
							}
							if !(c.HasType(TypeInstant) || c.HasType(TypeSorcery)) {
								continue
							}
							if removed, ok := p.RemoveFromGraveyard(c.ID()); ok {
								g.ExileCard(removed, sourceID)
							}
							break
						}
						// Return self from graveyard to battlefield.
						if removed, ok := p.RemoveFromGraveyard(sourceID); ok {
							g.PutOnBattlefield(removed, controller)
						}
						return nil
					}),
				ManaCostOf("{1}{B}"),
			),
		)
	})

	// Scathing Shadelock // Venomous Words {4}{B} // {B}
	// Creature — Snake Warlock // Sorcery
	// 4/6
	// TODO: implement
	Register("Scathing Shadelock // Venomous Words", func() Card {
		return NewCreature("Scathing Shadelock // Venomous Words", "{4}{B} // {B}", 4, 6,
			WithSubTypes("Snake", "Warlock", "//", "Sorcery"),
		)
	})

	// Scheming Silvertongue // Sign in Blood {1}{B} // {B}{B}
	// Creature — Vampire Warlock // Sorcery
	// 1/3
	// TODO: implement
	Register("Scheming Silvertongue // Sign in Blood", func() Card {
		return NewCreature("Scheming Silvertongue // Sign in Blood", "{1}{B} // {B}{B}", 1, 3,
			WithSubTypes("Vampire", "Warlock", "//", "Sorcery"),
		)
	})

	// Sneering Shadewriter {4}{B}
	// Creature — Vampire Warlock
	// 3/3
	// Flying
	// When this creature enters, each opponent loses 2 life and you gain 2 life.
	Register("Sneering Shadewriter", func() Card {
		return NewCreature("Sneering Shadewriter", "{4}{B}", 3, 3,
			WithSubTypes("Vampire", "Warlock"),
			WithKeyword(Flying),
			WithAbility(EntersBattlefieldTrigger(
				FuncEffect(
					"each opponent loses 2 life and you gain 2 life",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						for _, pl := range g.AllPlayers() {
							if pl.PlayerID() != controller {
								g.PlayerLoseLife(pl, 2)
							}
						}
						if you := g.GetPlayer(controller); you != nil {
							g.PlayerGainLife(you, 2)
						}
						return nil
					},
				),
				false,
			)),
		)
	})

	// Tragedy Feaster {2}{B}{B}
	// Creature — Demon
	// 7/6
	// Trample
	// Ward—Discard a card.
	// Infusion — At the beginning of your end step, sacrifice a permanent unless you gained life this turn.
	// TODO: implement
	Register("Tragedy Feaster", func() Card {
		return NewCreature("Tragedy Feaster", "{2}{B}{B}", 7, 6,
			WithSubTypes("Demon"),
		)
	})

	// Ulna Alley Shopkeep {2}{B}
	// Creature — Goblin Warlock
	// 2/3
	// Menace (This creature can't be blocked except by two or more creatures.)
	// Infusion — This creature gets +2/+0 as long as you gained life this turn.
	Register("Ulna Alley Shopkeep", func() Card {
		return NewCreature("Ulna Alley Shopkeep", "{2}{B}", 2, 3,
			WithSubTypes("Goblin", "Warlock"),
			WithKeyword(Menace),
			// Infusion — This creature gets +2/+0 as long as you gained life this turn.
			WithStaticAbility(FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
				src := g.FindPermanent(sourceID)
				if src == nil {
					return nil
				}
				if IfControllerGainedLifeThisTurn(g, src.Controller) {
					src.BoostPT(2, 0)
				}
				return nil
			})),
		)
	})

	// ===== RED CREATURES =====

	// Blazing Firesinger // Seething Song {2}{R} // {2}{R}
	// Creature — Dwarf Bard // Instant
	// 2/3
	// TODO: implement
	Register("Blazing Firesinger // Seething Song", func() Card {
		return NewCreature("Blazing Firesinger // Seething Song", "{2}{R} // {2}{R}", 2, 3,
			WithSubTypes("Dwarf", "Bard", "//", "Instant"),
		)
	})

	// Charging Strifeknight {2}{R}
	// Creature — Spirit Knight
	// 3/3
	// Haste
	// {T}, Discard a card: Draw a card.
	Register("Charging Strifeknight", func() Card {
		return NewCreature("Charging Strifeknight", "{2}{R}", 3, 3,
			WithSubTypes("Spirit", "Knight"),
			WithKeyword(Haste),
			// {T}, Discard a card: Draw a card.
			WithActivatedAbility(
				DrawCards(Fixed(1)),
				TapSourceCost(),
				WithCost(DiscardCost(1)),
			),
		)
	})

	// Emeritus of Conflict // Lightning Bolt {1}{R} // {R}
	// Creature — Human Wizard // Instant
	// 2/2
	// TODO: implement
	Register("Emeritus of Conflict // Lightning Bolt", func() Card {
		return NewCreature("Emeritus of Conflict // Lightning Bolt", "{1}{R} // {R}", 2, 2,
			WithSubTypes("Human", "Wizard", "//", "Instant"),
		)
	})

	// Expressive Firedancer {1}{R}
	// Creature — Human Sorcerer
	// 2/2
	// Opus — Whenever you cast an instant or sorcery spell, this creature gets +1/+1 until end of turn. If five or more mana was spent to cast that spell, this creature also gains double strike until end of turn.
	Register("Expressive Firedancer", func() Card {
		isInstantOrSorcery := NewCardFilter("instant or sorcery", func(c Card) bool {
			return c.HasType(TypeInstant) || c.HasType(TypeSorcery)
		})
		return NewCreature("Expressive Firedancer", "{1}{R}", 2, 2,
			WithSubTypes("Human", "Sorcerer"),
			// Opus — Whenever you cast an instant or sorcery spell, this creature gets
			// +1/+1 until end of turn. If five or more mana was spent to cast that spell,
			// this creature also gains double strike until end of turn.
			WithAbility(WheneverYouCastSpellTrigger(
				OpusEffect(
					"this creature gets +1/+1 until end of turn; if 5+ mana spent, also gains double strike",
					func(g *Game, sourceID, controller uuid.UUID, manaSpent int) error {
						boost := TemporaryBoost(sourceID, 1, 1)
						boost.SetSourceID(sourceID)
						g.AddContinuousEffect(boost)
						if manaSpent >= 5 {
							_ = GrantKeyword(DoubleStrike).Targeting(ToSource()).Until(EndOfTurn).Apply(g, sourceID, controller, nil)
						}
						return nil
					},
				),
				false,
				isInstantOrSorcery,
			)),
		)
	})

	// Garrison Excavator {3}{R}
	// Creature — Orc Sorcerer
	// 3/4
	// Menace (This creature can't be blocked except by two or more creatures.)
	// Whenever one or more cards leave your graveyard, create a 2/2 red and white Spirit creature token.
	Register("Garrison Excavator", func() Card {
		return NewCreature("Garrison Excavator", "{3}{R}", 3, 4,
			WithSubTypes("Orc", "Sorcerer"),
			WithKeyword(Menace),
			// XXX: "Whenever one or more cards leave your graveyard, create a 2/2 red and white Spirit creature token."
			// The engine has no EvtLeaveGraveyard event; this trigger cannot be implemented.
		)
	})

	// Goblin Glasswright // Craft with Pride {1}{R} // {R}
	// Creature — Goblin Sorcerer // Sorcery
	// 2/2
	// TODO: implement
	Register("Goblin Glasswright // Craft with Pride", func() Card {
		return NewCreature("Goblin Glasswright // Craft with Pride", "{1}{R} // {R}", 2, 2,
			WithSubTypes("Goblin", "Sorcerer", "//", "Sorcery"),
		)
	})

	// Maelstrom Artisan // Rocket Volley {1}{R}{R} // {1}{R}
	// Creature — Minotaur Sorcerer // Sorcery
	// 3/2
	// TODO: implement
	Register("Maelstrom Artisan // Rocket Volley", func() Card {
		return NewCreature("Maelstrom Artisan // Rocket Volley", "{1}{R}{R} // {1}{R}", 3, 2,
			WithSubTypes("Minotaur", "Sorcerer", "//", "Sorcery"),
		)
	})

	// Magmablood Archaic {2/R}{2/R}{2/R}
	// Creature — Avatar
	// 2/2
	// Trample, reach
	// Converge — This creature enters with a +1/+1 counter on it for each color of mana spent to cast it.
	// Whenever you cast an instant or sorcery spell, creatures you control get +1/+0 until end of turn for each color of mana spent to cast that spell.
	Register("Magmablood Archaic", func() Card {
		isInstantOrSorceryCard := NewCardFilter("instant or sorcery", func(c Card) bool {
			return c.HasType(TypeInstant) || c.HasType(TypeSorcery)
		})
		return NewCreature("Magmablood Archaic", "{2/R}{2/R}{2/R}", 2, 2,
			WithSubTypes("Avatar"),
			WithKeyword(Trample),
			WithKeyword(Reach),
			// Converge — This creature enters with a +1/+1 counter on it for each color of mana spent to cast it.
			WithAbility(EntersBattlefieldTrigger(
				FuncEffect("enter with a +1/+1 counter for each color of mana spent to cast",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						ctx := g.ResolvingCastContext()
						if ctx == nil {
							return nil
						}
						colors := ctx.DistinctColorsSpent()
						if colors <= 0 {
							return nil
						}
						perm := g.FindPermanent(sourceID)
						if perm == nil {
							return nil
						}
						perm.AddCounter(P1P1, colors)
						g.ApplyContinuousEffects()
						return nil
					},
				),
				false,
			)),
			// Whenever you cast an instant or sorcery spell, creatures you control get +1/+0 until end of turn
			// for each color of mana spent to cast that spell.
			WithAbility(WheneverYouCastSpellTrigger(
				OpusEffect("creatures you control get +1/+0 for each color of mana spent",
					func(g *Game, sourceID, controller uuid.UUID, manaSpent int) error {
						// Determine colors spent for the triggering spell.
						// Walk the stack to find the spell that triggered this.
						var numColors int
						objs := g.GetStack().Objects()
						for i := len(objs) - 1; i >= 0; i-- {
							obj := objs[i]
							if !obj.IsAbility && obj.CastContext != nil {
								numColors = obj.CastContext.DistinctColorsSpent()
								break
							}
						}
						if numColors <= 0 {
							return nil
						}
						src := g.FindPermanent(sourceID)
						if src == nil {
							return nil
						}
						for _, perm := range g.AllBattlefield() {
							if perm.Controller == controller && perm.HasAttr(AttrIsCreature) {
								ce := TemporaryBoost(perm.ID(), numColors, 0)
								ce.SetSourceID(sourceID)
								g.AddContinuousEffect(ce)
							}
						}
						return nil
					},
				),
				false,
				isInstantOrSorceryCard,
			)),
		)
	})

	// Mica, Reader of Ruins {3}{R}
	// Legendary Creature — Human Artificer
	// 4/4
	// Ward—Pay 3 life. (Whenever this creature becomes the target of a spell or ability an opponent controls, counter it unless that player pays 3 life.)
	// Whenever you cast an instant or sorcery spell, you may sacrifice an artifact. If you do, copy that spell and you may choose new targets for the copy.
	Register("Mica, Reader of Ruins", func() Card {
		isInstantOrSorcery := NewCardFilter("instant or sorcery", func(c Card) bool {
			return c.HasType(TypeInstant) || c.HasType(TypeSorcery)
		})
		return NewCreature("Mica, Reader of Ruins", "{3}{R}", 4, 4,
			WithSubTypes("Human", "Artificer"),
			WithSuperTypes(SuperLegendary),
			// XXX: Ward—Pay 3 life. The engine has no Ward mechanic implementation.
			// Whenever you cast an instant or sorcery spell, you may sacrifice an artifact.
			// If you do, copy that spell and you may choose new targets for the copy.
			WithAbility(WheneverYouCastSpellTrigger(
				FuncEffect("may sacrifice an artifact to copy that spell",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						// Check if controller has any artifacts.
						var artifacts []*Permanent
						for _, perm := range g.AllBattlefield() {
							if perm.Controller == controller && perm.HasAttr(AttrIsArtifact) {
								artifacts = append(artifacts, perm)
							}
						}
						if len(artifacts) == 0 {
							return nil
						}
						choice := p.ChooseMode([]string{"yes", "no"}, "sacrifice an artifact to copy the spell?")
						if choice != 0 {
							return nil
						}
						chosen := p.ChoosePermanent(artifacts, "choose artifact to sacrifice", g)
						if chosen == nil {
							return nil
						}
						g.Sacrifice(chosen)
						// Find the triggering spell on the stack.
						var spellSourceID uuid.UUID
						objs := g.GetStack().Objects()
						for i := len(objs) - 1; i >= 0; i-- {
							if !objs[i].IsAbility {
								spellSourceID = objs[i].SourceID
								break
							}
						}
						if spellSourceID == uuid.Nil {
							return nil
						}
						// XXX: Oracle says "you may choose new targets for the copy" (optional retarget).
						// CopySpellOnStack with false inherits targets, which is correct for the common case.
						g.CopySpellOnStack(spellSourceID, controller, false)
						return nil
					},
				),
				false,
				isInstantOrSorcery,
			)),
		)
	})

	// Molten-Core Maestro {1}{R}
	// Creature — Goblin Bard
	// 2/2
	// Menace
	// Opus — Whenever you cast an instant or sorcery spell, put a +1/+1 counter on this creature. If five or more mana was spent to cast that spell, add an amount of {R} equal to this creature's power.
	Register("Molten-Core Maestro", func() Card {
		isInstantOrSorcery := NewCardFilter("instant or sorcery", func(c Card) bool {
			return c.HasType(TypeInstant) || c.HasType(TypeSorcery)
		})
		return NewCreature("Molten-Core Maestro", "{1}{R}", 2, 2,
			WithSubTypes("Goblin", "Bard"),
			WithKeyword(Menace),
			// Opus — Whenever you cast an instant or sorcery spell, put a +1/+1 counter on
			// this creature. If five or more mana was spent to cast that spell, add an amount
			// of {R} equal to this creature's power.
			WithAbility(WheneverYouCastSpellTrigger(
				OpusEffect("put a +1/+1 counter; if 5+ mana spent, add {R} equal to power",
					func(g *Game, sourceID, controller uuid.UUID, manaSpent int) error {
						perm := g.FindPermanent(sourceID)
						if perm == nil {
							return nil
						}
						perm.AddCounter(P1P1, 1)
						g.ApplyContinuousEffects()
						if manaSpent >= 5 {
							p := g.GetPlayer(controller)
							if p == nil {
								return nil
							}
							power := perm.CurrentPower(g)
							if power > 0 {
								p.ManaPool().Add(Red, power)
							}
						}
						return nil
					},
				),
				false,
				isInstantOrSorcery,
			)),
		)
	})
	// Pigment Wrangler // Striking Palette {4}{R} // {R}
	// Creature — Orc Sorcerer // Sorcery
	// 4/4
	// TODO: implement
	Register("Pigment Wrangler // Striking Palette", func() Card {
		return NewCreature("Pigment Wrangler // Striking Palette", "{4}{R} // {R}", 4, 4,
			WithSubTypes("Orc", "Sorcerer", "//", "Sorcery"),
		)
	})

	// Rearing Embermare {4}{R}
	// Creature — Horse Beast
	// 4/5
	// Reach, haste
	Register("Rearing Embermare", func() Card {
		return NewCreature("Rearing Embermare", "{4}{R}", 4, 5,
			WithSubTypes("Horse", "Beast"),
			WithKeyword(Reach),
			WithKeyword(Haste),
		)
	})

	// Rubble Rouser {2}{R}
	// Creature — Dwarf Sorcerer
	// 1/4
	// When this creature enters, you may discard a card. If you do, draw a card.
	// {T}, Exile a card from your graveyard: Add {R}. When you do, this creature deals 1 damage to each opponent.
	Register("Rubble Rouser", func() Card {
		return NewCreature("Rubble Rouser", "{2}{R}", 1, 4,
			WithSubTypes("Dwarf", "Sorcerer"),
			WithAbility(EntersBattlefieldTrigger(
				FuncEffect(
					"you may discard a card; if you do, draw a card",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil || len(p.Hand()) == 0 {
							return nil
						}
						if !p.ChooseMayAbility("discard a card to draw a card") {
							return nil
						}
						chosen := p.ChooseCardsFromHand(1, "discard a card", g)
						for _, c := range chosen {
							g.PlayerDiscard(p, c.ID())
						}
						g.PlayerDrawCard(p)
						return nil
					},
				),
				false,
			)),
			WithActivatedAbility(
				AddMana(Red, 1),
				TapSourceCost(),
				WithCost(ExileFromGraveyardCost(1)),
				WithEffect(DealDamageToPlayers(Fixed(1), SelectEachOpponent())),
			),
		)
	})

	// Strife Scholar // Awaken the Ages {2}{R} // {5}{R}
	// Creature — Orc Sorcerer // Sorcery
	// 3/2
	// TODO: implement
	Register("Strife Scholar // Awaken the Ages", func() Card {
		return NewCreature("Strife Scholar // Awaken the Ages", "{2}{R} // {5}{R}", 3, 2,
			WithSubTypes("Orc", "Sorcerer", "//", "Sorcery"),
		)
	})

	// Tackle Artist {3}{R}
	// Creature — Orc Sorcerer
	// 4/3
	// Trample
	// Opus — Whenever you cast an instant or sorcery spell, put a +1/+1 counter on this creature. If five or more mana was spent to cast that spell, put two +1/+1 counters on this creature instead.
	// TODO: implement
	Register("Tackle Artist", func() Card {
		return NewCreature("Tackle Artist", "{3}{R}", 4, 3,
			WithSubTypes("Orc", "Sorcerer"),
		)
	})

	// Thunderdrum Soloist {1}{R}
	// Creature — Dwarf Bard
	// 1/3
	// Reach
	// Opus — Whenever you cast an instant or sorcery spell, this creature deals 1 damage to each opponent. If five or more mana was spent to cast that spell, this creature deals 3 damage to each opponent instead.
	// TODO: implement
	Register("Thunderdrum Soloist", func() Card {
		return NewCreature("Thunderdrum Soloist", "{1}{R}", 1, 3,
			WithSubTypes("Dwarf", "Bard"),
		)
	})

	// Zealous Lorecaster {5}{R}
	// Creature — Giant Sorcerer
	// 4/4
	// When this creature enters, return target instant or sorcery card from your graveyard to your hand.
	Register("Zealous Lorecaster", func() Card {
		return NewCreature("Zealous Lorecaster", "{5}{R}", 4, 4,
			WithSubTypes("Giant", "Sorcerer"),
			// When this creature enters, return target instant or sorcery card from your graveyard to your hand.
			WithAbility(EntersBattlefieldTrigger(
				ReturnFromGraveyardToHandTarget(),
				false,
			).AddTarget(TargetCardInYourGraveyard(IsInstantOrSorceryCard))),
		)
	})

	// ===== GREEN CREATURES =====

	// Aberrant Manawurm {3}{G}
	// Creature — Wurm
	// 2/5
	// Trample
	// Whenever you cast an instant or sorcery spell, this creature gets +X/+0 until end of turn, where X is the amount of mana spent to cast that spell.
	Register("Aberrant Manawurm", func() Card {
		isInstantOrSorcery := NewCardFilter("instant or sorcery", func(c Card) bool {
			return c.HasType(TypeInstant) || c.HasType(TypeSorcery)
		})
		return NewCreature("Aberrant Manawurm", "{3}{G}", 2, 5,
			WithSubTypes("Wurm"),
			WithKeyword(Trample),
			// Whenever you cast an instant or sorcery spell, this creature gets +X/+0 until end of turn,
			// where X is the amount of mana spent to cast that spell.
			WithAbility(WheneverYouCastSpellTrigger(
				OpusEffect("this creature gets +X/+0 where X is mana spent",
					func(g *Game, sourceID, controller uuid.UUID, manaSpent int) error {
						if manaSpent <= 0 {
							return nil
						}
						ce := TemporaryBoost(sourceID, manaSpent, 0)
						ce.SetSourceID(sourceID)
						g.AddContinuousEffect(ce)
						return nil
					},
				),
				false,
				isInstantOrSorcery,
			)),
		)
	})

	// Ambitious Augmenter {G}
	// Creature — Turtle Wizard
	// 1/1
	// Increment (Whenever you cast a spell, if the amount of mana you spent is greater than this creature's power or toughness, put a +1/+1 counter on this creature.)
	// When this creature dies, if it had one or more counters on it, create a 0/0 green and blue Fractal creature token, then put this creature's counters on that token.
	Register("Ambitious Augmenter", func() Card {
		return NewCreature("Ambitious Augmenter", "{G}", 1, 1,
			WithSubTypes("Turtle", "Wizard"),
			// Increment
			WithAbility(IncrementTrigger()),
			// When this creature dies, if it had one or more counters on it,
			// create a 0/0 green and blue Fractal creature token, then put
			// this creature's counters on that token.
			WithAbility(PutIntoGraveyardFromBattlefieldTrigger(
				FuncEffect(
					"if had counters, create Fractal token and move counters to it",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						lkiView := g.LookupObject(sourceID)
						if lkiView == nil {
							return nil
						}
						lki, ok := lkiView.(*PermanentLKI)
						if !ok || lki == nil {
							return nil
						}
						if lki.Snapshot == nil {
							return nil
						}
						totalCounters := 0
						for _, n := range lki.Snapshot.Counters {
							totalCounters += int(n)
						}
						if totalCounters == 0 {
							return nil
						}
						// Create a 0/0 green and blue Fractal creature token.
						token := NewToken("Fractal Token", 0, 0,
							[]CardType{TypeCreature},
							[]string{"Fractal"},
						)
						token.SetColorOverride([]Color{Green, Blue})
						token.SetOwner(controller)
						perm := g.PutOnBattlefield(token, controller)
						if perm == nil {
							return nil
						}
						// Put this creature's counters on that token.
						for ct, n := range lki.Snapshot.Counters {
							if n > 0 {
								g.AddCountersWithReplacement(perm, CounterType(ct), int(n), sourceID, false)
							}
						}
						g.ApplyContinuousEffects()
						return nil
					},
				),
				false,
			)),
		)
	})

	// Emeritus of Abundance // Regrowth {2}{G} // {1}{G}
	// Creature — Elf Druid // Sorcery
	// 3/4
	// TODO: implement
	Register("Emeritus of Abundance // Regrowth", func() Card {
		return NewCreature("Emeritus of Abundance // Regrowth", "{2}{G} // {1}{G}", 3, 4,
			WithSubTypes("Elf", "Druid", "//", "Sorcery"),
		)
	})

	// Emil, Vastlands Roamer {2}{G}
	// Legendary Creature — Elf Druid
	// 3/3
	// Creatures you control with +1/+1 counters on them have trample.
	// {4}{G}, {T}: Create a 0/0 green and blue Fractal creature token. Put X +1/+1 counters on it, where X is the number of differently named lands you control.
	Register("Emil, Vastlands Roamer", func() Card {
		return NewCreature("Emil, Vastlands Roamer", "{2}{G}", 3, 3,
			WithSubTypes("Elf", "Druid"),
			WithSuperTypes(SuperLegendary),
			// Creatures you control with +1/+1 counters on them have trample.
			WithStaticAbility(FuncContinuousEffect(LayerAbility, WhileOnBattlefield,
				func(g *Game, sourceID uuid.UUID) error {
					src := g.FindPermanent(sourceID)
					if src == nil {
						return nil
					}
					for _, perm := range g.AllBattlefield() {
						if perm.Controller == src.Controller && perm.HasAttr(AttrIsCreature) && perm.Counters[P1P1] > 0 {
							g.GrantAttr(perm.ID(), Trample)
						}
					}
					return nil
				},
			)),
			// {4}{G}, {T}: Create a 0/0 green and blue Fractal creature token and put X +1/+1 counters on it,
			// where X is the number of differently named lands you control.
			WithActivatedAbility(
				FuncEffect("create Fractal token with counters equal to differently named lands",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						// Count differently named lands you control.
						names := make(map[string]bool)
						for _, perm := range g.AllBattlefield() {
							if perm.Controller == controller && perm.HasAttr(AttrIsLand) {
								names[perm.Card.Name()] = true
							}
						}
						x := len(names)
						// Create a 0/0 green and blue Fractal creature token.
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
						return nil
					},
				),
				ManaCostOf("{4}{G}"),
				WithCost(TapSourceCost()),
			),
		)
	})

	// Environmental Scientist {1}{G}
	// Creature — Human Druid
	// 2/2
	// When this creature enters, you may search your library for a basic land card, reveal it, put it into your hand, then shuffle.
	Register("Environmental Scientist", func() Card {
		return NewCreature("Environmental Scientist", "{1}{G}", 2, 2,
			WithSubTypes("Human", "Druid"),
			// When this creature enters, you may search your library for a basic land card,
			// reveal it, put it into your hand, then shuffle.
			WithAbility(EntersBattlefieldTrigger(
				FuncEffect("search library for basic land, put it into hand",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						searchBasicLandToHandSOS(g, controller)
						return nil
					},
				),
				true, // "you may"
			)),
		)
	})

	// Hungry Graffalon {3}{G}
	// Creature — Giraffe
	// 3/4
	// Reach
	// Increment (Whenever you cast a spell, if the amount of mana you spent is greater than this creature's power or toughness, put a +1/+1 counter on this creature.)
	Register("Hungry Graffalon", func() Card {
		return NewCreature("Hungry Graffalon", "{3}{G}", 3, 4,
			WithSubTypes("Giraffe"),
			WithKeyword(Reach),
			WithAbility(IncrementTrigger()),
		)
	})

	// Infirmary Healer // Stream of Life {1}{G} // {X}{G}
	// Creature — Cat Cleric // Sorcery
	// 2/3
	// TODO: implement
	Register("Infirmary Healer // Stream of Life", func() Card {
		return NewCreature("Infirmary Healer // Stream of Life", "{1}{G} // {X}{G}", 2, 3,
			WithSubTypes("Cat", "Cleric", "//", "Sorcery"),
		)
	})

	// Mindful Biomancer {1}{G}
	// Creature — Dryad Druid
	// 2/2
	// When this creature enters, you gain 1 life.
	// {2}{G}: This creature gets +2/+2 until end of turn. Activate only once each turn.
	Register("Mindful Biomancer", func() Card {
		return NewCreature("Mindful Biomancer", "{1}{G}", 2, 2,
			WithSubTypes("Dryad", "Druid"),
			// When this creature enters, you gain 1 life.
			WithAbility(EntersBattlefieldTrigger(GainLife(1), false)),
			// {2}{G}: This creature gets +2/+2 until end of turn. Activate only once each turn.
			WithActivatedAbility(
				Boost(Fixed(2), Fixed(2)).Targeting(ToSource()),
				ManaCostOf("{2}{G}"),
				WithOncePerTurn(),
			),
		)
	})

	// Noxious Newt {1}{G}
	// Creature — Salamander
	// 1/2
	// Deathtouch
	// {T}: Add {G}.
	Register("Noxious Newt", func() Card {
		return NewCreature("Noxious Newt", "{1}{G}", 1, 2,
			WithSubTypes("Salamander"),
			WithKeyword(Deathtouch),
			WithManaAbility(Green),
		)
	})

	// Pestbrood Sloth {3}{G}
	// Creature — Plant Sloth
	// 4/4
	// Reach
	// When this creature dies, create two 1/1 black and green Pest creature tokens with "Whenever this token attacks, you gain 1 life."
	Register("Pestbrood Sloth", func() Card {
		makePestToken := func() Effect {
			return TokenWithAbilities(
				CreateColoredToken("Pest Token", 1, 1,
					[]Color{Black, Green},
					[]CardType{TypeCreature},
					[]string{"Pest"},
				),
				AttacksTrigger(GainLife(1), false),
			)
		}
		return NewCreature("Pestbrood Sloth", "{3}{G}", 4, 4,
			WithSubTypes("Plant", "Sloth"),
			WithKeyword(Reach),
			// When this creature dies, create two 1/1 black and green Pest creature tokens
			// with "Whenever this token attacks, you gain 1 life."
			// Two separate effect instances are used so each token gets its own ability
			// instance (sharing a single instance causes SetSource to overwrite).
			WithAbility(PutIntoGraveyardFromBattlefieldTrigger(
				CompositeEffects("create two Pest tokens",
					makePestToken(),
					makePestToken(),
				),
				false,
			)),
		)
	})

	// Shopkeeper's Bane {2}{G}
	// Creature — Badger Pest
	// 4/2
	// Trample
	// Whenever this creature attacks, you gain 2 life.
	Register("Shopkeeper's Bane", func() Card {
		return NewCreature("Shopkeeper's Bane", "{2}{G}", 4, 2,
			WithSubTypes("Badger", "Pest"),
			WithKeyword(Trample),
			WithAbility(AttacksTrigger(GainLife(2), false)),
		)
	})

	// Slumbering Trudge {X}{G}
	// Creature — Plant Beast
	// 6/6
	// This creature enters with a number of stun counters on it equal to three minus X. If X is 2 or less, it enters tapped. (If a permanent with a stun counter would become untapped, remove one from it instead.)
	// TODO: implement
	Register("Slumbering Trudge", func() Card {
		return NewCreature("Slumbering Trudge", "{X}{G}", 6, 6,
			WithSubTypes("Plant", "Beast"),
		)
	})

	// Studious First-Year // Rampant Growth {G} // {1}{G}
	// Creature — Bear Wizard // Sorcery
	// 1/1
	// TODO: implement
	Register("Studious First-Year // Rampant Growth", func() Card {
		return NewCreature("Studious First-Year // Rampant Growth", "{G} // {1}{G}", 1, 1,
			WithSubTypes("Bear", "Wizard", "//", "Sorcery"),
		)
	})

	// Tenured Concocter {4}{G}
	// Creature — Troll Druid
	// 4/5
	// Vigilance
	// Whenever this creature becomes the target of a spell or ability an opponent controls, you may draw a card.
	// Infusion — This creature gets +2/+0 as long as you gained life this turn.
	// TODO: implement
	Register("Tenured Concocter", func() Card {
		return NewCreature("Tenured Concocter", "{4}{G}", 4, 5,
			WithSubTypes("Troll", "Druid"),
		)
	})

	// Thornfist Striker {2}{G}
	// Creature — Elf Druid
	// 3/3
	// Ward {1} (Whenever this creature becomes the target of a spell or ability an opponent controls, counter it unless that player pays {1}.)
	// Infusion — Creatures you control get +1/+0 and have trample as long as you gained life this turn.
	// TODO: implement
	Register("Thornfist Striker", func() Card {
		return NewCreature("Thornfist Striker", "{2}{G}", 3, 3,
			WithSubTypes("Elf", "Druid"),
		)
	})

	// Topiary Lecturer {2}{G}
	// Creature — Elf Druid
	// 1/2
	// Increment (Whenever you cast a spell, if the amount of mana you spent is greater than this creature's power or toughness, put a +1/+1 counter on this creature.)
	// {T}: Add an amount of {G} equal to this creature's power.
	// TODO: implement
	Register("Topiary Lecturer", func() Card {
		return NewCreature("Topiary Lecturer", "{2}{G}", 1, 2,
			WithSubTypes("Elf", "Druid"),
		)
	})

	// Vastlands Scavenger // Bind to Life {1}{G}{G} // {4}{G}
	// Creature — Bear Druid // Instant
	// 4/4
	// TODO: implement
	Register("Vastlands Scavenger // Bind to Life", func() Card {
		return NewCreature("Vastlands Scavenger // Bind to Life", "{1}{G}{G} // {4}{G}", 4, 4,
			WithSubTypes("Bear", "Druid", "//", "Instant"),
		)
	})

	// Wildgrowth Archaic {2/G}{2/G}
	// Creature — Avatar
	// 0/0
	// Trample, reach
	// Converge — This creature enters with a +1/+1 counter on it for each color of mana spent to cast it.
	// Whenever you cast a creature spell, that creature enters with X additional +1/+1 counters on it, where X is the number of colors of mana spent to cast it.
	// XXX: {2/G} (two-generic-or-one-colored hybrid) mana is not supported by the engine's ManaCost
	// parser; it parses to zero cost, so no mana is recorded as spent and DistinctColorsSpent() is
	// always 0. Converge and the creature-spell trigger cannot function until the engine gains
	// support for generic-hybrid mana symbols.
	Register("Wildgrowth Archaic", func() Card {
		return NewCreature("Wildgrowth Archaic", "{2/G}{2/G}", 0, 0,
			WithSubTypes("Avatar"),
			WithKeyword(Trample),
			WithKeyword(Reach),
			// XXX: Converge counter ETB is not implemented — see XXX above.
			// XXX: "Whenever you cast a creature spell" trigger is not implemented.
		)
	})

	// ===== MULTICOLOR CREATURES =====

	// Abigale, Poet Laureate // Heroic Stanza {1}{W}{B} // {1}{W/B}
	// Legendary Creature — Bird Bard // Sorcery
	// 2/3
	// TODO: implement
	Register("Abigale, Poet Laureate // Heroic Stanza", func() Card {
		return NewCreature("Abigale, Poet Laureate // Heroic Stanza", "{1}{W}{B} // {1}{W/B}", 2, 3,
			WithSubTypes("Bird", "Bard", "//", "Sorcery"),
			WithSuperTypes(SuperLegendary),
		)
	})

	// Abstract Paintmage {U}{U/R}{R}
	// Creature — Djinn Sorcerer
	// 2/2
	// At the beginning of your first main phase, add {U}{R}. Spend this mana only to cast instant and sorcery spells.
	Register("Abstract Paintmage", func() Card {
		return NewCreature("Abstract Paintmage", "{U}{U/R}{R}", 2, 2,
			WithSubTypes("Djinn", "Sorcerer"),
			// At the beginning of your first main phase, add {U}{R}.
			// XXX: "Spend this mana only to cast instant and sorcery spells" restriction not enforced.
			WithAbility(BeginningOfFirstMainPhaseTrigger(
				CompositeEffects("add {U}{R}",
					AddMana(Blue, 1),
					AddMana(Red, 1),
				),
				false,
			)),
		)
	})

	// Aziza, Mage Tower Captain {R}{W}
	// Legendary Creature — Djinn Sorcerer
	// 2/2
	// Whenever you cast an instant or sorcery spell, you may tap three untapped creatures you control. If you do, copy that spell. You may choose new targets for the copy.
	Register("Aziza, Mage Tower Captain", func() Card {
		return NewCreature("Aziza, Mage Tower Captain", "{R}{W}", 2, 2,
			WithSubTypes("Djinn", "Sorcerer"),
			WithSuperTypes(SuperLegendary),
			// Whenever you cast an instant or sorcery spell, you may tap three untapped
			// creatures you control. If you do, copy that spell. You may choose new targets.
			WithAbility(WheneverYouCastSpellTrigger(
				FuncEffect(
					"you may tap three untapped creatures you control to copy that spell",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						// Collect untapped creatures the controller controls.
						var untapped []*Permanent
						for _, perm := range g.AllBattlefield() {
							if perm.Controller == controller &&
								perm.HasAttr(AttrIsCreature) &&
								!perm.Tapped {
								untapped = append(untapped, perm)
							}
						}
						if len(untapped) < 3 {
							return nil
						}
						choice := p.ChooseMode([]string{"yes", "no"},
							"tap three untapped creatures you control to copy the spell?")
						if choice != 0 {
							return nil
						}
						// Player chooses three untapped creatures to tap.
						for i := 0; i < 3; i++ {
							chosen := p.ChoosePermanent(untapped, "choose untapped creature to tap", g)
							if chosen == nil {
								return nil
							}
							g.TapPermanent(chosen)
							// Remove chosen from candidates.
							remaining := untapped[:0]
							for _, c := range untapped {
								if c.ID() != chosen.ID() {
									remaining = append(remaining, c)
								}
							}
							untapped = remaining
						}
						// Find the triggering spell on the stack (topmost non-ability).
						var spellSourceID uuid.UUID
						objs := g.GetStack().Objects()
						for i := len(objs) - 1; i >= 0; i-- {
							if !objs[i].IsAbility {
								spellSourceID = objs[i].SourceID
								break
							}
						}
						if spellSourceID == uuid.Nil {
							return nil
						}
						// XXX: Oracle says "you may choose new targets for the copy" (optional reprompt),
						// but the engine's CopySpellOnStack with true always reprompts, and ChooseTargets
						// defaults to the first candidate rather than preserving original targets.
						// Using false (inherit targets) is functionally correct for the common case.
						g.CopySpellOnStack(spellSourceID, controller, false)
						return nil
					},
				),
				false,
				IsInstantOrSorceryCard,
			)),
		)
	})

	// Berta, Wise Extrapolator {2}{G}{U}
	// Legendary Creature — Frog Druid
	// 1/4
	// Increment (Whenever you cast a spell, if the amount of mana you spent is greater than this creature's power or toughness, put a +1/+1 counter on this creature.)
	// Whenever one or more +1/+1 counters are put on Berta, add one mana of any color.
	// {X}, {T}: Create a 0/0 green and blue Fractal creature token and put X +1/+1 counters on it.
	Register("Berta, Wise Extrapolator", func() Card {
		return NewCreature("Berta, Wise Extrapolator", "{2}{G}{U}", 1, 4,
			WithSubTypes("Frog", "Druid"),
			WithSuperTypes(SuperLegendary),
			// Increment
			WithAbility(IncrementTrigger()),
			// XXX: "Whenever one or more +1/+1 counters are put on Berta, add one mana
			// of any color." — the engine has no EvtCounterAdded event, so this trigger
			// cannot be implemented without an engine change.
			// {X}, {T}: Create a 0/0 green and blue Fractal creature token and put X +1/+1 counters on it.
			WithActivatedAbility(
				FuncEffect(
					"create Fractal token with X +1/+1 counters",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						x := g.XValue()
						token := NewToken("Fractal Token", 0, 0,
							[]CardType{TypeCreature},
							[]string{"Fractal"},
						)
						token.SetColorOverride([]Color{Green, Blue})
						token.SetOwner(controller)
						perm := g.PutOnBattlefield(token, controller)
						if perm == nil {
							return nil
						}
						if x > 0 {
							g.AddCountersWithReplacement(perm, P1P1, x, sourceID, false)
							g.ApplyContinuousEffects()
						}
						return nil
					},
				),
				TapSourceCost(),
				WithCost(ManaCostOf("{X}")),
			),
		)
	})

	// Blech, Loafing Pest {1}{B}{G}
	// Legendary Creature — Pest
	// 3/4
	// Whenever you gain life, put a +1/+1 counter on each Pest, Bat, Insect, Snake, and Spider you control.
	Register("Blech, Loafing Pest", func() Card {
		isPestBatInsectSnakeSpider := func(perm *Permanent, _ *Game) bool {
			return perm.HasSubType("Pest") || perm.HasSubType("Bat") ||
				perm.HasSubType("Insect") || perm.HasSubType("Snake") ||
				perm.HasSubType("Spider")
		}
		return NewCreature("Blech, Loafing Pest", "{1}{B}{G}", 3, 4,
			WithSubTypes("Pest"),
			WithSuperTypes(SuperLegendary),
			// Whenever you gain life, put a +1/+1 counter on each Pest, Bat, Insect, Snake, and Spider you control.
			WithAbility(WheneverYouGainLifeTrigger(
				FuncEffect("put +1/+1 counter on each Pest/Bat/Insect/Snake/Spider you control",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						for _, perm := range g.AllBattlefield() {
							if perm.Controller == controller && isPestBatInsectSnakeSpider(perm, g) {
								g.AddCountersWithReplacement(perm, P1P1, 1, sourceID, false)
							}
						}
						return nil
					},
				),
				false,
			)),
		)
	})

	// Bogwater Lumaret {B}{G}
	// Creature — Spirit Frog
	// 2/2
	// Whenever this creature or another creature you control enters, you gain 1 life.
	Register("Bogwater Lumaret", func() Card {
		return NewCreature("Bogwater Lumaret", "{B}{G}", 2, 2,
			WithSubTypes("Spirit", "Frog"),
			// Whenever this creature or another creature you control enters, you gain 1 life.
			WithAbility(WheneverPermanentEntersBattlefieldTrigger(
				GainLife(1),
				false,
				IsCreature,
			).AndConditionData(EventSourceControlledByController{})),
		)
	})

	// Colorstorm Stallion {1}{U}{R}
	// Creature — Elemental Horse
	// 3/3
	// Ward {1}, haste
	// Opus — Whenever you cast an instant or sorcery spell, this creature gets +1/+1 until end of turn. If five or more mana was spent to cast that spell, create a token that's a copy of this creature.
	Register("Colorstorm Stallion", func() Card {
		isInstantOrSorcery := NewCardFilter("instant or sorcery", func(c Card) bool {
			return c.HasType(TypeInstant) || c.HasType(TypeSorcery)
		})
		return NewCreature("Colorstorm Stallion", "{1}{U}{R}", 3, 3,
			WithSubTypes("Elemental", "Horse"),
			// XXX: Ward {1} — the engine has no Ward mechanic implementation.
			WithKeyword(Haste),
			// Opus — Whenever you cast an instant or sorcery spell, this creature
			// gets +1/+1 until end of turn. If five or more mana was spent to cast
			// that spell, create a token that's a copy of this creature.
			WithAbility(WheneverYouCastSpellTrigger(
				OpusEffect(
					"gets +1/+1 until EOT; if 5+ mana, create a token copy",
					func(g *Game, sourceID, controller uuid.UUID, manaSpent int) error {
						perm := g.FindPermanent(sourceID)
						if perm == nil {
							return nil
						}
						// +1/+1 until end of turn.
						ce := TemporaryBoost(perm.ID(), 1, 1)
						ce.SetSourceID(sourceID)
						g.AddContinuousEffect(ce)
						// If five or more mana was spent, create a token copy.
						if manaSpent >= 5 {
							tokenCard := perm.Card.Copy()
							g.PutOnBattlefield(tokenCard, controller)
						}
						return nil
					},
				),
				false,
				isInstantOrSorcery,
			)),
		)
	})

	// Colossus of the Blood Age {4}{R}{W}
	// Artifact Creature — Construct
	// 6/6
	// When this creature enters, it deals 3 damage to each opponent and you gain 3 life.
	// When this creature dies, discard any number of cards, then draw that many cards plus one.
	Register("Colossus of the Blood Age", func() Card {
		return NewCreature("Colossus of the Blood Age", "{4}{R}{W}", 6, 6,
			WithSubTypes("Construct"),
			WithCardType(TypeArtifact),
			// When this creature enters, it deals 3 damage to each opponent and you gain 3 life.
			WithAbility(EntersBattlefieldTrigger(
				FuncEffect("deals 3 damage to each opponent and you gain 3 life",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						for _, opp := range g.AllPlayers() {
							if opp.PlayerID() != controller {
								g.PlayerLoseLife(opp, 3)
							}
						}
						g.PlayerGainLife(p, 3)
						return nil
					},
				),
				false,
			)),
			// When this creature dies, discard any number of cards, then draw that many cards plus one.
			WithAbility(PutIntoGraveyardFromBattlefieldTrigger(
				FuncEffect("discard any number, draw that many plus one",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						hand := p.Hand()
						if len(hand) == 0 {
							// Discard 0, draw 1.
							g.PlayerDrawCard(p)
							return nil
						}
						discarded := p.ChooseCardsFromHand(len(hand), "discard any number", g)
						for _, c := range discarded {
							g.PlayerDiscard(p, c.ID())
						}
						n := len(discarded) + 1
						for i := 0; i < n; i++ {
							g.PlayerDrawCard(p)
						}
						return nil
					},
				),
				false,
			)),
		)
	})

	// Conciliator's Duelist {W}{W}{B}{B}
	// Creature — Kor Warlock
	// 4/3
	// When this creature enters, draw a card. Each player loses 1 life.
	// Repartee — Whenever you cast an instant or sorcery spell that targets a creature, exile up to one target creature. Return that card to the battlefield under its owner's control at the beginning of the next end step.
	Register("Conciliator's Duelist", func() Card {
		return NewCreature("Conciliator's Duelist", "{W}{W}{B}{B}", 4, 3,
			WithSubTypes("Kor", "Warlock"),
			// When this creature enters, draw a card. Each player loses 1 life.
			WithAbility(EntersBattlefieldTrigger(
				FuncEffect(
					"draw a card; each player loses 1 life",
					EffectProperties{Outcome: OutcomeUnknown},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p != nil {
							g.PlayerDrawCard(p)
						}
						for _, player := range g.AllPlayers() {
							player.LoseLife(1)
						}
						return nil
					},
				),
				false,
			)),
			// Repartee — Whenever you cast an instant or sorcery spell that targets
			// a creature, exile up to one target creature. Return that card to the
			// battlefield under its owner's control at the beginning of the next end step.
			WithAbility(WheneverYouCastInstantOrSorceryTargetingCreatureTrigger(
				ExileTargetReturnAtEndStep(nil),
				false,
			).AddTarget(TargetUpToOneCreature())),
		)
	})

	// Cuboid Colony {G}{U}
	// Creature — Insect
	// 1/1
	// Flash
	// Flying, trample
	// Increment (Whenever you cast a spell, if the amount of mana you spent is greater than this creature's power or toughness, put a +1/+1 counter on this creature.)
	Register("Cuboid Colony", func() Card {
		return NewCreature("Cuboid Colony", "{G}{U}", 1, 1,
			WithSubTypes("Insect"),
			WithKeyword(Flash),
			WithKeyword(Flying),
			WithKeyword(Trample),
			WithAbility(IncrementTrigger()),
		)
	})

	// Elemental Mascot {1}{U}{R}
	// Creature — Elemental Bird
	// 1/4
	// Flying, vigilance
	// Opus — Whenever you cast an instant or sorcery spell, this creature gets +1/+0 until end of turn. If five or more mana was spent to cast that spell, exile the top card of your library. You may play that card until the end of your next turn.
	Register("Elemental Mascot", func() Card {
		isInstantOrSorcery := NewCardFilter("instant or sorcery", func(c Card) bool {
			return c.HasType(TypeInstant) || c.HasType(TypeSorcery)
		})
		return NewCreature("Elemental Mascot", "{1}{U}{R}", 1, 4,
			WithSubTypes("Elemental", "Bird"),
			WithKeyword(Flying),
			WithKeyword(Vigilance),
			// Opus — Whenever you cast an instant or sorcery spell, this creature gets
			// +1/+0 until end of turn. If five or more mana was spent to cast that spell,
			// exile the top card of your library. You may play that card until the end of
			// your next turn.
			WithAbility(WheneverYouCastSpellTrigger(
				OpusEffect(
					"this creature gets +1/+0 until end of turn; if 5+ mana spent, exile top card — may play until end of next turn",
					func(g *Game, sourceID, controller uuid.UUID, manaSpent int) error {
						boost := TemporaryBoost(sourceID, 1, 0)
						boost.SetSourceID(sourceID)
						g.AddContinuousEffect(boost)
						if manaSpent >= 5 {
							p := g.GetPlayer(controller)
							if p == nil {
								return nil
							}
							topCards := g.RemoveTopN(p, 1)
							if len(topCards) == 0 {
								return nil
							}
							card := topCards[0]
							g.ExileCard(card, sourceID)
							// XXX: "until the end of your next turn" duration is not
							// supported for cast-from-exile permissions; the permission
							// persists until the card leaves exile (indefinite).
							g.GrantCastFromExile(controller, card.ID(), false)
						}
						return nil
					},
				),
				false,
				isInstantOrSorcery,
			)),
		)
	})

	// Essenceknit Scholar {B}{B/G}{G}
	// Creature — Dryad Warlock
	// 3/1
	// When this creature enters, create a 1/1 black and green Pest creature token with "Whenever this token attacks, you gain 1 life."
	// At the beginning of your end step, if a creature died under your control this turn, draw a card.
	Register("Essenceknit Scholar", func() Card {
		pestTokenWithAttack := TokenWithAbilities(
			CreateColoredToken("Pest Token", 1, 1,
				[]Color{Black, Green},
				[]CardType{TypeCreature},
				[]string{"Pest"},
			),
			AttacksTrigger(GainLife(1), false),
		)
		return NewCreature("Essenceknit Scholar", "{B}{B/G}{G}", 3, 1,
			WithSubTypes("Dryad", "Warlock"),
			// When this creature enters, create a 1/1 black and green Pest creature token
			// with "Whenever this token attacks, you gain 1 life."
			WithAbility(EntersBattlefieldTrigger(
				pestTokenWithAttack,
				false,
			)),
			// At the beginning of your end step, if a creature died under your control this turn, draw a card.
			// XXX: engine tracks global creature deaths (both players); "under your control" is approximated.
			WithAbility(NewTriggered(EvtEndStep, false,
				FuncEffect("if a creature died under your control this turn, draw a card",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						if g.CreatureDeaths() > 0 {
							p := g.GetPlayer(controller)
							if p != nil {
								g.PlayerDrawCard(p)
							}
						}
						return nil
					},
				)).SetConditionData(EventPlayerIsController{})),
		)
	})

	// Fractal Mascot {4}{G}{U}
	// Creature — Fractal Elk
	// 6/6
	// Trample
	// When this creature enters, tap target creature an opponent controls. Put a stun counter on it. (If a permanent with a stun counter would become untapped, remove one from it instead.)
	// XXX: Stun counter (untap-prevention counter type) not in engine; tap is implemented but stun counter effect is not.
	Register("Fractal Mascot", func() Card {
		return NewCreature("Fractal Mascot", "{4}{G}{U}", 6, 6,
			WithSubTypes("Fractal", "Elk"),
			WithKeyword(Trample),
			WithAbility(EntersBattlefieldTrigger(
				FuncEffect(
					"tap target creature an opponent controls",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						opp := g.GetOpponent(controller)
						if opp == nil {
							return nil
						}
						var candidates []*Permanent
						candidates = append(candidates, g.FilterBattlefield(And(
							ControlledBy(opp.PlayerID()),
							IsCreature,
						))...)
						if len(candidates) == 0 {
							return nil
						}
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						chosen := p.ChoosePermanent(candidates, "tap target creature an opponent controls", g)
						if chosen == nil {
							return nil
						}
						g.TapPermanent(chosen)
						// XXX: put a stun counter on it (engine lacks stun counter type)
						return nil
					},
				),
				false,
			)),
		)
	})

	// Fractal Tender {3}{G}{U}
	// Creature — Elf Wizard
	// 3/3
	// Ward {2}
	// Increment (Whenever you cast a spell, if the amount of mana you spent is greater than this creature's power or toughness, put a +1/+1 counter on this creature.)
	// At the beginning of each end step, if you put a counter on this creature this turn, create a 0/0 green and blue Fractal creature token and put three +1/+1 counters on it.
	Register("Fractal Tender", func() Card {
		// incrementAndFlagEffect adds a +1/+1 counter when the Increment condition is
		// met and marks the creature with a Charge counter as a "counter-placed-this-turn"
		// flag. The Charge counter is a repurposed flag marker — Fractal Tender has no
		// normal use for Charge counters, so this is safe. The end-step trigger checks
		// for the flag and clears it after creating the Fractal token.
		incrementAndFlagEffect := OpusEffect(
			"if mana spent > this creature's power or toughness, put a +1/+1 counter on this; mark flag",
			func(g *Game, sourceID, controller uuid.UUID, manaSpent int) error {
				perm := g.FindPermanent(sourceID)
				if perm == nil {
					return nil
				}
				pow := perm.CurrentPower(g)
				tou := perm.CurrentToughness(g)
				if manaSpent > pow || manaSpent > tou {
					g.AddCountersWithReplacement(perm, P1P1, 1, sourceID, false)
					// Set a Charge counter as a "counter placed this turn" flag.
					perm.AddCounter(Charge, 1)
				}
				return nil
			},
		)
		createFractalToken := FuncEffect(
			"if counter placed this turn: create 0/0 green and blue Fractal token with three +1/+1 counters",
			EffectProperties{Outcome: OutcomeBenefit},
			func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
				perm := g.FindPermanent(sourceID)
				if perm == nil {
					return nil
				}
				if perm.Counters[Charge] == 0 {
					return nil
				}
				// Clear the flag.
				perm.RemoveCounter(Charge, int(perm.Counters[Charge]))
				token := NewToken("Fractal Token", 0, 0,
					[]CardType{TypeCreature}, []string{"Fractal"})
				token.SetOwner(controller)
				token.SetColorOverride([]Color{Green, Blue})
				tokenPerm := g.PutOnBattlefield(token, controller)
				if tokenPerm == nil {
					return nil
				}
				g.AddCountersWithReplacement(tokenPerm, P1P1, 3, sourceID, false)
				return nil
			},
		)
		return NewCreature("Fractal Tender", "{3}{G}{U}", 3, 3,
			WithSubTypes("Elf", "Wizard"),
			// XXX: Ward {2}. The engine has no Ward mechanic implementation.
			WithAbility(WheneverYouCastSpellTrigger(incrementAndFlagEffect, false)),
			WithAbility(BeginningOfEachEndStepTrigger(createFractalToken, false).
				SetConditionData(EventPlayerIsController{})),
		)
	})

	// Geometer's Arthropod {G}{U}
	// Creature — Fractal Crab
	// 1/4
	// Whenever you cast a spell with {X} in its mana cost, look at the top X cards of your library. Put one of them into your hand and the rest on the bottom of your library in a random order.
	Register("Geometer's Arthropod", func() Card {
		return NewCreature("Geometer's Arthropod", "{G}{U}", 1, 4,
			WithSubTypes("Fractal", "Crab"),
			WithAbility(WheneverYouCastSpellTrigger(
				FuncEffect(
					"look at top X cards, put one in hand and rest on bottom",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						// Find the triggering spell on the stack to get its X value.
						var x int
						objs := g.GetStack().Objects()
						for i := len(objs) - 1; i >= 0; i-- {
							if !objs[i].IsAbility {
								x = objs[i].XValue
								break
							}
						}
						if x <= 0 {
							return nil
						}
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						revealed := g.RemoveTopN(p, x)
						if len(revealed) == 0 {
							return nil
						}
						chosen := p.ChooseCardFromLibrary(revealed, "put into hand", g)
						var rest []Card
						for _, c := range revealed {
							if chosen == nil || c.ID() != chosen.ID() {
								rest = append(rest, c)
							} else {
								p.AddToHand(c)
								chosen = nil // mark as taken
							}
						}
						g.PutOnBottomInRandomOrder(p, rest)
						return nil
					},
				),
				false,
			).SetCondition(func(evt *GameEvent, g GameReader, _, controllerID uuid.UUID) bool {
				// Condition: the cast spell has {X} in its mana cost.
				if evt.PlayerID != controllerID {
					return false
				}
				card := g.FindCardAnywhere(evt.SourceID)
				if card == nil {
					return false
				}
				return card.ManaCost().HasX
			})),
		)
	})

	// Hardened Academic {R}{W}
	// Creature — Bird Cleric
	// 2/1
	// Flying, haste
	// Discard a card: This creature gains lifelink until end of turn.
	// Whenever one or more cards leave your graveyard, put a +1/+1 counter on target creature you control.
	// XXX: "Whenever one or more cards leave your graveyard" trigger not implemented (engine lacks graveyard-leave event).
	Register("Hardened Academic", func() Card {
		return NewCreature("Hardened Academic", "{R}{W}", 2, 1,
			WithSubTypes("Bird", "Cleric"),
			WithKeyword(Flying),
			WithKeyword(Haste),
			WithActivatedAbility(
				GrantKeyword(Lifelink).Targeting(ToSource()).Until(EndOfTurn),
				DiscardCost(1),
			),
		)
	})

	// Imperious Inkmage {1}{W}{B}
	// Creature — Orc Warlock
	// 3/3
	// Vigilance
	// When this creature enters, surveil 2. (Look at the top two cards of your library, then put any number of them into your graveyard and the rest on top of your library in any order.)
	Register("Imperious Inkmage", func() Card {
		return NewCreature("Imperious Inkmage", "{1}{W}{B}", 3, 3,
			WithSubTypes("Orc", "Warlock"),
			WithKeyword(Vigilance),
			WithAbility(EntersBattlefieldTrigger(surveilEffect(2), false)),
		)
	})

	// Inkling Mascot {W}{B}
	// Creature — Inkling Cat
	// 2/2
	// Repartee — Whenever you cast an instant or sorcery spell that targets a creature,
	// this creature gains flying until end of turn. Surveil 1. (Look at the top card of
	// your library. You may put it into your graveyard.)
	Register("Inkling Mascot", func() Card {
		return NewCreature("Inkling Mascot", "{W}{B}", 2, 2,
			WithSubTypes("Inkling", "Cat"),
			WithAbility(NewTriggered(EvtSpellCast, false,
				// XXX: Surveil 1 is not implemented in the engine; only the
				// "gains flying until end of turn" half is applied here.
				GrantKeyword(Flying).Targeting(ToSource()).Until(EndOfTurn),
			).SetCondition(reparteeCondition)),
		)
	})
	// Kirol, History Buff // Pack a Punch {R}{W} // {1}{R}{W}
	// Legendary Creature — Vampire Cleric // Sorcery
	// 2/3
	// TODO: implement
	Register("Kirol, History Buff // Pack a Punch", func() Card {
		return NewCreature("Kirol, History Buff // Pack a Punch", "{R}{W} // {1}{R}{W}", 2, 3,
			WithSubTypes("Vampire", "Cleric", "//", "Sorcery"),
			WithSuperTypes(SuperLegendary),
		)
	})

	// Lluwen, Exchange Student // Pest Friend {2}{B}{G} // {B/G}
	// Legendary Creature — Elf Druid // Sorcery
	// 3/4
	// TODO: implement
	Register("Lluwen, Exchange Student // Pest Friend", func() Card {
		return NewCreature("Lluwen, Exchange Student // Pest Friend", "{2}{B}{G} // {B/G}", 3, 4,
			WithSubTypes("Elf", "Druid", "//", "Sorcery"),
			WithSuperTypes(SuperLegendary),
		)
	})

	// Lorehold, the Historian {3}{R}{W}
	// Legendary Creature — Elder Dragon
	// 5/5
	// Flying, haste
	// Each instant and sorcery card in your hand has miracle {2}. (You may cast a card for its miracle cost when you draw it if it's the first card you drew this turn.)
	// At the beginning of each opponent's upkeep, you may discard a card. If you do, draw a card.
	Register("Lorehold, the Historian", func() Card {
		return NewCreature("Lorehold, the Historian", "{3}{R}{W}", 5, 5,
			WithSubTypes("Elder", "Dragon"),
			WithSuperTypes(SuperLegendary),
			WithKeyword(Flying),
			WithKeyword(Haste),
			// XXX: "Each instant and sorcery card in your hand has miracle {2}" not implemented —
			// engine does not support granting miracle costs to cards in hand.
			// At the beginning of each opponent's upkeep, you may discard a card. If you do, draw a card.
			WithAbility(NewTriggered(EvtUpkeep, false,
				FuncEffect(
					"you may discard a card; if you do, draw a card",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil || len(p.Hand()) == 0 {
							return nil
						}
						if !p.ChooseMayAbility("discard a card to draw a card") {
							return nil
						}
						chosen := p.ChooseCardsFromHand(1, "discard a card", g)
						for _, c := range chosen {
							g.PlayerDiscard(p, c.ID())
						}
						g.PlayerDrawCard(p)
						return nil
					},
				),
			).SetConditionData(EventPlayerIsNotController{})),
		)
	})

	// Nita, Forum Conciliator {1}{W}{B}
	// Legendary Creature — Human Advisor
	// 2/3
	// Whenever you cast a spell you don't own, put a +1/+1 counter on each creature you control.
	// {2}, Sacrifice another creature: Exile target instant or sorcery card from an opponent's graveyard. You may cast it this turn, and mana of any type can be spent to cast that spell. If that spell would be put into a graveyard, exile it instead. Activate only as a sorcery.
	Register("Nita, Forum Conciliator", func() Card {
		// XXX: "Whenever you cast a spell you don't own" trigger not implemented — engine
		// does not track spell ownership separately from controller.
		instantOrSorceryCard := NewCardFilter("instant or sorcery card", func(c Card) bool {
			return c.HasType(TypeInstant) || c.HasType(TypeSorcery)
		})
		return NewCreature("Nita, Forum Conciliator", "{1}{W}{B}", 2, 3,
			WithSubTypes("Human", "Advisor"),
			WithSuperTypes(SuperLegendary),
			// {2}, Sacrifice another creature: Exile target instant or sorcery card from an
			// opponent's graveyard. You may cast it this turn, and mana of any type can be
			// spent to cast that spell. If that spell would be put into a graveyard, exile it
			// instead. Activate only as a sorcery.
			WithActivatedAbility(
				FuncEffect(
					"exile target instant or sorcery from an opponent's graveyard; may cast it this turn with any mana",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						// Collect instant/sorcery cards from all opponents' graveyards.
						var candidates []Card
						for _, opp := range g.AllPlayers() {
							if opp.PlayerID() == controller {
								continue
							}
							for _, c := range opp.Graveyard() {
								if instantOrSorceryCard.Match(c) {
									candidates = append(candidates, c)
								}
							}
						}
						if len(candidates) == 0 {
							return nil
						}
						chosen := p.ChooseCardFromLibrary(candidates, "exile target instant or sorcery card from an opponent's graveyard", g)
						if chosen == nil {
							return nil
						}
						// Remove from owner's graveyard and exile it.
						for _, opp := range g.AllPlayers() {
							if removed, ok := opp.RemoveFromGraveyard(chosen.ID()); ok {
								g.ExileCard(removed, sourceID)
								g.GrantCastFromExile(controller, removed.ID(), true)
								g.AddExileIfWouldGoToGraveyardThisTurn(removed.ID(), sourceID)
								break
							}
						}
						return nil
					},
				),
				ManaCostOf("{2}"),
				WithCost(SacrificeCreatureCost()),
				WithSorcerySpeed(),
			),
		)
	})

	// Old-Growth Educator {2}{B}{G}
	// Creature — Treefolk Druid
	// 4/4
	// Vigilance, reach
	// Infusion — When this creature enters, put two +1/+1 counters on it if you gained life this turn.
	Register("Old-Growth Educator", func() Card {
		return NewCreature("Old-Growth Educator", "{2}{B}{G}", 4, 4,
			WithSubTypes("Treefolk", "Druid"),
			WithKeyword(Vigilance),
			WithKeyword(Reach),
			// Infusion — When this creature enters, put two +1/+1 counters on it if you
			// gained life this turn.
			WithAbility(EntersBattlefieldTrigger(
				InfusionEffect("put two +1/+1 counters on this creature",
					FuncEffect("put two +1/+1 counters on this creature",
						EffectProperties{Outcome: OutcomeBenefit},
						func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							perm := g.FindPermanent(sourceID)
							if perm == nil {
								return nil
							}
							perm.AddCounter(P1P1, 2)
							g.ApplyContinuousEffects()
							return nil
						},
					),
				),
				false,
			)),
		)
	})
	// Paradox Surveyor {G}{G/U}{U}
	// Creature — Elf Druid
	// 3/3
	// Reach
	// When this creature enters, look at the top five cards of your library. You may reveal a land card or a card with {X} in its mana cost from among them and put it into your hand. Put the rest on the bottom of your library in a random order.
	Register("Paradox Surveyor", func() Card {
		landOrXCostCard := NewCardFilter("land card or card with {X} in its mana cost", func(c Card) bool {
			if c.HasType(TypeLand) {
				return true
			}
			return c.ManaCost().HasX
		})
		return NewCreature("Paradox Surveyor", "{G}{G/U}{U}", 3, 3,
			WithSubTypes("Elf", "Druid"),
			WithKeyword(Reach),
			// When this creature enters, look at the top five cards of your library. You may
			// reveal a land card or a card with {X} in its mana cost from among them and put
			// it into your hand. Put the rest on the bottom of your library in a random order.
			WithAbility(EntersBattlefieldTrigger(
				FuncEffect(
					"look at top 5; may put a land or X-cost card into your hand; rest on bottom in random order",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						chosen, _ := g.RevealAndPickFromTop(p, p, 5, landOrXCostCard, true,
							"reveal a land card or a card with {X} in its mana cost and put it into your hand")
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
						g.PutOnBottomInRandomOrder(p, rest)
						return nil
					},
				),
				false,
			)),
		)
	})

	// Pest Mascot {1}{B}{G}
	// Creature — Pest Ape
	// 2/3
	// Trample
	// Whenever you gain life, put a +1/+1 counter on this creature.
	Register("Pest Mascot", func() Card {
		return NewCreature("Pest Mascot", "{1}{B}{G}", 2, 3,
			WithSubTypes("Pest", "Ape"),
			WithKeyword(Trample),
			WithAbility(WheneverYouGainLifeTrigger(
				AddCounters(P1P1, Fixed(1)).Targeting(ToSource()),
				false,
			)),
		)
	})

	// Practiced Scrollsmith {R}{R/W}{W}
	// Creature — Dwarf Cleric
	// 3/2
	// First strike
	// When this creature enters, exile target noncreature, nonland card from your graveyard. Until the end of your next turn, you may cast that card.
	Register("Practiced Scrollsmith", func() Card {
		isNonCreatureNonLandCard := NewCardFilter("noncreature, nonland card", func(c Card) bool {
			return !c.HasType(TypeCreature) && !c.HasType(TypeLand)
		})
		return NewCreature("Practiced Scrollsmith", "{R}{R/W}{W}", 3, 2,
			WithSubTypes("Dwarf", "Cleric"),
			WithKeyword(FirstStrike),
			// When this creature enters, exile target noncreature, nonland card from
			// your graveyard. Until the end of your next turn, you may cast that card.
			WithAbility(EntersBattlefieldTrigger(
				FuncEffect("exile target noncreature, nonland card from your graveyard; until end of your next turn you may cast it",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						cardID := targets[0]
						card, ok := p.RemoveFromGraveyard(cardID)
						if !ok {
							return nil
						}
						g.ExileCard(card, sourceID)
						// XXX: "until the end of your next turn" duration is not
						// supported for cast-from-exile permissions; the permission
						// persists until the card leaves exile (indefinite).
						g.GrantCastFromExile(controller, card.ID(), false)
						return nil
					}),
				false,
			).AddTarget(TargetCardInYourGraveyard(isNonCreatureNonLandCard))),
		)
	})

	// Prismari, the Inspiration {5}{U}{R}
	// Legendary Creature — Elder Dragon
	// 7/7
	// Flying
	// Ward—Pay 5 life.
	// Instant and sorcery spells you cast have storm. (Whenever you cast an instant or sorcery spell, copy it for each spell cast before it this turn. You may choose new targets for the copies.)
	// TODO: implement
	Register("Prismari, the Inspiration", func() Card {
		return NewCreature("Prismari, the Inspiration", "{5}{U}{R}", 7, 7,
			WithSubTypes("Elder", "Dragon"),
			WithSuperTypes(SuperLegendary),
		)
	})

	// Pterafractyl {X}{G}{U}
	// Creature — Dinosaur Fractal
	// 1/0
	// Flying
	// This creature enters with X +1/+1 counters on it.
	// When this creature enters, you gain 2 life.
	Register("Pterafractyl", func() Card {
		return NewCreature("Pterafractyl", "{X}{G}{U}", 1, 0,
			WithSubTypes("Dinosaur", "Fractal"),
			WithKeyword(Flying),
			WithAbility(EntersWithXCounters(P1P1)),
			WithAbility(ETBEffect(GainLife(2))),
		)
	})

	// Quandrix, the Proof {4}{G}{U}
	// Legendary Creature — Elder Dragon
	// 6/6
	// Flying, trample
	// Cascade (When you cast this spell, exile cards from the top of your library until you exile a nonland card that costs less. You may cast it without paying its mana cost. Put the exiled cards on the bottom in a random order.)
	// Instant and sorcery spells you cast from your hand have cascade.
	// TODO: implement
	Register("Quandrix, the Proof", func() Card {
		return NewCreature("Quandrix, the Proof", "{4}{G}{U}", 6, 6,
			WithSubTypes("Elder", "Dragon"),
			WithSuperTypes(SuperLegendary),
		)
	})

	// Sanar, Unfinished Genius // Wild Idea {U}{R} // {3}{U}{R}
	// Legendary Creature — Goblin Sorcerer // Sorcery
	// 0/4
	// TODO: implement
	Register("Sanar, Unfinished Genius // Wild Idea", func() Card {
		return NewCreature("Sanar, Unfinished Genius // Wild Idea", "{U}{R} // {3}{U}{R}", 0, 4,
			WithSubTypes("Goblin", "Sorcerer", "//", "Sorcery"),
			WithSuperTypes(SuperLegendary),
		)
	})

	// Scolding Administrator {W}{B}
	// Creature — Dwarf Cleric
	// 2/2
	// Menace (This creature can't be blocked except by two or more creatures.)
	// Repartee — Whenever you cast an instant or sorcery spell that targets a creature, put a +1/+1 counter on this creature.
	// When this creature dies, if it had counters on it, put those counters on up to one target creature.
	// TODO: implement
	Register("Scolding Administrator", func() Card {
		return NewCreature("Scolding Administrator", "{W}{B}", 2, 2,
			WithSubTypes("Dwarf", "Cleric"),
		)
	})

	// Silverquill, the Disputant {2}{W}{B}
	// Legendary Creature — Elder Dragon
	// 4/4
	// Flying, vigilance
	// Each instant and sorcery spell you cast has casualty 1. (As you cast that spell, you may sacrifice a creature with power 1 or greater. When you do, copy the spell and you may choose new targets for the copy.)
	// TODO: implement
	Register("Silverquill, the Disputant", func() Card {
		return NewCreature("Silverquill, the Disputant", "{2}{W}{B}", 4, 4,
			WithSubTypes("Elder", "Dragon"),
			WithSuperTypes(SuperLegendary),
		)
	})

	// Snooping Page {1}{W}{B}
	// Creature — Human Cleric
	// 2/3
	// Repartee — Whenever you cast an instant or sorcery spell that targets a creature, this creature can't be blocked this turn.
	// Whenever this creature deals combat damage to a player, you draw a card and lose 1 life.
	// TODO: implement
	Register("Snooping Page", func() Card {
		return NewCreature("Snooping Page", "{1}{W}{B}", 2, 3,
			WithSubTypes("Human", "Cleric"),
		)
	})

	// Spectacular Skywhale {2}{U}{R}
	// Creature — Elemental Whale
	// 1/4
	// Flying
	// Opus — Whenever you cast an instant or sorcery spell, this creature gets +3/+0 until end of turn. If five or more mana was spent to cast that spell, put three +1/+1 counters on this creature instead.
	// TODO: implement
	Register("Spectacular Skywhale", func() Card {
		return NewCreature("Spectacular Skywhale", "{2}{U}{R}", 1, 4,
			WithSubTypes("Elemental", "Whale"),
		)
	})

	// Spirit Mascot {R}{W}
	// Creature — Spirit Ox
	// 2/2
	// Whenever one or more cards leave your graveyard, put a +1/+1 counter on this creature.
	// XXX: "Whenever one or more cards leave your graveyard" trigger not implemented (engine lacks EvtLeaveGraveyard event).
	Register("Spirit Mascot", func() Card {
		return NewCreature("Spirit Mascot", "{R}{W}", 2, 2,
			WithSubTypes("Spirit", "Ox"),
		)
	})

	// Stadium Tidalmage {2}{U}{R}
	// Creature — Djinn Sorcerer
	// 4/4
	// Whenever this creature enters or attacks, you may draw a card. If you do, discard a card.
	Register("Stadium Tidalmage", func() Card {
		drawDiscard := FuncEffect(
			"you may draw a card, if you do discard a card",
			EffectProperties{Outcome: OutcomeBenefit},
			func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
				p := g.GetPlayer(controller)
				if p == nil {
					return nil
				}
				if !p.ChooseMayAbility("draw a card then discard a card") {
					return nil
				}
				drawn, ok := g.PlayerDrawCard(p)
				if !ok || drawn == nil {
					return nil
				}
				currentHand := p.Hand()
				if len(currentHand) == 0 {
					return nil
				}
				chosen := p.ChooseCardsFromHand(1, "discard a card", g)
				if len(chosen) == 0 {
					return nil
				}
				g.PlayerDiscard(p, chosen[0].ID())
				return nil
			},
		)
		return NewCreature("Stadium Tidalmage", "{2}{U}{R}", 4, 4,
			WithSubTypes("Djinn", "Sorcerer"),
			WithAbility(EntersBattlefieldTrigger(drawDiscard, false)),
			WithAbility(AttacksTrigger(drawDiscard, false)),
		)
	})

	// Startled Relic Sloth {2}{R}{W}
	// Creature — Sloth Beast
	// 4/4
	// Trample, lifelink
	// At the beginning of combat on your turn, exile up to one target card from a graveyard.
	Register("Startled Relic Sloth", func() Card {
		return NewCreature("Startled Relic Sloth", "{2}{R}{W}", 4, 4,
			WithSubTypes("Sloth", "Beast"),
			WithKeyword(Trample),
			WithKeyword(Lifelink),
			WithAbility(NewTriggered(EvtBeginCombat, true,
				FuncEffect("exile up to one target card from a graveyard",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						var candidates []Card
						for _, pl := range g.AllPlayers() {
							candidates = append(candidates, pl.Graveyard()...)
						}
						if len(candidates) == 0 {
							return nil
						}
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						chosen := p.ChooseCardFromLibrary(candidates, "exile up to one card from a graveyard", g)
						if chosen == nil {
							return nil
						}
						for _, pl := range g.AllPlayers() {
							if removed, ok := pl.RemoveFromGraveyard(chosen.ID()); ok {
								g.ExileCard(removed, sourceID)
								break
							}
						}
						return nil
					},
				)).SetConditionData(EventPlayerIsController{})),
		)
	})

	// Stirring Honormancer {2}{W}{W/B}{B}
	// Creature — Rhino Bard
	// 4/5
	// When this creature enters, look at the top X cards of your library, where X is the number of creatures you control. Put one of those cards into your hand and the rest into your graveyard.
	Register("Stirring Honormancer", func() Card {
		return NewCreature("Stirring Honormancer", "{2}{W}{W/B}{B}", 4, 5,
			WithSubTypes("Rhino", "Bard"),
			WithAbility(EntersBattlefieldTrigger(
				FuncEffect(
					"look at top X cards, put one into hand, rest into graveyard",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						x := g.CountBattlefield(And(IsCreature, ControlledBy(controller)))
						if x <= 0 {
							return nil
						}
						revealed := g.RemoveTopN(p, x)
						if len(revealed) == 0 {
							return nil
						}
						chosen := p.ChooseCardFromLibrary(revealed, "put into hand", g)
						for _, c := range revealed {
							if chosen != nil && c.ID() == chosen.ID() {
								p.AddToHand(c)
								chosen = nil
							} else {
								p.AddToGraveyard(c)
							}
						}
						return nil
					},
				),
				false,
			)),
		)
	})

	// Tam, Observant Sequencer // Deep Sight {2}{G}{U} // {G}{U}
	// Legendary Creature — Gorgon Wizard // Sorcery
	// 4/3
	// TODO: implement
	Register("Tam, Observant Sequencer // Deep Sight", func() Card {
		return NewCreature("Tam, Observant Sequencer // Deep Sight", "{2}{G}{U} // {G}{U}", 4, 3,
			WithSubTypes("Gorgon", "Wizard", "//", "Sorcery"),
			WithSuperTypes(SuperLegendary),
		)
	})

	// Teacher's Pest {B}{G}
	// Creature — Skeleton Pest
	// 1/1
	// Menace (This creature can't be blocked except by two or more creatures.)
	// Whenever this creature attacks, you gain 1 life.
	// {B}{G}: Return this card from your graveyard to the battlefield tapped.
	Register("Teacher's Pest", func() Card {
		return NewCreature("Teacher's Pest", "{B}{G}", 1, 1,
			WithSubTypes("Skeleton", "Pest"),
			WithKeyword(Menace),
			// Whenever this creature attacks, you gain 1 life.
			WithAbility(AttacksTrigger(GainLife(1), false)),
			// {B}{G}: Return this card from your graveyard to the battlefield tapped.
			WithGraveyardActivatedAbility(
				FuncEffect("return this card from your graveyard to the battlefield tapped",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						removed, ok := p.RemoveFromGraveyard(sourceID)
						if !ok {
							return nil
						}
						perm := g.PutOnBattlefield(removed, controller)
						if perm != nil {
							perm.Tapped = true
						}
						return nil
					},
				),
				ManaCostOf("{B}{G}"),
			),
		)
	})

	// Witherbloom, the Balancer {6}{B}{G}
	// Legendary Creature — Elder Dragon
	// 5/5
	// Affinity for creatures (This spell costs {1} less to cast for each creature you control.)
	// Flying, deathtouch
	// Instant and sorcery spells you cast have affinity for creatures.
	Register("Witherbloom, the Balancer", func() Card {
		return NewCreature("Witherbloom, the Balancer", "{6}{B}{G}", 5, 5,
			WithSubTypes("Elder", "Dragon"),
			WithSuperTypes(SuperLegendary),
			// Affinity for creatures: costs {1} less to cast for each creature you control.
			WithSelfCostReduction(AmountByPermanentCount(IsCreature), nil),
			WithKeyword(Flying),
			WithKeyword(Deathtouch),
			// Instant and sorcery spells you cast have affinity for creatures.
			WithStaticAbility(ReduceSpellCostStatic(
				SpellsOr(SpellHasType(TypeInstant), SpellHasType(TypeSorcery)),
				AmountByPermanentCount(IsCreature),
				nil,
			)),
		)
	})

	// Zaffai and the Tempests {5}{U}{R}
	// Legendary Creature — Human Bard Sorcerer
	// 5/7
	// Once during each of your turns, you may cast an instant or sorcery spell from your hand without paying its mana cost.
	Register("Zaffai and the Tempests", func() Card {
		return NewCreature("Zaffai and the Tempests", "{5}{U}{R}", 5, 7,
			WithSubTypes("Human", "Bard", "Sorcerer"),
			WithSuperTypes(SuperLegendary),
			// Once during each of your turns, you may cast an instant or sorcery spell from
			// your hand without paying its mana cost.
			WithActivatedAbility(
				FuncEffect(
					"cast an instant or sorcery spell from your hand without paying its mana cost",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						var candidates []Card
						for _, c := range p.Hand() {
							if c.HasType(TypeInstant) || c.HasType(TypeSorcery) {
								candidates = append(candidates, c)
							}
						}
						if len(candidates) == 0 {
							return nil
						}
						chosen := p.ChooseCardFromLibrary(candidates, "cast an instant or sorcery from your hand without paying its mana cost", g)
						if chosen == nil {
							return nil
						}
						// Gather targets for the chosen spell.
						var castTargets []uuid.UUID
						for _, a := range chosen.Abilities() {
							sa, ok := a.(*SpellAbility)
							if !ok {
								continue
							}
							for _, t := range sa.Targets() {
								t.Reset()
								possible := t.Possible(controller, chosen, g)
								if len(possible) == 0 {
									continue
								}
								selected := p.ChooseTargets(possible, t.Min(), t.Max(), g)
								if err := t.Choose(controller, chosen, g, selected); err != nil {
									continue
								}
								castTargets = append(castTargets, selected...)
							}
							break
						}
						return g.CastCardFromZoneWithoutPaying(controller, chosen.ID(), ZoneHand, castTargets, 0)
					},
				),
				ManaCostOf("{0}"),
				WithOncePerTurn(),
				WithSorcerySpeed(),
			),
		)
	})

	// ===== COLORLESS CREATURES =====

	// Biblioplex Tomekeeper {4}
	// Artifact Creature — Construct
	// 3/4
	// When this creature enters, choose up to one —
	// • Target creature becomes prepared. (Only creatures with prepare spells can become prepared.)
	// • Target creature becomes unprepared.
	Register("Biblioplex Tomekeeper", func() Card {
		// hasPreparedSpellFilter matches creatures that have a prepared spell.
		hasPreparedSpellFilter := NewPermanentFilter("creature with a prepared spell", func(perm *Permanent, _ *Game) bool {
			return HasPreparedSpell(perm.Card)
		})
		return NewCreature("Biblioplex Tomekeeper", "{4}", 3, 4,
			WithSubTypes("Construct"),
			WithCardType(TypeArtifact),
			// When this creature enters, choose up to one —
			// • Target creature becomes prepared.
			// • Target creature becomes unprepared.
			// "Choose up to one" is modeled with three modes: do nothing (mode 2),
			// becomes prepared (mode 0), becomes unprepared (mode 1).
			WithAbility(EntersBattlefieldTrigger(nil, false).WithModes(
				Mode{
					Label:   "Target creature becomes prepared",
					Targets: []Target{TargetCreature(hasPreparedSpellFilter)},
					Effects: []Effect{FuncEffect(
						"target creature becomes prepared",
						EffectProperties{Outcome: OutcomeBenefit},
						func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
							if len(targets) == 0 {
								return nil
							}
							g.SetPrepared(targets[0], true)
							return nil
						},
					)},
				},
				Mode{
					Label:   "Target creature becomes unprepared",
					Targets: []Target{TargetCreature()},
					Effects: []Effect{FuncEffect(
						"target creature becomes unprepared",
						EffectProperties{Outcome: OutcomeDetriment},
						func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
							if len(targets) == 0 {
								return nil
							}
							g.SetPrepared(targets[0], false)
							return nil
						},
					)},
				},
				Mode{
					Label:   "Do nothing",
					Targets: nil,
					Effects: nil,
				},
			)),
		)
	})

	// Mage Tower Referee {2}
	// Artifact Creature — Construct
	// 2/1
	// Whenever you cast a multicolored spell, put a +1/+1 counter on this creature.
	Register("Mage Tower Referee", func() Card {
		isMulticoloredCard := NewCardFilter("multicolored", func(c Card) bool {
			return len(c.ManaCost().Colors()) >= 2
		})
		return NewCreature("Mage Tower Referee", "{2}", 2, 1,
			WithSubTypes("Construct"),
			WithCardType(TypeArtifact),
			// Whenever you cast a multicolored spell, put a +1/+1 counter on this creature.
			WithAbility(WheneverYouCastSpellTrigger(
				AddCounters(P1P1, Fixed(1)).Targeting(ToSource()),
				false,
				isMulticoloredCard,
			)),
		)
	})

	// Page, Loose Leaf {2}
	// Legendary Artifact Creature — Construct
	// 0/2
	// {T}: Add {C}.
	// Grandeur — Discard another card named Page, Loose Leaf: Reveal cards from the top of your library until you reveal an instant or sorcery card. Put that card into your hand and the rest on the bottom of your library in a random order.
	// TODO: implement
	Register("Page, Loose Leaf", func() Card {
		return NewCreature("Page, Loose Leaf", "{2}", 0, 2,
			WithSubTypes("Construct"),
			WithSuperTypes(SuperLegendary),
			WithCardType(TypeArtifact),
		)
	})

	// Rancorous Archaic {5}
	// Creature — Avatar
	// 2/2
	// Trample, reach
	// Converge — This creature enters with a +1/+1 counter on it for each color of mana spent to cast it.
	Register("Rancorous Archaic", func() Card {
		return NewCreature("Rancorous Archaic", "{5}", 2, 2,
			WithSubTypes("Avatar"),
			WithKeyword(Trample),
			WithKeyword(Reach),
			// Converge — enters with a +1/+1 counter for each distinct color of mana spent to cast it.
			// EntersWithComputedCounters runs inside PutOnBattlefield while resolvingCastContext is still set.
			WithAbility(EntersWithComputedCounters(P1P1, func(g *Game, _ *Permanent) int {
				ctx := g.ResolvingCastContext()
				if ctx == nil {
					return 0
				}
				return ctx.DistinctColorsSpent()
			})),
		)
	})

	// Sundering Archaic {6}
	// Creature — Avatar
	// 3/3
	// Converge — When this creature enters, exile target nonland permanent an opponent controls with mana value less than or equal to the number of colors of mana spent to cast this creature.
	// {2}: Put target card from a graveyard on the bottom of its owner's library.
	Register("Sundering Archaic", func() Card {
		return NewCreature("Sundering Archaic", "{6}", 3, 3,
			WithSubTypes("Avatar"),
			// Converge — capture colors-spent during PutOnBattlefield (while resolvingCastContext is live)
			// so the ETB trigger effect can read it after the cast context has been cleared.
			WithAbility(EntersWithComputedCounters(Charge, func(g *Game, perm *Permanent) int {
				ctx := g.ResolvingCastContext()
				if ctx == nil {
					return 0
				}
				return ctx.DistinctColorsSpent()
			})),
			// Converge — When this creature enters, exile target nonland permanent an opponent controls
			// with mana value less than or equal to the number of colors of mana spent to cast this creature.
			WithAbility(EntersBattlefieldTrigger(
				FuncEffect("exile target nonland permanent an opponent controls with MV <= colors spent",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 || targets[0] == uuid.Nil {
							return nil
						}
						src := g.FindPermanent(sourceID)
						if src == nil {
							return nil
						}
						colors := int(src.Counters[Charge])
						// Remove the Charge counters used for Converge bookkeeping.
						src.RemoveCounter(Charge, colors)
						target := g.FindPermanent(targets[0])
						if target == nil {
							return nil
						}
						if target.HasType(TypeLand) {
							return nil
						}
						if target.Card.ManaCost().CMC() > colors {
							return nil
						}
						g.ExilePermanent(target)
						return nil
					},
				),
				false,
			).AddTarget(TargetPermanentOpponentControls(Not(IsLand)))),
			// XXX: {2}: Put target card from a graveyard on the bottom of its owner's library.
			// Not tested; skipping for now.
		)
	})

	// The Dawning Archaic {10}
	// Legendary Creature — Avatar
	// 7/7
	// This spell costs {1} less to cast for each instant and sorcery card in your graveyard.
	// Reach
	// Whenever The Dawning Archaic attacks, you may cast target instant or sorcery card from your graveyard without paying its mana cost. If that spell would be put into your graveyard, exile it instead.
	Register("The Dawning Archaic", func() Card {
		return NewCreature("The Dawning Archaic", "{10}", 7, 7,
			WithSubTypes("Avatar"),
			WithSuperTypes(SuperLegendary),
			WithKeyword(Reach),
			WithSelfCostReduction(
				AmountByGraveyardCount(IsInstantOrSorceryCard),
				nil,
			),
			// XXX: attack trigger "cast instant/sorcery from graveyard for free; exile instead of graveyard" not implemented.
		)
	})

	// Transcendent Archaic {7}
	// Creature — Avatar
	// 6/6
	// Vigilance
	// Converge — When this creature enters, you may draw X cards, where X is the number of colors of mana spent to cast this spell. If you draw one or more cards this way, discard two cards.
	Register("Transcendent Archaic", func() Card {
		return NewCreature("Transcendent Archaic", "{7}", 6, 6,
			WithSubTypes("Avatar"),
			WithKeyword(Vigilance),
			WithAbility(ETBEffect(FuncEffect(
				"converge: you may draw X cards (X=colors spent), discard 2 if drew any",
				EffectProperties{Outcome: OutcomeBenefit, DrawCount: 1},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					// Converge: count only actual colors (not colorless) from ColorsSpent.
					x := 0
					ctx := g.ResolvingCastContext()
					if ctx != nil {
						for c, v := range ctx.ColorsSpent {
							if v > 0 && c != Colorless {
								x++
							}
						}
					}
					if x == 0 {
						return nil
					}
					p := g.GetPlayer(controller)
					if p == nil {
						return nil
					}
					if !p.ChooseMayAbility("draw X cards where X is the number of colors spent") {
						return nil
					}
					drawn := 0
					for i := 0; i < x; i++ {
						card, ok := g.PlayerDrawCard(p)
						if !ok || card == nil {
							break
						}
						drawn++
					}
					if drawn >= 1 {
						for i := 0; i < 2; i++ {
							hand := p.Hand()
							if len(hand) == 0 {
								break
							}
							chosen := p.ChooseCardsFromHand(1, "discard a card", g)
							for _, c := range chosen {
								g.PlayerDiscard(p, c.ID())
							}
						}
					}
					return nil
				},
			))),
		)
	})

}
