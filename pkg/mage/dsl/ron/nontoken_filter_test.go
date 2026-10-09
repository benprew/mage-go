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

// These probes retain filters because other card capabilities remain unsupported.
func TestRONNontokenFilterCardTargets(t *testing.T) {
	for _, card := range []struct{ set, name string }{{"ATQ", "Bronze Tablet"}, {"DRK", "Dance of Many"}} {
		record := staticBaseRecord(t, card.set, card.name)
		rule := record.Fields["Rules"].Args[2]
		if card.name == "Dance of Many" {
			rule = record.Fields["Rules"].Args[0]
		}
		targeted := rule.Args[1]
		filter := targeted.Args[0].Args[0].Args[0]
		if card.name == "Bronze Tablet" {
			filter = filter.Args[0].Args[0]
		}
		targeted.Args[0].Args[0].Args[0] = filter
		targeted.Args[1] = nameScopeNode(t, `ActionList([TapPermanent(Ref_TargetPermanent)])`)
		rule.Args[1] = targeted
		factory := staticBaseSubrule(t, record, rule)
		for _, state := range []string{"live", "token", "returned", "phased", "phase-cycle", "control", "source-left", "clone", "copy"} {
			t.Run(card.name+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				targetCard := mage.NewCreature("Subject", "", 2, 3)
				if state == "token" {
					targetCard = mage.NewToken("Subject", 2, 3, []core.CardType{core.TypeCreature}, nil)
				}
				targetCard.SetOwner(b.PlayerID())
				target := g.PutOnBattlefield(targetCard, b.PlayerID())
				sourceCard := factory()
				sourceCard.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(sourceCard, a.PlayerID())
				if card.name == "Bronze Tablet" {
					a.ManaPool().Add(core.Colorless, 4)
					err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()})
					if state == "token" {
						if err == nil || a.ManaPool().TotalMana() != 4 || g.StackSize() != 0 || g.FindPermanent(source.ID()).Tapped {
							t.Fatal("illegal token target spent resources")
						}
						return
					}
					if err != nil {
						t.Fatal(err)
					}
					if a.ManaPool().TotalMana() != 0 || !g.FindPermanent(source.ID()).Tapped {
						t.Fatal("activation cost not paid")
					}
				} else {
					g.PutTriggersOnStack()
					if state == "token" {
						if g.StackSize() != 0 {
							t.Fatal("token-only battlefield supplied creature target")
						}
						return
					}
					if g.StackSize() != 1 {
						t.Fatal("missing entry trigger")
					}
				}
				if g.FindPermanent(target.ID()).Tapped {
					t.Fatal("effect ran at announcement")
				}
				switch state {
				case "returned":
					g.BouncePermanentToHand(target)
					b.RemoveFromHand(targetCard.ID())
					g.PutOnBattlefield(targetCard, b.PlayerID())
				case "phased":
					g.PhaseOut(target.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(target.ID()))
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: target.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				case "source-left":
					g.BouncePermanentToHand(source)
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				}
				clone := g.Clone()
				if state == "clone" {
					clone.PhaseOut(target.ID())
				}
				g.ResolveStack()
				if state == "phased" {
					g.PhaseIn([]uuid.UUID{target.ID()})
				}
				want := state != "returned" && state != "phased"
				if g.FindPermanent(target.ID()).Tapped != want {
					t.Fatal("wrong target identity or legality")
				}
				if clone.FindPermanentIncludingPhased(target.ID()).Tapped {
					t.Fatal("original changed pending clone")
				}
				clone.ResolveStack()
				if clone.FindPermanentIncludingPhased(target.ID()).Tapped != (want && state != "clone") {
					t.Fatal("clone lost target identity or legality")
				}
			})
		}
	}
}

