package ron

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONIndefiniteCharacteristicDeclarations(t *testing.T) {
	for _, modification := range []string{`SetColor(SimpleColorList([Green]))`, `SetColor(SimpleColorList([]))`, `AddAbility([Flying])`, `AddAbility([Flying, FirstStrike])`, `AddAbility([Landwalk(IsLandType(Island))])`} {
		factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]), Targeted([TargetPermanent(AnyPermanent)], ActionList([CreatePermanentLayerEffect(Ref_TargetPermanent, [`+modification+`])])))`)
		first, second := factory(), factory()
		if err := mage.ValidateCardDefinition(first.Name(), first); err != nil {
			t.Fatal(err)
		}
		if first.Abilities()[0] == second.Abilities()[0] {
			t.Fatal("shared ability")
		}
	}
}

func TestRONIndefiniteCharacteristicInvalid(t *testing.T) {
	for _, body := range []string{
		`CreatePermanentLayerEffect(Ref_TargetPermanent, [SetColor(SimpleColorList([Green]))])`,
		`CreatePermanentLayerEffect(ThisPermanent, [SetColor(TheChosenColor)])`,
		`CreatePermanentLayerEffect(ThisPermanent, [SetColor(SimpleColorList([Colorless]))])`,
		`CreatePermanentLayerEffect(ThisPermanent, [SetColor(SimpleColorList([Green, Green]))])`,
		`CreatePermanentLayerEffect(ThisPermanent, [SetColor(SimpleColorList([Green()]))])`,
		`CreatePermanentLayerEffect(ThisPermanent, [AddAbility([])])`,
		`CreatePermanentLayerEffect(ThisPermanent, [AddAbility([Flying()])])`,
		`CreatePermanentLayerEffect(ThisPermanent, [AddAbility([Flying, Unknown])])`,
		`CreatePermanentLayerEffect(ThisPermanent, [AddColor(SimpleColorList([Green]))])`,
	} {
		src := fmt.Sprintf(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [Activated(PayMana([]), ActionList([%s]))])`, body)
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestRONAislingIndefiniteColor(t *testing.T) {
	for _, blocking := range []bool{false, true} {
		for _, state := range []string{"normal", "source-left", "returned", "phased", "phase-cycle", "control", "copy"} {
			t.Run(fmt.Sprint(blocking)+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, "LEG", "Aisling Leprechaun")
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				otherCard := mage.NewCreature("Other", "{R}{U}", 2, 8)
				otherCard.SetOwner(b.PlayerID())
				other := g.PutOnBattlefield(otherCard, b.PlayerID())
				event := core.GameEvent{Type: core.EvtCreatureBlockedBy, SourceID: source.ID(), TargetID: other.ID(), PlayerID: a.PlayerID()}
				if blocking {
					event.Type = core.EvtDeclaredBlocker
				}
				g.FireEvent(event)
				g.PutTriggersOnStack()
				if g.StackSize() != 1 || slices.Contains(other.Colors(), core.Green) {
					t.Fatal("wrong trigger timing")
				}
				switch state {
				case "source-left":
					g.BouncePermanentToHand(source)
				case "returned":
					g.BouncePermanentToHand(other)
					b.RemoveFromHand(otherCard.ID())
					g.PutOnBattlefield(otherCard, b.PlayerID())
				case "phased":
					g.PhaseOut(other.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(other.ID()))
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: other.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				}
				clone := g.Clone()
				for _, game := range []*mage.Game{g, clone} {
					game.ResolveStack()
					if state == "phased" {
						game.PhaseIn([]uuid.UUID{other.ID()})
					}
					p := game.FindPermanent(other.ID())
					changed := state != "returned" && state != "phased"
					if slices.Contains(p.Colors(), core.Green) != changed || slices.Contains(p.Colors(), core.Red) == changed || slices.Contains(p.Colors(), core.Blue) == changed {
						t.Fatal("color setting or identity failed")
					}
					game.RunStepWithPriority(core.Cleanup)
					if slices.Contains(game.FindPermanent(other.ID()).Colors(), core.Green) != changed {
						t.Fatal("color expired at cleanup")
					}
					if changed {
						if err := mage.ApplyEffect(game, mage.Until(core.EndOfTurn, mage.SetColors(mage.This, core.White)), other.ID(), b.PlayerID(), nil); err != nil {
							t.Fatal(err)
						}
						if !slices.Contains(game.FindPermanent(other.ID()).Colors(), core.White) || slices.Contains(game.FindPermanent(other.ID()).Colors(), core.Green) {
							t.Fatal("later timestamp lost")
						}
						game.RunStepWithPriority(core.Cleanup)
						if !slices.Contains(game.FindPermanent(other.ID()).Colors(), core.Green) {
							t.Fatal("indefinite color lost")
						}
					}
				}
			})
		}
	}
}

