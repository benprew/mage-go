package jumpstart

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage"
	. "github.com/benprew/mage-go/pkg/mage/core"
)

func init() {
	registerArtifacts()
}

func registerArtifacts() {

	// Aether Spellbomb {1}
	// Artifact
	// {U}, Sacrifice this artifact: Return target creature to its owner's hand.
	// {1}, Sacrifice this artifact: Draw a card.
	Register("Aether Spellbomb", func() Card {
		return NewArtifact("Aether Spellbomb", "{1}",
			WithActivatedAbility(
				ReturnToHandTarget(),
				ManaCostOf("{U}"),
				WithCost(SacrificeSourceCost()),
				WithTarget(TargetCreature()),
			),
			WithActivatedAbility(
				DrawCards(Fixed(1)),
				GenericCost(1),
				WithCost(SacrificeSourceCost()),
			),
		)
	})

	// Arcane Encyclopedia {3}
	// Artifact — Book
	// {3}, {T}: Draw a card.
	Register("Arcane Encyclopedia", func() Card {
		return NewArtifact("Arcane Encyclopedia", "{3}",
			WithSubTypes("Book"),
			WithActivatedAbility(
				DrawCards(Fixed(1)),
				GenericCost(3),
				WithCost(Tap()),
			),
		)
	})

	// Bubbling Cauldron {2}
	// Artifact
	// {1}, {T}, Sacrifice a creature: You gain 4 life.
	// {1}, {T}, Sacrifice a creature named Festering Newt: Each opponent loses 4 life. You gain life equal to the life lost this way.
	Register("Bubbling Cauldron", func() Card {
		return NewArtifact("Bubbling Cauldron", "{2}",
			WithActivatedAbility(
				GainLife(4),
				GenericCost(1),
				WithCost(Tap()),
				WithCost(SacrificeCreatureCost()),
			),
			WithActivatedAbility(
				FuncEffect("each opponent loses 4 life; you gain that much life",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						total := 0
						for _, p := range g.AllPlayers() {
							if p.PlayerID() == controller {
								continue
							}
							before := p.Life()
							p.LoseLife(4)
							lost := before - p.Life()
							if lost > 0 {
								total += lost
							}
						}
						if total > 0 {
							if you := g.GetPlayer(controller); you != nil {
								g.PlayerGainLife(you, total)
							}
						}
						return nil
					}),
				GenericCost(1),
				WithCost(Tap()),
				WithCost(SacrificeMatchingCost(And(IsCreature, Named("Festering Newt")), "Sacrifice a creature named Festering Newt")),
			),
		)
	})

	// Chromatic Sphere {1}
	// Artifact
	// {1}, {T}, Sacrifice this artifact: Add one mana of any color. Draw a card.
	Register("Chromatic Sphere", func() Card {
		return NewArtifact("Chromatic Sphere", "{1}",
			WithActivatedAbility(
				CompositeEffects("add one mana of any color, then draw a card",
					AddAnyMana(1, Colorless),
					DrawCards(Fixed(1)),
				),
				GenericCost(1),
				WithCost(Tap()),
				WithCost(SacrificeSourceCost()),
			),
		)
	})

	// Dreamstone Hedron {6}
	// Artifact
	// {T}: Add {C}{C}{C}.
	// {3}, {T}, Sacrifice this artifact: Draw three cards.
	Register("Dreamstone Hedron", func() Card {
		return NewArtifact("Dreamstone Hedron", "{6}",
			WithMultiManaAbility(ManaProduction{Color: Colorless, Amount: 3}),
			WithActivatedAbility(
				DrawCards(Fixed(3)),
				GenericCost(3),
				WithCost(Tap()),
				WithCost(SacrificeSourceCost()),
			),
		)
	})

	// Guardian Idol {2}
	// Artifact
	// This artifact enters tapped.
	// {T}: Add {C}.
	// {2}: This artifact becomes a 2/2 Golem artifact creature until end of turn.
	Register("Guardian Idol", func() Card {
		return NewArtifact("Guardian Idol", "{2}",
			WithKeyword(EntersTapped),
			WithMultiManaAbility(ManaProduction{Color: Colorless, Amount: 1}),
			WithActivatedAbility(
				FuncEffect("becomes a 2/2 Golem artifact creature until end of turn",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						eff := TemporaryAnimate(sourceID, 2, 2)
						eff.SetSourceID(sourceID)
						g.AddContinuousEffect(eff)
						g.ApplyContinuousEffects()
						return nil
					}),
				GenericCost(2),
			),
		)
	})

	// Hedron Archive {4}
	// Artifact
	// {T}: Add {C}{C}.
	// {2}, {T}, Sacrifice this artifact: Draw two cards.
	Register("Hedron Archive", func() Card {
		return NewArtifact("Hedron Archive", "{4}",
			WithMultiManaAbility(ManaProduction{Color: Colorless, Amount: 2}),
			WithActivatedAbility(
				DrawCards(Fixed(2)),
				GenericCost(2),
				WithCost(Tap()),
				WithCost(SacrificeSourceCost()),
			),
		)
	})

	// Herald's Horn {3}
	// Artifact
	// As this artifact enters, choose a creature type.
	// Creature spells you cast of the chosen type cost {1} less to cast.
	// At the beginning of your upkeep, look at the top card of your library. If it's a creature card of the chosen type, you may reveal it and put it into your hand.
	Register("Herald's Horn", func() Card {
		return NewArtifact("Herald's Horn", "{3}",
			WithAbility(ETBChooseCreatureType(heraldsHornCreatureTypes)),
			WithStaticAbility(ReduceSpellCostStatic(
				SpellsAnd(SpellHasType(TypeCreature), SpellSubTypeMatchesChosen()),
				FixedAmount(1), nil,
			)),
			WithAbility(BeginningOfUpkeepTrigger(FuncEffect(
				"look at the top card of your library; if it is a creature card of the chosen type, you may reveal it and put it into your hand",
				EffectProperties{Outcome: OutcomeBenefit},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					src := g.FindPermanent(sourceID)
					if src == nil || src.ChosenSubtype == "" {
						return nil
					}
					you := g.GetPlayer(controller)
					if you == nil {
						return nil
					}
					top := g.RevealTopN(you, 1)
					if len(top) == 0 {
						return nil
					}
					card := top[0]
					if !card.HasType(TypeCreature) || !card.HasSubType(src.ChosenSubtype) {
						return nil
					}
					if !you.ChooseMayAbility("reveal top card and put it into your hand?") {
						return nil
					}
					taken := g.RemoveTopN(you, 1)
					if len(taken) == 0 {
						return nil
					}
					you.AddToHand(taken[0])
					return nil
				},
			), false)),
		)
	})

	// Mana Geode {3}
	// Artifact
	// When this artifact enters, scry 1.
	// {T}: Add one mana of any color.
	Register("Mana Geode", func() Card {
		return NewArtifact("Mana Geode", "{3}",
			WithAnyColorMana(),
			WithAbility(EntersBattlefieldTrigger(Scry(Fixed(1)), false)),
		)
	})

	// Marauder's Axe {2}
	// Artifact — Equipment
	// Equipped creature gets +2/+0.
	// Equip {2} ({2}: Attach to target creature you control. Equip only as a sorcery.)
	Register("Marauder's Axe", func() Card {
		return NewEquipment("Marauder's Axe", "{2}",
			WithStaticAbility(BoostAttached(2, 0, AttachEquipment)),
			WithAbility(NewEquipAbility(ManaCostOf("{2}"))),
		)
	})

	// Pirate's Cutlass {3}
	// Artifact — Equipment
	// When this Equipment enters, attach it to target Pirate you control.
	// Equipped creature gets +2/+1.
	// Equip {2} ({2}: Attach to target creature you control. Equip only as a sorcery.)
	Register("Pirate's Cutlass", func() Card {
		return NewEquipment("Pirate's Cutlass", "{3}",
			WithStaticAbility(BoostAttached(2, 1, AttachEquipment)),
			WithAbility(EntersBattlefieldTrigger(
				FuncEffect("attach to target Pirate you control",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						pirates := g.FilterBattlefield(And(IsCreature, HasSubType("Pirate"), ControlledBy(controller)))
						if len(pirates) == 0 {
							return nil
						}
						p := g.GetPlayer(controller)
						var chosen *Permanent
						if p != nil {
							chosen = p.ChoosePermanent(pirates, "attach Pirate's Cutlass to a Pirate", g)
						}
						if chosen == nil {
							chosen = pirates[0]
						}
						g.Attach(sourceID, chosen.ID())
						return nil
					}),
				false,
			)),
			WithAbility(NewEquipAbility(ManaCostOf("{2}"))),
		)
	})

	// Prophetic Prism {2}
	// Artifact
	// When this artifact enters, draw a card.
	// {1}, {T}: Add one mana of any color.
	Register("Prophetic Prism", func() Card {
		return NewArtifact("Prophetic Prism", "{2}",
			WithAbility(EntersBattlefieldTrigger(DrawCards(Fixed(1)), false)),
			WithActivatedAbility(
				AddAnyMana(1, Colorless),
				GenericCost(1),
				WithCost(Tap()),
			),
		)
	})

	// Rogue's Gloves {2}
	// Artifact — Equipment
	// Whenever equipped creature deals combat damage to a player, you may draw a card.
	// Equip {2} ({2}: Attach to target creature you control. Equip only as a sorcery.)
	Register("Rogue's Gloves", func() Card {
		return NewEquipment("Rogue's Gloves", "{2}",
			WithAbility(NewTriggered(EvtDamageDealt, true,
				FuncEffect("you may draw a card",
					EffectProperties{Outcome: OutcomeBenefit, DrawCount: 1},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						g.PlayerDrawCard(p)
						return nil
					}),
			).SetCondition(func(evt *GameEvent, g GameReader, sourceID, controllerID uuid.UUID) bool {
				if !evt.Flag {
					return false
				}
				if g.GetPlayer(evt.TargetID) == nil {
					return false
				}
				src := g.FindPermanent(sourceID)
				if src == nil {
					return false
				}
				return src.IsAttached() && src.AttachedTo == evt.SourceID
			})),
			WithAbility(NewEquipAbility(ManaCostOf("{2}"))),
		)
	})

	// Scroll of Avacyn {1}
	// Artifact
	// {1}, Sacrifice this artifact: Draw a card. If you control an Angel, you gain 5 life.
	Register("Scroll of Avacyn", func() Card {
		return NewArtifact("Scroll of Avacyn", "{1}",
			WithActivatedAbility(
				CompositeEffects("draw a card; if you control an Angel, you gain 5 life",
					DrawCards(Fixed(1)),
					FuncEffect("if you control an Angel, you gain 5 life",
						EffectProperties{Outcome: OutcomeBenefit},
						func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							p := g.GetPlayer(controller)
							if p == nil {
								return nil
							}
							for range g.FilterBattlefield(And(IsCreature, HasSubType("Angel"), ControlledBy(controller))) {
								g.PlayerGainLife(p, 5)
								return nil
							}
							return nil
						}),
				),
				GenericCost(1),
				WithCost(SacrificeSourceCost()),
			),
		)
	})

	// Terrarion {1}
	// Artifact
	// This artifact enters tapped.
	// {2}, {T}, Sacrifice this artifact: Add two mana in any combination of colors.
	// When this artifact is put into a graveyard from the battlefield, draw a card.
	Register("Terrarion", func() Card {
		return NewArtifact("Terrarion", "{1}",
			WithKeyword(EntersTapped),
			WithActivatedAbility(
				AddAnyMana(2, Colorless),
				GenericCost(2),
				WithCost(Tap()),
				WithCost(SacrificeSourceCost()),
			),
			WithAbility(DiesTrigger(
				DrawCards(Fixed(1)), false,
			)),
		)
	})

	// Unstable Obelisk {3}
	// Artifact
	// {T}: Add {C}.
	// {7}, {T}, Sacrifice this artifact: Destroy target permanent.
	Register("Unstable Obelisk", func() Card {
		return NewArtifact("Unstable Obelisk", "{3}",
			WithMultiManaAbility(ManaProduction{Color: Colorless, Amount: 1}),
			WithActivatedAbility(
				DestroyTargetPermanent(),
				GenericCost(7),
				WithCost(Tap()),
				WithCost(SacrificeSourceCost()),
				WithTarget(TargetPermanent()),
			),
		)
	})

	// Warmonger's Chariot {2}
	// Artifact — Equipment
	// Equipped creature gets +2/+2.
	// As long as equipped creature has defender, it can attack as though it didn't have defender.
	// Equip {3} ({3}: Attach to target creature you control. Equip only as a sorcery.)
	Register("Warmonger's Chariot", func() Card {
		return NewEquipment("Warmonger's Chariot", "{2}",
			WithStaticAbility(
				BoostAttached(2, 2, AttachEquipment),
				AttachedEffect(LayerAbility, func(g *Game, source, target *Permanent) error {
					if target.HasKeyword(Defender) {
						g.GrantAttr(target.ID(), AttrCanAttack)
					}
					return nil
				}),
			),
			WithAbility(NewEquipAbility(ManaCostOf("{3}"))),
		)
	})

}

