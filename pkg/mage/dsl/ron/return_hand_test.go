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

func castRONReturn(t *testing.T, g *mage.Game, a *ronModePlayer, card mage.Card, target uuid.UUID) {
	t.Helper()
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Blue, 2)
	a.ManaPool().Add(core.Red, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target}, 0); err != nil {
		t.Fatal(err)
	}
}

func TestRONReturnHandCards(t *testing.T) {
	for _, name := range []string{"Unsummon", "Boomerang", "Active Volcano", "Flash Flood"} {
		sets := []string{"LEG"}
		if name == "Unsummon" {
			sets = []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"}
		}
		modes := []int{0}
		if name == "Active Volcano" || name == "Flash Flood" {
			modes = []int{0, 1}
		}
		for _, set := range sets {
			for _, mode := range modes {
				t.Run(fmt.Sprintf("%s/%s/%d", name, set, mode), func(t *testing.T) {
					g, a, b := ronModalGame(mode)
					victimName := "Air Elemental"
					if name == "Boomerang" {
						victimName = "Forest"
					}
					if name == "Flash Flood" {
						victimName = "Shivan Dragon"
					}
					if mode == 1 {
						victimName = "Island"
						if name == "Flash Flood" {
							victimName = "Mountain"
						}
					}
					victim := repositoryCard(t, "LEB", victimName)
					victim.SetOwner(b.PlayerID())
					p := g.PutOnBattlefield(victim, a.PlayerID())
					castRONReturn(t, g, a, repositoryCard(t, set, name), p.ID())
					clone := g.Clone()
					clone.ResolveTopOfStack()
					if g.FindPermanent(p.ID()) == nil || len(b.Hand()) != 0 {
						t.Fatal("clone changed original")
					}
					g.ResolveTopOfStack()
					if g.FindPermanent(p.ID()) != nil {
						t.Fatal("victim stayed on battlefield")
					}
					destroy := modes[len(modes)-1] == 1 && mode == 0
					if destroy {
						if len(b.Graveyard()) != 1 || len(b.Hand()) != 0 {
							t.Fatal("destroy mode did not use owner's graveyard")
						}
					} else if len(b.Hand()) != 1 || b.Hand()[0].ID() != victim.ID() || len(b.Graveyard()) != 0 {
						t.Fatal("return did not use owner's hand")
					}
					if len(a.Hand()) != 0 || len(a.Graveyard()) != 1 {
						t.Fatal("spell or victim used wrong zone")
					}
					if len(modes) == 2 && a.prompts != 1 {
						t.Fatal("mode selected at resolution")
					}
				})
			}
		}
	}
}

func TestRONReturnHandIdentityAndLegality(t *testing.T) {
	for _, change := range []string{"control", "depart-return", "absent", "phase", "phase-cycle", "shroud", "regeneration"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronModalGame(0)
			victim := repositoryCard(t, "LEB", "Air Elemental")
			victim.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(victim, b.PlayerID())
			castRONReturn(t, g, a, repositoryCard(t, "LEB", "Unsummon"), p.ID())
			switch change {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "depart-return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(victim, b.PlayerID())
			case "absent":
				g.RemoveFromBattlefield(p)
			case "phase":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "shroud":
				g.AddContinuousEffect(mage.TemporaryKeyword(p.ID(), core.Shroud))
			case "regeneration":
				g.AddRegenerationShield(p.ID())
			}
			g.ResolveTopOfStack()
			succeeds := change == "control" || change == "phase-cycle" || change == "regeneration"
			want := 0
			if succeeds {
				want = 1
			}
			if len(b.Hand()) != want || len(b.Graveyard()) != 0 || len(a.Hand()) != 0 || len(a.Graveyard()) != 1 {
				t.Fatal("wrong zone outcome")
			}
			if change == "phase" {
				g.PhaseIn([]uuid.UUID{p.ID()})
			}
			if !succeeds && change != "absent" && g.FindPermanent(victim.ID()) == nil {
				t.Fatal("illegal target was moved")
			}
		})
	}
}

