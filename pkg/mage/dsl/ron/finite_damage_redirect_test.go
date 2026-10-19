package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const finiteRedirectRON = `CreateFutureReplaceWouldDealDamage(NextAmountOfDamageThatWouldBeDealtThisTurnToRecipient(Integer(1),Permanent(ThisPermanent)),[DealToPlayerInstead(OwnerOfPermanent(ThisPermanent))])`

func TestRONFiniteDamageRedirect(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			record := staticBaseRecord(t, set, "Personal Incarnation")
			rules := record.Fields["Rules"]
			rules.Args = rules.Args[:1]
			record.Fields["Rules"] = rules
			factory, err := Compile(record)
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			card := factory()
			card.SetOwner(a.PlayerID())
			subject := g.PutOnBattlefield(card, b.PlayerID())
			source := g.PutOnBattlefield(mage.NewCreature("Damage source", "", 1, 20), b.PlayerID())
			if err := g.ActivateAbilityByIndex(b.PlayerID(), subject.ID(), 0, nil); err == nil {
				t.Fatal("controller activated owner ability")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), subject.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.DealDamageToPermanent(subject, 1, source.ID())
			if a.Life() != 20 || subject.Damage != 1 {
				t.Fatal("rule installed before resolution")
			}
			g.ResolveStack()
			baseline := g.Clone()
			for _, game := range []*mage.Game{g, g.Clone()} {
				game.DealDamageToPermanent(game.FindPermanent(subject.ID()), 3, source.ID())
				if game.GetPlayer(a.PlayerID()).Life() != 19 || game.GetPlayer(b.PlayerID()).Life() != 20 || game.FindPermanent(subject.ID()).Damage != 3 {
					t.Fatal("split lost saved owner or budget")
				}
				game.DealDamageToPermanent(game.FindPermanent(subject.ID()), 1, source.ID())
				if game.FindPermanent(subject.ID()).Damage != 4 {
					t.Fatal("budget reused")
				}
			}
			if baseline.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("clone changed")
			}
			if _, err := Compile(staticBaseRecord(t, set, "Personal Incarnation")); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRONFiniteDamageRedirectInvalid(t *testing.T) {
	for _, src := range []string{
		strings.Replace(finiteRedirectRON, "Integer(1)", "Integer(0)", 1),
		strings.Replace(finiteRedirectRON, "Integer(1)", "Integer(-1)", 1),
		strings.Replace(finiteRedirectRON, "Integer(1)", "ValueX", 1),
		strings.Replace(finiteRedirectRON, "Permanent(ThisPermanent)", "Player(You)", 1),
		strings.Replace(finiteRedirectRON, "Permanent(ThisPermanent)", "Permanent(Ref_TargetPermanent)", 1),
		strings.Replace(finiteRedirectRON, "[DealToPlayerInstead(OwnerOfPermanent(ThisPermanent))]", "[]", 1),
		strings.Replace(finiteRedirectRON, "OwnerOfPermanent(ThisPermanent)", "Trigger_ThatPlayer", 1),
		strings.Replace(finiteRedirectRON, "Integer(1),Permanent(ThisPermanent)", "Integer(1)", 1),
		strings.Replace(finiteRedirectRON, "DealToPlayerInstead(OwnerOfPermanent(ThisPermanent))", "DealToPlayerInstead(You,You)", 1),
	} {
		nodes, err := Parse(strings.NewReader(src), "invalid-finite.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := compileEffect(nodes[0], scope{permanent: true, resolving: true}); err == nil {
			t.Fatal("invalid declaration accepted", src)
		}
	}
	nodes, err := Parse(strings.NewReader(finiteRedirectRON), "finite.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, ctx := range []scope{{permanent: true}, {permanent: true, resolving: true, enteringSource: true}, {permanent: true, resolving: true, drawReplacementBody: true}} {
		if _, err := compileEffect(nodes[0], ctx); err == nil {
			t.Fatal("invalid context accepted")
		}
	}
}

