package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONDestructionReplacementClergy(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEG", "Clergy of the Holy Nimbus")
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	g.DestroyPermanent(p)
	if g.FindPermanent(p.ID()) == nil || !g.FindPermanent(p.ID()).Tapped {
		t.Fatal("missing static regeneration")
	}
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, nil); err == nil {
		t.Fatal("controller activated opponent ability")
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 1, nil); err == nil {
		t.Fatal("unpaid activation")
	}
	b.ManaPool().Add(core.Colorless, 1)
	if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 1, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.DestroyPermanent(g.FindPermanent(p.ID()))
	if g.FindPermanent(p.ID()) != nil {
		t.Fatal("regenerated despite prohibition")
	}
}

func TestRONDestructionReplacementPyramids(t *testing.T) {
	for _, change := range []string{"none", "return", "phase", "source-exit", "copy", "clone"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronModalGame(0)
			sourceCard := repositoryCard(t, "ARN", "Pyramids")
			sourceCard.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(sourceCard, a.PlayerID())
			landCard := repositoryCard(t, "LEB", "Island")
			landCard.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(landCard, b.PlayerID())
			p.Damage = 2
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{p.ID()}); err == nil {
				t.Fatal("unpaid activation")
			}
			a.ManaPool().Add(core.Colorless, 2)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{p.ID()}); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "return":
				g.RemoveFromBattlefield(p)
				p = g.PutOnBattlefield(p.Card, b.PlayerID())
			case "phase":
				g.PhaseOut(p.ID())
			case "source-exit":
				g.RemoveFromBattlefield(source)
			case "copy":
				g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false)
			}
			clone := g.Clone()
			clone.ResolveStack()
			g.ResolveStack()
			if change == "phase" {
				g.PhaseIn([]uuid.UUID{p.ID()})
				p = g.FindPermanent(p.ID())
			}
			g.DestroyPermanent(p)
			if change == "return" || change == "phase" {
				if g.FindPermanent(p.ID()) != nil {
					t.Fatal("illegal target installed shield")
				}
				return
			}
			if g.FindPermanent(p.ID()) == nil || g.FindPermanent(p.ID()).Damage != 0 || g.FindPermanent(p.ID()).Tapped {
				t.Fatal("wrong destruction replacement")
			}
			if change == "clone" {
				clone.DestroyPermanent(clone.FindPermanent(p.ID()))
				if clone.FindPermanent(p.ID()) == nil {
					t.Fatal("clone lost its shield")
				}
			}
			g.DestroyPermanent(g.FindPermanent(p.ID()))
			if change == "copy" {
				if g.FindPermanent(p.ID()) == nil {
					t.Fatal("copy did not install separate shield")
				}
				g.DestroyPermanent(g.FindPermanent(p.ID()))
			}
			if g.FindPermanent(p.ID()) != nil {
				t.Fatal("shield was not consumed")
			}
		})
	}
}

func TestRONDestructionReplacementInvalid(t *testing.T) {
	for _, rule := range []string{
		`ReplaceWouldDestroy(PermanentWouldBeDestroyed(HostPermanent),[RegeneratePermanent(WouldBeDestroyed_ThatPermanent)])`,
		`ReplaceWouldDestroy(PermanentWouldBeDestroyed(ThisPermanent),[RegeneratePermanent(ThisPermanent)])`,
		`ReplaceWouldDestroy(PermanentWouldBeDestroyed(ThisPermanent),[])`,
		`ReplaceWouldDestroy(PermanentWouldBeDestroyed(ThisPermanent),[CancelDestroy])`,
		`Activated(PayMana([]),ActionList([CreateFutureReplaceWouldDestroy(NextTimePermanentWouldBeDestroyedThisTurn(ThisPermanent),[CancelDestroy])]))`,
		`Activated(PayMana([]),ActionList([CreateFutureReplaceWouldDestroy(NextTimePermanentWouldBeDestroyedThisTurn(ThisPermanent),[CancelDestroy(),RemoveAllDamageFromPermanent(WouldBeDestroyed_ThatPermanent)])]))`,
		`Activated(PayMana([]),ActionList([RemoveAllDamageFromPermanent(WouldBeDestroyed_ThatPermanent)]))`,
	} {
		assertConditionalRejected(t, rule)
	}
}

func TestRONDestructionReplacementAuraMode(t *testing.T) {
	for _, change := range []string{"none", "reattach", "shroud", "phase"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronModalGame(0)
			card := repositoryCard(t, "ARN", "Pyramids")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			land := repositoryCard(t, "LEB", "Island")
			land.SetOwner(b.PlayerID())
			host := g.PutOnBattlefield(land, b.PlayerID())
			auraCard := repositoryCard(t, "LEB", "Wild Growth")
			auraCard.SetOwner(b.PlayerID())
			aura := g.PutOnBattlefield(auraCard, b.PlayerID())
			g.Attach(aura.ID(), host.ID())
			a.ManaPool().Add(core.Colorless, 2)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{aura.ID()}); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "reattach":
				aura.AttachedTo = uuid.Nil
			case "shroud":
				aura.GrantBaseAttr(core.Shroud)
			case "phase":
				g.PhaseOut(aura.ID())
			}
			g.ResolveStack()
			if (g.FindPermanentIncludingPhased(aura.ID()) == nil) != (change == "none") {
				t.Fatal("Aura target restriction did not recheck")
			}
		})
	}
}

