package astral

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage"
	. "github.com/benprew/mage-go/pkg/mage/core"
)

func init() {
	registerCreatures()
}

func applyFaerieDragonAction(g *Game, sourceID, controllerID uuid.UUID, action int) error {
	switch action {
	case 0: // Berserk
		return ApplyEffect(g, ApplyToRandomPermanent(BerserkEffect(), IsCreature), sourceID, controllerID, nil)
	case 1: // Twiddle
		permanent := g.RandomPermanent(Or(IsArtifact, IsCreature, IsLand))
		player := g.GetPlayer(controllerID)
		if permanent == nil || player == nil || !player.ChooseMayAbility("tap or untap the chosen permanent") {
			return nil
		}
		return ApplyEffect(g, TwiddleEffect(), sourceID, controllerID, []uuid.UUID{permanent.ID()})
	case 2: // Blood Lust
		return ApplyEffect(g, ApplyToRandomPermanent(BloodLustEffect(), IsCreature), sourceID, controllerID, nil)
	case 3: // Lifelace
		return ApplyEffect(g, ApplyToRandomSpellOrPermanent(LaceEffect(Green)), sourceID, controllerID, nil)
	case 4: // Purelace
		return ApplyEffect(g, ApplyToRandomSpellOrPermanent(LaceEffect(White)), sourceID, controllerID, nil)
	case 5: // Chaoslace
		return ApplyEffect(g, ApplyToRandomSpellOrPermanent(LaceEffect(Red)), sourceID, controllerID, nil)
	case 6: // Lightning Bolt
		return ApplyEffect(g, ApplyToRandomDamageTarget(LightningBoltEffect()), sourceID, controllerID, nil)
	case 7: // Flying Carpet
		return ApplyEffect(g, ApplyToRandomPermanent(FlyingCarpetEffect(), IsCreature), sourceID, controllerID, nil)
	case 8: // Giant Growth
		return ApplyEffect(g, ApplyToRandomPermanent(GiantGrowthEffect(), IsCreature), sourceID, controllerID, nil)
	case 9: // Helm of Chatzuk
		return ApplyEffect(g, ApplyToRandomPermanent(HelmOfChatzukEffect(), IsCreature), sourceID, controllerID, nil)
	case 10: // Deathlace
		return ApplyEffect(g, ApplyToRandomSpellOrPermanent(LaceEffect(Black)), sourceID, controllerID, nil)
	case 11: // Thoughtlace
		return ApplyEffect(g, ApplyToRandomSpellOrPermanent(LaceEffect(Blue)), sourceID, controllerID, nil)
	case 12: // Hurr Jackal
		return ApplyEffect(g, ApplyToRandomPermanent(HurrJackalEffect(), IsCreature), sourceID, controllerID, nil)
	case 13: // Tawnos's Wand
		return ApplyEffect(g, ApplyToRandomPermanent(TawnosWandEffect(), And(IsCreature, HasPowerLTE(2))), sourceID, controllerID, nil)
	case 14: // Staff of Zegon
		return ApplyEffect(g, ApplyToRandomPermanent(StaffOfZegonEffect(), IsCreature), sourceID, controllerID, nil)
	case 15: // Unsummon
		return ApplyEffect(g, ApplyToRandomPermanent(UnsummonEffect(), IsCreature), sourceID, controllerID, nil)
	case 16: // Prodigal Sorcerer
		return ApplyEffect(g, ApplyToRandomDamageTarget(ProdigalSorcererEffect()), sourceID, controllerID, nil)
	case 17: // Sorceress Queen
		return ApplyEffect(g, ApplyToRandomPermanent(SorceressQueenEffect(), And(IsCreature, NotID(sourceID))), sourceID, controllerID, nil)
	case 18: // Swords to Plowshares
		return ApplyEffect(g, ApplyToRandomPermanent(SwordsToPlowsharesEffect(), IsCreature), sourceID, controllerID, nil)
	case 19: // Lesser Werewolf
		return ApplyEffect(g, ApplyToRandomPermanent(LesserWerewolfCounterEffect(), IsCreature), sourceID, controllerID, nil)
	}
	return nil
}

