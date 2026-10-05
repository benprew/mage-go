package ron

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONMillstone(t *testing.T) {
	for _, recipient := range []string{"self", "opponent"} {
		for _, count := range []int{0, 1, 5} {
			for _, change := range []string{"none", "departed", "returned", "phased", "control", "copy"} {
				t.Run(fmt.Sprintf("%s/%d/%s", recipient, count, change), func(t *testing.T) {
					g, a, b := ronGame()
					fillRONLibrary(t, a, 5)
					fillRONLibrary(t, b, count)
					player := mage.Player(b)
					if recipient == "self" {
						player = a
					}
					before := slices.Clone(player.Library())
					card := repositoryCard(t, "ATQ", "Millstone")
					card.SetOwner(a.PlayerID())
					if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
						t.Fatal(err)
					}
					source := g.PutOnBattlefield(card, a.PlayerID())
					targets := []uuid.UUID{player.PlayerID()}
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, targets); err == nil {
						t.Fatal("unpaid activation accepted")
					}
					if source.Tapped || g.StackSize() != 0 {
						t.Fatal("payment did not roll back")
					}
					a.ManaPool().Add(core.Colorless, 2)
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{uuid.New()}); err == nil {
						t.Fatal("invalid player target accepted")
					}
					if source.Tapped || a.ManaPool().TotalMana() != 2 {
						t.Fatal("illegal target spent costs")
					}
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, targets); err != nil {
						t.Fatal(err)
					}
					if !g.FindPermanent(source.ID()).Tapped || a.ManaPool().TotalMana() != 0 || len(player.Library()) != len(before) {
						t.Fatal("cost or action timing")
					}
					switch change {
					case "departed", "returned":
						g.BouncePermanentToHand(g.FindPermanent(source.ID()))
						if change == "returned" {
							a.RemoveFromHand(card.ID())
							g.PutOnBattlefield(card, b.PlayerID())
						}
					case "phased":
						g.PhaseOut(source.ID())
					case "control":
						g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					case "copy":
						if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
							t.Fatal("copy failed")
						}
					}
					clone := g.Clone()
					g.ResolveStack()
					amount := min(2, len(before))
					if change == "copy" {
						amount = min(4, len(before))
					}
					if len(player.Library()) != len(before)-amount || len(player.Graveyard()) != amount {
						t.Fatal("wrong library or graveyard count")
					}
					for i, c := range before[:amount] {
						if player.Graveyard()[i].ID() != c.ID() {
							t.Fatal("did not mill from library top in order")
						}
					}
					if len(clone.GetPlayer(player.PlayerID()).Library()) != len(before) {
						t.Fatal("clone library changed")
					}
					clone.ResolveStack()
					if len(clone.GetPlayer(player.PlayerID()).Library()) != len(before)-amount {
						t.Fatal("clone lost player binding")
					}
					if recipient == "opponent" && len(a.Library()) != 5 {
						t.Fatal("milled ability controller")
					}
				})
			}
		}
	}
}

