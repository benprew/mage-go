package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func swapCombat(t *testing.T) (*Game, []*Permanent, []*Permanent) {
	t.Helper()
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	attackers := []*Permanent{g.PutOnBattlefield(NewCreature("First attacker", "", 2, 10), a.PlayerID()), g.PutOnBattlefield(NewCreature("Second attacker", "", 3, 10), a.PlayerID())}
	blockers := []*Permanent{g.PutOnBattlefield(NewCreature("First blocker", "", 1, 10), b.PlayerID()), g.PutOnBattlefield(NewCreature("Second blocker", "", 1, 10), b.PlayerID())}
	for i, attacker := range attackers {
		g.combat.AddAttackerIncarnation(attacker.ID(), attacker.incarnationID, b.PlayerID())
		g.combat.AddBlockerIncarnation(blockers[i].ID(), blockers[i].incarnationID, attacker.ID())
	}
	return g, attackers, blockers
}

func TestBlockAssignmentSwapRestrictions(t *testing.T) {
	for _, change := range []string{"ordinary", "tapped", "flying", "reach", "protection", "landwalk", "unblockable", "cant-block", "block-only", "menace", "two-blockers", "empty", "both-empty", "band", "capacity", "extra-capacity", "unlimited", "rule-unlimited", "limit", "duplicate", "not-blocking", "attacker-phase", "blocker-phase", "blocker-return", "control", "clone"} {
		t.Run(change, func(t *testing.T) {
			g, attackers, blockers := swapCombat(t)
			a, b := g.players[0].PlayerID(), g.players[1].PlayerID()
			subjects := BindPermanentSet("subjects")
			ctx := &EffectContext{Game: g, Controller: a}
			g.resolution.createdTokenBindings = map[*CreatedTokensBinding][]capturedPermanent{subjects: {{blockers[0].ID(), blockers[0].incarnationID}, {blockers[1].ID(), blockers[1].incarnationID}}}
			want := true
			switch change {
			case "tapped":
				blockers[0].Tapped = true
				want = false
			case "flying":
				attackers[1].GrantBaseAttr(core.Flying)
				want = false
			case "reach":
				attackers[1].GrantBaseAttr(core.Flying)
				blockers[0].GrantBaseAttr(core.Reach)
			case "protection":
				ability := ProtectionFromCardType(core.TypeCreature)
				attackers[1].Card.AddAbility(ability)
				attackers[1].RuntimeAbilities = append(attackers[1].RuntimeAbilities, ability)
				want = false
			case "landwalk":
				attackers[1].GrantBaseAttr(core.Forestwalk)
				g.PutOnBattlefield(NewLand("Forest", WithSubTypes("Forest")), b)
				want = false
			case "unblockable":
				attackers[1].GrantBaseAttr(core.UnblockableKW)
				want = false
			case "cant-block":
				g.effects.AddCanBlockOnly(blockers[0].ID(), Not(AnyPermanent))
				want = false
			case "block-only":
				g.effects.AddCanBlockOnly(blockers[0].ID(), HasKeywordFilter(core.Flying))
				want = false
			case "menace":
				attackers[1].GrantBaseAttr(core.Menace)
				want = false
			case "two-blockers":
				attackers[1].GrantBaseAttr(core.Menace)
				extra := g.PutOnBattlefield(NewCreature("Extra", "", 1, 10), b)
				g.combat.AddBlockerIncarnation(extra.ID(), extra.incarnationID, attackers[1].ID())
			case "empty":
				g.combat.RemoveFromCombat(attackers[0].ID())
			case "both-empty":
				for _, p := range attackers {
					g.combat.RemoveFromCombat(p.ID())
				}
			case "attacker-phase":
				g.PhaseOut(attackers[0].ID())
			case "band":
				g.combat.AddBand([]uuid.UUID{attackers[0].ID(), attackers[1].ID()})
				g.combat.AddBlockerIncarnation(blockers[0].ID(), blockers[0].incarnationID, attackers[1].ID())
				g.combat.AddBlockerIncarnation(blockers[1].ID(), blockers[1].incarnationID, attackers[0].ID())
				attackers[1].GrantBaseAttr(core.Flying)
			case "capacity", "extra-capacity", "unlimited", "rule-unlimited":
				extra := g.PutOnBattlefield(NewCreature("Extra attacker", "", 1, 10), a)
				g.combat.AddAttackerIncarnation(extra.ID(), extra.incarnationID, b)
				g.combat.AddBlockerIncarnation(blockers[1].ID(), blockers[1].incarnationID, extra.ID())
				switch change {
				case "capacity":
					want = false
				case "extra-capacity":
					blockers[0].GrantBaseAttr(core.CanBlockAdditional)
				case "unlimited":
					blockers[0].GrantBaseAttr(core.CanBlockAny)
				case "rule-unlimited":
					g.effects.blockAnyNumberRules = map[uuid.UUID]bool{blockers[0].ID(): true}
				}
			case "limit":
				g.PutOnBattlefield(NewEnchantment("Limit", "", WithAbility(Static(LimitBlockDeclarations(0)))), a)
			case "duplicate":
				g.resolution.createdTokenBindings[subjects] = g.resolution.createdTokenBindings[subjects][:1]
				want = false
			case "not-blocking":
				g.combat.RemoveFromCombat(blockers[0].ID())
				want = false
			case "blocker-phase":
				g.PhaseOut(blockers[0].ID())
				want = false
			case "blocker-return":
				g.RemoveFromBattlefield(blockers[0])
				g.PutOnBattlefield(blockers[0].Card, b)
				want = false
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: a, TargetID: blockers[0].ID(), ControllerID: a, Duration: core.Indefinite})
				want = false
			}
			before := cloneCombat(g.combat)
			original := g
			if change == "clone" {
				g = g.Clone()
				ctx.Game = g
			}
			if err := TrySwapBlockingAssignments(FromPermanentSet(subjects)).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if !want {
				for i, group := range before.Groups {
					if !slices.Equal(group.BlockerIDs, g.combat.Groups[i].BlockerIDs) {
						t.Fatal("failed swap changed combat")
					}
				}
			} else if change == "empty" || change == "attacker-phase" {
				if g.combat.IsBlocking(blockers[1].ID()) || !slices.Contains(g.combat.GroupFor(attackers[1].ID()).BlockerIDs, blockers[0].ID()) {
					t.Fatal("empty assignment was not swapped")
				}
			} else if change == "both-empty" {
				if g.combat.IsBlocking(blockers[0].ID()) || g.combat.IsBlocking(blockers[1].ID()) {
					t.Fatal("empty swap kept blocking status")
				}
			} else if change == "band" {
				for _, attacker := range attackers {
					if len(g.combat.GroupFor(attacker.ID()).BlockerIDs) != 2 {
						t.Fatal("swap lost band membership")
					}
				}
			} else {
				if !slices.Contains(g.combat.GroupFor(attackers[0].ID()).BlockerIDs, blockers[1].ID()) || !slices.Contains(g.combat.GroupFor(attackers[1].ID()).BlockerIDs, blockers[0].ID()) {
					t.Fatal("assignments did not swap")
				}
			}
			if change == "clone" && !slices.Equal(original.combat.Groups[0].BlockerIDs, before.Groups[0].BlockerIDs) {
				t.Fatal("clone changed original")
			}
		})
	}
}