func registerCreatures() {

	// Aswan Jaguar {1}{G}{G}
	// Summon Jaguar
	// 2/2
	// When Aswan Jaguar comes into play, choose a random creature type from those in target opponent's deck.
	// {G}{G}, {T}: Bury target creature of the chosen type.
	Register("Aswan Jaguar", func() Card {
		return NewCreature("Aswan Jaguar", "{1}{G}{G}", 2, 2,
			WithSubTypes("Jaguar"),
			WithAbility(
				EntersBattlefieldTrigger(
					ChooseRandomCreatureSubtypeFromTargetLibrary(), false,
				).AddTarget(TargetOpponent()),
			),
			WithActivatedAbility(
				DestroyTargetNoRegen(), ManaCostOf("{G}{G}"),
				WithCost(Tap()),
				WithTarget(TargetCreatureOfSourceChosenSubtype()),
			),
		)
	})

	// Faerie Dragon {2}{G}{G}
	// Summon Dragon
	// 1/3
	// Flying
	// {1}{G}{G}: Play a random effect.
	Register("Faerie Dragon", func() Card {
		return NewCreature("Faerie Dragon", "{2}{G}{G}", 1, 3,
			WithSubTypes("Dragon"), WithKeyword(Flying),
			WithActivatedAbility(
				FuncEffect("play a random effect", EffectProperties{}, func(g *Game, sourceID, controllerID uuid.UUID, _ []uuid.UUID) error {
					return applyFaerieDragonAction(g, sourceID, controllerID, g.RandIntn(20))
				}),
				ManaCostOf("{1}{G}{G}"),
			),
		)
	})

	// Goblin Polka Band {R}{R}
	// Summon Goblin
	// 1/1
	// {2}, {T}, Pay {R} for each target: Tap any number of random target creatures. Goblins tapped in this way do not untap during their controllers' next untap phases.
	Register("Goblin Polka Band", func() Card {
		return NewCreature("Goblin Polka Band", "{R}{R}", 1, 1,
			WithSubTypes("Goblin"),
			WithActivatedAbility(
				FuncEffect("tap any number of random target creatures", EffectProperties{}, func(g *Game, _, _ uuid.UUID, targets []uuid.UUID) error {
					for _, targetID := range targets {
						perm := g.FindPermanent(targetID)
						if perm == nil || perm.Tapped {
							continue
						}
						isGoblin := perm.HasSubType("Goblin")
						g.TapPermanent(perm)
						if isGoblin {
							g.SkipNextUntap(perm.ID())
						}
					}
					return nil
				}),
				GenericCost(2),
				WithCost(Tap()),
				WithCost(ManaCostPerTarget("{R}")),
				WithTarget(TargetRandom(TargetUpToNCreatures(100))),
			),
		)
	})

	// Prismatic Dragon {2}{W}{W}
	// Summon Dragon
	// 2/3
	// Flying
	// During your upkeep, Prismatic Dragon becomes a random color permanently.
	// {2}: Prismatic Dragon becomes a random color permanently.
	Register("Prismatic Dragon", func() Card {
		return NewCreature("Prismatic Dragon", "{2}{W}{W}", 2, 3,
			WithSubTypes("Dragon"),
			WithKeyword(Flying),
			WithAbility(BeginningOfUpkeepTrigger(ChangeSourceToRandomColor(), false)),
			WithActivatedAbility(ChangeSourceToRandomColor(), GenericCost(2)),
		)
	})

	// Rainbow Knights {W}{W}
	// Summon Knights
	// 2/1
	// When Rainbow Knights comes into play, it gains protection from a random color permanently.
	// {1}: First strike until end of turn.
	// {W}{W}: +0/+0, +1/+0 or +2/+0 until end of turn chosen at random.
	Register("Rainbow Knights", func() Card {
		return NewCreature("Rainbow Knights", "{W}{W}", 2, 1,
			WithSubTypes("Knights"),
			WithAbility(EntersBattlefieldTrigger(FuncEffect(
				"gains protection from a random color permanently",
				EffectProperties{Outcome: OutcomeBenefit},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					return ApplyEffect(
						g,
						GrantAbility(ProtectionFromColor(g.RandomColor())).Targeting(ToSource()).Until(Indefinite),
						sourceID,
						controller,
						nil,
					)
				},
			), false)),
			WithActivatedAbility(
				GrantKeyword(FirstStrike).Targeting(ToSource()).Until(EndOfTurn),
				GenericCost(1),
			),
			WithActivatedAbility(FuncEffect(
				"gets +0/+0, +1/+0, or +2/+0 at random until end of turn",
				EffectProperties{Outcome: OutcomeBenefit},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					return ApplyEffect(
						g,
						Boost(Fixed(g.RandIntn(3)), Fixed(0)).Targeting(ToSource()).Until(EndOfTurn),
						sourceID,
						controller,
						nil,
					)
				},
			), ManaCostOf("{W}{W}")),
		)
	})
}
