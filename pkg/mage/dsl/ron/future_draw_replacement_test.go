package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONFutureDrawReplacementGameplay(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([ManaCostGeneric(1)]),Targeted([TargetPlayer(AnyPlayer)],ActionList([CreateFutureReplaceWouldDraw(NextTimePlayerWouldDrawACardThisTurn(Ref_TargetPlayer),[GainLife(Integer(2)),PlayerAction(Ref_TargetPlayer,DrawACard)])])))`)
	g, a, b := ronGame()
	g.SetTurn(3)
	card := factory()
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	seedRONDrawLibrary(a)
	seedRONDrawLibrary(b)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err == nil {
		t.Fatal("failed payment accepted")
	}
	g.PlayerDrawCard(b)
	if len(b.Hand()) != 1 || a.Life() != 20 {
		t.Fatal("failed payment installed replacement")
	}
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.PlayerDrawCard(b)
	if len(b.Hand()) != 2 || a.Life() != 20 {
		t.Fatal("announcement installed replacement")
	}
	g.DestroyPermanent(p)
	g.ResolveStack()
	clone := g.Clone()
	g.PlayerDrawCard(a)
	g.PlayerDrawCard(b)
	if a.Life() != 22 || b.Life() != 20 || len(b.Hand()) != 3 || g.StackSize() != 0 {
		t.Fatal("saved player, controller, target or generated draw")
	}
	clone.PlayerDrawCard(clone.GetPlayer(b.PlayerID()))
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || a.Life() != 22 {
		t.Fatal("clone shared consumption")
	}
	g.PlayerDrawCard(b)
	if a.Life() != 22 || len(b.Hand()) != 4 {
		t.Fatal("one-shot used twice")
	}
}

func TestRONFutureDrawReplacementCardEvents(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"ARN", "Aladdin's Lamp"}, {"3ED", "Aladdin's Lamp"}, {"ARN", "Ring of Ma'rûf"}} {
		t.Run(tc.set+tc.name, func(t *testing.T) {
			record := staticBaseRecord(t, tc.set, tc.name)
			factory, err := Compile(record)
			if err != nil {
				t.Fatal(err)
			}
			if err = mage.ValidateCardDefinition(tc.name, factory()); err != nil {
				t.Fatal(err)
			}
			installation := record.Fields["Rules"].Args[0].Args[1].Args[0].Args[0]
			installation.Args[1] = nameScopeNode(t, `[GainLife(Integer(2)),DrawACard]`)
			rule := record.Fields["Rules"].Args[0]
			if rule.Name == "ActivatedWithModifiers" {
				rule.Name = "Activated"
				rule.Args = rule.Args[:2]
			}
			rule.Args[1].Args[0].Args = []Node{installation}
			record.Fields["Rules"] = Node{Kind: List, Args: []Node{rule}}
			factory, err = Compile(record)
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			g.SetTurn(3)
			card := factory()
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			seedRONDrawLibrary(a)
			seedRONDrawLibrary(b)
			a.ManaPool().Add(core.Colorless, 5)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if g.FindPermanent(p.ID()) != nil {
				g.DestroyPermanent(p)
			}
			g.PlayerDrawCard(b)
			g.PlayerDrawCard(a)
			if a.Life() != 22 || b.Life() != 20 || len(a.Hand()) != 1 || len(b.Hand()) != 1 {
				t.Fatal("real event scope")
			}
		})
	}
}

func TestRONFutureDrawReplacementTargetIdentity(t *testing.T) {
	for _, state := range []string{"live", "return", "phase", "shroud", "control", "expire"} {
		t.Run(state, func(t *testing.T) {
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreateFutureReplaceWouldDraw(NextTimePlayerWouldDrawACardThisTurn(You),[TapPermanent(Ref_TargetPermanent),GainLife(Integer(2))])])))`)
			g, a, b := ronGame()
			g.SetTurn(3)
			seedRONDrawLibrary(a)
			source := factory()
			source.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(source, a.PlayerID())
			card := mage.NewCreature("Target", "", 2, 3)
			card.SetOwner(b.PlayerID())
			victim := g.PutOnBattlefield(card, b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			switch state {
			case "return":
				g.DestroyPermanent(victim)
				victim = g.PutOnBattlefield(card, b.PlayerID())
			case "phase":
				g.PhaseOut(victim.ID())
			case "shroud":
				g.AddContinuousEffect(mage.TemporaryKeyword(victim.ID(), core.Shroud))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: p.ID(), TargetID: victim.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "expire":
				g.RunStepWithPriority(core.Cleanup)
			}
			g.PlayerDrawCard(a)
			want := state == "live" || state == "shroud" || state == "control"
			if victim.Tapped != want {
				t.Fatal("saved target identity or new legality check", state)
			}
			if state == "expire" {
				if a.Life() != 20 || len(a.Hand()) != 1 {
					t.Fatal("RON rule did not expire")
				}
			} else if a.Life() != 22 || len(a.Hand()) != 0 {
				t.Fatal("independent body instruction was lost")
			}
		})
	}
}