func TestRONFiniteDamageRedirectDelayed(t *testing.T) {
	for _, state := range []string{"live", "departed", "returned", "phase-cycle"} {
		t.Run(state, func(t *testing.T) {
			factory := ronDelayedCombatFactory(t, `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+finiteRedirectRON+`]))]))`)
			g, a, b := ronGame()
			card := factory()
			card.SetOwner(a.PlayerID())
			subject := g.PutOnBattlefield(card, b.PlayerID())
			source := g.PutOnBattlefield(mage.NewCreature("Damage source", "", 1, 20), b.PlayerID())
			saved := g.CaptureDamageSource(subject.ID())
			if err := g.ActivateAbilityByIndex(b.PlayerID(), subject.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			switch state {
			case "departed", "returned":
				g.BouncePermanentToHand(subject)
				if state == "returned" {
					a.RemoveFromHand(card.ID())
					subject = g.PutOnBattlefield(card, b.PlayerID())
				}
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(subject.ID()))
			}
			for range 2 {
				refs := g.WaitingDelayedPermanentSources()
				if len(refs) != 1 || !refs[0].SameObject(saved) || a.Life() != 20 {
					t.Fatal("delayed waiting read lost source or changed state")
				}
			}
			baseline := g.Clone()
			for _, game := range []*mage.Game{g, g.Clone()} {
				game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				game.PutTriggersOnStack()
				game.ResolveStack()
				if p := game.FindPermanent(subject.ID()); p != nil {
					game.DealDamageToPermanent(p, 2, source.ID())
				}
				want := 19
				if state == "departed" || state == "returned" {
					want = 20
				}
				if game.GetPlayer(a.PlayerID()).Life() != want {
					t.Fatal("delayed rule followed returned source or lost owner")
				}
			}
			if baseline.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("clone changed")
			}
		})
	}
}

func TestRONFiniteDamageRedirectTarget(t *testing.T) {
	factory := ronDelayedCombatFactory(t, `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreateFutureReplaceWouldDealDamage(NextAmountOfDamageThatWouldBeDealtThisTurnToRecipient(Integer(2),Permanent(Ref_TargetPermanent)),[DealToPlayerInstead(OwnerOfPermanent(Ref_TargetPermanent))])])))`)
	g, a, b := ronGame()
	card := factory()
	card.SetOwner(b.PlayerID())
	source := g.PutOnBattlefield(card, b.PlayerID())
	targetCard := mage.NewCreature("Target", "", 1, 20)
	targetCard.SetOwner(a.PlayerID())
	target := g.PutOnBattlefield(targetCard, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(source)
	g.ResolveStack()
	g.DealDamageToPermanent(target, 3, target.ID())
	if a.Life() != 18 || b.Life() != 20 || target.Damage != 1 {
		t.Fatal("target binding used source or its controller")
	}
}

func TestRONFiniteDamageRedirectCopy(t *testing.T) {
	record := staticBaseRecord(t, "LEA", "Personal Incarnation")
	rules := record.Fields["Rules"]
	rules.Args = rules.Args[:1]
	record.Fields["Rules"] = rules
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	card := factory()
	card.SetOwner(a.PlayerID())
	subject := g.PutOnBattlefield(card, b.PlayerID())
	source := g.PutOnBattlefield(mage.NewCreature("Damage source", "", 1, 20), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), subject.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("ability copy failed")
	}
	baseline := g.Clone()
	for _, game := range []*mage.Game{g, g.Clone()} {
		game.ResolveStack()
		game.DealDamageToPermanent(game.FindPermanent(subject.ID()), 3, source.ID())
		if game.GetPlayer(a.PlayerID()).Life() != 18 || game.GetPlayer(b.PlayerID()).Life() != 20 || game.FindPermanent(subject.ID()).Damage != 1 {
			t.Fatal("copy changed owner or shared installation budget")
		}
	}
	if baseline.GetPlayer(a.PlayerID()).Life() != 20 || baseline.StackSize() != 2 {
		t.Fatal("pending clone changed")
	}
}