func TestRONDeepSpawnPayment(t *testing.T) {
	for _, scenario := range []string{"pay", "decline", "short", "double", "double-short", "zero", "departed", "returned", "phased", "phase-cycle", "control", "copy"} {
		t.Run(scenario, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: scenario != "decline"}
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
			g := mage.NewGame(a, b)
			g.SetTurn(2)
			count := 6
			if scenario == "short" {
				count = 1
			}
			if scenario == "double-short" {
				count = 3
			}
			fillRONLibrary(t, a.BasePlayer, count)
			fillRONLibrary(t, b.BasePlayer, 6)
			original := slices.Clone(a.Library())
			card := repositoryCard(t, "FEM", "Deep Spawn")
			card.SetOwner(b.PlayerID())
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(card, a.PlayerID())
			if !source.HasKeyword(core.Trample) {
				t.Fatal("missing trample")
			}
			queueRONUpkeep(g, b)
			if g.StackSize() != 0 {
				t.Fatal("opponent upkeep triggered")
			}
			queueRONUpkeep(g, a)
			if g.StackSize() != 1 || a.offers != 0 || len(a.Library()) != count {
				t.Fatal("payment ran before resolution")
			}
			amount := 2
			if scenario == "double" || scenario == "double-short" || scenario == "zero" {
				amount = 4
				if scenario == "zero" {
					amount = 0
				}
				adjusted := amount
				g.AddMillReplacement(source.ID(), func(_ *mage.Game, _ uuid.UUID, _ int) int { return adjusted })
			}
			switch scenario {
			case "departed", "returned":
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				if scenario == "returned" {
					b.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "phased":
				g.PhaseOut(source.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(source.ID()))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			g.ResolveStack()
			paid := scenario != "decline" && count >= amount
			milled := 0
			if paid {
				milled = amount
			}
			if len(a.Library()) != count-milled {
				t.Fatal("wrong payment amount or payer")
			}
			for i, c := range original[:milled] {
				if a.Graveyard()[i].ID() != c.ID() {
					t.Fatal("wrong paid cards")
				}
			}
			offers := 1
			if count < amount {
				offers = 0
			}
			if a.offers != offers {
				t.Fatalf("payment offers=%d, want %d", a.offers, offers)
			}
			if scenario == "copy" {
				if len(b.Library()) != 4 || b.offers != 1 {
					t.Fatal("copy did not use its own controller")
				}
			} else if len(b.Library()) != 6 || b.offers != 0 {
				t.Fatal("wrong player paid")
			}
			present := g.FindPermanentIncludingPhased(source.ID()) != nil
			if present != (scenario != "departed" && (paid || scenario == "returned")) {
				t.Fatal("wrong source incarnation sacrificed")
			}
			if !paid && len(b.Graveyard()) != 1 {
				t.Fatal("sacrifice did not use owner graveyard")
			}
			if len(clone.GetPlayer(a.PlayerID()).Library()) != count {
				t.Fatal("payment changed clone")
			}
		})
	}
}

func TestRONDeepSpawnUnpaidIdentity(t *testing.T) {
	for _, change := range []string{"returned", "phased", "phase-cycle", "control"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "FEM", "Deep Spawn")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			queueRONUpkeep(g, a)
			switch change {
			case "returned":
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
			case "phased":
				g.PhaseOut(source.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(source.ID()))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			g.ResolveStack()
			want := change != "phase-cycle"
			if (g.FindPermanentIncludingPhased(source.ID()) != nil) != want {
				t.Fatal("unpaid sacrifice used wrong identity or control")
			}
		})
	}
}

func TestRONDeepSpawnShroudUntap(t *testing.T) {
	g, a, _ := ronGame()
	card := repositoryCard(t, "FEM", "Deep Spawn")
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, nil); err == nil {
		t.Fatal("unpaid shroud accepted")
	}
	a.ManaPool().Add(core.Blue, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, nil); err != nil {
		t.Fatal(err)
	}
	if source.Tapped || source.HasKeyword(core.Shroud) {
		t.Fatal("actions ran before resolution")
	}
	g.ResolveStack()
	if !g.FindPermanent(source.ID()).Tapped || !g.FindPermanent(source.ID()).HasKeyword(core.Shroud) {
		t.Fatal("shroud or tap missing")
	}
	g.RunStepWithPriority(core.Cleanup)
	if g.FindPermanent(source.ID()).HasKeyword(core.Shroud) {
		t.Fatal("shroud did not expire")
	}
	clone := g.Clone()
	g.RunStepWithPriority(core.Untap)
	if !g.FindPermanent(source.ID()).Tapped {
		t.Fatal("next untap not skipped")
	}
	g.RunStepWithPriority(core.Untap)
	if g.FindPermanent(source.ID()).Tapped || !clone.FindPermanent(source.ID()).Tapped {
		t.Fatal("untap lifetime or clone isolation")
	}
}

