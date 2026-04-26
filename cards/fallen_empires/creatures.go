package fallen_empires

import (
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage"
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"github.com/google/uuid"
)

func init() {
	registerCreatures()
}

// moneychangerFactory creates Icatian Moneychanger.
// Sacrifice ability reads counters before sacrificing, so it's handled in FuncEffect
// rather than using SacrificeSourceCost (which would destroy the permanent before
// the effect could read its counter count).
func moneychangerFactory() Card {
	return NewCreature("Icatian Moneychanger", "{W}", 0, 2,
		WithSubTypes("Human"),
		// Enters with three credit counters
		WithAbility(ETBEffect(AddCounters(Credit, Fixed(3)).Targeting(ToSource()))),
		// When this creature enters, it deals 3 damage to you
		WithAbility(EntersBattlefieldTrigger(
			DealDamageToPlayers(Fixed(3), SelectController()), false,
		)),
		// At the beginning of your upkeep, put a credit counter on it
		WithAbility(BeginningOfUpkeepTrigger(
			AddCounters(Credit, Fixed(1)).Targeting(ToSource()), false,
		)),
		// Sacrifice: gain 1 life per credit counter (only during upkeep)
		WithActivatedAbility(
			Pipeline("sacrifice, gain 1 life for each credit counter",
				EffectProperties{Outcome: OutcomeBenefit},
				SnapshotPermanent(SelectSource, "src"),
				SnapshotSourceCounter(Credit, "credits"),
				SacrificeSourceStep(),
				IfElse("gain life if credits > 0",
					&VarGTCond{Name: "credits", Value: 0},
					GainLifeFromVar("src.controller", "credits"),
					nil,
				),
			),
			GenericCost(0),
			WithUpkeepOnly(),
		),
	)
}

