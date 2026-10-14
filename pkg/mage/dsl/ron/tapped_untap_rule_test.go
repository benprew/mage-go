package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTappedUntapRuleGremlins(t *testing.T) {
	for _, change := range []string{"none", "already-tapped", "source-untap-before", "source-retap-before", "source-return-before", "source-phase-before", "illegal", "target-return-before", "copy", "source-untap-after", "source-phase-cycle-after", "target-phase-cycle-after", "control", "remove-abilities", "stun"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "ATQ", "Phyrexian Gremlins")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Black, 1)
			a.ManaPool().Add(core.Colorless, 2)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			source := g.MutablePermanent(card.ID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			tc := mage.NewArtifact("Subject", "")
			tc.SetOwner(b.PlayerID())
			target := g.PutOnBattlefield(tc, b.PlayerID())
			creature := g.PutOnBattlefield(mage.NewCreature("Illegal", "", 1, 1), b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{creature.ID()}); err == nil {
				t.Fatal("nonartifact accepted")
			}
			if g.FindPermanent(source.ID()).Tapped {
				t.Fatal("illegal activation paid")
			}
			if change == "already-tapped" {
				g.TapPermanent(target)
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{target.ID()}); err != nil {
				t.Fatal(err)
			}
			if target.Tapped && change != "already-tapped" {
				t.Fatal("tap occurred at announcement")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{target.ID()}); err == nil {
				t.Fatal("tap cost paid twice")
			}
			switch change {
			case "source-untap-before", "source-retap-before":
				g.UntapPermanent(g.FindPermanent(source.ID()))
				if change == "source-retap-before" {
					g.TapPermanent(g.FindPermanent(source.ID()))
				}
			case "source-return-before":
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				a.RemoveFromHand(card.ID())
				g.TapPermanent(g.PutOnBattlefield(card, a.PlayerID()))
			case "source-phase-before":
				g.PhaseOut(source.ID())
			case "illegal":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.GrantKeywordModification(mage.This, core.Shroud)), target.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "target-return-before":
				g.BouncePermanentToHand(g.FindPermanent(target.ID()))
				b.RemoveFromHand(tc.ID())
				target = g.PutOnBattlefield(tc, b.PlayerID())
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			g.ResolveStack()
			clone.ResolveStack()
			blocked := change != "source-untap-before" && change != "source-return-before" && change != "source-phase-before" && change != "illegal" && change != "target-return-before"
			switch change {
			case "source-untap-after":
				g.UntapPermanent(g.FindPermanent(source.ID()))
				g.TapPermanent(g.FindPermanent(source.ID()))
				blocked = false
			case "source-phase-cycle-after":
				g.PhaseIn(g.PhaseOut(source.ID()))
				blocked = false
			case "target-phase-cycle-after":
				g.PhaseIn(g.PhaseOut(target.ID()))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "remove-abilities":
				g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
				g.AddContinuousEffect(mage.RemoveAllAbilities(target.ID(), core.Indefinite))
			case "stun":
				g.MutablePermanent(source.ID()).Counters[core.Stun] = 1
				if g.UntapPermanent(g.FindPermanent(source.ID())) {
					t.Fatal("stun")
				}
			}
			g.SetActivePlayerIndex(1)
			g.RunStepWithPriority(core.Untap)
			if g.FindPermanent(target.ID()).Tapped != blocked {
				t.Fatalf("blocked %v, want %v", g.FindPermanent(target.ID()).Tapped, blocked)
			}
			clone.SetActivePlayerIndex(1)
			clone.RunStepWithPriority(core.Untap)
			cloneBlocked := change != "source-untap-before" && change != "source-return-before" && change != "source-phase-before" && change != "illegal" && change != "target-return-before"
			if clone.FindPermanent(target.ID()).Tapped != cloneBlocked {
				t.Fatal("clone result")
			}
		})
	}
}