func TestBlockAssignmentSwapEvents(t *testing.T) {
	for _, shared := range []bool{false, true} {
		t.Run(map[bool]string{false: "distinct", true: "same-attacker"}[shared], func(t *testing.T) {
			g, attackers, blockers := swapCombat(t)
			if shared {
				g.combat.RemoveFromCombat(blockers[1].ID())
				g.combat.AddBlockerIncarnation(blockers[1].ID(), blockers[1].incarnationID, attackers[0].ID())
			}
			counts := make(map[core.EventType]int)
			blocks := 0
			options := make([]CardOption, 0, 7)
			for _, kind := range []core.EventType{core.EvtDeclaredBlocker, core.EvtCreatureBlockedBy, core.EvtCombatPair, core.EvtBlockerGroup, core.EvtCreatureBecomesBlocked, core.EvtBlockersDecl, core.EvtAttacksUnblocked} {
				options = append(options, WithAbility(NewTriggered(kind, false).SetCondition(func(event *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
					counts[event.Type]++
					if event.Type == core.EvtDeclaredBlocker && event.Flag {
						blocks++
					}
					for i, blocker := range blockers {
						attacker := attackers[1-i]
						if shared {
							attacker = attackers[0]
						}
						if !slices.Contains(g.combat.GroupFor(attacker.ID()).BlockerIDs, blocker.ID()) || !g.wasBlockedByThisTurn(capturedPermanent{blocker.ID(), blocker.incarnationID}, attacker) {
							t.Fatal("observer saw partial assignment or history")
						}
					}
					return false
				})))
			}
			g.PutOnBattlefield(NewEnchantment("Watcher", "", options...), g.players[0].PlayerID())
			subjects := BindPermanentSet("pair")
			g.resolution.createdTokenBindings = map[*CreatedTokensBinding][]capturedPermanent{subjects: {{blockers[0].ID(), blockers[0].incarnationID}, {blockers[1].ID(), blockers[1].incarnationID}}}
			if err := TrySwapBlockingAssignments(FromPermanentSet(subjects)).Apply(&EffectContext{Game: g, Controller: g.players[0].PlayerID()}); err != nil {
				t.Fatal(err)
			}
			wantGroups := 4
			if shared {
				wantGroups = 3
			}
			for kind, want := range map[core.EventType]int{core.EvtDeclaredBlocker: 2, core.EvtCreatureBlockedBy: 2, core.EvtCombatPair: 4, core.EvtBlockerGroup: wantGroups, core.EvtCreatureBecomesBlocked: 0, core.EvtBlockersDecl: 0, core.EvtAttacksUnblocked: 0} {
				if counts[kind] != want {
					t.Fatalf("event %v: got %d, want %d", kind, counts[kind], want)
				}
			}
			if blocks != 2 || g.combat.BlockedAlone != uuid.Nil {
				t.Fatal("wrong new-block or declaration-only history")
			}
			for _, attacker := range attackers {
				if !g.combat.GroupFor(attacker.ID()).Blocked {
					t.Fatal("swap unblocked an attacker")
				}
			}
		})
	}
}

