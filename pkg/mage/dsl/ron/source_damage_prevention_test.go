package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const sourceDamagePrevention = `CreatePreventDamageUntil(DamageWouldBeDealtByPermanent(ThisPermanent),[PreventThatDamage],UntilEndOfTurn)`

func TestRONSourceDamagePreventionKryShield(t *testing.T) {
	for _, state := range []string{"normal", "unpaid", "opponent", "noncreature", "shroud-before", "shroud-pending", "control-pending", "phase-pending", "return-pending", "shield-leaves", "shield-phase", "shield-return", "shield-control", "control-after", "shroud-after", "phase-after", "return-after"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			shieldCard := repositoryCard(t, "LEG", "Kry Shield")
			shieldCard.SetOwner(a.PlayerID())
			a.AddToHand(shieldCard)
			if err := g.CastSpellByID(a.PlayerID(), shieldCard.ID(), nil, 0); err == nil {
				t.Fatal("unpaid cast succeeded")
			}
			a.ManaPool().Add(core.Colorless, 2)
			if err := g.CastSpellByID(a.PlayerID(), shieldCard.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			shield := g.FindPermanent(shieldCard.ID())
			card := mage.NewCreature("Source", "{3}", 2, 20)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 1, 20), b.PlayerID())
			target := p.ID()
			switch state {
			case "opponent":
				target = victim.ID()
			case "noncreature":
				target = shield.ID()
			case "shroud-before":
				p.GrantBaseAttr(core.Shroud)
			}
			if state != "unpaid" {
				a.ManaPool().Add(core.Colorless, 2)
			}
			err := g.ActivateAbilityByIndex(a.PlayerID(), shield.ID(), 0, []uuid.UUID{target})
			if state == "unpaid" || state == "opponent" || state == "noncreature" || state == "shroud-before" {
				if err == nil || g.StackSize() != 0 || shield.Tapped {
					t.Fatal("illegal activation paid a cost")
				}
				if state != "unpaid" && a.ManaPool().TotalMana() != 2 {
					t.Fatal("failed activation spent mana")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !g.FindPermanent(shield.ID()).Tapped || a.ManaPool().TotalMana() != 0 {
				t.Fatal("activation costs not paid")
			}
			saved := g.CaptureDamageSource(p.ID())
			g.DealDamageToPlayerFromSource(b, 1, saved)
			if b.Life() != 19 || p.CurrentToughness(g) != 20 {
				t.Fatal("effect started before resolution")
			}
			active := true
			switch state {
			case "shroud-pending":
				g.FindPermanent(p.ID()).GrantBaseAttr(core.Shroud)
				active = false
			case "control-pending":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				active = false
			case "phase-pending":
				g.PhaseOut(p.ID())
				active = false
			case "return-pending":
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
				g.PutOnBattlefield(card, a.PlayerID())
				active = false
			case "shield-leaves", "shield-return":
				g.RemoveFromBattlefield(g.FindPermanent(shield.ID()))
				if state == "shield-return" {
					g.PutOnBattlefield(shieldCard, b.PlayerID())
				}
			case "shield-phase":
				g.PhaseOut(shield.ID())
			case "shield-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: shield.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			pending := g.Clone()
			g.ResolveStack()
			if state == "phase-pending" {
				g.PhaseIn([]uuid.UUID{p.ID()})
			}
			wantT := 20
			if active {
				wantT = 23
			}
			if g.FindPermanent(p.ID()).CurrentToughness(g) != wantT {
				t.Fatal("wrong toughness or illegal target resolved")
			}
			switch state {
			case "control-after":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "shroud-after":
				g.FindPermanent(p.ID()).GrantBaseAttr(core.Shroud)
			case "phase-after":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "return-after":
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
				g.PutOnBattlefield(card, a.PlayerID())
			}
			source := g.CaptureDamageSource(p.ID())
			for _, game := range []*mage.Game{g, g.Clone()} {
				game.DealSimultaneousDamage(mage.PlayerDamageAssignment(game.GetPlayer(b.PlayerID()), 2, saved), mage.PermanentDamageAssignment(game.FindPermanent(victim.ID()), 3, saved))
				wantLife, wantDamage := 17, 3
				if active {
					wantLife, wantDamage = 19, 0
				}
				if game.GetPlayer(b.PlayerID()).Life() != wantLife || game.FindPermanent(victim.ID()).Damage != wantDamage {
					t.Fatal("wrong exact-source batch prevention")
				}
				game.DealDamageToPlayerFromSource(game.GetPlayer(b.PlayerID()), 1, source)
				if !active || state == "return-after" {
					wantLife--
				}
				if game.GetPlayer(b.PlayerID()).Life() != wantLife {
					t.Fatal("prevention followed a returned visit or was consumed")
				}
				game.DealDamageToPlayer(game.GetPlayer(b.PlayerID()), 1, victim.ID())
				if game.GetPlayer(b.PlayerID()).Life() != wantLife-1 {
					t.Fatal("other source was prevented")
				}
			}
			if pending.GetPlayer(b.PlayerID()).Life() != 19 || pending.StackSize() != 1 {
				t.Fatal("pending clone changed")
			}
			pending.ResolveStack()
			g.GetEffects().ClearReplacementsEndOfTurn()
			before := b.Life()
			g.DealDamageToPlayerFromSource(b, 1, saved)
			if b.Life() != before-1 {
				t.Fatal("prevention survived cleanup")
			}
		})
	}
}