func TestRONReturnHandCopyAndInstances(t *testing.T) {
	g, a, b := ronModalGame(1)
	firstCard, secondCard := repositoryCard(t, "LEB", "Island"), repositoryCard(t, "LEB", "Island")
	firstCard.SetOwner(b.PlayerID())
	secondCard.SetOwner(b.PlayerID())
	first, second := g.PutOnBattlefield(firstCard, b.PlayerID()), g.PutOnBattlefield(secondCard, b.PlayerID())
	castRONReturn(t, g, a, repositoryCard(t, "LEG", "Active Volcano"), first.ID())
	original := g.StackPeek()
	clone := g.Clone()
	a.targets = []uuid.UUID{second.ID()}
	a.mode = 0
	copied := g.CopyStackObjectDirect(original, a.PlayerID(), true)
	if copied == nil || copied.ModeChoice != 1 || copied.Targets[0] != second.ID() || original.Targets[0] != first.ID() || a.prompts != 1 {
		t.Fatal("copy lost mode or target isolation")
	}
	g.ResolveTopOfStack()
	if g.FindPermanent(second.ID()) != nil || g.FindPermanent(first.ID()) == nil {
		t.Fatal("copy used wrong target")
	}
	g.ResolveTopOfStack()
	if len(b.Hand()) != 2 || len(a.Graveyard()) != 1 {
		t.Fatal("original lost its target")
	}
	clone.ResolveTopOfStack()
	if clone.FindPermanent(second.ID()) == nil || len(clone.GetPlayer(b.PlayerID()).Hand()) != 1 {
		t.Fatal("clone shared copy state")
	}
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(AnyPermanent)], ActionList([PutPermanentIntoItsOwnersHand(Ref_TargetPermanent)])))`)
	x, y := factory(), factory()
	if x.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == y.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
		t.Fatal("factories shared target declaration")
	}
	for _, card := range []mage.Card{x, y} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRONReturnHandRollback(t *testing.T) {
	for _, reason := range []string{"unpaid", "wrong-type", "wrong-subtype", "phased"} {
		t.Run(reason, func(t *testing.T) {
			g, a, b := ronModalGame(0)
			victim := repositoryCard(t, "LEB", "Forest")
			name, set := "Unsummon", "LEB"
			if reason == "unpaid" || reason == "phased" {
				victim = repositoryCard(t, "LEB", "Air Elemental")
			}
			if reason == "wrong-subtype" {
				name, set = "Active Volcano", "LEG"
				a.mode = 1
			}
			p := g.PutOnBattlefield(victim, b.PlayerID())
			if reason == "phased" {
				g.PhaseOut(p.ID())
			}
			card := repositoryCard(t, set, name)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if reason != "unpaid" {
				a.ManaPool().Add(core.Blue, 1)
				a.ManaPool().Add(core.Red, 1)
			}
			beforeBlue, beforeRed := a.ManaPool().Count(core.Blue), a.ManaPool().Count(core.Red)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{p.ID()}, 0); err == nil {
				t.Fatal("illegal cast succeeded")
			}
			if g.StackSize() != 0 || len(a.Hand()) != 1 || len(a.Graveyard()) != 0 || a.ManaPool().Count(core.Blue) != beforeBlue || a.ManaPool().Count(core.Red) != beforeRed {
				t.Fatal("failed cast changed resources")
			}
		})
	}
}

func TestRONReturnHandInvalid(t *testing.T) {
	for _, body := range []string{
		`PutPermanentIntoItsOwnersHand`, `PutPermanentIntoItsOwnersHand()`, `PutPermanentIntoItsOwnersHand(Ref_TargetPermanent, Ref_TargetPermanent)`,
		`PutPermanentIntoItsOwnersHand([Ref_TargetPermanent])`, `PutPermanentIntoItsOwnersHand(Ref_TargetPermanent())`,
		`PutPermanentIntoItsOwnersHand(Ref_TargetPlayer)`, `PutPermanentIntoItsOwnersHand(ThisPermanent)`, `PutPermanentIntoItsOwnersHand(HostPermanent)`, `PutPermanentIntoItsOwnersHand(Trigger_ThatPermanent)`,
	} {
		source := `Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Instant], Subtypes: []), Rules: [SpellActions(Targeted([TargetPermanent(AnyPermanent)], ActionList([` + body + `])))])`
		nodes, err := Parse(strings.NewReader(source), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
	for _, rule := range []string{
		`SpellActions(ActionList([PutPermanentIntoItsOwnersHand(Ref_TargetPermanent)]))`,
		`SpellActions(Targeted([TargetPlayer(AnyPlayer)], ActionList([PutPermanentIntoItsOwnersHand(Ref_TargetPermanent)])))`,
		`SpellActions(Modal_ChooseOne([Targeted([TargetPermanent(AnyPermanent)], ActionList([])), ActionList([PutPermanentIntoItsOwnersHand(Ref_TargetPermanent)])]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Instant], Subtypes: []), Rules: [`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONReturnHandInventory(t *testing.T) {
	source := `Card(Name: "Return", Typeline: (Supertypes: [], Cardtypes: [Instant], Subtypes: []), Rules: [SpellActions(Targeted([TargetPermanent(AnyPermanent)], ActionList([PutPermanentIntoItsOwnersHand(Ref_TargetPermanent)])))])`
	report, err := InspectFS(fstest.MapFS{"return.ron": {Data: []byte(source)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 1 {
		t.Fatal("return declaration not compiled")
	}
	found := false
	for _, s := range report.Signatures {
		if s.Name == "Ref_TargetPermanent" {
			found = true
			if s.Role != "permanent_reference" {
				t.Fatal("return read has wrong role")
			}
		}
	}
	if !found {
		t.Fatal("return read missing")
	}
}

func TestRONReturnHandSourceAndHostContexts(t *testing.T) {
	for _, change := range []string{"current-host", "departed-source", "returned-host", "returned-source"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			nodes, err := Parse(strings.NewReader(`Card(Name: "Return Aura", Typeline: (Supertypes: [], Cardtypes: [Enchantment], Subtypes: [Aura]), Rules: [EnchantPermanent(IsCardtype(Creature)), Activated(SacrificePermanent(ThisPermanent), ActionList([PutPermanentIntoItsOwnersHand(HostPermanent)]))])`), "return-aura.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			aura := factory()
			aura.SetOwner(a.PlayerID())
			if err := mage.ValidateCardDefinition(aura.Name(), aura); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(aura, a.PlayerID())
			firstCard, secondCard := repositoryCard(t, "LEB", "Air Elemental"), repositoryCard(t, "LEB", "Air Elemental")
			firstCard.SetOwner(b.PlayerID())
			secondCard.SetOwner(b.PlayerID())
			first, second := g.PutOnBattlefield(firstCard, b.PlayerID()), g.PutOnBattlefield(secondCard, b.PlayerID())
			g.Attach(source.ID(), first.ID())
			if change == "current-host" {
				g.Attach(source.ID(), second.ID())
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if len(b.Hand()) != 0 || g.FindPermanent(source.ID()) != nil || len(a.Graveyard()) != 1 {
				t.Fatal("sacrifice or return timing incorrect")
			}
			if change == "returned-host" {
				g.RemoveFromBattlefield(first)
				g.PutOnBattlefield(firstCard, b.PlayerID())
			}
			if change == "returned-source" {
				a.RemoveFromGraveyard(aura.ID())
				g.PutOnBattlefield(aura, a.PlayerID())
				g.Attach(aura.ID(), second.ID())
			}
			g.ResolveStack()
			if change == "returned-host" {
				if len(b.Hand()) != 0 || g.FindPermanent(firstCard.ID()) == nil {
					t.Fatal("old attachment followed returned host")
				}
			} else {
				wanted := firstCard.ID()
				if change == "current-host" {
					wanted = secondCard.ID()
				}
				if len(b.Hand()) != 1 || b.Hand()[0].ID() != wanted {
					t.Fatal("attachment LKI used wrong host")
				}
			}
			if change == "returned-source" && g.FindPermanent(second.ID()) == nil {
				t.Fatal("old ability followed new source attachment")
			}
		})
	}
	for _, context := range []string{"activation", "entry-trigger"} {
		t.Run(context, func(t *testing.T) {
			rule := `Activated(TapPermanent(ThisPermanent), ActionList([PutPermanentIntoItsOwnersHand(ThisPermanent)]))`
			if context == "entry-trigger" {
				rule = `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)), ActionList([PutPermanentIntoItsOwnersHand(ThisPermanent)]))`
			}
			factory := permanentScopeFactory(t, "Artifact", rule)
			g, a, _ := ronGame()
			card := factory()
			card.SetOwner(a.PlayerID())
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(card, a.PlayerID())
			if context == "activation" {
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if !source.Tapped {
					t.Fatal("tap cost not paid")
				}
			}
			g.ResolveStack()
			if len(a.Hand()) != 1 || a.Hand()[0].ID() != card.ID() || g.FindPermanent(source.ID()) != nil {
				t.Fatal("source reference did not return source")
			}
		})
	}
}
