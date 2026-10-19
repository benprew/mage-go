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

func TestRONDrawDeadlineNafsAsp(t *testing.T) {
	for _, pay := range []bool{false, true} {
		g, a, b := ronGame()
		card := repositoryCard(t, "ARN", "Nafs Asp")
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		a.ManaPool().Add(core.Green, 1)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		source := g.FindPermanent(card.ID())
		for range 2 {
			g.DealDamageToPlayer(b, 1, source.ID())
		}
		b.ManaPool().Add(core.Colorless, 2)
		if len(g.GetPlayerPayments(b.PlayerID())) != 0 {
			t.Fatal("obligation before trigger resolution")
		}
		g.PutTriggersOnStack()
		g.ResolveStack()
		offers := g.GetPlayerPayments(b.PlayerID())
		if len(offers) != 2 {
			t.Fatal("damage events did not create separate obligations")
		}
		if pay {
			if err := g.TakePlayerPayment(b.PlayerID(), offers[0].ID); err != nil {
				t.Fatal(err)
			}
		}
		g.RemoveFromBattlefield(source)
		g.RunStepWithPriority(core.Cleanup)
		g.SetActivePlayerIndex(1)
		b.AddToLibrary(mage.NewLand("Draw card"))
		g.RunStepWithPriority(core.Draw)
		want := 16
		if pay {
			want = 17
		}
		if b.Life() != want || len(b.Hand()) != 1 || len(g.GetPlayerPayments(b.PlayerID())) != 0 {
			t.Fatalf("life=%d want=%d hand=%d", b.Life(), want, len(b.Hand()))
		}
	}
}

const drawDeadlineRON = `CreateFutureTrigger_UnlessPlayerPaysManaBefore(AtTheBeginningOfPlayersNextDrawStep(You),ActionList([PlayerAction(You,LoseLife(Integer(1)))]),You,[ManaCostGeneric(1)])`

func TestRONDrawDeadlineInvalid(t *testing.T) {
	for _, src := range []string{
		strings.Replace(drawDeadlineRON, "NextDrawStep(You)", "NextUpkeep(You)", 1),
		strings.Replace(drawDeadlineRON, "NextDrawStep(You)", "NextDrawStep(Opponent)", 1),
		strings.ReplaceAll(drawDeadlineRON, "You", "Trigger_ThatPlayer"),
		strings.Replace(drawDeadlineRON, "[ManaCostGeneric(1)]", "[ManaCostX]", 1),
		strings.Replace(drawDeadlineRON, "[ManaCostGeneric(1)]", "[ManaCostGeneric(-1)]", 1),
		strings.Replace(drawDeadlineRON, "[PlayerAction(You,LoseLife(Integer(1)))]", "[]", 1),
		strings.Replace(drawDeadlineRON, "ActionList([PlayerAction(You,LoseLife(Integer(1)))])", "LoseLife(Integer(1))", 1),
		strings.Replace(drawDeadlineRON, "NextDrawStep(You)", "NextDrawStep(You,You)", 1),
		strings.Replace(drawDeadlineRON, ",You,[ManaCostGeneric(1)]", ",You", 1),
	} {
		nodes, err := Parse(strings.NewReader(src), "invalid-deadline.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := compileEffect(nodes[0], scope{permanent: true, resolving: true}); err == nil {
			t.Fatal("invalid declaration accepted", src)
		}
	}
	nodes, err := Parse(strings.NewReader(drawDeadlineRON), "deadline.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, ctx := range []scope{{permanent: true}, {permanent: true, resolving: true, enteringSource: true}, {permanent: true, resolving: true, drawReplacementBody: true}} {
		if _, err := compileEffect(nodes[0], ctx); err == nil {
			t.Fatal("invalid context accepted")
		}
	}
}

