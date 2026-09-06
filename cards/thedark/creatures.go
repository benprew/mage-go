package thedark

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage"
	. "github.com/benprew/mage-go/pkg/mage/core"
)

func init() {
	registerCreatures()
}

func registerCreatures() {

	// ===== WHITE CREATURES =====

	// Exorcist {W}{W}
	// Creature — Human Cleric
	// 1/1
	// {1}{W}, {T}: Destroy target black creature.
	Register("Exorcist", func() Card {
		return NewCreature("Exorcist", "{W}{W}", 1, 1,
			WithSubTypes("Human", "Cleric"),
			WithActivatedAbility(
				DestroyTarget(),
				ManaCostOf("{1}{W}"),
				WithCost(Tap()),
				WithTarget(TargetCreature(HasColorFilter(Black))),
			),
		)
	})

	// Knights of Thorn {3}{W}
	// Creature — Human Knight
	// 2/2
	// Protection from red; banding (Any creatures with banding, and up to one without, can attack in a band. Bands are blocked as a group. If any creatures with banding you control are blocking or being blocked by a creature, you divide that creature's combat damage, not its controller, among any of the creatures it's being blocked by or is blocking.)
	Register("Knights of Thorn", func() Card {
		return NewCreature("Knights of Thorn", "{3}{W}", 2, 2,
			WithSubTypes("Human", "Knight"),
			WithKeyword(Banding),
			WithAbility(ProtectionFromColor(Red)),
		)
	})

	// Miracle Worker {W}
	// Creature — Human Cleric
	// 1/1
	// {T}: Destroy target Aura attached to a creature you control.
	Register("Miracle Worker", func() Card {
		return NewCreature("Miracle Worker", "{W}", 1, 1,
			WithSubTypes("Human", "Cleric"),
			WithActivatedAbility(
				DestroyTarget(),
				Tap(),
				WithTarget(TargetAuraAttachedToCreatureYouControl()),
			),
		)
	})

	// Preacher {1}{W}{W}
	// Creature — Human Cleric
	// 1/1
	// You may choose not to untap this creature during your untap step.
	// {T}: For as long as this creature remains tapped, gain control of target creature of an opponent's choice they control.
	Register("Preacher", func() Card {
		return NewCreature("Preacher", "{1}{W}{W}", 1, 1,
			WithSubTypes("Human", "Cleric"),
			WithActivatedAbility(
				GainControl().While(ControlSourceExists{}, ControlSourceTapped{}).TapMaintained(),
				Tap(),
				WithTarget(TargetCreatureOpponentControls()),
			),
		)
	})

	// Squire {1}{W}
	// Creature — Human Soldier
	// 1/2
	Register("Squire", func() Card {
		return NewCreature("Squire", "{1}{W}", 1, 2,
			WithSubTypes("Human", "Soldier"),
		)
	})

	// Witch Hunter {2}{W}{W}
	// Creature — Human Cleric
	// 1/1
	// {T}: This creature deals 1 damage to target player or planeswalker.
	// {1}{W}{W}, {T}: Return target creature an opponent controls to its owner's hand.
	Register("Witch Hunter", func() Card {
		return NewCreature("Witch Hunter", "{2}{W}{W}", 1, 1,
			WithSubTypes("Human", "Cleric"),
			WithActivatedAbility(
				DealDamage(Fixed(1)),
				Tap(),
				WithTarget(TargetPlayerOrPlaneswalker()),
			),
			WithActivatedAbility(
				ReturnToHandTarget(),
				ManaCostOf("{1}{W}{W}"),
				WithCost(Tap()),
				WithTarget(TargetCreatureOpponentControls()),
			),
		)
	})

	// ===== BLUE CREATURES =====

	// Drowned {1}{U}
	// Creature — Zombie
	// 1/1
	// {B}: Regenerate this creature.
	Register("Drowned", func() Card {
		return NewCreature("Drowned", "{1}{U}", 1, 1,
			WithSubTypes("Zombie"),
			WithActivatedAbility(
				RegenerateSource(),
				ManaCostOf("{B}"),
			),
		)
	})

	// Electric Eel {U}
	// Creature — Fish
	// 1/1
	// When this creature enters, it deals 1 damage to you.
	// {R}{R}: This creature gets +2/+0 until end of turn and deals 1 damage to you.
	Register("Electric Eel", func() Card {
		return NewCreature("Electric Eel", "{U}", 1, 1,
			WithSubTypes("Fish"),
			WithAbility(EntersBattlefieldTrigger(DealDamageToPlayers(Fixed(1), SelectController()), false)),
			WithActivatedAbility(
				Boost(Fixed(2), Fixed(0)).Targeting(ToSource()),
				ManaCostOf("{R}{R}"),
				WithEffect(DealDamageToPlayers(Fixed(1), SelectController())),
			),
		)
	})

	// Giant Shark {5}{U}
	// Creature — Shark
	// 4/4
	// This creature can't attack unless defending player controls an Island.
	// Whenever this creature blocks or becomes blocked by a creature that has been dealt damage this turn, this creature gets +2/+0 and gains trample until end of turn.
	// When you control no Islands, sacrifice this creature.
	Register("Giant Shark", func() Card {
		return NewCreature("Giant Shark", "{5}{U}", 4, 4,
			WithSubTypes("Shark"),
			WithStaticAbility(PreventFromAttackingIfDefendingPlayerControls(HasSubType("Island"))),
			WithAbility(SacrificeUnlessLand("Island")),
			WithAbility(BlocksOrBlockedByDamagedCreatureTrigger(
				CompositeEffects("gets +2/+0 and gains trample until end of turn",
					Boost(Fixed(2), Fixed(0)).Targeting(ToSource()).Until(EndOfTurn),
					GrantKeyword(Trample).Targeting(ToSource()),
				),
			)),
		)
	})

	// Merfolk Assassin {U}{U}
	// Creature — Merfolk Assassin
	// 1/2
	// {T}: Destroy target creature with islandwalk.
	Register("Merfolk Assassin", func() Card {
		return NewCreature("Merfolk Assassin", "{U}{U}", 1, 2,
			WithSubTypes("Merfolk", "Assassin"),
			WithActivatedAbility(
				DestroyTarget(),
				Tap(),
				WithTarget(TargetCreature(HasKeywordFilter(Islandwalk))),
			),
		)
	})

	// Water Wurm {U}
	// Creature — Wurm
	// 1/1
	// This creature gets +0/+1 as long as an opponent controls an Island.
	Register("Water Wurm", func() Card {
		return NewCreature("Water Wurm", "{U}", 1, 1,
			WithSubTypes("Wurm"),
			WithStaticAbility(BoostSelf(0, 1, WhileOpponentControls(HasSubType("Island")))),
		)
	})

	// ===== BLACK CREATURES =====

	// Banshee {2}{B}{B}
	// Creature — Spirit
	// 0/1
	// {X}, {T}: This creature deals half X damage, rounded down, to any target, and half X damage, rounded up, to you.
	Register("Banshee", func() Card {
		return NewCreature("Banshee", "{2}{B}{B}", 0, 1,
			WithSubTypes("Spirit"),
			WithActivatedAbility(
				DealDamage(HalfXRoundedDown()),
				XManaCost(),
				WithCost(Tap()),
				WithEffect(DealDamageToPlayers(HalfXRoundedUp(), SelectController())),
				WithTarget(TargetDamageAnyTarget()),
			),
		)
	})

	// Bog Rats {B}
	// Creature — Rat
	// 1/1
	// This creature can't be blocked by Walls.
	Register("Bog Rats", func() Card {
		return NewCreature("Bog Rats", "{B}", 1, 1,
			WithSubTypes("Rat"),
			WithKeyword(CantBeBlockedByWalls),
		)
	})

	// Eater of the Dead {4}{B}
	// Creature — Horror
	// 3/4
	// {0}: If this creature is tapped, exile target creature card from a graveyard and untap this creature.
	Register("Eater of the Dead", func() Card {
		return NewCreature("Eater of the Dead", "{4}{B}", 3, 4,
			WithSubTypes("Horror"),
			WithActivatedAbility(
				IfSourceTapped(CompositeEffects(
					"exile target creature card from a graveyard and untap this creature",
					ExileTargetCardFromGraveyard(),
					UntapSource(),
				)),
				ManaCostOf("{0}"),
				WithTarget(TargetCreatureCardInAnyGraveyard()),
			),
		)
	})

	// Frankenstein's Monster {X}{B}{B}
	// Creature — Zombie
	// 0/1
	// As this creature enters, exile X creature cards from your graveyard. If you can't, put this creature into its owner's graveyard instead of onto the battlefield. For each creature card exiled this way, this creature enters with a +2/+0, +1/+1, or +0/+2 counter on it.
	Register("Frankenstein's Monster", func() Card {
		return NewCreature("Frankenstein's Monster", "{X}{B}{B}", 0, 1,
			WithSubTypes("Zombie"),
			WithAbility(FrankensteinsMonsterAbility()),
		)
	})

	// Grave Robbers {1}{B}{B}
	// Creature — Human Rogue
	// 1/1
	// {B}, {T}: Exile target artifact card from a graveyard. You gain 2 life.
	Register("Grave Robbers", func() Card {
		return NewCreature("Grave Robbers", "{1}{B}{B}", 1, 1,
			WithSubTypes("Human", "Rogue"),
			WithActivatedAbility(
				ExileTargetCardFromGraveyardStep(),
				ManaCostOf("{B}"),
				WithCost(Tap()),
				WithTarget(TargetCardInAnyGraveyard(IsArtifactCard)),
				WithEffect(GainLife(2)),
			),
		)
	})

	// Nameless Race {3}{B}
	// Creature
	// */*
	// Trample
	// As this creature enters, pay any amount of life. The amount you pay can't be more than the total number of white nontoken permanents your opponents control plus the total number of white cards in their graveyards.
	// Nameless Race's power and toughness are each equal to the life paid as it entered.
	Register("Nameless Race", func() Card {
		return NewCreature("Nameless Race", "{3}{B}", 0, 0,
			WithKeyword(Trample),
			WithAbility(NamelessRaceAbility()),
			WithStaticAbility(FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
				perm := g.FindPermanent(sourceID)
				if perm != nil {
					perm.BasePTOverride = &[2]int{perm.StoredValue, perm.StoredValue}
				}
				return nil
			})),
		)
	})

	// The Fallen {1}{B}{B}{B}
	// Creature — Zombie
	// 2/3
	// At the beginning of your upkeep, this creature deals 1 damage to each opponent and planeswalker it has dealt damage to this game.
	Register("The Fallen", func() Card {
		return NewCreature("The Fallen", "{1}{B}{B}{B}", 2, 3,
			WithSubTypes("Zombie"),
			WithAbility(BeginningOfUpkeepTrigger(TheFallenUpkeepEffect(), false)),
		)
	})

	// ===== RED CREATURES =====

	// Fire Drake {1}{R}{R}
	// Creature — Drake
	// 1/2
	// Flying
	// {R}: This creature gets +1/+0 until end of turn. Activate only once each turn.
	Register("Fire Drake", func() Card {
		return NewCreature("Fire Drake", "{1}{R}{R}", 1, 2,
			WithSubTypes("Drake"),
			WithKeyword(Flying),
			WithActivatedAbility(
				Boost(Fixed(1), Fixed(0)).Targeting(ToSource()),
				ManaCostOf("{R}"),
				WithOncePerTurn(),
			),
		)
	})

	// Goblin Digging Team {R}
	// Creature — Goblin
	// 1/1
	// {T}, Sacrifice this creature: Destroy target Wall.
	Register("Goblin Digging Team", func() Card {
		return NewCreature("Goblin Digging Team", "{R}", 1, 1,
			WithSubTypes("Goblin"),
			WithActivatedAbility(
				DestroyTarget(),
				Tap(),
				WithCost(SacrificeSourceCost()),
				WithTarget(TargetCreature(HasSubType("Wall"))),
			),
		)
	})

	// Goblin Hero {2}{R}
	// Creature — Goblin
	// 2/2
	Register("Goblin Hero", func() Card {
		return NewCreature("Goblin Hero", "{2}{R}", 2, 2,
			WithSubTypes("Goblin"),
		)
	})

	// Goblin Wizard {2}{R}{R}
	// Creature — Goblin Wizard
	// 1/1
	// {T}: You may put a Goblin permanent card from your hand onto the battlefield.
	// {R}: Target Goblin gains protection from white until end of turn.
	Register("Goblin Wizard", func() Card {
		return NewCreature("Goblin Wizard", "{2}{R}{R}", 1, 1,
			WithSubTypes("Goblin", "Wizard"),
			WithActivatedAbility(
				PutFromHandOntoBattlefield(AndCardFilter(HasSubTypeCardFilter("Goblin"), IsPermanentCard)),
				Tap(),
			),
			WithActivatedAbility(
				GrantProtectionTarget(White),
				ManaCostOf("{R}"),
				WithTarget(TargetCreature(HasSubType("Goblin"))),
			),
		)
	})

	// Goblins of the Flarg {R}
	// Creature — Goblin Warrior
	// 1/1
	// Mountainwalk (This creature can't be blocked as long as defending player controls a Mountain.)
	// When you control a Dwarf, sacrifice this creature.
	Register("Goblins of the Flarg", func() Card {
		return NewCreature("Goblins of the Flarg", "{R}", 1, 1,
			WithSubTypes("Goblin", "Warrior"),
			WithKeyword(Mountainwalk),
			WithAbility(SacrificeIfControls("Dwarf")),
		)
	})

	// Orc General {2}{R}
	// Creature — Orc Warrior
	// 2/2
	// {T}, Sacrifice another Orc or Goblin: Other Orc creatures get +1/+1 until end of turn.
	Register("Orc General", func() Card {
		return NewCreature("Orc General", "{2}{R}", 2, 2,
			WithSubTypes("Orc", "Warrior"),
			WithActivatedAbility(
				Boost(Fixed(1), Fixed(1)).Targeting(ToOtherMatching(HasSubType("Orc"))).Until(EndOfTurn),
				Tap(),
				WithCost(SacrificeMatchingCost(Or(HasSubType("Orc"), HasSubType("Goblin")), "Sacrifice another Orc or Goblin")),
			),
		)
	})

	// ===== GREEN CREATURES =====

	// Elves of Deep Shadow {G}
	// Creature — Elf Druid
	// 1/1
	// {T}: Add {B}. This creature deals 1 damage to you.
	Register("Elves of Deep Shadow", func() Card {
		return NewCreature("Elves of Deep Shadow", "{G}", 1, 1,
			WithSubTypes("Elf", "Druid"),
			WithManaAbility(Black, DealDamageToPlayers(Fixed(1), SelectController())),
		)
	})

	// Lurker {2}{G}
	// Creature — Beast
	// 2/3
	// This creature can't be the target of spells unless it attacked or blocked this turn.
	Register("Lurker", func() Card {
		return NewCreature("Lurker", "{2}{G}", 2, 3,
			WithSubTypes("Beast"),
			WithKeyword(AttrCantBeTargetedBySpellsUnlessAttackedOrBlocked),
		)
	})

	// Niall Silvain {G}{G}{G}
	// Creature — Ouphe
	// 2/2
	// {G}{G}{G}{G}, {T}: Regenerate target creature.
	Register("Niall Silvain", func() Card {
		return NewCreature("Niall Silvain", "{G}{G}{G}", 2, 2,
			WithSubTypes("Ouphe"),
			WithActivatedAbility(
				RegenerateTarget(),
				ManaCostOf("{G}{G}{G}{G}"),
				WithCost(Tap()),
				WithTarget(TargetCreature()),
			),
		)
	})

	// People of the Woods {G}{G}
	// Creature — Human
	// 1/*
	// People of the Woods's toughness is equal to the number of Forests you control.
	Register("People of the Woods", func() Card {
		return NewCreature("People of the Woods", "{G}{G}", 1, 0,
			WithSubTypes("Human"),
			WithStaticAbility(ToughnessEqualsControlledCount(And(IsLand, HasSubType("Forest")))),
		)
	})

	// Savaen Elves {G}
	// Creature — Elf
	// 1/1
	// {G}{G}, {T}: Destroy target Aura attached to a land.
	Register("Savaen Elves", func() Card {
		return NewCreature("Savaen Elves", "{G}", 1, 1,
			WithSubTypes("Elf"),
			WithActivatedAbility(
				DestroyTarget(),
				ManaCostOf("{G}{G}"),
				WithCost(Tap()),
				WithTarget(TargetPermanent(IsAuraOnLand)),
			),
		)
	})

	// Scarwood Bandits {2}{G}{G}
	// Creature — Human Rogue
	// 2/2
	// Forestwalk (This creature can't be blocked as long as defending player controls a Forest.)
	// {2}{G}, {T}: Unless an opponent pays {2}, gain control of target artifact for as long as this creature remains on the battlefield.
	Register("Scarwood Bandits", func() Card {
		return NewCreature("Scarwood Bandits", "{2}{G}{G}", 2, 2,
			WithSubTypes("Human", "Rogue"),
			WithKeyword(Forestwalk),
			WithActivatedAbility(
				UnlessTargetPays(
					SelectOpponent(),
					ManaCostOf("{2}"),
					"Pay {2} to prevent Scarwood Bandits from gaining control of this artifact?",
					GainControl().While(ControlSourceExists{}),
				),
				ManaCostOf("{2}{G}"),
				WithCost(Tap()),
				WithTarget(TargetArtifact()),
			),
		)
	})

	// Scarwood Hag {1}{G}
	// Creature — Hag
	// 1/1
	// {G}{G}{G}{G}, {T}: Target creature gains forestwalk until end of turn. (It can't be blocked as long as defending player controls a Forest.)
	// {T}: Target creature loses forestwalk until end of turn.
	Register("Scarwood Hag", func() Card {
		return NewCreature("Scarwood Hag", "{1}{G}", 1, 1,
			WithSubTypes("Hag"),
			WithActivatedAbility(
				GrantKeyword(Forestwalk),
				ManaCostOf("{G}{G}{G}{G}"),
				WithCost(Tap()),
				WithTarget(TargetCreature()),
			),
			WithActivatedAbility(
				RevokeKeyword(Forestwalk),
				Tap(),
				WithTarget(TargetCreature()),
			),
		)
	})

	// Scavenger Folk {G}
	// Creature — Human
	// 1/1
	// {G}, {T}, Sacrifice this creature: Destroy target artifact.
	Register("Scavenger Folk", func() Card {
		return NewCreature("Scavenger Folk", "{G}", 1, 1,
			WithSubTypes("Human"),
			WithActivatedAbility(
				DestroyTarget(),
				ManaCostOf("{G}"),
				WithCost(Tap()),
				WithCost(SacrificeSourceCost()),
				WithTarget(TargetArtifact()),
			),
		)
	})

	// Spitting Slug {1}{G}{G}
	// Creature — Slug
	// 2/4
	// Whenever this creature blocks or becomes blocked, you may pay {1}{G}. If you do, this creature gains first strike until end of turn. Otherwise, each creature blocking or blocked by this creature gains first strike until end of turn.
	Register("Spitting Slug", func() Card {
		return NewCreature("Spitting Slug", "{1}{G}{G}", 2, 4,
			WithSubTypes("Slug"),
			WithAbility(BlocksOrBecomesBlockedTrigger(
				IfPlayerPays(
					SelectController(),
					ManaCostOf("{1}{G}"),
					"Pay {1}{G} for first strike?",
					GrantKeyword(FirstStrike).Targeting(ToSource()).Until(EndOfTurn),
					GrantKeyword(FirstStrike).Targeting(ToBlockingOrBlockedBySource()).Until(EndOfTurn),
				),
			)),
		)
	})

	// Tracker {2}{G}
	// Creature — Human
	// 2/2
	// {G}{G}, {T}: This creature deals damage equal to its power to target creature. That creature deals damage equal to its power to this creature.
	Register("Tracker", func() Card {
		return NewCreature("Tracker", "{2}{G}", 2, 2,
			WithSubTypes("Human"),
			WithActivatedAbility(
				FightTarget(),
				ManaCostOf("{G}{G}"),
				WithCost(Tap()),
				WithTarget(TargetCreature()),
			),
		)
	})

	// Whippoorwill {G}
	// Creature — Bird
	// 1/1
	// {G}{G}, {T}: Target creature can't be regenerated this turn. Damage that would be dealt to that creature this turn can't be prevented or dealt instead to another permanent or player. When the creature dies this turn, exile the creature.
	Register("Whippoorwill", func() Card {
		return NewCreature("Whippoorwill", "{G}", 1, 1,
			WithSubTypes("Bird"),
			WithActivatedAbility(
				CompositeEffects(
					"Target creature can't be regenerated this turn. Damage that would be dealt to that creature this turn can't be prevented or dealt instead to another permanent or player. When the creature dies this turn, exile the creature.",
					GrantKeyword(CantRegenerate).Targeting(ToTarget()).Until(EndOfTurn),
					GrantKeyword(AttrDamageCantBePreventedOrRedirected).Targeting(ToTarget()).Until(EndOfTurn),
					OnTargetDiesThisTurn(ExileEventSourceEffect()),
				),
				ManaCostOf("{G}{G}"),
				WithCost(Tap()),
				WithTarget(TargetCreature()),
			),
		)
	})

	// Wormwood Treefolk {3}{G}{G}
	// Creature — Treefolk
	// 4/4
	// {G}{G}: This creature gains forestwalk until end of turn and deals 2 damage to you. (It can't be blocked as long as defending player controls a Forest.)
	// {B}{B}: This creature gains swampwalk until end of turn and deals 2 damage to you. (It can't be blocked as long as defending player controls a Swamp.)
	Register("Wormwood Treefolk", func() Card {
		return NewCreature("Wormwood Treefolk", "{3}{G}{G}", 4, 4,
			WithSubTypes("Treefolk"),
			WithActivatedAbility(
				GrantKeyword(Forestwalk).Targeting(ToSource()),
				ManaCostOf("{G}{G}"),
				WithEffect(DealDamageToPlayers(Fixed(2), SelectController())),
			),
			WithActivatedAbility(
				GrantKeyword(Swampwalk).Targeting(ToSource()),
				ManaCostOf("{B}{B}"),
				WithEffect(DealDamageToPlayers(Fixed(2), SelectController())),
			),
		)
	})

	// ===== MULTICOLOR CREATURES =====

	// Marsh Goblins {B}{R}
	// Creature — Goblin
	// 1/1
	// Swampwalk (This creature can't be blocked as long as defending player controls a Swamp.)
	Register("Marsh Goblins", func() Card {
		return NewCreature("Marsh Goblins", "{B}{R}", 1, 1,
			WithSubTypes("Goblin"),
			WithKeyword(Swampwalk),
		)
	})

	// Scarwood Goblins {R}{G}
	// Creature — Goblin
	// 2/2
	Register("Scarwood Goblins", func() Card {
		return NewCreature("Scarwood Goblins", "{R}{G}", 2, 2,
			WithSubTypes("Goblin"),
		)
	})

	// ===== COLORLESS CREATURES =====

	// Coal Golem {5}
	// Artifact Creature — Golem
	// 3/3
	// {3}, Sacrifice this creature: Add {R}{R}{R}.
	Register("Coal Golem", func() Card {
		return NewCreature("Coal Golem", "{5}", 3, 3,
			WithSubTypes("Golem"),
			WithCardType(TypeArtifact),
			WithActivatedAbility(
				AddMana(Red, 3),
				GenericCost(3),
				WithCost(SacrificeSourceCost()),
			),
		)
	})

	// Necropolis {5}
	// Artifact Creature — Wall
	// 0/1
	// Defender (This creature can't attack.)
	// Exile a creature card from your graveyard: Put X +0/+1 counters on this creature, where X is the exiled card's mana value.
	Register("Necropolis", func() Card {
		return NewCreature("Necropolis", "{5}", 0, 1,
			WithSubTypes("Wall"),
			WithCardType(TypeArtifact),
			WithKeyword(Defender),
			WithActivatedAbility(
				AddCounters(P0P1, ExiledCardManaValue()).Targeting(ToSource()),
				ExileMatchingCardFromGraveyardCost(IsCreatureCard, "Exile a creature card from your graveyard"),
			),
		)
	})

	// Scarecrow {5}
	// Artifact Creature — Scarecrow
	// 2/2
	// {6}, {T}: Prevent all damage that would be dealt to you this turn by creatures with flying.
	Register("Scarecrow", func() Card {
		return NewCreature("Scarecrow", "{5}", 2, 2,
			WithSubTypes("Scarecrow"),
			WithCardType(TypeArtifact),
			WithActivatedAbility(
				PreventDamageToYouByCreaturesWithFlying(),
				ManaCostOf("{6}"),
				WithCost(Tap()),
			),
		)
	})

}