func TestRONSourceDamagePreventionInvalid(t *testing.T) {
	for _, action := range []string{
		strings.Replace(sourceDamagePrevention, "DamageWouldBeDealtByPermanent(ThisPermanent)", "DamageWouldBeDealtByPermanent", 1),
		strings.Replace(sourceDamagePrevention, "ThisPermanent", "Ref_TargetPermanent", 1),
		strings.Replace(sourceDamagePrevention, "ThisPermanent", "You", 1),
		strings.Replace(sourceDamagePrevention, "ThisPermanent", "ThisPermanent()", 1),
		strings.Replace(sourceDamagePrevention, "[PreventThatDamage]", "[]", 1),
		strings.Replace(sourceDamagePrevention, "[PreventThatDamage]", "[PreventThatDamage()]", 1),
		strings.Replace(sourceDamagePrevention, "[PreventThatDamage]", "[PreventThatDamage,PreventThatDamage]", 1),
		strings.Replace(sourceDamagePrevention, "UntilEndOfTurn", "UntilEndOfCombat", 1),
		strings.Replace(sourceDamagePrevention, "UntilEndOfTurn", "UntilEndOfTurn()", 1),
	} {
		n := nameScopeNode(t, action)
		if _, err := compileEffect(n, scope{permanent: true, resolving: true}); err == nil {
			t.Fatal("accepted", action)
		}
	}
	for _, ctx := range []scope{{permanent: true}, {permanent: true, resolving: true, enteringSource: true}, {permanent: true, resolving: true, drawReplacementBody: true}} {
		if _, err := compileEffect(nameScopeNode(t, sourceDamagePrevention), ctx); err == nil {
			t.Fatal("accepted invalid context")
		}
	}
}