func TestRONDrawDeadlineDamageFacts(t *testing.T) {
	for _, change := range []string{"phase", "return", "control", "copy", "two-players", "prevent"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "ARN", "Nafs Asp")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			if change == "prevent" {
				if err := mage.ApplyEffect(g, mage.PreventNextDamageFromPermanent(mage.This, mage.You, mage.IsCreature), source.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			g.DealDamageToPlayer(b, 1, source.ID())
			if change == "two-players" {
				g.DealDamageToPlayer(a, 1, source.ID())
			}
			g.PutTriggersOnStack()
			if change == "prevent" {
				g.ResolveStack()
				if len(g.DelayedTriggers()) != 0 || b.Life() != 20 {
					t.Fatal("prevented damage triggered obligation")
				}
				return
			}
			switch change {
			case "phase":
				g.PhaseOut(source.ID())
			case "return":
				g.RemoveFromBattlefield(source)
				g.PutOnBattlefield(card, b.PlayerID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			for _, game := range []*mage.Game{g, clone} {
				game.ResolveStack()
				game.GetPlayer(a.PlayerID()).ManaPool().Add(core.Colorless, 3)
				game.GetPlayer(b.PlayerID()).ManaPool().Add(core.Colorless, 3)
				want := 1
				if change == "copy" {
					want = 2
				}
				if len(game.GetPlayerPayments(b.PlayerID())) != want {
					t.Fatal("lost damage recipient")
				}
				game.FireEvent(core.GameEvent{Type: core.EvtDrawStep, PlayerID: b.PlayerID()})
				game.PutTriggersOnStack()
				game.ResolveStack()
				if game.GetPlayer(b.PlayerID()).Life() != 19-want {
					t.Fatal("wrong unpaid loss")
				}
				if change == "two-players" {
					if len(game.GetPlayerPayments(a.PlayerID())) != 1 {
						t.Fatal("one recipient erased another")
					}
					game.FireEvent(core.GameEvent{Type: core.EvtDrawStep, PlayerID: a.PlayerID()})
					game.PutTriggersOnStack()
					game.ResolveStack()
					if game.GetPlayer(a.PlayerID()).Life() != 18 {
						t.Fatal("second recipient lost")
					}
				}
			}
		})
	}
}

func TestRONDrawDeadlineLoadAndCombat(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/ARN.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.HasPrefix(line, `Card(Name: "Nafs Asp",`) {
			record = line
		}
	}
	files := fstest.MapFS{"ARN.ron": &fstest.MapFile{Data: []byte(record)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	first, err := registry.CreateCard("Nafs Asp")
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.CreateCard("Nafs Asp")
	if err != nil || first.ID() == second.ID() || first.Abilities()[0] == second.Abilities()[0] {
		t.Fatal("shared card factory state")
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal("inventory", err)
	}
	found := false
	for _, sig := range report.Signatures {
		found = found || sig.Name == "AtTheBeginningOfPlayersNextDrawStep" && sig.Role == "delayed_step_event"
	}
	if !found {
		t.Fatal("missing deadline event role")
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([MissingAction]))])`)}
	if LoadInto(registry, files, "*.ron") == nil {
		t.Fatal("published partial collection")
	}
	if _, err := registry.CreateCard("Nafs Asp"); err != nil {
		t.Fatal("failed collection load erased registry")
	}
	g, a, b := ronGame()
	first.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(first, a.PlayerID())
	source.RevokeBaseAttr(core.AttrSummonSick)
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{source.ID()})
	g.ExecuteCombatDamage()
	g.PutTriggersOnStack()
	g.ResolveStack()
	b.ManaPool().Add(core.Colorless, 1)
	offers := g.GetPlayerPayments(b.PlayerID())
	if b.Life() != 19 || len(offers) != 1 {
		t.Fatal("combat damage did not create obligation")
	}
	paid := false
	g.SetOnPriority(func(_ *mage.Game, index int, _ bool) mage.PriorityAction {
		if index == 1 && !paid {
			paid = true
			return mage.PriorityAction{Type: mage.PriorityTakePlayerPayment, PlayerPaymentID: offers[0].ID}
		}
		return mage.PriorityAction{Type: mage.PriorityPass}
	})
	g.RunStepWithPriority(core.Upkeep)
	if !paid || len(g.DelayedTriggers()) != 0 || b.Life() != 19 {
		t.Fatal("priority payment failed")
	}
}
