package ron

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONLifeTotalTimingAndSubject(t *testing.T) {
	for _, half := range []string{"HalfRoundedDown", "HalfRoundedUp"} {
		t.Run(half, func(t *testing.T) {
			factory := permanentScopeFactory(t, "Instant", fmt.Sprintf(`SpellActions(Targeted([TargetPlayer(AnyPlayer)], ActionList([PlayerAction(Ref_TargetPlayer, LoseLife(%s(LifeTotalOfPlayer(LoseLife_ThatPlayer)))), PlayerAction(You, GainLife(LifeTotalOfPlayer(Ref_TargetPlayer))), PlayerAction(You, LoseLife(%s(LifeTotalOfPlayer(LoseLife_ThatPlayer))))])))`, half, half))
			g, a, b := ronGame()
			card := factory()
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
				t.Fatal(err)
			}
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("announcement read changed life")
			}
			b.SetLife(13)
			g.AddPreventionShield(b.PlayerID(), 100)
			clone := g.Clone()
			g.ResolveStack()
			wantA, wantB := 14, 7
			if half == "HalfRoundedUp" {
				wantA, wantB = 13, 6
			}
			if a.Life() != wantA || b.Life() != wantB {
				t.Fatalf("instruction timing or subject: %d/%d", a.Life(), b.Life())
			}
			clone.GetPlayer(b.PlayerID()).SetLife(9)
			clone.ResolveStack()
			ca, cb := 13, 5
			if half == "HalfRoundedUp" {
				ca, cb = 12, 4
			}
			if clone.GetPlayer(a.PlayerID()).Life() != ca || clone.GetPlayer(b.PlayerID()).Life() != cb || a.Life() != wantA {
				t.Fatal("clone shared subject or value")
			}
		})
	}
}

func TestRONLifeTotalCopiesAndInstances(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPlayer(AnyPlayer)], ActionList([PlayerAction(Ref_TargetPlayer,LoseLife(HalfRoundedUp(LifeTotalOfPlayer(LoseLife_ThatPlayer))))])))`)
	g, a, b := ronGame()
	for _, target := range []uuid.UUID{a.PlayerID(), b.PlayerID()} {
		card := factory()
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target}, 0); err != nil {
			t.Fatal(err)
		}
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if a.Life() != 10 || b.Life() != 5 {
		t.Fatalf("instances or copies share state: %d/%d", a.Life(), b.Life())
	}
}

func TestRONLifeTotalInvalid(t *testing.T) {
	for _, rule := range []string{
		`SpellActions(ActionList([GainLife(LifeTotalOfPlayer(LoseLife_ThatPlayer))]))`,
		`SpellActions(ActionList([LoseLife(Integer(1)),GainLife(LifeTotalOfPlayer(LoseLife_ThatPlayer))]))`,
		`SpellActions(ActionList([LoseLife(LifeTotalOfPlayer(LoseLife_ThatPlayer()))]))`,
		`SpellActions(ActionList([LoseLife(LifeTotalOfPlayer(Ref_TargetPlayer))]))`,
		`SpellActions(ActionList([LoseLife(LifeTotalOfPlayer(ThisPermanent))]))`,
		`SpellActions(ActionList([LoseLife(LifeTotalOfPlayer)]))`,
		`SpellActions(ActionList([LoseLife(LifeTotalOfPlayer(You,You))]))`,
		`AsPermanentEnters(ThisPermanent,[LoseLife(LifeTotalOfPlayer(ActionPlayer))])`,
		`SpellActions(ActionList([LoseLife(LifeTotalOfPlayer(ActionPlayer))]))`,
		`SpellActions(ActionList([PlayerActions(You,[LoseLife(Integer(1))]),LoseLife(LifeTotalOfPlayer(ActionPlayer))]))`,
		`CreatePermanentLayerEffect(ThisPermanent,[AdjustPTX(+1,+1,LifeTotalOfPlayer(You))])`,
		`Activated(PayLife(LifeTotalOfPlayer(You)),ActionList([GainLife(Integer(1))]))`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[` + rule + `])`
		nodes, err := Parse(strings.NewReader(src), "invalid-life.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONLifeTotalInventory(t *testing.T) {
	src := `Card(Name:"Life",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([LoseLife(LifeTotalOfPlayer(LoseLife_ThatPlayer))]))])`
	report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 1 {
		t.Fatalf("compile issues: %+v", report.Issues)
	}
	for _, s := range report.Signatures {
		if s.Name == "LoseLife_ThatPlayer" && s.Role != "player_reference" {
			t.Fatalf("wrong role: %s", s.Role)
		}
	}
}