func TestBlockAssignmentSwapDefinitionAndWaiting(t *testing.T) {
	for _, effect := range []ResolvingEffect{
		TrySwapBlockingAssignments(nil),
		TrySwapBlockingAssignments(FromPermanentSet(BindPermanentSet("missing"))),
		DealDamageEach(This, DamageRecipients(OnePlayer(nil), MatchingPermanents(IsCreature)), Fixed(2)),
		DealDamageEach(This, DamageRecipients(OnePlayer(FromPlayer(BindPlayer("missing"))), MatchingPermanents(IsCreature)), Fixed(2)),
	} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("accepted invalid declaration")
		}
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	body := TrySwapBlockingAssignments(WithoutPermanent(MatchingPermanents(IsCreature), This))
	if err := ApplyEffect(g, AtNext(EachEndStep, body), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	saved := g.CaptureDamageSource(source.ID())
	g.RemoveFromBattlefield(source)
	g.zones.ClearLKI()
	got := g.WaitingDelayedPermanentSources()
	if len(got) != 1 || !got[0].SameObject(saved) {
		t.Fatal("waiting query lost explicit source")
	}
	playerSet := OnePlayer(ControllerOfWithLKI(This))
	if !playerSetReadsSourcePermanent(playerSet) {
		t.Fatal("singleton player set lost explicit source")
	}
	if len(OnePlayer(FromPlayer(BindPlayer("absent"))).resolvePlayers(&EffectContext{Game: g})) != 0 {
		t.Fatal("unbound singleton supplied a player")
	}
}