func TestRONTappedUntapRuleOptionalAndSimultaneous(t *testing.T) {
	a := &ronStoragePlayer{ronCounterPayer: &ronCounterPayer{BasePlayer: mage.NewBasePlayer("A")}}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	card := repositoryCard(t, "ATQ", "Phyrexian Gremlins")
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	source.RevokeBaseAttr(core.AttrSummonSick)
	target := g.PutOnBattlefield(mage.NewArtifact("Target", ""), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{target.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.RunStepWithPriority(core.Untap)
	if !g.FindPermanent(source.ID()).Tapped || !g.FindPermanent(target.ID()).Tapped || a.untapCalls != 1 {
		t.Fatal("declined untap")
	}
	a.untap = true
	g.RunStepWithPriority(core.Untap)
	if g.FindPermanent(source.ID()).Tapped || !g.FindPermanent(target.ID()).Tapped {
		t.Fatal("restriction must be read before simultaneous untap")
	}
	g.RunStepWithPriority(core.Untap)
	if g.FindPermanent(target.ID()).Tapped {
		t.Fatal("expired restriction")
	}
}

func TestRONTappedUntapRuleInvalid(t *testing.T) {
	good := `CreatePermanentRuleEffectUntil(Ref_TargetPermanent,[DoesntUntapDuringControllersUntap],ForAsLongAsPermanentRemainsTapped(ThisPermanent))`
	for _, bad := range []string{
		strings.Replace(good, `DoesntUntapDuringControllersUntap`, `DoesntUntapDuringControllersUntap()`, 1),
		strings.Replace(good, `[DoesntUntapDuringControllersUntap]`, `[]`, 1),
		strings.Replace(good, `[DoesntUntapDuringControllersUntap]`, `[DoesntUntapDuringControllersUntap,CantBlock]`, 1),
		strings.Replace(good, `ForAsLongAsPermanentRemainsTapped(ThisPermanent)`, `UntilEndOfTurn`, 1),
		strings.Replace(good, `ForAsLongAsPermanentRemainsTapped(ThisPermanent)`, `ForAsLongAsPermanentRemainsTapped(Ref_TargetPermanent)`, 1),
		strings.Replace(good, `ForAsLongAsPermanentRemainsTapped(ThisPermanent)`, `ForAsLongAsPermanentRemainsTapped(ThisPermanent())`, 1),
		strings.Replace(good, `ForAsLongAsPermanentRemainsTapped(ThisPermanent)`, `ForAsLongAsPermanentRemainsTapped(ThisPermanent,You)`, 1),
		strings.Replace(good, `Ref_TargetPermanent`, `ThePermanentTappedThisWay`, 1),
		strings.Replace(good, `Ref_TargetPermanent`, `Trigger_ThatPermanent`, 1),
		strings.Replace(good, `Ref_TargetPermanent`, `You`, 1),
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),Targeted([TargetPermanent(AnyPermanent)],ActionList([` + bad + `])))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
	for _, rule := range []string{
		`SpellActions(Targeted([TargetPermanent(AnyPermanent)],ActionList([` + good + `])))`,
		`AsPermanentEnters(ThisPermanent,[CreatePermanentRuleEffectUntil(ThisPermanent,[DoesntUntapDuringControllersUntap],ForAsLongAsPermanentRemainsTapped(ThisPermanent))])`,
		`Activated(CreatePermanentRuleEffectUntil(ThisPermanent,[DoesntUntapDuringControllersUntap],ForAsLongAsPermanentRemainsTapped(ThisPermanent)),ActionList([GainLife(Integer(1))]))`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[` + rule + `])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONTappedUntapRuleInventory(t *testing.T) {
	node := chosenDiscardRecord(t, "ATQ", "Phyrexian Gremlins")
	entries := map[string]*InventorySignature{}
	walkInventory(node, "Phyrexian Gremlins", "Card", "card", "collection", false, entries)
	subject, source, operation := false, false, false
	for _, entry := range entries {
		if entry.Name == "Ref_TargetPermanent" && strings.Contains(entry.Context, "CreatePermanentRuleEffectUntil") {
			subject = entry.Role == "permanent_reference"
		}
		if entry.Name == "ThisPermanent" && strings.Contains(entry.Context, "ForAsLongAsPermanentRemainsTapped") {
			source = entry.Role == "permanent_reference"
		}
		if entry.Name == "DoesntUntapDuringControllersUntap" {
			operation = entry.Role == "permanent_rule_operation"
		}
	}
	if !subject || !source || !operation {
		t.Fatalf("roles: subject %v source %v operation %v", subject, source, operation)
	}
}