func TestRONDestructionReplacementCasting(t *testing.T) {
	for name, set := range map[string]string{"Pyramids": "ARN", "Clergy of the Holy Nimbus": "LEG"} {
		t.Run(name, func(t *testing.T) {
			g, a, _ := ronGame()
			g.SetActivePlayerIndex(0)
			g.SetStep(core.PrecombatMain)
			card := repositoryCard(t, set, name)
			rebuildRONDefinition(t, card)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid casting succeeded")
			}
			if len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("failed casting changed zones")
			}
			a.ManaPool().Add(core.White, 1)
			a.ManaPool().Add(core.Colorless, 6)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if g.FindPermanent(card.ID()) == nil || len(a.Hand()) != 0 {
				t.Fatal("paid casting failed")
			}
		})
	}
}

func TestRONDestructionReplacementStrictDeclarations(t *testing.T) {
	static := `ReplaceWouldDestroy(PermanentWouldBeDestroyed(ThisPermanent),[RegeneratePermanent(WouldBeDestroyed_ThatPermanent)])`
	future := `Activated(PayMana([]),ActionList([CreateFutureReplaceWouldDestroy(NextTimePermanentWouldBeDestroyedThisTurn(ThisPermanent),[CancelDestroy,RemoveAllDamageFromPermanent(WouldBeDestroyed_ThatPermanent)])]))`
	for _, rule := range []string{static, future} {
		for _, replace := range [][2]string{
			{"ThisPermanent", "ThisPermanent()"},
			{"ThisPermanent", "You"},
			{"ThisPermanent", "Ref_TargetPermanent"},
			{"WouldBeDestroyed_ThatPermanent", "WouldBeDestroyed_ThatPermanent()"},
			{"WouldBeDestroyed_ThatPermanent", "ThisPermanent"},
			{"WouldBeDestroyed_ThatPermanent", "You"},
			{"(ThisPermanent)", "()"},
			{"(ThisPermanent)", "(ThisPermanent,ThisPermanent)"},
			{"(WouldBeDestroyed_ThatPermanent)", "()"},
			{"(WouldBeDestroyed_ThatPermanent)", "(WouldBeDestroyed_ThatPermanent,ThisPermanent)"},
		} {
			assertConditionalRejected(t, strings.Replace(rule, replace[0], replace[1], 1))
		}
	}
	for _, permission := range []string{`OnlyOtherPlayersMayActivate`, `OnlyOtherPlayersMayActivate()`, `OnlyOtherPlayersMayActivate(AnyPlayer)`, `OnlyOtherPlayersMayActivate(Opponent())`, `OnlyOtherPlayersMayActivate(Opponent,Opponent)`, `And([OnlyOtherPlayersMayActivate(Opponent),OtherPlayersMayActivate(AnyPlayer)])`} {
		assertConditionalRejected(t, `ActivatedWithModifiers(PayMana([]),ActionList([GainLife(Integer(1))]),`+permission+`)`)
	}
}

func TestRONDestructionReplacementClergyLifecycle(t *testing.T) {
	for _, change := range []string{"damage", "cleanup", "source-control", "copy", "clone"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", "Clergy of the Holy Nimbus")
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			if change == "damage" {
				g.DealDamageToPermanent(p, 1, uuid.Nil)
				g.CheckStateBasedActions()
				if g.FindPermanent(p.ID()) == nil || g.FindPermanent(p.ID()).Damage != 0 || !g.FindPermanent(p.ID()).Tapped {
					t.Fatal("lethal damage did not regenerate")
				}
				return
			}
			if change == "source-control" {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.EndOfTurn})
				g.ApplyContinuousEffects()
				b.ManaPool().Add(core.Colorless, 1)
				if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 1, nil); err == nil {
					t.Fatal("new controller activated")
				}
				a.ManaPool().Add(core.Colorless, 1)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, nil); err != nil {
					t.Fatal(err)
				}
			} else {
				b.ManaPool().Add(core.Colorless, 1)
				if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 1, nil); err != nil {
					t.Fatal(err)
				}
			}
			if change == "copy" {
				g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
			}
			clone := g.Clone()
			clone.ResolveStack()
			g.ResolveStack()
			if change == "cleanup" {
				g.RunStepWithPriority(core.Cleanup)
			}
			g.DestroyPermanent(g.FindPermanent(p.ID()))
			if (g.FindPermanent(p.ID()) != nil) != (change == "cleanup") {
				t.Fatal("wrong prohibition lifetime")
			}
			if change == "clone" {
				clone.DestroyPermanent(clone.FindPermanent(p.ID()))
				if clone.FindPermanent(p.ID()) != nil {
					t.Fatal("clone lost pending prohibition")
				}
			}
		})
	}
}

func TestRONDestructionReplacementInventory(t *testing.T) {
	files := fstest.MapFS{}
	for name, set := range map[string]string{"Pyramids": "ARN", "Clergy of the Holy Nimbus": "LEG"} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.Contains(line, `Name: "`+name+`"`) {
				files[name+".ron"] = &fstest.MapFile{Data: []byte(line)}
			}
		}
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 2 || len(report.Issues) != 0 {
		t.Fatalf("report = %+v", report)
	}
	roles := map[string]bool{}
	for _, s := range report.Signatures {
		roles[s.Role+"/"+s.Name] = true
	}
	for _, key := range []string{"destruction_replacement_event/PermanentWouldBeDestroyed", "destruction_replacement_event/NextTimePermanentWouldBeDestroyedThisTurn", "destruction_replacement_operation/CancelDestroy", "destruction_replacement_operation/RemoveAllDamageFromPermanent", "destruction_event_reference/WouldBeDestroyed_ThatPermanent", "player_filter/Opponent"} {
		if !roles[key] {
			t.Fatalf("missing role %s", key)
		}
	}
}