func TestRONCocoonCompleteLifecycle(t *testing.T) {
	g, a, b := ronGame()
	hostCard := mage.NewCreature("Host", "", 2, 8)
	hostCard.SetOwner(a.PlayerID())
	host := g.PutOnBattlefield(hostCard, a.PlayerID())
	wrong := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 8), b.PlayerID())
	card := repositoryCard(t, "LEG", "Cocoon")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Green, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{wrong.ID()}, 0); err == nil {
		t.Fatal("opponent host accepted")
	}
	if a.ManaPool().TotalMana() != 1 || len(a.Hand()) != 1 {
		t.Fatal("illegal cast spent resources")
	}
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	aura := g.FindPermanent(card.ID())
	if aura == nil || !g.FindPermanent(host.ID()).Tapped || aura.Counters[core.Pupa] != 3 {
		t.Fatal("entry instructions failed")
	}
	for i := range 3 {
		g.RunStepWithPriority(core.Untap)
		if !g.FindPermanent(host.ID()).Tapped {
			t.Fatal("host untapped with pupa")
		}
		g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
		g.ResolveStack()
		if g.FindPermanent(aura.ID()) == nil || g.FindPermanent(aura.ID()).Counters[core.Pupa] != (2-i) || g.FindPermanent(host.ID()).HasKeyword(core.Flying) {
			t.Fatal("counter payment failed")
		}
	}
	g.RunStepWithPriority(core.Untap)
	if g.FindPermanent(host.ID()).Tapped {
		t.Fatal("host stayed tapped without pupa")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
	g.ResolveStack()
	if g.FindPermanent(aura.ID()) == nil {
		t.Fatal("opponent upkeep triggered")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	clone := g.Clone()
	for _, game := range []*mage.Game{g, clone} {
		game.ResolveStack()
		p := game.FindPermanent(host.ID())
		if game.FindPermanent(aura.ID()) != nil || !p.HasKeyword(core.Flying) || p.Counters[core.P1P1] != 1 {
			t.Fatal("sacrifice lost host grant")
		}
		game.RunStepWithPriority(core.Cleanup)
		game.PhaseIn(game.PhaseOut(host.ID()))
		if !game.FindPermanent(host.ID()).HasKeyword(core.Flying) {
			t.Fatal("grant lost across cleanup or phasing")
		}
	}
	g.BouncePermanentToHand(g.FindPermanent(host.ID()))
	a.RemoveFromHand(hostCard.ID())
	g.PutOnBattlefield(hostCard, a.PlayerID())
	if g.FindPermanent(host.ID()).HasKeyword(core.Flying) || !clone.FindPermanent(host.ID()).HasKeyword(core.Flying) {
		t.Fatal("returned host or clone inherited mutation")
	}
}

func TestRONAislingNativeCombat(t *testing.T) {
	for _, blocking := range []bool{false, true} {
		g, a, b := ronGame()
		source := g.PutOnBattlefield(repositoryCard(t, "LEG", "Aisling Leprechaun"), a.PlayerID())
		other := g.PutOnBattlefield(mage.NewCreature("Other", "{R}", 0, 8), b.PlayerID())
		source.RevokeBaseAttr(core.AttrSummonSick)
		other.RevokeBaseAttr(core.AttrSummonSick)
		if blocking {
			g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{other.ID()})
			g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: source.ID(), AttackerID: other.ID()}})
		} else {
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{source.ID()})
			g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: other.ID(), AttackerID: source.ID()}})
		}
		g.ResolveStack()
		if !slices.Equal(g.FindPermanent(other.ID()).Colors(), []core.Color{core.Green}) {
			t.Fatal("native blocking did not set green")
		}
	}
}

func TestRONIndefiniteKeywordIdentity(t *testing.T) {
	for _, state := range []string{"normal", "returned", "phased", "phase-cycle", "source-left", "control", "copy"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(AnyPermanent)],ActionList([CreatePermanentLayerEffect(Ref_TargetPermanent,[AddAbility([Flying, FirstStrike])])])))`)
			source := g.PutOnBattlefield(factory(), a.PlayerID())
			card := mage.NewCreature("Recipient", "", 2, 8)
			card.SetOwner(b.PlayerID())
			target := g.PutOnBattlefield(card, b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
				t.Fatal(err)
			}
			switch state {
			case "returned":
				g.BouncePermanentToHand(target)
				b.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, b.PlayerID())
			case "phased":
				g.PhaseOut(target.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(target.ID()))
			case "source-left":
				g.BouncePermanentToHand(source)
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: target.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			for _, game := range []*mage.Game{g, clone} {
				game.ResolveStack()
				if state == "phased" {
					game.PhaseIn([]uuid.UUID{target.ID()})
				}
				granted := state != "returned" && state != "phased"
				for _, keyword := range []core.Attr{core.Flying, core.FirstStrike} {
					if game.FindPermanent(target.ID()).HasKeyword(keyword) != granted {
						t.Fatal("grant followed invalid identity")
					}
				}
				if granted {
					if err := mage.ApplyEffect(game, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), target.ID(), b.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					if game.FindPermanent(target.ID()).HasKeyword(core.Flying) {
						t.Fatal("later removal lost")
					}
					game.RunStepWithPriority(core.Cleanup)
					if !game.FindPermanent(target.ID()).HasKeyword(core.Flying) {
						t.Fatal("indefinite grant expired")
					}
				}
			}
		})
	}
}

func TestRONIndefiniteFixedColorLists(t *testing.T) {
	for _, tc := range []struct {
		list   string
		colors []core.Color
	}{
		{`[]`, nil}, {`[White, Blue, Black, Red, Green]`, []core.Color{core.White, core.Blue, core.Black, core.Red, core.Green}},
	} {
		g, a, b := ronGame()
		factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(AnyPermanent)],ActionList([CreatePermanentLayerEffect(Ref_TargetPermanent,[SetColor(SimpleColorList(`+tc.list+`))])])))`)
		source := g.PutOnBattlefield(factory(), a.PlayerID())
		target := g.PutOnBattlefield(mage.NewCreature("Recipient", "{R}", 2, 8), b.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if !slices.Equal(g.FindPermanent(target.ID()).Colors(), tc.colors) {
			t.Fatal("wrong fixed colors")
		}
		g.RunStepWithPriority(core.Cleanup)
		if !slices.Equal(g.FindPermanent(target.ID()).Colors(), tc.colors) {
			t.Fatal("fixed colors expired")
		}
	}
}
