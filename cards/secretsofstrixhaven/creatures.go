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
// Repartee — Whenever you cast an instant or sorcery spell that targets a creature, create a 1/1 white and black Inkling creature token with flying.
// TODO: implement
	Register("Informed Inkwright", func() Card {
		return NewCreature("Informed Inkwright", "{1}{W}", 2, 2,
			WithSubTypes("Human", "Wizard"),
		)
	})

// Inkshape Demonstrator {3}{W}
// Creature — Elephant Cleric
// 3/4
// Ward {2} (Whenever this creature becomes the target of a spell or ability an opponent controls, counter it unless that player pays {2}.)
// Repartee — Whenever you cast an instant or sorcery spell that targets a creature, this creature gets +1/+0 and gains lifelink until end of turn.
// TODO: implement
	Register("Inkshape Demonstrator", func() Card {
		return NewCreature("Inkshape Demonstrator", "{3}{W}", 3, 4,
			WithSubTypes("Elephant", "Cleric"),
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
// TODO: implement
	Register("Rehearsed Debater", func() Card {
		return NewCreature("Rehearsed Debater", "{2}{W}", 3, 3,
			WithSubTypes("Djinn", "Bard"),
		)
	})

// Shattered Acolyte {1}{W}
// Creature — Dwarf Warlock
// 2/2
// Lifelink
// {1}, Sacrifice this creature: Destroy target artifact or enchantment.
// TODO: implement
	Register("Shattered Acolyte", func() Card {
		return NewCreature("Shattered Acolyte", "{1}{W}", 2, 2,
			WithSubTypes("Dwarf", "Warlock"),
		)
	})

// Soaring Stoneglider {2}{W}
// Creature — Elephant Cleric
// 4/3
// As an additional cost to cast this spell, exile two cards from your graveyard or pay {1}{W}.
// Flying, vigilance
// TODO: implement
	Register("Soaring Stoneglider", func() Card {
		return NewCreature("Soaring Stoneglider", "{2}{W}", 4, 3,
			WithSubTypes("Elephant", "Cleric"),
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
// TODO: implement
	Register("Stone Docent", func() Card {
		return NewCreature("Stone Docent", "{1}{W}", 3, 1,
			WithSubTypes("Spirit", "Chimera"),
		)
	})

// Summoned Dromedary {3}{W}
// Creature — Spirit Camel
// 4/3
// Vigilance
// {1}{W}: Return this card from your graveyard to your hand. Activate only as a sorcery.
// TODO: implement
	Register("Summoned Dromedary", func() Card {
		return NewCreature("Summoned Dromedary", "{3}{W}", 4, 3,
			WithSubTypes("Spirit", "Camel"),
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
// TODO: implement
	Register("Deluge Virtuoso", func() Card {
		return NewCreature("Deluge Virtuoso", "{2}{U}", 2, 2,
			WithSubTypes("Human", "Wizard"),
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
// TODO: implement
	Register("Exhibition Tidecaller", func() Card {
		return NewCreature("Exhibition Tidecaller", "{U}", 0, 2,
			WithSubTypes("Djinn", "Wizard"),
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
			WithAbility(EntersBattlefieldTrigger(
				FuncEffect("return up to one other target creature to its owner's hand",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						target := targets[0]
						if target == uuid.Nil {
							return nil
						}
						perm := g.FindPermanent(target)
						if perm == nil {
							return nil
						}
						owner := g.GetPlayer(perm.Card.Owner())
						if owner == nil {
							return nil
						}
						g.RemoveFromBattlefield(perm)
						owner.AddToHand(perm.Card)
						return nil
					},
				),
				false,
			).AddTarget(TargetUpToOneCreature())),
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
// TODO: implement
	Register("Muse Seeker", func() Card {
		return NewCreature("Muse Seeker", "{1}{U}", 1, 2,
			WithSubTypes("Elf", "Wizard"),
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
// TODO: implement
	Register("Forum Necroscribe", func() Card {
		return NewCreature("Forum Necroscribe", "{5}{B}", 5, 4,
			WithSubTypes("Troll", "Warlock"),
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
// Repartee — Whenever you cast an instant or sorcery spell that targets a creature, put a +1/+1 counter on this creature.
// TODO: implement
	Register("Lecturing Scornmage", func() Card {
		return NewCreature("Lecturing Scornmage", "{B}", 1, 1,
			WithSubTypes("Human", "Warlock"),
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
// TODO: implement
	Register("Melancholic Poet", func() Card {
		return NewCreature("Melancholic Poet", "{1}{B}", 2, 2,
			WithSubTypes("Elf", "Bard"),
		)
	})

// Moseo, Vein's New Dean {2}{B}
// Legendary Creature — Bird Skeleton Warlock
// 2/1
// Flying
// When Moseo enters, create a 1/1 black and green Pest creature token with "Whenever this token attacks, you gain 1 life."
// Infusion — At the beginning of your end step, if you gained life this turn, return up to one target creature card with mana value X or less from your graveyard to the battlefield, where X is the amount of life you gained this turn.
// TODO: implement
	Register("Moseo, Vein's New Dean", func() Card {
		return NewCreature("Moseo, Vein's New Dean", "{2}{B}", 2, 1,
			WithSubTypes("Bird", "Skeleton", "Warlock"),
			WithSuperTypes(SuperLegendary),
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
			WithGraveyardActivatedAbility(
				ReturnSourceFromGraveyardToBattlefield(),
				ManaCostOf("{1}{B}"),
				WithCost(ExileFromGraveyardCost(1)),
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
// TODO: implement
	Register("Sneering Shadewriter", func() Card {
		return NewCreature("Sneering Shadewriter", "{4}{B}", 3, 3,
			WithSubTypes("Vampire", "Warlock"),
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
// TODO: implement
	Register("Ulna Alley Shopkeep", func() Card {
		return NewCreature("Ulna Alley Shopkeep", "{2}{B}", 2, 3,
			WithSubTypes("Goblin", "Warlock"),
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
// TODO: implement
	Register("Expressive Firedancer", func() Card {
		return NewCreature("Expressive Firedancer", "{1}{R}", 2, 2,
			WithSubTypes("Human", "Sorcerer"),
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
// TODO: implement
	Register("Molten-Core Maestro", func() Card {
		return NewCreature("Molten-Core Maestro", "{1}{R}", 2, 2,
			WithSubTypes("Goblin", "Bard"),
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
// TODO: implement
	Register("Rearing Embermare", func() Card {
		return NewCreature("Rearing Embermare", "{4}{R}", 4, 5,
			WithSubTypes("Horse", "Beast"),
		)
	})

// Rubble Rouser {2}{R}
// Creature — Dwarf Sorcerer
// 1/4
// When this creature enters, you may discard a card. If you do, draw a card.
// {T}, Exile a card from your graveyard: Add {R}. When you do, this creature deals 1 damage to each opponent.
// TODO: implement
	Register("Rubble Rouser", func() Card {
		return NewCreature("Rubble Rouser", "{2}{R}", 1, 4,
			WithSubTypes("Dwarf", "Sorcerer"),
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
// TODO: implement
	Register("Zealous Lorecaster", func() Card {
		return NewCreature("Zealous Lorecaster", "{5}{R}", 4, 4,
			WithSubTypes("Giant", "Sorcerer"),
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
// TODO: implement
	Register("Ambitious Augmenter", func() Card {
		return NewCreature("Ambitious Augmenter", "{G}", 1, 1,
			WithSubTypes("Turtle", "Wizard"),
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
// TODO: implement
	Register("Hungry Graffalon", func() Card {
		return NewCreature("Hungry Graffalon", "{3}{G}", 3, 4,
			WithSubTypes("Giraffe"),
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
		pestTokenWithAttack := TokenWithAbilities(
			CreateColoredToken("Pest Token", 1, 1,
				[]Color{Black, Green},
				[]CardType{TypeCreature},
				[]string{"Pest"},
			),
			AttacksTrigger(GainLife(1), false),
		)
		return NewCreature("Pestbrood Sloth", "{3}{G}", 4, 4,
			WithSubTypes("Plant", "Sloth"),
			WithKeyword(Reach),
			// When this creature dies, create two 1/1 black and green Pest creature tokens
			// with "Whenever this token attacks, you gain 1 life."
			WithAbility(PutIntoGraveyardFromBattlefieldTrigger(
				CompositeEffects("create two Pest tokens",
					pestTokenWithAttack,
					pestTokenWithAttack,
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
// TODO: implement
	Register("Shopkeeper's Bane", func() Card {
		return NewCreature("Shopkeeper's Bane", "{2}{G}", 4, 2,
			WithSubTypes("Badger", "Pest"),
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
// TODO: implement
	Register("Wildgrowth Archaic", func() Card {
		return NewCreature("Wildgrowth Archaic", "{2/G}{2/G}", 0, 0,
			WithSubTypes("Avatar"),
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
// TODO: implement
	Register("Berta, Wise Extrapolator", func() Card {
		return NewCreature("Berta, Wise Extrapolator", "{2}{G}{U}", 1, 4,
			WithSubTypes("Frog", "Druid"),
			WithSuperTypes(SuperLegendary),
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
// TODO: implement
	Register("Colorstorm Stallion", func() Card {
		return NewCreature("Colorstorm Stallion", "{1}{U}{R}", 3, 3,
			WithSubTypes("Elemental", "Horse"),
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
// TODO: implement
	Register("Conciliator's Duelist", func() Card {
		return NewCreature("Conciliator's Duelist", "{W}{W}{B}{B}", 4, 3,
			WithSubTypes("Kor", "Warlock"),
		)
	})

// Cuboid Colony {G}{U}
// Creature — Insect
// 1/1
// Flash
// Flying, trample
// Increment (Whenever you cast a spell, if the amount of mana you spent is greater than this creature's power or toughness, put a +1/+1 counter on this creature.)
// TODO: implement
	Register("Cuboid Colony", func() Card {
		return NewCreature("Cuboid Colony", "{G}{U}", 1, 1,
			WithSubTypes("Insect"),
		)
	})

// Elemental Mascot {1}{U}{R}
// Creature — Elemental Bird
// 1/4
// Flying, vigilance
// Opus — Whenever you cast an instant or sorcery spell, this creature gets +1/+0 until end of turn. If five or more mana was spent to cast that spell, exile the top card of your library. You may play that card until the end of your next turn.
// TODO: implement
	Register("Elemental Mascot", func() Card {
		return NewCreature("Elemental Mascot", "{1}{U}{R}", 1, 4,
			WithSubTypes("Elemental", "Bird"),
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
// TODO: implement
	Register("Fractal Mascot", func() Card {
		return NewCreature("Fractal Mascot", "{4}{G}{U}", 6, 6,
			WithSubTypes("Fractal", "Elk"),
		)
	})

// Fractal Tender {3}{G}{U}
// Creature — Elf Wizard
// 3/3
// Ward {2}
// Increment (Whenever you cast a spell, if the amount of mana you spent is greater than this creature's power or toughness, put a +1/+1 counter on this creature.)
// At the beginning of each end step, if you put a counter on this creature this turn, create a 0/0 green and blue Fractal creature token and put three +1/+1 counters on it.
// TODO: implement
	Register("Fractal Tender", func() Card {
		return NewCreature("Fractal Tender", "{3}{G}{U}", 3, 3,
			WithSubTypes("Elf", "Wizard"),
		)
	})

// Geometer's Arthropod {G}{U}
// Creature — Fractal Crab
// 1/4
// Whenever you cast a spell with {X} in its mana cost, look at the top X cards of your library. Put one of them into your hand and the rest on the bottom of your library in a random order.
// TODO: implement
	Register("Geometer's Arthropod", func() Card {
		return NewCreature("Geometer's Arthropod", "{G}{U}", 1, 4,
			WithSubTypes("Fractal", "Crab"),
		)
	})

// Hardened Academic {R}{W}
// Creature — Bird Cleric
// 2/1
// Flying, haste
// Discard a card: This creature gains lifelink until end of turn.
// Whenever one or more cards leave your graveyard, put a +1/+1 counter on target creature you control.
// TODO: implement
	Register("Hardened Academic", func() Card {
		return NewCreature("Hardened Academic", "{R}{W}", 2, 1,
			WithSubTypes("Bird", "Cleric"),
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
// Repartee — Whenever you cast an instant or sorcery spell that targets a creature, this creature gains flying until end of turn. Surveil 1. (Look at the top card of your library. You may put it into your graveyard.)
// TODO: implement
	Register("Inkling Mascot", func() Card {
		return NewCreature("Inkling Mascot", "{W}{B}", 2, 2,
			WithSubTypes("Inkling", "Cat"),
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
// TODO: implement
	Register("Lorehold, the Historian", func() Card {
		return NewCreature("Lorehold, the Historian", "{3}{R}{W}", 5, 5,
			WithSubTypes("Elder", "Dragon"),
			WithSuperTypes(SuperLegendary),
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
// TODO: implement
	Register("Old-Growth Educator", func() Card {
		return NewCreature("Old-Growth Educator", "{2}{B}{G}", 4, 4,
			WithSubTypes("Treefolk", "Druid"),
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
// TODO: implement
	Register("Practiced Scrollsmith", func() Card {
		return NewCreature("Practiced Scrollsmith", "{R}{R/W}{W}", 3, 2,
			WithSubTypes("Dwarf", "Cleric"),
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
// TODO: implement
	Register("Pterafractyl", func() Card {
		return NewCreature("Pterafractyl", "{X}{G}{U}", 1, 0,
			WithSubTypes("Dinosaur", "Fractal"),
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
// TODO: implement
	Register("Spirit Mascot", func() Card {
		return NewCreature("Spirit Mascot", "{R}{W}", 2, 2,
			WithSubTypes("Spirit", "Ox"),
		)
	})

// Stadium Tidalmage {2}{U}{R}
// Creature — Djinn Sorcerer
// 4/4
// Whenever this creature enters or attacks, you may draw a card. If you do, discard a card.
// TODO: implement
	Register("Stadium Tidalmage", func() Card {
		return NewCreature("Stadium Tidalmage", "{2}{U}{R}", 4, 4,
			WithSubTypes("Djinn", "Sorcerer"),
		)
	})

// Startled Relic Sloth {2}{R}{W}
// Creature — Sloth Beast
// 4/4
// Trample, lifelink
// At the beginning of combat on your turn, exile up to one target card from a graveyard.
// TODO: implement
	Register("Startled Relic Sloth", func() Card {
		return NewCreature("Startled Relic Sloth", "{2}{R}{W}", 4, 4,
			WithSubTypes("Sloth", "Beast"),
		)
	})

// Stirring Honormancer {2}{W}{W/B}{B}
// Creature — Rhino Bard
// 4/5
// When this creature enters, look at the top X cards of your library, where X is the number of creatures you control. Put one of those cards into your hand and the rest into your graveyard.
// TODO: implement
	Register("Stirring Honormancer", func() Card {
		return NewCreature("Stirring Honormancer", "{2}{W}{W/B}{B}", 4, 5,
			WithSubTypes("Rhino", "Bard"),
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
// TODO: implement
	Register("Teacher's Pest", func() Card {
		return NewCreature("Teacher's Pest", "{B}{G}", 1, 1,
			WithSubTypes("Skeleton", "Pest"),
		)
	})

// Witherbloom, the Balancer {6}{B}{G}
// Legendary Creature — Elder Dragon
// 5/5
// Affinity for creatures (This spell costs {1} less to cast for each creature you control.)
// Flying, deathtouch
// Instant and sorcery spells you cast have affinity for creatures.
// TODO: implement
	Register("Witherbloom, the Balancer", func() Card {
		return NewCreature("Witherbloom, the Balancer", "{6}{B}{G}", 5, 5,
			WithSubTypes("Elder", "Dragon"),
			WithSuperTypes(SuperLegendary),
		)
	})

// Zaffai and the Tempests {5}{U}{R}
// Legendary Creature — Human Bard Sorcerer
// 5/7
// Once during each of your turns, you may cast an instant or sorcery spell from your hand without paying its mana cost.
// TODO: implement
	Register("Zaffai and the Tempests", func() Card {
		return NewCreature("Zaffai and the Tempests", "{5}{U}{R}", 5, 7,
			WithSubTypes("Human", "Bard", "Sorcerer"),
			WithSuperTypes(SuperLegendary),
		)
	})


	// ===== COLORLESS CREATURES =====

// Biblioplex Tomekeeper {4}
// Artifact Creature — Construct
// 3/4
// When this creature enters, choose up to one —
// • Target creature becomes prepared. (Only creatures with prepare spells can become prepared.)
// • Target creature becomes unprepared.
// TODO: implement
	Register("Biblioplex Tomekeeper", func() Card {
		return NewCreature("Biblioplex Tomekeeper", "{4}", 3, 4,
			WithSubTypes("Construct"),
			WithCardType(TypeArtifact),
		)
	})

// Mage Tower Referee {2}
// Artifact Creature — Construct
// 2/1
// Whenever you cast a multicolored spell, put a +1/+1 counter on this creature.
// TODO: implement
	Register("Mage Tower Referee", func() Card {
		return NewCreature("Mage Tower Referee", "{2}", 2, 1,
			WithSubTypes("Construct"),
			WithCardType(TypeArtifact),
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
// TODO: implement
	Register("Rancorous Archaic", func() Card {
		return NewCreature("Rancorous Archaic", "{5}", 2, 2,
			WithSubTypes("Avatar"),
		)
	})

// Sundering Archaic {6}
// Creature — Avatar
// 3/3
// Converge — When this creature enters, exile target nonland permanent an opponent controls with mana value less than or equal to the number of colors of mana spent to cast this creature.
// {2}: Put target card from a graveyard on the bottom of its owner's library.
// TODO: implement
	Register("Sundering Archaic", func() Card {
		return NewCreature("Sundering Archaic", "{6}", 3, 3,
			WithSubTypes("Avatar"),
		)
	})

// The Dawning Archaic {10}
// Legendary Creature — Avatar
// 7/7
// This spell costs {1} less to cast for each instant and sorcery card in your graveyard.
// Reach
// Whenever The Dawning Archaic attacks, you may cast target instant or sorcery card from your graveyard without paying its mana cost. If that spell would be put into your graveyard, exile it instead.
// TODO: implement
	Register("The Dawning Archaic", func() Card {
		return NewCreature("The Dawning Archaic", "{10}", 7, 7,
			WithSubTypes("Avatar"),
			WithSuperTypes(SuperLegendary),
		)
	})

// Transcendent Archaic {7}
// Creature — Avatar
// 6/6
// Vigilance
// Converge — When this creature enters, you may draw X cards, where X is the number of colors of mana spent to cast this spell. If you draw one or more cards this way, discard two cards.
// TODO: implement
	Register("Transcendent Archaic", func() Card {
		return NewCreature("Transcendent Archaic", "{7}", 6, 6,
			WithSubTypes("Avatar"),
		)
	})

}