func TestRONSourceDamagePreventionInstances(t *testing.T) {
	g, a, b := ronGame()
	firstCard := repositoryCard(t, "LEG", "Kry Shield")
	secondCard := repositoryCard(t, "LEG", "Kry Shield")
	first := g.PutOnBattlefield(firstCard, a.PlayerID())
	second := g.PutOnBattlefield(secondCard, b.PlayerID())
	x := g.PutOnBattlefield(mage.NewCreature("X", "{1}", 1, 20), a.PlayerID())
	y := g.PutOnBattlefield(mage.NewCreature("Y", "{4}", 2, 20), b.PlayerID())
	a.ManaPool().Add(core.Colorless, 2)
	b.ManaPool().Add(core.Colorless, 2)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), first.ID(), 0, []uuid.UUID{x.ID()}); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), second.ID(), 0, []uuid.UUID{y.ID()}); err != nil {
		t.Fatal(err)
	}
	pending := g.Clone()
	for _, game := range []*mage.Game{g, pending} {
		game.ResolveStack()
		if game.FindPermanent(x.ID()).CurrentToughness(game) != 21 || game.FindPermanent(y.ID()).CurrentToughness(game) != 24 {
			t.Fatal("factories shared target or value bindings")
		}
		game.DealDamageToPlayer(game.GetPlayer(a.PlayerID()), 2, y.ID())
		game.DealDamageToPlayer(game.GetPlayer(b.PlayerID()), 3, x.ID())
		if game.GetPlayer(a.PlayerID()).Life() != 20 || game.GetPlayer(b.PlayerID()).Life() != 20 || len(game.WaitingPreventionPermanentSources()) != 2 {
			t.Fatal("instances lost independent sources")
		}
	}
	g.RunStepWithPriority(core.Cleanup)
	g.DealDamageToPlayer(g.GetPlayer(b.PlayerID()), 1, x.ID())
	if g.GetPlayer(b.PlayerID()).Life() != 19 || g.FindPermanent(x.ID()).CurrentToughness(g) != 20 || pending.FindPermanent(x.ID()).CurrentToughness(pending) != 21 {
		t.Fatal("cleanup did not expire both effects independently")
	}
}

func TestRONSourceDamagePreventionDelayed(t *testing.T) {
	g, a, b := ronGame()
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+sourceDamagePrevention+`]))]))`)
	card := factory()
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	saved := g.CaptureDamageSource(p.ID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	a.RemoveFromHand(card.ID())
	g.PutOnBattlefield(card, b.PlayerID())
	if len(g.WaitingDelayedPermanentSources()) != 1 {
		t.Fatal("waiting body lost source declaration")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.ResolveStack()
	g.DealDamageToPlayerFromSource(b, 2, saved)
	if b.Life() != 20 {
		t.Fatal("RON delayed prevention lost LKI")
	}
	g.DealDamageToPlayer(a, 1, p.ID())
	if a.Life() != 19 {
		t.Fatal("RON delayed prevention followed returned source")
	}
}

func TestRONSourceDamagePreventionInventory(t *testing.T) {
	nodes, err := Parse(strings.NewReader(sourceDamagePrevention), "source.ron")
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string]*InventorySignature{}
	walkInventory(nodes[0], "Test", "Action", "action", "resolving", true, entries)
	found := false
	for _, entry := range entries {
		if entry.Name == "ThisPermanent" {
			found = true
			if entry.Role != "permanent_reference" {
				t.Fatalf("role=%s", entry.Role)
			}
		}
	}
	if !found {
		t.Fatal("missing source reference")
	}
}

func TestRONSourceDamagePreventionCopy(t *testing.T) {
	g, a, b := ronGame()
	shield := g.PutOnBattlefield(repositoryCard(t, "LEG", "Kry Shield"), a.PlayerID())
	x := g.PutOnBattlefield(mage.NewCreature("X", "{1}", 1, 20), a.PlayerID())
	y := g.PutOnBattlefield(mage.NewCreature("Y", "{4}", 2, 20), b.PlayerID())
	a.ManaPool().Add(core.Colorless, 2)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), shield.ID(), 0, []uuid.UUID{x.ID()}); err != nil {
		t.Fatal(err)
	}
	phased := g.PhaseOut(x.ID())
	copyObject := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), true)
	if copyObject == nil || len(copyObject.Targets) != 1 || copyObject.Targets[0] != y.ID() {
		t.Fatal("copy lost separate controller or target")
	}
	g.PhaseIn(phased)
	g.ResolveStack()
	if g.FindPermanent(x.ID()).CurrentToughness(g) != 21 || g.FindPermanent(y.ID()).CurrentToughness(g) != 24 {
		t.Fatal("copy crossed value or target bindings")
	}
	g.DealDamageToPlayer(a, 2, y.ID())
	g.DealDamageToPlayer(b, 3, x.ID())
	if a.Life() != 20 || b.Life() != 20 || len(g.WaitingPreventionPermanentSources()) != 2 {
		t.Fatal("copy crossed prevention identities")
	}
}