func TestRONMillDeclarations(t *testing.T) {
	for _, amount := range []string{"", "ValueX", "2", "Integer", "Integer()", "Integer(You)", "Integer(2,3)", "Integer(-1)"} {
		for _, context := range []string{"action", "cost"} {
			t.Run(context+"/"+amount, func(t *testing.T) {
				body := fmt.Sprintf(`MillNumberCards(%s)`, amount)
				if context == "cost" {
					body = `MayCost(` + body + `), Unless(CostWasPaid, [GainLife(Integer(1))])`
				}
				src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([` + body + `]))])`
				nodes, err := Parse(strings.NewReader(src), "invalid-mill.ron")
				if err != nil {
					t.Fatal(err)
				}
				if _, err := Compile(nodes[0]); err == nil {
					t.Fatal("invalid declaration accepted")
				}
			})
		}
	}
	for _, body := range []string{`MayCost(MillNumberCards(Integer(0))), Unless(CostWasPaid,[GainLife(Integer(1))])`, `PlayerAction(Ref_TargetPlayer,MillNumberCards(Integer(2)))`, `PlayerAction(ThisPermanent,MillNumberCards(Integer(2)))`, `MayCost(MillNumberCards(Integer(2)))`, `EachPlayerAction(AnyPlayer,MillNumberCards(Integer(2)))`} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([` + body + `]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid-mill.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestRONMillScopesAndInstances(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerActions(Ref_TargetPlayer,[MillNumberCards(Integer(1)),PlayerAction(ActionPlayer,MillNumberCards(Integer(1)))]),MillNumberCards(Integer(0))])))`)
	first, second := factory(), factory()
	if first.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == second.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
		t.Fatal("shared targets")
	}
	g, a, b := ronGame()
	fillRONLibrary(t, a, 4)
	fillRONLibrary(t, b, 4)
	source := g.PutOnBattlefield(first, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if len(a.Library()) != 4 || len(b.Library()) != 2 {
		t.Fatal("player action scope escaped")
	}
}

func TestRONMillStrictLoadAndInventory(t *testing.T) {
	records := fstest.MapFS{}
	for _, set := range []string{"ATQ", "3ED", "FEM"} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.HasPrefix(line, `Card(Name: "Millstone",`) || strings.HasPrefix(line, `Card(Name: "Deep Spawn",`) {
				records[set+".ron"] = &fstest.MapFile{Data: []byte(line)}
			}
		}
	}
	registry, err := LoadFS(records, "*.ron")
	if err != nil || registry == nil || len(records) != 3 {
		t.Fatalf("real records did not load: %v", err)
	}

	src := `Card(Name:"Mill",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([MayCost(MillNumberCards(Integer(2))),Unless(CostWasPaid,[MillNumberCards(Integer(1))])]))])`
	fs := fstest.MapFS{"mill.ron": {Data: []byte(src)}}
	if _, err := LoadFS(fs, "*.ron"); err != nil {
		t.Fatal(err)
	}
	report, err := InspectFS(fs, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, role := range []string{"resolution_cost", "action"} {
		found := false
		for _, sig := range report.Signatures {
			if sig.Name == "MillNumberCards" && sig.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing mill role %s", role)
		}
	}
	fs["bad.ron"] = &fstest.MapFile{Data: []byte(strings.ReplaceAll(src, `Integer(2)`, `Integer(-1)`))}
	if registry, err := LoadFS(fs, "*.ron"); err == nil || registry != nil {
		t.Fatal("partial collection loaded")
	}
}