func TestRONFutureDrawReplacementSavedX(t *testing.T) {
	source := `Card(Name:"Future spell",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),ManaCost:[ManaCostX],Rules:[SpellActions(ActionList([CreateFutureReplaceWouldDraw(NextTimePlayerWouldDrawACardThisTurn(You),[GainLife(ValueX)])]))])`
	nodes, err := Parse(strings.NewReader(source), "x.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	g, a, _ := ronGame()
	g.SetTurn(3)
	card := factory()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Colorless, 3)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 3); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.SetXValue(77)
	g.PlayerDrawCard(a)
	if a.Life() != 23 || g.XValue() != 77 || a.DrewFromEmpty() {
		t.Fatal("saved X or replacement timing")
	}
}

func TestRONFutureDrawReplacementInvalid(t *testing.T) {
	valid := `CreateFutureReplaceWouldDraw(NextTimePlayerWouldDrawACardThisTurn(You),[GainLife(Integer(1))])`
	for _, action := range []string{
		`CreateFutureReplaceWouldDraw`, `CreateFutureReplaceWouldDraw()`,
		strings.Replace(valid, "You)", "You())", 1),
		strings.Replace(valid, "You)", "Ref_TargetPlayer)", 1),
		strings.Replace(valid, "You)", "AnyPlayer)", 1),
		strings.Replace(valid, "You)", "WouldDrawACard_ThatPlayer)", 1),
		strings.Replace(valid, "NextTimePlayerWouldDrawACardThisTurn", "NextTimePlayerWouldDrawACard", 1),
		strings.Replace(valid, "[GainLife(Integer(1))]", "[]", 1),
		strings.Replace(valid, "GainLife(Integer(1))", "UnknownAction", 1),
		strings.Replace(valid, "GainLife(Integer(1))", "DrawACard", 1) + `,RevealTheCardDrawnThisWay`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([` + action + `]))])`
		registry, err := LoadFS(fstest.MapFS{"invalid.ron": {Data: []byte(src)}}, "*.ron")
		if err == nil || registry != nil {
			t.Fatal("invalid declaration accepted", action)
		}
	}
}

func TestRONFutureDrawReplacementInventory(t *testing.T) {
	nodes, err := Parse(strings.NewReader(`CreateFutureReplaceWouldDraw(NextTimePlayerWouldDrawACardThisTurn(You),[DrawACard])`), "inventory.ron")
	if err != nil {
		t.Fatal(err)
	}
	entries := make(map[string]*InventorySignature)
	walkInventory(nodes[0], "Rule", "action", "action", "action", true, entries)
	for name, role := range map[string]string{"NextTimePlayerWouldDrawACardThisTurn": "future_draw_replacement_event", "You": "player_reference", "DrawACard": "action"} {
		found := false
		for _, entry := range entries {
			if entry.Name == name {
				found = true
				if entry.Role != role {
					t.Fatalf("%s role=%s", name, entry.Role)
				}
			}
		}
		if !found {
			t.Fatal("missing signature", name)
		}
	}
}