func TestRONLifeTotalSourceFragments(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		record := staticBaseRecord(t, set, "Personal Incarnation")
		action := record.Fields["Rules"].Args[1].Args[1].Args[0].Args[0].Args[1]
		if action.Name != "LoseLife" {
			t.Fatal("wrong source fragment")
		}
		actions, err := compilePlayerEffect(action, scope{resolving: true})
		if err != nil {
			t.Fatal(err)
		}
		g, a, b := ronGame()
		a.SetLife(15)
		if err := mage.ApplyEffect(g, actions[0](instanceBindings{}), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if a.Life() != 7 || b.Life() != 20 {
			t.Fatal("source fragment did not retain declared rounding")
		}
		if _, err := Compile(record); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRONLifeTotalSourceLifetime(t *testing.T) {
	for _, state := range []string{"live", "control", "departed", "returned", "phased", "copy"} {
		t.Run(state, func(t *testing.T) {
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([PlayerAction(ControllerOfPermanent(ThisPermanent),LoseLife(HalfRoundedUp(LifeTotalOfPlayer(LoseLife_ThatPlayer))))]))`)
			g, a, b := ronGame()
			card := factory()
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			a.SetLife(15)
			b.SetLife(11)
			switch state {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "departed", "returned":
				g.DestroyPermanent(p)
				if state == "returned" {
					a.RemoveFromGraveyard(card.ID())
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "phased":
				g.PhaseOut(p.ID())
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			for _, game := range []*mage.Game{g, clone} {
				game.ResolveStack()
				wantA, wantB := 7, 11
				if state == "control" {
					wantA, wantB = 15, 5
				}
				if state == "copy" {
					wantA = 3
				}
				if game.GetPlayer(a.PlayerID()).Life() != wantA || game.GetPlayer(b.PlayerID()).Life() != wantB {
					t.Fatalf("source visit %s: %d/%d", state, game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
				}
			}
		})
	}
}

func TestRONLifeTotalNonpositiveAndMissing(t *testing.T) {
	for _, life := range []int{-3, 0, 1, 7} {
		for _, action := range []string{"GainLife", "LoseLife"} {
			g, a, b := ronGame()
			a.SetLife(life)
			factory := permanentScopeFactory(t, "Instant", fmt.Sprintf(`SpellActions(ActionList([%s(LifeTotalOfPlayer(You))]))`, action))
			card := factory()
			effect := card.Abilities()[0].(*mage.ActionDefinition)
			if err := mage.ApplyEffect(g, mage.Sequence(effect.Effects()...), card.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			want := life
			if life > 0 {
				if action == "GainLife" {
					want = 2 * life
				} else {
					want = 0
				}
			}
			if a.Life() != want || b.Life() != 20 {
				t.Fatalf("%s %d: %d", action, life, a.Life())
			}
		}
	}
	g, a, b := ronGame()
	factory := permanentScopeFactory(t, "Instant", `SpellActions(ActionList([PlayerAction(ThePlayerWithTheMostLife,LoseLife(LifeTotalOfPlayer(LoseLife_ThatPlayer)))]))`)
	card := factory()
	if err := mage.ApplyEffect(g, mage.Sequence(card.Abilities()[0].(*mage.ActionDefinition).Effects()...), card.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("absent subject used another player")
	}
}

func TestRONLifeTotalScopedSubgameFragment(t *testing.T) {
	record := staticBaseRecord(t, "ARN", "Shahrazad")
	action := record.Fields["Rules"].Args[0].Args[0].Args[0].Args[1].Args[1]
	if action.Name != "LoseLife" {
		t.Fatal("wrong subgame fragment")
	}
	plan := &playerActionsPlan{name: "subgame action player"}
	effects, err := compilePlayerEffect(action, scope{resolving: true, actionPlayer: plan, player: playerReferencePlan{action: plan}})
	if err != nil {
		t.Fatal(err)
	}
	binding := mage.BindPlayer(plan.name)
	bindings := instanceBindings{actionPlayers: map[*playerActionsPlan]*mage.PlayerBinding{plan: binding}}
	g, a, b := ronGame()
	b.SetLife(15)
	effect := mage.Sequence(mage.SnapshotPlayer(mage.ActivePlayer, binding), effects[0](bindings))
	g.SetActivePlayerIndex(1)
	if err := mage.ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 || b.Life() != 7 {
		t.Fatal("action player or round up differs")
	}
	if _, err := Compile(record); err == nil {
		t.Fatal("subgame was silently accepted")
	}
}

func TestRONLifeTotalPlayerActions(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerActions(Ref_TargetPlayer,[LoseLife(HalfRoundedUp(LifeTotalOfPlayer(ActionPlayer))),PlayerAction(You,LoseLife(HalfRoundedDown(LifeTotalOfPlayer(LoseLife_ThatPlayer)))),GainLife(LifeTotalOfPlayer(ActionPlayer))])])))`)
	g, a, b := ronGame()
	b.SetLife(15)
	card := factory()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{uuid.New()}, 0); err == nil {
		t.Fatal("invalid player accepted")
	}
	if len(a.Hand()) != 1 || a.Life() != 20 || b.Life() != 15 || !g.Stack().IsEmpty() {
		t.Fatal("illegal announcement changed state")
	}
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 10 || b.Life() != 14 {
		t.Fatalf("nested subject escaped or action player changed: %d/%d", a.Life(), b.Life())
	}
}