func TestRONMillModifiersAndPaymentFrames(t *testing.T) {
	for _, cost := range []bool{false, true} {
		t.Run(fmt.Sprint(cost), func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
			g := mage.NewGame(a, mage.NewBasePlayer("B"))
			fillRONLibrary(t, a.BasePlayer, 5)
			body := `MillNumberCards(Integer(2))`
			if cost {
				body = `MayCost(MillNumberCards(Integer(2))), If(CostWasPaid,[GainLife(Integer(1))]), MayCost(MillNumberCards(Integer(2))), Unless(CostWasPaid,[GainLife(Integer(3))])`
			}
			card := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([`+body+`]))`)()
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			first := g.PutOnBattlefield(mage.NewArtifact("First modifier", ""), a.PlayerID())
			secondCard := mage.NewArtifact("Second modifier", "")
			secondCard.SetOwner(a.PlayerID())
			second := g.PutOnBattlefield(secondCard, a.PlayerID())
			g.AddMillReplacement(first.ID(), func(_ *mage.Game, _ uuid.UUID, n int) int { return n + 1 })
			g.AddMillReplacement(second.ID(), func(_ *mage.Game, _ uuid.UUID, n int) int { return n * 2 })
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if cost {
				if len(a.Library()) != 5 || a.Life() != 23 || a.offers != 0 {
					t.Fatal("unaffordable adjusted costs partly paid")
				}
				g.DestroyPermanent(second)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				if len(a.Library()) != 2 || a.Life() != 27 || a.offers != 1 {
					t.Fatal("independent payment results or recheck failed")
				}
			} else if len(a.Library()) != 0 || len(a.Graveyard()) != 5 {
				t.Fatal("modified action did not mill as much as possible")
			}
		})
	}
}

func TestRONDeepSpawnNativeUpkeep(t *testing.T) {
	a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	g.SetTurn(2)
	fillRONLibrary(t, a.BasePlayer, 4)
	card := repositoryCard(t, "FEM", "Deep Spawn")
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	g.SetActivePlayerIndex(1)
	g.RunStepWithPriority(core.Upkeep)
	if len(a.Library()) != 4 || a.offers != 0 {
		t.Fatal("wrong native upkeep")
	}
	g.SetActivePlayerIndex(0)
	g.RunStepWithPriority(core.Upkeep)
	if len(a.Library()) != 2 || a.offers != 1 || g.FindPermanent(source.ID()) == nil || g.StackSize() != 0 {
		t.Fatal("native upkeep did not pay and retain source")
	}
}

type ronMillOrderPlayer struct {
	*ronPaymentPlayer
	orders int
}

func (p *ronMillOrderPlayer) ChooseCardFromLibrary(cards []mage.Card, _ string, _ mage.GameReader) mage.Card {
	p.orders++
	return cards[len(cards)-1]
}

func TestRONMillRealCardOwnerOrder(t *testing.T) {
	for _, name := range []string{"Millstone", "Deep Spawn"} {
		t.Run(name, func(t *testing.T) {
			a := &ronMillOrderPlayer{ronPaymentPlayer: &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			g.SetTurn(2)
			fillRONLibrary(t, a.BasePlayer, 3)
			original := slices.Clone(a.Library())
			set := "FEM"
			if name == "Millstone" {
				set = "ATQ"
			}
			card := repositoryCard(t, set, name)
			card.SetOwner(b.PlayerID())
			source := g.PutOnBattlefield(card, b.PlayerID())
			if name == "Millstone" {
				b.ManaPool().Add(core.Colorless, 2)
				if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, []uuid.UUID{a.PlayerID()}); err != nil {
					t.Fatal(err)
				}
			} else {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: source.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				queueRONUpkeep(g, a)
			}
			if a.orders != 0 {
				t.Fatal("ordering happened before resolution")
			}
			g.ResolveStack()
			if a.orders != 1 || len(a.Graveyard()) != 2 || a.Graveyard()[0].ID() != original[1].ID() || a.Graveyard()[1].ID() != original[0].ID() {
				t.Fatal("real card ignored library owner's order")
			}
			if name == "Deep Spawn" && (a.offers != 1 || g.FindPermanent(source.ID()) == nil) {
				t.Fatal("ordering changed payment outcome")
			}
		})
	}
}

func TestRONMillExplicitPayer(t *testing.T) {
	for _, accept := range []bool{false, true} {
		a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: !accept}
		b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: accept}
		g := mage.NewGame(a, b)
		fillRONLibrary(t, a.BasePlayer, 3)
		fillRONLibrary(t, b.BasePlayer, 3)
		factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerMayCost(Ref_TargetPlayer,MillNumberCards(Integer(2))),If(CostWasPaid,[GainLife(Integer(1))])])))`)
		card := factory()
		card.SetOwner(a.PlayerID())
		source := g.PutOnBattlefield(card, a.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		wantLibrary, wantLife := 3, 20
		if accept {
			wantLibrary, wantLife = 1, 21
		}
		if len(a.Library()) != 3 || len(b.Library()) != wantLibrary || a.Life() != wantLife || b.Life() != 20 || a.offers != 0 || b.offers != 1 {
			t.Fatal("explicit payer or continuation scope changed")
		}
	}
}
