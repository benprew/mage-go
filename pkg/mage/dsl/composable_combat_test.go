package dsl_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

// --- from any_attack_batch_test.go ---
func TestAnyAttackBatchPublicDeclaration(t *testing.T) {
	count := dsl.BindNumber("declared attackers")
	pattern := dsl.OneOrMoreCreaturesAttack().WithCount(count)
	card := dsl.NewEnchantment("Attack observer", "", dsl.WithAbility(dsl.WhenEvent(
		pattern, dsl.GainLife(dsl.You, dsl.FromNumber(count)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if info := dsl.InspectDefinition(pattern); info.Role != "event-pattern" || len(info.Children) != 1 {
		t.Fatalf("inspection = %+v", info)
	}
}

// --- from blocked_attackers_set_test.go ---
func TestBlockedAttackersOfPublicDeclaration(t *testing.T) {
	blocker := dsl.TargetPermanent("blocker", dsl.IsCreature)
	card := dsl.NewInstant("Count blocked attackers", "", dsl.WithAbility(dsl.Spell(
		dsl.GainLife(dsl.You, dsl.CountPermanents(dsl.BlockedAttackersOf(blocker))),
		dsl.WithTargets(blocker))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from blocked_defender_test.go ---
func TestBecomesBlockedDefenderPublicDeclaration(t *testing.T) {
	defender := dsl.BindPlayer("defending player")
	card := dsl.NewCreature("Synthetic blocker watcher", "", 2, 2, dsl.WithAbility(dsl.WhenEvent(
		dsl.BecomesBlocked(dsl.This).WithDefendingPlayer(defender),
		dsl.Discard(dsl.FromPlayer(defender), dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from blockers_set_test.go ---
func TestBlockersOfPublicDeclaration(t *testing.T) {
	attacker := dsl.TargetPermanent("attacker", dsl.IsCreature)
	card := dsl.NewInstant("Count blockers", "", dsl.WithAbility(dsl.Spell(
		dsl.GainLife(dsl.You, dsl.CountPermanents(dsl.BlockersOf(attacker))),
		dsl.WithTargets(attacker))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from combat_boost_migration_test.go ---
func TestCombatBoostTypedDefinitions(t *testing.T) {
	for _, tc := range []struct {
		name             string
		power, toughness int
	}{{"Army of Allah", 2, 0}, {"Piety", 0, 3}, {"Morale", 1, 1}} {
		t.Run(tc.name, func(t *testing.T) {
			card, err := mage.CreateCard(tc.name)
			if err != nil {
				t.Fatal(err)
			}
			if err := mage.ValidateCardDefinition(tc.name, card); err != nil {
				t.Fatal(err)
			}
			ability := mage.UnwrapAbility(card.Abilities()[0]).(*mage.SpellAbility)
			if len(ability.Targets()) != 0 {
				t.Fatal("combat collection must not target")
			}
			metadata := mage.InspectDefinition(ability.Effects()[0])
			if metadata.Role != "until" {
				t.Fatalf("effect role = %q, want until", metadata.Role)
			}
			props := metadata.Properties
			if !props.Mass || props.Outcome != mage.OutcomeBenefit || props.PowerBoost != tc.power || props.ToughnessBoost != tc.toughness {
				t.Fatalf("combat boost metadata = %+v", props)
			}
		})
	}
}

func TestCombatBoostResolutionMembership(t *testing.T) {
	for _, tc := range []struct {
		name             string
		power, toughness int
		blocking         bool
	}{{"Army of Allah", 2, 0, false}, {"Piety", 0, 3, true}, {"Morale", 1, 1, false}} {
		for _, opponentCasts := range []bool{false, true} {
			label := tc.name + "/combatant casts"
			if opponentCasts {
				label = tc.name + "/opponent casts"
			}
			t.Run(label, func(t *testing.T) {
				a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
				g := mage.NewGame(a, b)
				caster := a.PlayerID()
				if opponentCasts {
					caster = b.PlayerID()
				}
				card, err := mage.CreateCard(tc.name)
				if err != nil {
					t.Fatal(err)
				}
				card.SetOwner(caster)
				makeCreature := func(name string) *mage.Permanent {
					c := mage.NewCreature(name, "", 2, 2)
					c.SetOwner(a.PlayerID())
					return g.PutOnBattlefield(c, a.PlayerID())
				}
				joinCombat := func(p *mage.Permanent) {
					if tc.blocking {
						attacker := uuid.New()
						g.GetCombat().AddAttacker(attacker, a.PlayerID())
						g.GetCombat().AddBlocker(p.ID(), attacker)
					} else {
						g.GetCombat().AddAttacker(p.ID(), b.PlayerID())
					}
				}
				member := makeCreature("Member")
				removed := makeCreature("Removed before resolution")
				later := makeCreature("Joins after resolution")
				joinCombat(member)
				joinCombat(removed)
				ability := mage.UnwrapAbility(card.Abilities()[0]).(*mage.SpellAbility)
				g.PushStack(&mage.StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: caster, Effects: ability.Effects()})
				g.RemoveFromCombat(removed.ID())
				g.ResolveStack()
				joinCombat(later)
				g.RemoveFromCombat(member.ID())
				check := func(game *mage.Game, id uuid.UUID, power, toughness int) {
					t.Helper()
					game.ApplyContinuousEffects()
					p := game.FindPermanent(id)
					if p.CurrentPower(game) != power || p.CurrentToughness(game) != toughness {
						t.Fatalf("%s P/T = %d/%d, want %d/%d", p.Card.Name(), p.CurrentPower(game), p.CurrentToughness(game), power, toughness)
					}
				}
				check(g, member.ID(), 2+tc.power, 2+tc.toughness)
				check(g, removed.ID(), 2, 2)
				check(g, later.ID(), 2, 2)
				clone := g.Clone()
				clone.GetEffects().RemoveEndOfTurn()
				check(clone, member.ID(), 2, 2)
				check(g, member.ID(), 2+tc.power, 2+tc.toughness)
				g.RemoveFromBattlefield(g.FindPermanent(member.ID()))
				g.PutOnBattlefield(member.Card, a.PlayerID())
				check(g, member.ID(), 2, 2)
			})
		}
	}
}

// --- from combat_history_test.go ---
func TestCombatHistoryPublicDeclaration(t *testing.T) {
	card := dsl.NewCreature("Combat History Watcher", "{2}", 2, 2,
		dsl.WithAbility(dsl.At(dsl.EachEndOfCombat,
			dsl.IfState(dsl.SourceAttackedOrBlockedThisCombatCondition(),
				dsl.GainLife(dsl.You, dsl.Fixed(1))))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from combat_removal_test.go ---
func TestCombatRemovalComposition(t *testing.T) {
	saved := dsl.BindPermanent("subject")
	target := dsl.TargetPermanent("subject", dsl.IsCreature)
	body := dsl.Sequence(dsl.Snapshot(dsl.This, saved), dsl.RemoveFromCombat(dsl.From(saved)))
	for _, card := range []mage.Card{
		dsl.NewCreature("State", "", 1, 1, dsl.WithAbility(dsl.WhenState(dsl.SourceHasCounters(dsl.Tide, 4), body))),
		dsl.NewCreature("Activate", "", 1, 1, dsl.WithAbility(dsl.Activate(dsl.Costs(), body))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.At(dsl.YourUpkeep, body))),
		dsl.NewSorcery("Target", "", dsl.WithAbility(dsl.Spell(dsl.RemoveFromCombat(target), dsl.WithTargets(target)))),
		dsl.NewSorcery("Loop", "", dsl.WithAbility(dsl.Spell(dsl.ForEach(dsl.MatchingPermanents(dsl.IsCreature), saved, dsl.RemoveFromCombat(dsl.From(saved)))))),
		dsl.NewSorcery("Mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Remove", dsl.RemoveFromCombat(target), dsl.WithTargets(target)), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}

// --- from combat_source_prevention_test.go ---
func TestCombatSourcePreventionPublicDeclaration(t *testing.T) {
	blocker := dsl.TargetPermanent("creature", dsl.IsCreature)
	card := dsl.NewInstant("Combat prevention", "", dsl.WithAbility(dsl.Spell(
		dsl.IfPermanentBlocking(blocker, dsl.Sequence(
			dsl.PreventCombatDamageBy(blocker),
			dsl.PreventCombatDamageByEach(dsl.BlockedAttackersOf(blocker)))),
		dsl.WithTargets(blocker))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from combat_state_branch_test.go ---
func TestCombatStateBranchPublicDefinition(t *testing.T) {
	target := dsl.TargetPermanent("creature", dsl.IsCreature)
	card := dsl.NewInstant("Combat check", "", dsl.WithAbility(dsl.Spell(dsl.Sequence(
		dsl.IfPermanentAttacking(target, dsl.GainLife(dsl.You, dsl.Fixed(1))),
		dsl.IfPermanentBlocking(target, dsl.GainLife(dsl.You, dsl.Fixed(1))),
	), dsl.WithTargets(target))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from composable_combat_damage_batch_test.go ---
func TestCombatDamageBatchAuthorSurface(t *testing.T) {
	player := dsl.BindPlayer("damaged player")
	var card dsl.Card = dsl.NewEnchantment("Combat Batch Watcher", "{G}",
		dsl.WithAbility(dsl.WhenEvent(
			dsl.OneOrMoreCreaturesYouControlDealCombatDamageToPlayer().WithPlayer(player),
			dsl.GainLife(dsl.FromPlayer(player), dsl.Fixed(1)),
		)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestCombatDamageBatchSubtypeAuthorSurface(t *testing.T) {
	var card dsl.Card = dsl.NewEnchantment("Non-Human Watcher", "{G}",
		dsl.WithAbility(dsl.WhenEvent(
			dsl.OneOrMoreCreaturesYouControlDealCombatDamageToPlayer().WithoutSubtype("Human"),
			dsl.Draw(dsl.You, dsl.Fixed(1)),
		)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from composable_combat_damage_event_test.go ---
func TestCombatDamageEventAuthorSurface(t *testing.T) {
	player := dsl.BindPlayer("damaged player")
	amount := dsl.BindNumber("combat damage")
	var card dsl.Card = dsl.NewCreature("Combat Watcher", "{2}{G}", 2, 2,
		dsl.WithAbility(dsl.WhenEvent(
			dsl.DealsCombatDamageToPlayer(dsl.This, player).WithAmount(amount),
			dsl.GainLife(dsl.FromPlayer(player), dsl.FromNumber(amount)),
		)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from composable_combat_opponents_test.go ---
func TestCombatOpponentsAuthorSurface(t *testing.T) {
	var card dsl.Card = dsl.NewCreature("Combat Watcher", "{1}{G}", 2, 4,
		dsl.WithAbility(dsl.WhenEvent(dsl.BlocksOrBecomesBlockedBy(dsl.This),
			dsl.Until(dsl.EndOfTurn,
				dsl.GrantEachKeyword(dsl.BlockingOrBlockedBy(dsl.This), dsl.FirstStrike)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestBlockerGroupWithoutSubtypeAuthorSurface(t *testing.T) {
	var card dsl.Card = dsl.NewCreature("Group watcher", "{1}{G}", 2, 4,
		dsl.WithAbility(dsl.WhenEvent(
			dsl.BlocksOrBecomesBlockedBy(dsl.This).WithoutSubtype("Wall"),
			dsl.GainLife(dsl.You, dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestCombatPairAuthorSurface(t *testing.T) {
	opponent := dsl.BindPermanent("opposing creature")
	var card dsl.Card = dsl.NewCreature("Pair watcher", "{1}{G}", 2, 4,
		dsl.WithAbility(dsl.WhenEvent(
			dsl.BlocksOrBecomesBlockedByCreature(dsl.This).WithoutSubtype("Wall").WithOpponent(opponent),
			dsl.Tap(dsl.From(opponent)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from defending_control_attack_test.go ---
func TestDefendingPlayerControlAttackPublicDeclaration(t *testing.T) {
	card := dsl.NewCreature("Attacker", "", 2, 2,
		dsl.WithAbility(dsl.Static(dsl.CanAttackOnlyIfDefendingPlayerControls(dsl.And(dsl.IsLand, dsl.HasSubType("Island"))))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from permanent_attacking_test.go ---
func TestPermanentAttackingPublicDefinition(t *testing.T) {
	card := dsl.NewCreature("Attacking guardian", "", 2, 2,
		dsl.WithAbility(dsl.Static(dsl.While(
			dsl.PermanentIsAttacking(dsl.This),
			dsl.ModifyPT(dsl.This, dsl.Fixed(0), dsl.Fixed(2)),
		))),
	)
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from permanent_blocking_test.go ---
func TestPermanentBlockingPublicDefinition(t *testing.T) {
	card := dsl.NewAura("Blocking cloak", "",
		dsl.WithAbility(dsl.Static(dsl.While(
			dsl.PermanentIsBlocking(dsl.AttachedPermanent),
			dsl.GrantKeyword(dsl.AttachedPermanent, dsl.Shroud),
		))),
	)
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