func registerCreatures() {

	// ===== WHITE CREATURES =====

	// Combat Medic {2}{W}
	// Creature — Human Cleric Soldier
	// 0/2
	// {1}{W}: Prevent the next 1 damage that would be dealt to any target this turn.
	Register("Combat Medic", withExpansion(func() Card {
		return NewCreature("Combat Medic", "{2}{W}", 0, 2,
			WithSubTypes("Human", "Cleric", "Soldier"),
			WithActivatedAbility(
				PreventDamageToTarget(Fixed(1)),
				ManaCostOf("{1}{W}"),
				WithTarget(TargetAnyTarget()),
			),
		)
	}))

	// Farrel's Zealot {1}{W}{W}
	// Creature — Human
	// 2/2
	// Whenever this creature attacks and isn't blocked, you may have it deal 3 damage to target creature. If you do, this creature assigns no combat damage this turn.
	Register("Farrel's Zealot", withExpansion(func() Card {
		return NewCreature("Farrel's Zealot", "{1}{W}{W}", 2, 2,
			WithSubTypes("Human"),
			// TODO: convert to pipeline — needs ChoosePermanentStep + DealDamageToPermanent + RemoveFromCombat steps
			WithAbility(NewTriggered(EvtBlockersDecl, true,
				FuncEffect("deal 3 damage to target creature, assign no combat damage",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						src := g.FindPermanent(sourceID)
						if src == nil {
							return nil
						}
						// Choose target creature
						candidates := g.FilterBattlefield(IsCreature)
						if len(candidates) == 0 {
							return nil
						}
						player := g.GetPlayer(controller)
						chosen := player.ChoosePermanent(candidates, "deal 3 damage to target creature", g)
						if chosen == nil {
							return nil
						}
						g.DealDamageToPermanent(chosen, 3, sourceID)
						// Remove from combat so it assigns no combat damage
						g.RemoveFromCombat(sourceID)
						return nil
					}),
			).
				SetConditionData(SourceIsUnblockedAttacker{})),
		)
	}))

	// Farrelite Priest {1}{W}{W}
	// Creature — Human Cleric
	// 1/3
	// {1}: Add {W}. If this ability has been activated four or more times this turn, sacrifice this creature at the beginning of the next end step.
	// TODO: implement
	Register("Farrelite Priest", withExpansion(func() Card {
		return NewCreature("Farrelite Priest", "{1}{W}{W}", 1, 3,
			WithSubTypes("Human", "Cleric"),
		)
	}))

	// Hand of Justice {5}{W}
	// Creature — Avatar
	// 2/6
	// {T}, Tap three untapped white creatures you control: Destroy target creature.
	// TODO: implement
	Register("Hand of Justice", withExpansion(func() Card {
		return NewCreature("Hand of Justice", "{5}{W}", 2, 6,
			WithSubTypes("Avatar"),
		)
	}))

	// Icatian Infantry {W}
	// Creature — Human Soldier
	// 1/1
	// {1}: This creature gains first strike until end of turn.
	// {1}: This creature gains banding until end of turn. (Any creatures with banding, and up to one without, can attack in a band. Bands are blocked as a group. If any creatures with banding you control are blocking or being blocked by a creature, you divide that creature's combat damage, not its controller, among any of the creatures it's being blocked by or is blocking.)
	Register("Icatian Infantry", withExpansion(func() Card {
		return NewCreature("Icatian Infantry", "{W}", 1, 1,
			WithSubTypes("Human", "Soldier"),
			WithActivatedAbility(
				GrantKeyword(FirstStrike).Targeting(ToSource()),
				GenericCost(1),
			),
			WithActivatedAbility(
				GrantKeyword(Banding).Targeting(ToSource()),
				GenericCost(1),
			),
		)
	}))

	// Icatian Javelineers {W}
	// Creature — Human Soldier
	// 1/1
	// This creature enters with a javelin counter on it.
	// {T}, Remove a javelin counter from this creature: It deals 1 damage to any target.
	Register("Icatian Javelineers", withExpansion(func() Card {
		return NewCreature("Icatian Javelineers", "{W}", 1, 1,
			WithSubTypes("Human", "Soldier"),
			// Enters with a javelin counter
			WithAbility(ETBEffect(AddCounters(Javelin, Fixed(1)).Targeting(ToSource()))),
			// {T}, Remove a javelin counter: deals 1 damage to any target
			WithActivatedAbility(
				DealDamage(Fixed(1)),
				TapSourceCost(),
				WithCost(RemoveCountersCost(Javelin, 1)),
				WithTarget(TargetAnyTarget()),
			),
		)
	}))

	// Icatian Lieutenant {W}{W}
	// Creature — Human Soldier
	// 1/2
	// {1}{W}: Target Soldier creature gets +1/+0 until end of turn.
	Register("Icatian Lieutenant", withExpansion(func() Card {
		return NewCreature("Icatian Lieutenant", "{W}{W}", 1, 2,
			WithSubTypes("Human", "Soldier"),
			WithActivatedAbility(
				Boost(Fixed(1), Fixed(0)),
				ManaCostOf("{1}{W}"),
				WithTarget(TargetCreature(HasSubType("Soldier"))),
			),
		)
	}))

	// Icatian Moneychanger {W}
	// Creature — Human
	// 0/2
	// This creature enters with three credit counters on it.
	// When this creature enters, it deals 3 damage to you.
	// At the beginning of your upkeep, put a credit counter on this creature.
	// Sacrifice this creature: You gain 1 life for each credit counter on this creature. Activate only during your upkeep.
	Register("Icatian Moneychanger", withExpansion(moneychangerFactory))

	// Icatian Phalanx {4}{W}
	// Creature — Human Soldier
	// 2/4
	// Banding (Any creatures with banding, and up to one without, can attack in a band. Bands are blocked as a group. If any creatures with banding you control are blocking or being blocked by a creature, you divide that creature's combat damage, not its controller, among any of the creatures it's being blocked by or is blocking.)
	Register("Icatian Phalanx", withExpansion(func() Card {
		return NewCreature("Icatian Phalanx", "{4}{W}", 2, 4,
			WithSubTypes("Human", "Soldier"),
			WithKeyword(Banding),
		)
	}))

	// Icatian Priest {W}
	// Creature — Human Cleric
	// 1/1
	// {1}{W}{W}: Target creature gets +1/+1 until end of turn.
	Register("Icatian Priest", withExpansion(func() Card {
		return NewCreature("Icatian Priest", "{W}", 1, 1,
			WithSubTypes("Human", "Cleric"),
			WithActivatedAbility(
				Boost(Fixed(1), Fixed(1)),
				ManaCostOf("{1}{W}{W}"),
				WithTarget(TargetCreature()),
			),
		)
	}))

	// Icatian Scout {W}
	// Creature — Human Soldier Scout
	// 1/1
	// {1}, {T}: Target creature gains first strike until end of turn.
	Register("Icatian Scout", withExpansion(func() Card {
		return NewCreature("Icatian Scout", "{W}", 1, 1,
			WithSubTypes("Human", "Soldier", "Scout"),
			WithActivatedAbility(
				GrantKeyword(FirstStrike),
				GenericCost(1),
				WithCost(TapSourceCost()),
				WithTarget(TargetCreature()),
			),
		)
	}))

	// Icatian Skirmishers {3}{W}
	// Creature — Human Soldier
	// 1/1
	// First strike; banding (Any creatures with banding, and up to one without, can attack in a band. Bands are blocked as a group. If any creatures with banding you control are blocking or being blocked by a creature, you divide that creature's combat damage, not its controller, among any of the creatures it's being blocked by or is blocking.)
	// Whenever this creature attacks, all creatures banded with it gain first strike until end of turn.
	// TODO: implement
	Register("Icatian Skirmishers", withExpansion(func() Card {
		return NewCreature("Icatian Skirmishers", "{3}{W}", 1, 1,
			WithSubTypes("Human", "Soldier"),
		)
	}))

	// Order of Leitbur {W}{W}
	// Creature — Human Cleric Knight
	// 2/1
	// Protection from black
	// {W}: This creature gains first strike until end of turn.
	// {W}{W}: This creature gets +1/+0 until end of turn.
	Register("Order of Leitbur", withExpansion(func() Card {
		return NewCreature("Order of Leitbur", "{W}{W}", 2, 1,
			WithSubTypes("Human", "Cleric", "Knight"),
			WithAbility(ProtectionFromColor(Black)),
			WithActivatedAbility(
				GrantKeyword(FirstStrike).Targeting(ToSource()),
				ManaCostOf("{W}"),
			),
			WithActivatedAbility(
				Boost(Fixed(1), Fixed(0)).Targeting(ToSource()),
				ManaCostOf("{W}{W}"),
			),
		)
	}))

	// ===== BLUE CREATURES =====

	// Deep Spawn {5}{U}{U}{U}
	// Creature — Homarid
	// 6/6
	// Trample
	// At the beginning of your upkeep, sacrifice this creature unless you mill two cards.
	// {U}: This creature gains shroud until end of turn and doesn't untap during your next untap step. Tap this creature. (A creature with shroud can't be the target of spells or abilities.)
	// TODO: implement
	Register("Deep Spawn", withExpansion(func() Card {
		return NewCreature("Deep Spawn", "{5}{U}{U}{U}", 6, 6,
			WithSubTypes("Homarid"),
		)
	}))

	// Homarid {2}{U}
	// Creature — Homarid
	// 2/2
	// This creature enters with a tide counter on it.
	// At the beginning of your upkeep, put a tide counter on this creature.
	// As long as there is exactly one tide counter on this creature, it gets -1/-1.
	// As long as there are exactly three tide counters on this creature, it gets +1/+1.
	// Whenever there are four or more tide counters on this creature, remove all tide counters from it.
	Register("Homarid", withExpansion(func() Card {
		return NewCreature("Homarid", "{2}{U}", 2, 2,
			WithSubTypes("Homarid"),
			// ETB with 1 tide counter
			WithAbility(ETBEffect(AddCounters(Tide, Fixed(1)).Targeting(ToSource()))),
			// At the beginning of your upkeep, put a tide counter on this creature.
			// Also handle "whenever there are four or more tide counters, remove all."
			// TODO: convert to pipeline — needs AddCounters + conditional counter reset (threshold check)
			WithAbility(BeginningOfUpkeepTrigger(
				FuncEffect("put a tide counter on Homarid and check for reset",
					EffectProperties{Outcome: OutcomeUnknown},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						src := g.FindPermanent(sourceID)
						if src == nil {
							return nil
						}
						src.AddCounter(Tide, 1)
						// Whenever there are four or more tide counters, remove all
						if src.Counters[Tide] >= 4 {
							src.RemoveCounter(Tide, int(src.Counters[Tide]))
						}
						return nil
					}),
				false,
			)),
			// As long as there is exactly one tide counter, -1/-1
			// As long as there are exactly three tide counters, +1/+1
			WithStaticAbility(FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
				src := g.FindPermanent(sourceID)
				if src == nil {
					return nil
				}
				count := src.Counters[Tide]
				switch count {
				case 1:
					src.BoostPT(-1, -1)
				case 3:
					src.BoostPT(1, 1)
				}
				return nil
			})),
		)
	}))

	// Homarid Shaman {2}{U}{U}
	// Creature — Homarid Shaman
	// 2/1
	// {U}: Tap target green creature.
	Register("Homarid Shaman", withExpansion(func() Card {
		return NewCreature("Homarid Shaman", "{2}{U}{U}", 2, 1,
			WithSubTypes("Homarid", "Shaman"),
			WithActivatedAbility(
				TapTarget(),
				ManaCostOf("{U}"),
				WithTarget(TargetCreature(HasColorFilter(Green))),
			),
		)
	}))

	// Homarid Warrior {4}{U}
	// Creature — Homarid Warrior
	// 3/3
	// {U}: This creature gains shroud until end of turn and doesn't untap during your next untap step. Tap it. (A creature with shroud can't be the target of spells or abilities.)
	// TODO: implement
	Register("Homarid Warrior", withExpansion(func() Card {
		return NewCreature("Homarid Warrior", "{4}{U}", 3, 3,
			WithSubTypes("Homarid", "Warrior"),
		)
	}))

	// River Merfolk {U}{U}
	// Creature — Merfolk
	// 2/1
	// {U}: This creature gains mountainwalk until end of turn. (It can't be blocked as long as defending player controls a Mountain.)
	Register("River Merfolk", withExpansion(func() Card {
		return NewCreature("River Merfolk", "{U}{U}", 2, 1,
			WithSubTypes("Merfolk"),
			WithActivatedAbility(
				GrantKeyword(Mountainwalk).Targeting(ToSource()),
				ManaCostOf("{U}"),
			),
		)
	}))

	// Seasinger {1}{U}{U}
	// Creature — Merfolk
	// 0/1
	// When you control no Islands, sacrifice this creature.
	// You may choose not to untap this creature during your untap step.
	// {T}: Gain control of target creature whose controller controls an Island for as long as you control this creature and this creature remains tapped.
	// TODO: implement
	Register("Seasinger", withExpansion(func() Card {
		return NewCreature("Seasinger", "{1}{U}{U}", 0, 1,
			WithSubTypes("Merfolk"),
		)
	}))

	// Svyelunite Priest {1}{U}
	// Creature — Merfolk Cleric
	// 1/1
	// {U}{U}, {T}: Target creature gains shroud until end of turn. Activate only during your upkeep. (It can't be the target of spells or abilities.)
	Register("Svyelunite Priest", withExpansion(func() Card {
		return NewCreature("Svyelunite Priest", "{1}{U}", 1, 1,
			WithSubTypes("Merfolk", "Cleric"),
			WithActivatedAbility(
				GrantKeyword(Shroud),
				ManaCostOf("{U}{U}"),
				WithCost(TapSourceCost()),
				WithTarget(TargetCreature()),
				WithUpkeepOnly(),
			),
		)
	}))

	// Vodalian Knights {1}{U}{U}
	// Creature — Merfolk Knight
	// 2/2
	// First strike
	// This creature can't attack unless defending player controls an Island.
	// When you control no Islands, sacrifice this creature.
	// {U}: This creature gains flying until end of turn.
	Register("Vodalian Knights", withExpansion(func() Card {
		return NewCreature("Vodalian Knights", "{1}{U}{U}", 2, 2,
			WithSubTypes("Merfolk", "Knight"),
			WithKeyword(FirstStrike),
			// When you control no Islands, sacrifice this creature.
			WithAbility(SacrificeUnlessLand("Island")),
			// Can't attack unless defending player controls an Island.
			WithStaticAbility(PreventFromAttackingIfDefendingPlayerControls(And(IsLand, HasSubType("Island")))),
			// {U}: This creature gains flying until end of turn.
			WithActivatedAbility(
				GrantKeyword(Flying).Targeting(ToSource()),
				ManaCostOf("{U}"),
			),
		)
	}))

	// Vodalian Mage {2}{U}
	// Creature — Merfolk Wizard
	// 1/1
	// {U}, {T}: Counter target spell unless its controller pays {1}.
	// TODO: implement
	Register("Vodalian Mage", withExpansion(func() Card {
		return NewCreature("Vodalian Mage", "{2}{U}", 1, 1,
			WithSubTypes("Merfolk", "Wizard"),
		)
	}))

	// Vodalian Soldiers {1}{U}
	// Creature — Merfolk Soldier
	// 1/2
	Register("Vodalian Soldiers", withExpansion(func() Card {
		return NewCreature("Vodalian Soldiers", "{1}{U}", 1, 2,
			WithSubTypes("Merfolk", "Soldier"),
		)
	}))

	// Vodalian War Machine {1}{U}{U}
	// Creature — Wall
	// 0/4
	// Defender (This creature can't attack.)
	// Tap an untapped Merfolk you control: This creature can attack this turn as though it didn't have defender.
	// Tap an untapped Merfolk you control: This creature gets +2/+1 until end of turn.
	// When this creature dies, destroy all Merfolk tapped this turn to pay for its abilities.
	// TODO: implement
	Register("Vodalian War Machine", withExpansion(func() Card {
		return NewCreature("Vodalian War Machine", "{1}{U}{U}", 0, 4,
			WithSubTypes("Wall"),
		)
	}))

	// ===== BLACK CREATURES =====

	// Armor Thrull {2}{B}
	// Creature — Thrull
	// 1/3
	// {T}, Sacrifice this creature: Put a +1/+2 counter on target creature.
	Register("Armor Thrull", withExpansion(func() Card {
		return NewCreature("Armor Thrull", "{2}{B}", 1, 3,
			WithSubTypes("Thrull"),
			WithActivatedAbility(
				AddCounters(P1P2, Fixed(1)),
				TapSourceCost(),
				WithCost(SacrificeSourceCost()),
				WithTarget(TargetCreature()),
			),
		)
	}))

	// Basal Thrull {B}{B}
	// Creature — Thrull
	// 1/2
	// {T}, Sacrifice this creature: Add {B}{B}.
	Register("Basal Thrull", withExpansion(func() Card {
		return NewCreature("Basal Thrull", "{B}{B}", 1, 2,
			WithSubTypes("Thrull"),
			WithActivatedAbility(
				AddMana(Black, 2),
				TapSourceCost(),
				WithCost(SacrificeSourceCost()),
			),
		)
	}))

	// Derelor {3}{B}
	// Creature — Thrull
	// 4/4
	// Black spells you cast cost {B} more to cast.
	// TODO: implement
	Register("Derelor", withExpansion(func() Card {
		return NewCreature("Derelor", "{3}{B}", 4, 4,
			WithSubTypes("Thrull"),
			// Black spells you cast cost {B} more to cast.
			// Using IncreaseSpellCostForColor which adds generic cost globally.
			// This is the closest approximation with the existing engine API.
			WithStaticAbility(IncreaseSpellCostForColor(Black, 1)),
		)
	}))

	// Ebon Praetor {4}{B}{B}
	// Creature — Avatar Praetor
	// 5/5
	// First strike, trample
	// At the beginning of your upkeep, put a -2/-2 counter on this creature.
	// Sacrifice a creature: Remove a -2/-2 counter from this creature. If the sacrificed creature was a Thrull, put a +1/+0 counter on this creature. Activate only during your upkeep and only once each turn.
	// TODO: implement
	Register("Ebon Praetor", withExpansion(func() Card {
		return NewCreature("Ebon Praetor", "{4}{B}{B}", 5, 5,
			WithSubTypes("Avatar", "Praetor"),
		)
	}))

	// Initiates of the Ebon Hand {B}
	// Creature — Cleric
	// 1/1
	// {1}: Add {B}. If this ability has been activated four or more times this turn, sacrifice this creature at the beginning of the next end step.
	// TODO: implement
	Register("Initiates of the Ebon Hand", withExpansion(func() Card {
		return NewCreature("Initiates of the Ebon Hand", "{B}", 1, 1,
			WithSubTypes("Cleric"),
		)
	}))

	// Mindstab Thrull {1}{B}{B}
	// Creature — Thrull
	// 2/2
	// Whenever this creature attacks and isn't blocked, you may sacrifice it. If you do, defending player discards three cards.
	Register("Mindstab Thrull", withExpansion(func() Card {
		return NewCreature("Mindstab Thrull", "{1}{B}{B}", 2, 2,
			WithSubTypes("Thrull"),
			WithAbility(NewTriggered(EvtBlockersDecl, true,
				SacrificeSource(),
				DiscardCards(Fixed(3)).Targeting(SelectDefendingPlayer()),
			).SetConditionData(SourceIsUnblockedAttacker{})),
		)
	}))

	// Necrite {1}{B}{B}
	// Creature — Thrull
	// 2/2
	// Whenever this creature attacks and isn't blocked, you may sacrifice it. If you do, destroy target creature defending player controls. It can't be regenerated.
	Register("Necrite", withExpansion(func() Card {
		return NewCreature("Necrite", "{1}{B}{B}", 2, 2,
			WithSubTypes("Thrull"),
			// TODO: convert to pipeline — needs SacrificeSourceStep + ChoosePermanentStep (defending player's creatures) + DestroyGatheredNoRegen
			WithAbility(NewTriggered(EvtBlockersDecl, true,
				FuncEffect("sacrifice, destroy target creature defending player controls",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						src := g.FindPermanent(sourceID)
						if src == nil {
							return nil
						}
						// Find creatures defending player controls
						defending := g.NonActivePlayerObj()
						if defending == nil {
							return nil
						}
						candidates := g.FilterBattlefield(And(ControlledBy(defending.PlayerID()), IsCreature))
						if len(candidates) == 0 {
							return nil
						}
						player := g.GetPlayer(controller)
						chosen := player.ChoosePermanent(candidates, "destroy target creature", g)
						if chosen == nil {
							return nil
						}
						g.Sacrifice(src)
						chosen.GrantBaseAttr(CantRegenerate)
						g.DestroyPermanent(chosen)
						return nil
					}),
			).
				SetConditionData(SourceIsUnblockedAttacker{})),
		)
	}))

	// Order of the Ebon Hand {B}{B}
	// Creature — Cleric Knight
	// 2/1
	// Protection from white
	// {B}: This creature gains first strike until end of turn.
	// {B}{B}: This creature gets +1/+0 until end of turn.
	Register("Order of the Ebon Hand", withExpansion(func() Card {
		return NewCreature("Order of the Ebon Hand", "{B}{B}", 2, 1,
			WithSubTypes("Cleric", "Knight"),
			WithAbility(ProtectionFromColor(White)),
			WithActivatedAbility(
				GrantKeyword(FirstStrike).Targeting(ToSource()),
				ManaCostOf("{B}"),
			),
			WithActivatedAbility(
				Boost(Fixed(1), Fixed(0)).Targeting(ToSource()),
				ManaCostOf("{B}{B}"),
			),
		)
	}))

	// Thrull Champion {4}{B}
	// Creature — Thrull
	// 2/2
	// Thrull creatures get +1/+1.
	// {T}: Gain control of target Thrull for as long as you control this creature.
	Register("Thrull Champion", withExpansion(func() Card {
		return NewCreature("Thrull Champion", "{4}{B}", 2, 2,
			WithSubTypes("Thrull"),
			// Thrull creatures get +1/+1 (lord effect, doesn't boost self)
			WithStaticAbility(BoostOtherControlledCreatures(1, 1, HasSubType("Thrull"))),
		)
	}))

	// Thrull Wizard {2}{B}
	// Creature — Thrull Wizard
	// 1/1
	// {1}{B}: Counter target black spell unless that spell's controller pays {B} or {3}.
	// TODO: implement
	Register("Thrull Wizard", withExpansion(func() Card {
		return NewCreature("Thrull Wizard", "{2}{B}", 1, 1,
			WithSubTypes("Thrull", "Wizard"),
		)
	}))

	// ===== RED CREATURES =====

	// Brassclaw Orcs {2}{R}
	// Creature — Orc
	// 3/2
	// This creature can't block creatures with power 2 or greater.
	Register("Brassclaw Orcs", withExpansion(func() Card {
		return NewCreature("Brassclaw Orcs", "{2}{R}", 3, 2,
			WithSubTypes("Orc"),
			WithStaticAbility(FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
				for _, p := range g.AllBattlefield() {
					if p.HasType(TypeCreature) && p.CurrentPower(g) >= 2 {
						g.PreventBlockPair(sourceID, p.ID())
					}
				}
				return nil
			})),
		)
	}))

	// Dwarven Armorer {R}
	// Creature — Dwarf
	// 0/2
	// {R}, {T}, Discard a card: Put a +0/+1 counter or a +1/+0 counter on target creature.
	// TODO: implement
	Register("Dwarven Armorer", withExpansion(func() Card {
		return NewCreature("Dwarven Armorer", "{R}", 0, 2,
			WithSubTypes("Dwarf"),
		)
	}))

	// Dwarven Lieutenant {R}{R}
	// Creature — Dwarf Soldier
	// 1/2
	// {1}{R}: Target Dwarf creature gets +1/+0 until end of turn.
	Register("Dwarven Lieutenant", withExpansion(func() Card {
		return NewCreature("Dwarven Lieutenant", "{R}{R}", 1, 2,
			WithSubTypes("Dwarf", "Soldier"),
			WithActivatedAbility(
				Boost(Fixed(1), Fixed(0)),
				ManaCostOf("{1}{R}"),
				WithTarget(TargetCreature(HasSubType("Dwarf"))),
			),
		)
	}))

	// Dwarven Soldier {1}{R}
	// Creature — Dwarf Soldier
	// 2/1
	// Whenever this creature blocks or becomes blocked by one or more Orcs, this creature gets +0/+2 until end of turn.
	Register("Dwarven Soldier", withExpansion(func() Card {
		return NewCreature("Dwarven Soldier", "{1}{R}", 2, 1,
			WithSubTypes("Dwarf", "Soldier"),
			// Trigger when this creature blocks an Orc
			WithAbility(NewTriggered(EvtDeclaredBlocker, false,
				Boost(Fixed(0), Fixed(2)).Targeting(ToSource()),
			).
				SetConditionData(OrTriggerCond{Conditions: []TriggerConditionData{
					AndTriggerCond{Conditions: []TriggerConditionData{
						EventSourceIsSelf{},
						EventTargetHasSubType{SubType: "Orc"},
					}},
					AndTriggerCond{Conditions: []TriggerConditionData{
						EventTargetIsSelf{},
						EventSourceHasSubType{SubType: "Orc"},
					}},
				}})),
		)
	}))

	// Goblin Chirurgeon {R}
	// Creature — Goblin Shaman
	// 0/2
	// Sacrifice a Goblin: Regenerate target creature.
	// TODO: implement
	Register("Goblin Chirurgeon", withExpansion(func() Card {
		return NewCreature("Goblin Chirurgeon", "{R}", 0, 2,
			WithSubTypes("Goblin", "Shaman"),
		)
	}))

	// Goblin Flotilla {2}{R}
	// Creature — Goblin
	// 2/2
	// Islandwalk (This creature can't be blocked as long as defending player controls an Island.)
	// At the beginning of each combat, unless you pay {R}, whenever this creature blocks or becomes blocked by a creature this combat, that creature gains first strike until end of turn.
	// TODO: implement
	Register("Goblin Flotilla", withExpansion(func() Card {
		return NewCreature("Goblin Flotilla", "{2}{R}", 2, 2,
			WithSubTypes("Goblin"),
		)
	}))

	// Orcish Captain {R}
	// Creature — Orc Warrior
	// 1/1
	// {1}: Flip a coin. If you win the flip, target Orc creature gets +2/+0 until end of turn. If you lose the flip, it gets -0/-2 until end of turn.
	// TODO: implement
	Register("Orcish Captain", withExpansion(func() Card {
		return NewCreature("Orcish Captain", "{R}", 1, 1,
			WithSubTypes("Orc", "Warrior"),
		)
	}))

	// Orcish Spy {R}
	// Creature — Orc Rogue
	// 1/1
	// {T}: Look at the top three cards of target player's library.
	// TODO: implement
	Register("Orcish Spy", withExpansion(func() Card {
		return NewCreature("Orcish Spy", "{R}", 1, 1,
			WithSubTypes("Orc", "Rogue"),
		)
	}))

	// Orcish Veteran {2}{R}
	// Creature — Orc
	// 2/2
	// This creature can't block white creatures with power 2 or greater.
	// {R}: This creature gains first strike until end of turn.
	Register("Orcish Veteran", withExpansion(func() Card {
		return NewCreature("Orcish Veteran", "{2}{R}", 2, 2,
			WithSubTypes("Orc"),
			// Can't block white creatures with power 2 or greater
			WithStaticAbility(FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
				for _, p := range g.AllBattlefield() {
					if p.HasType(TypeCreature) && p.CurrentPower(g) >= 2 {
						isWhite := false
						for _, c := range p.Colors() {
							if c == White {
								isWhite = true
								break
							}
						}
						if isWhite {
							g.PreventBlockPair(sourceID, p.ID())
						}
					}
				}
				return nil
			})),
			WithActivatedAbility(
				GrantKeyword(FirstStrike).Targeting(ToSource()),
				ManaCostOf("{R}"),
			),
		)
	}))

	// Orgg {3}{R}{R}
	// Creature — Orgg
	// 6/6
	// Trample
	// This creature can't attack if defending player controls an untapped creature with power 3 or greater.
	// This creature can't block creatures with power 3 or greater.
	Register("Orgg", withExpansion(func() Card {
		return NewCreature("Orgg", "{3}{R}{R}", 6, 6,
			WithSubTypes("Orgg"),
			WithKeyword(Trample),
			// Can't attack if defending player controls an untapped creature with power 3 or greater
			WithStaticAbility(FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
				who := g.NonActivePlayerObj()
				whoID := who.PlayerID()
				if g.AnyBattlefield(And(ControlledBy(whoID), IsCreature, IsUntapped, HasPowerGTE(3))) {
					g.RevokeAttr(sourceID, AttrCanAttack)
				}
				return nil
			})),
			// Can't block creatures with power 3 or greater
			WithStaticAbility(FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
				for _, p := range g.AllBattlefield() {
					if p.HasType(TypeCreature) && p.CurrentPower(g) >= 3 {
						g.PreventBlockPair(sourceID, p.ID())
					}
				}
				return nil
			})),
		)
	}))

	// ===== GREEN CREATURES =====

	// Elvish Farmer {1}{G}
	// Creature — Elf
	// 0/2
	// At the beginning of your upkeep, put a spore counter on this creature.
	// Remove three spore counters from this creature: Create a 1/1 green Saproling creature token.
	// Sacrifice a Saproling: You gain 2 life.
	// NOTE: Third ability (Sacrifice a Saproling) skipped — requires SacrificeMatchingCost engine work.
	Register("Elvish Farmer", withExpansion(func() Card {
		return NewCreature("Elvish Farmer", "{1}{G}", 0, 2,
			WithSubTypes("Elf"),
			WithAbility(BeginningOfUpkeepTrigger(
				AddCounters(Spore, Fixed(1)).Targeting(ToSource()), false,
			)),
			WithActivatedAbility(
				CreateToken("Saproling", 1, 1, []CardType{TypeCreature}, []string{"Saproling"}),
				RemoveCountersCost(Spore, 3),
			),
		)
	}))

	// Elvish Hunter {1}{G}
	// Creature — Elf Archer
	// 1/1
	// {1}{G}, {T}: Target creature doesn't untap during its controller's next untap step.
	Register("Elvish Hunter", withExpansion(func() Card {
		return NewCreature("Elvish Hunter", "{1}{G}", 1, 1,
			WithSubTypes("Elf", "Archer"),
			// TODO: convert to pipeline — needs DoesNotUntapStep (creates FuncContinuousEffect with expiry)
			WithActivatedAbility(
				FuncEffect("target creature doesn't untap during its controller's next untap step",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						targetID := targets[0]
						// In 2-player, "controller's next untap step" is 2 game turns from now
						expiryTurn := g.CurrentTurn() + 2
						ce := FuncContinuousEffect(LayerAbility, Indefinite, func(g *Game, _ uuid.UUID) error {
							p := g.FindPermanent(targetID)
							if p != nil {
								g.GrantAttr(p.ID(), AttrDoesNotUntap)
							}
							return nil
						}, func(g *Game, _ uuid.UUID) bool {
							return g.CurrentTurn() <= expiryTurn && g.FindPermanent(targetID) != nil
						})
						ce.SetSourceID(sourceID)
						g.AddContinuousEffect(ce)
						return nil
					}),
				ManaCostOf("{1}{G}"),
				WithCost(TapSourceCost()),
				WithTarget(TargetCreature()),
			),
		)
	}))

	// Elvish Scout {G}
	// Creature — Elf Scout
	// 1/1
	// {G}, {T}: Untap target attacking creature you control. Prevent all combat damage that would be dealt to and dealt by it this turn.
	// TODO: implement
	Register("Elvish Scout", withExpansion(func() Card {
		return NewCreature("Elvish Scout", "{G}", 1, 1,
			WithSubTypes("Elf", "Scout"),
		)
	}))

	// Feral Thallid {3}{G}{G}{G}
	// Creature — Fungus
	// 6/3
	// At the beginning of your upkeep, put a spore counter on this creature.
	// Remove three spore counters from this creature: Regenerate this creature.
	Register("Feral Thallid", withExpansion(func() Card {
		return NewCreature("Feral Thallid", "{3}{G}{G}{G}", 6, 3,
			WithSubTypes("Fungus"),
			WithAbility(BeginningOfUpkeepTrigger(
				AddCounters(Spore, Fixed(1)).Targeting(ToSource()), false,
			)),
			WithActivatedAbility(
				RegenerateSource(),
				RemoveCountersCost(Spore, 3),
			),
		)
	}))

	// Spore Flower {G}{G}
	// Creature — Fungus
	// 0/1
	// At the beginning of your upkeep, put a spore counter on this creature.
	// Remove three spore counters from this creature: Prevent all combat damage that would be dealt this turn.
	Register("Spore Flower", withExpansion(func() Card {
		return NewCreature("Spore Flower", "{G}{G}", 0, 1,
			WithSubTypes("Fungus"),
			WithAbility(BeginningOfUpkeepTrigger(
				AddCounters(Spore, Fixed(1)).Targeting(ToSource()), false,
			)),
			WithActivatedAbility(
				PreventAllCombatDamage(),
				RemoveCountersCost(Spore, 3),
			),
		)
	}))

	// Thallid {G}
	// Creature — Fungus
	// 1/1
	// At the beginning of your upkeep, put a spore counter on this creature.
	// Remove three spore counters from this creature: Create a 1/1 green Saproling creature token.
	Register("Thallid", withExpansion(func() Card {
		return NewCreature("Thallid", "{G}", 1, 1,
			WithSubTypes("Fungus"),
			WithAbility(BeginningOfUpkeepTrigger(
				AddCounters(Spore, Fixed(1)).Targeting(ToSource()), false,
			)),
			WithActivatedAbility(
				CreateToken("Saproling", 1, 1, []CardType{TypeCreature}, []string{"Saproling"}),
				RemoveCountersCost(Spore, 3),
			),
		)
	}))

	// Thallid Devourer {1}{G}{G}
	// Creature — Fungus
	// 2/2
	// At the beginning of your upkeep, put a spore counter on this creature.
	// Remove three spore counters from this creature: Create a 1/1 green Saproling creature token.
	// Sacrifice a Saproling: This creature gets +1/+2 until end of turn.
	// NOTE: Third ability (Sacrifice a Saproling) skipped — requires SacrificeMatchingCost engine work.
	Register("Thallid Devourer", withExpansion(func() Card {
		return NewCreature("Thallid Devourer", "{1}{G}{G}", 2, 2,
			WithSubTypes("Fungus"),
			WithAbility(BeginningOfUpkeepTrigger(
				AddCounters(Spore, Fixed(1)).Targeting(ToSource()), false,
			)),
			WithActivatedAbility(
				CreateToken("Saproling", 1, 1, []CardType{TypeCreature}, []string{"Saproling"}),
				RemoveCountersCost(Spore, 3),
			),
		)
	}))

	// Thelonite Druid {2}{G}
	// Creature — Human Cleric Druid
	// 1/1
	// {1}{G}, {T}, Sacrifice a creature: Forests you control become 2/3 creatures until end of turn. They're still lands.
	// TODO: implement
	Register("Thelonite Druid", withExpansion(func() Card {
		return NewCreature("Thelonite Druid", "{2}{G}", 1, 1,
			WithSubTypes("Human", "Cleric", "Druid"),
		)
	}))

	// Thelonite Monk {2}{G}{G}
	// Creature — Insect Monk Cleric
	// 1/2
	// {T}, Sacrifice a green creature: Target land becomes a Forest. (This effect lasts indefinitely.)
	// TODO: implement
	Register("Thelonite Monk", withExpansion(func() Card {
		return NewCreature("Thelonite Monk", "{2}{G}{G}", 1, 2,
			WithSubTypes("Insect", "Monk", "Cleric"),
		)
	}))

	// Thorn Thallid {1}{G}{G}
	// Creature — Fungus
	// 2/2
	// At the beginning of your upkeep, put a spore counter on this creature.
	// Remove three spore counters from this creature: It deals 1 damage to any target.
	Register("Thorn Thallid", withExpansion(func() Card {
		return NewCreature("Thorn Thallid", "{1}{G}{G}", 2, 2,
			WithSubTypes("Fungus"),
			WithAbility(BeginningOfUpkeepTrigger(
				AddCounters(Spore, Fixed(1)).Targeting(ToSource()), false,
			)),
			WithActivatedAbility(
				DealDamage(Fixed(1)),
				RemoveCountersCost(Spore, 3),
				WithTarget(TargetAnyTarget()),
			),
		)
	}))
}