func TestRONNontokenFilterSacrifice(t *testing.T) {
	record := staticBaseRecord(t, "LEB", "Lich")
	filter := record.Fields["Rules"].Args[3].Args[1].Args[0].Args[0].Args[0].Args[1]
	for _, state := range []string{"pay", "token", "phased", "opponent", "mana"} {
		t.Run(state, func(t *testing.T) {
			rule := nameScopeNode(t, `Activated(And([PayMana([ManaCostGeneric(1)]),SacrificeNumberPermanents(Integer(2),AnyPermanent)]),ActionList([GainLife(Integer(1))]))`)
			rule.Args[0].Args[0].Args[1].Args[1] = nameScopeNode(t, `And([IsNonToken,IsCardtype(Creature)])`)
			rule.Args[0].Args[0].Args[1].Args[1].Args[0].Args[0] = filter
			factory := staticBaseSubrule(t, record, rule)
			g, a, b := ronGame()
			source := g.PutOnBattlefield(factory(), a.PlayerID())
			firstCard := mage.NewCreature("First", "", 2, 3)
			firstCard.SetOwner(b.PlayerID())
			first := g.PutOnBattlefield(firstCard, a.PlayerID())
			secondCard := mage.NewCreature("Second", "", 2, 3)
			if state == "token" {
				secondCard = mage.NewToken("Second", 2, 3, []core.CardType{core.TypeCreature}, nil)
			}
			secondCard.SetOwner(b.PlayerID())
			second := g.PutOnBattlefield(secondCard, a.PlayerID())
			if state == "phased" {
				g.PhaseOut(second.ID())
			}
			if state == "opponent" {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: second.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			if state != "mana" {
				a.ManaPool().Add(core.Colorless, 1)
			}
			err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil)
			if state != "pay" {
				if err == nil || g.StackSize() != 0 || g.FindPermanent(first.ID()) == nil || g.FindPermanentIncludingPhased(second.ID()) == nil || len(b.Graveyard()) != 0 {
					t.Fatal("failed sacrifice payment changed state")
				}
				if state != "mana" && a.ManaPool().TotalMana() != 1 {
					t.Fatal("failed sacrifice spent mana")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(b.Graveyard()) != 2 || a.Life() != 20 || a.ManaPool().TotalMana() != 0 {
				t.Fatal("wrong cost timing or owner")
			}
			clone := g.Clone()
			g.ResolveStack()
			if a.Life() != 21 || clone.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("wrong resolution or clone isolation")
			}
		})
	}
}

func TestRONNontokenFilterCollectionsAndLayers(t *testing.T) {
	for _, context := range []string{"collection", "static", "resolving"} {
		t.Run(context, func(t *testing.T) {
			rule := `Activated(PayMana([]),ActionList([TapEachPermanent(And([IsNonToken,IsCardtype(Creature)]))]))`
			if context == "static" {
				rule = `EachPermanentLayerEffect(And([IsNonToken,IsCardtype(Creature)]),[AdjustPT(1,1)])`
			}
			if context == "resolving" {
				rule = `Activated(PayMana([]),ActionList([If(PermanentPassesFilter(ThisPermanent,IsNonToken),[GainLife(Integer(2))])]))`
			}
			factory := permanentScopeFactory(t, "Artifact", rule)
			g, a, b := ronGame()
			source := g.PutOnBattlefield(factory(), a.PlayerID())
			normal := g.PutOnBattlefield(mage.NewCreature("Same", "", 2, 3), b.PlayerID())
			token := g.PutOnBattlefield(mage.NewToken("Same", 2, 3, []core.CardType{core.TypeCreature}, nil), b.PlayerID())
			if context == "static" {
				checkConditionalPT(t, g, normal.ID(), 3, 4)
				checkConditionalPT(t, g, token.ID(), 2, 3)
				g.PhaseOut(source.ID())
				g.ApplyContinuousEffects()
				checkConditionalPT(t, g, normal.ID(), 2, 3)
				g.PhaseIn([]uuid.UUID{source.ID()})
				g.ApplyContinuousEffects()
				checkConditionalPT(t, g, normal.ID(), 3, 4)
				return
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if context == "collection" {
				late := g.PutOnBattlefield(mage.NewCreature("Late", "", 2, 3), a.PlayerID())
				phased := g.PutOnBattlefield(mage.NewCreature("Phased", "", 2, 3), a.PlayerID())
				g.PhaseOut(phased.ID())
				g.ResolveStack()
				if !g.FindPermanent(normal.ID()).Tapped || !g.FindPermanent(late.ID()).Tapped || g.FindPermanent(token.ID()).Tapped || g.FindPermanentIncludingPhased(phased.ID()).Tapped {
					t.Fatal("wrong current collection")
				}
			} else {
				g.BouncePermanentToHand(source)
				g.ResolveStack()
				if a.Life() != 22 {
					t.Fatal("departure lost nontoken LKI")
				}
			}
		})
	}
}

func TestRONNontokenFilterDeclarations(t *testing.T) {
	for _, filter := range []string{`IsNonToken()`, `IsNonToken(Integer(1))`, `IsNonToken(AnyPermanent,AnyPermanent)`} {
		for _, rule := range []string{
			`SpellActions(Targeted([TargetPermanent(%s)],ActionList([TapPermanent(Ref_TargetPermanent)])))`,
			`SpellActions(ActionList([TapEachPermanent(%s)]))`,
			`Activated(SacrificeNumberPermanents(Integer(1),%s),ActionList([GainLife(Integer(1))]))`,
			`EachPermanentLayerEffect(%s,[AdjustPT(1,1)])`,
			`Activated(PayMana([]),ActionList([If(PermanentPassesFilter(ThisPermanent,%s),[GainLife(Integer(1))])]))`,
		} {
			kind := "Artifact"
			if strings.HasPrefix(rule, "SpellActions") {
				kind = "Instant"
			}
			text := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[` + kind + `],Subtypes:[]),Rules:[` + fmt.Sprintf(rule, filter) + `])`
			if registry, err := LoadFS(fstest.MapFS{"bad.ron": {Data: []byte(text)}}, "*.ron"); err == nil || registry != nil {
				t.Fatal("accepted invalid filter", text)
			} else if !strings.Contains(err.Error(), "IsNonToken") {
				t.Fatal("invalid declaration failed for another reason", err)
			}
		}
	}
	for _, rule := range []string{`Activated(IsNonToken,ActionList([GainLife(Integer(1))]))`, `Activated(PayMana([]),ActionList([IsNonToken]))`, `SpellActions(Targeted([TargetPlayer(IsNonToken)],ActionList([GainLife(Integer(1))])))`} {
		kind := "Artifact"
		if strings.HasPrefix(rule, "SpellActions") {
			kind = "Instant"
		}
		text := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[` + kind + `],Subtypes:[]),Rules:[` + rule + `])`
		if registry, err := LoadFS(fstest.MapFS{"bad.ron": {Data: []byte(text)}}, "*.ron"); err == nil || registry != nil {
			t.Fatal("accepted wrong role", text)
		} else if !strings.Contains(err.Error(), "IsNonToken") {
			t.Fatal("wrong role failed for another reason", err)
		}
	}
	text := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([TapEachPermanent(IsNonToken)]))])`
	report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(text)}}, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal("inventory", err, report.Issues)
	}
	found := false
	for _, sig := range report.Signatures {
		if sig.Name == "IsNonToken" && sig.Role == "permanent_filter" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing permanent filter role")
	}
}