// heraldsHornCreatureTypes is the menu of creature types offered to the
// controller of Herald's Horn for its "as it enters, choose a creature type"
// replacement. Magic does not actually constrain this choice — any creature
// type is legal — but ETBChooseCreatureType requires a finite option list.
// This list covers the common tribal types found across MTG; cards in the
// Jumpstart pool whose tribes appear here exercise the cost reducer and the
// upkeep reveal naturally.
var heraldsHornCreatureTypes = []string{
	"Advisor", "Aetherborn", "Ally", "Angel", "Antelope", "Ape", "Archer", "Archon",
	"Artificer", "Assassin", "Assembly-Worker", "Atog", "Aurochs", "Avatar", "Azra",
	"Badger", "Barbarian", "Basilisk", "Bat", "Bear", "Beast", "Beeble", "Berserker",
	"Bird", "Blinkmoth", "Boar", "Bringer", "Brushwagg", "Camarid", "Camel", "Caribou",
	"Carrier", "Cat", "Centaur", "Cephalid", "Chimera", "Citizen", "Cleric", "Cockatrice",
	"Construct", "Coward", "Crab", "Crocodile", "Cyclops", "Dauthi", "Demon", "Deserter",
	"Devil", "Dinosaur", "Djinn", "Dragon", "Drake", "Dreadnought", "Drone", "Druid",
	"Dryad", "Dwarf", "Efreet", "Egg", "Elder", "Eldrazi", "Elemental", "Elephant",
	"Elf", "Elk", "Eye", "Faerie", "Ferret", "Fish", "Flagbearer", "Fox", "Frog",
	"Fungus", "Gargoyle", "Germ", "Giant", "Gnome", "Goat", "Goblin", "God", "Golem",
	"Gorgon", "Graveborn", "Gremlin", "Griffin", "Hag", "Harpy", "Hellion", "Hippo",
	"Hippogriff", "Homarid", "Homunculus", "Horror", "Horse", "Hound", "Human", "Hydra",
	"Hyena", "Illusion", "Imp", "Incarnation", "Insect", "Jackal", "Jellyfish", "Juggernaut",
	"Kavu", "Kirin", "Kithkin", "Knight", "Kobold", "Kor", "Kraken", "Lamia", "Lammasu",
	"Leech", "Leviathan", "Lhurgoyf", "Licid", "Lizard", "Manticore", "Masticore", "Mercenary",
	"Merfolk", "Metathran", "Minion", "Minotaur", "Mole", "Monger", "Mongoose", "Monk",
	"Monkey", "Moonfolk", "Mouse", "Mutant", "Myr", "Mystic", "Naga", "Nautilus",
	"Nephilim", "Nightmare", "Nightstalker", "Ninja", "Noggle", "Nomad", "Nymph", "Octopus",
	"Ogre", "Ooze", "Orb", "Orc", "Orgg", "Ouphe", "Ox", "Oyster", "Pegasus", "Pentavite",
	"Pest", "Phelddagrif", "Phoenix", "Pilot", "Pincher", "Pirate", "Plant", "Praetor",
	"Prism", "Processor", "Rabbit", "Rat", "Rebel", "Reflection", "Rhino", "Rigger",
	"Rogue", "Sable", "Salamander", "Samurai", "Sand", "Saproling", "Satyr", "Scarecrow",
	"Scion", "Scorpion", "Scout", "Serf", "Serpent", "Servo", "Shade", "Shaman",
	"Shapeshifter", "Sheep", "Siren", "Skeleton", "Slith", "Sliver", "Slug", "Snake",
	"Soldier", "Soltari", "Spawn", "Specter", "Spellshaper", "Sphinx", "Spider", "Spike",
	"Spirit", "Splinter", "Sponge", "Squid", "Squirrel", "Starfish", "Surrakar", "Survivor",
	"Tetravite", "Thalakos", "Thopter", "Thrull", "Treefolk", "Trilobite", "Triskelavite",
	"Troll", "Turtle", "Unicorn", "Vampire", "Vedalken", "Viashino", "Volver", "Wall",
	"Warlock", "Warrior", "Weird", "Werewolf", "Whale", "Wizard", "Wolf", "Wolverine",
	"Wombat", "Worm", "Wraith", "Wurm", "Yeti", "Zombie", "Zubera",
}
