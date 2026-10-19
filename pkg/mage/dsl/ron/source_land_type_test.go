package ron

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONSourceLandTypeGaea(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, change := range []string{"none", "source-phase-before", "source-cycle-before", "source-return-before", "source-exit-before", "target-phase-before", "target-cycle-before", "target-return-before", "shroud", "source-phase-after", "source-cycle-after", "source-return-after", "source-control", "source-abilities", "source-untap", "target-cycle-after", "target-return-after", "target-phased-source-exit", "copy", "second-instance", "cleanup"} {
			t.Run(set+"/"+change, func(t *testing.T) {
				record := staticBaseRecord(t, set, "Gaea's Liege")
				factory := staticBaseSubrule(t, record, record.Fields["Rules"].Args[0], record.Fields["Rules"].Args[2])
				g, a, b := ronGame()
				g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID())
				g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), b.PlayerID())
				card := factory()
				card.SetOwner(a.PlayerID())
				castRONSpell(t, g, a, card)
				pendingCast := g.Clone()
				pendingCast.ResolveStack()
				if g.FindPermanent(card.ID()) != nil {
					t.Fatal("clone resolved original cast")
				}
				g.ResolveStack()
				source := g.FindPermanent(card.ID())
				source.RevokeBaseAttr(core.AttrSummonSick)
				targetCard := mage.NewLand("Snow land", mage.WithSuperTypes(core.SuperSnow), mage.WithSubTypes("Island", "Desert"))
				targetCard.SetOwner(b.PlayerID())
				target := g.PutOnBattlefield(targetCard, b.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{source.ID()}); err == nil || source.Tapped {
					t.Fatal("illegal target paid tap cost")
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{target.ID()}); err != nil {
					t.Fatal(err)
				}
				if !g.FindPermanent(source.ID()).Tapped || target.HasSubType("Forest") {
					t.Fatal("activation timing")
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{target.ID()}); err == nil || g.StackSize() != 1 {
					t.Fatal("tapped source activated")
				}
				switch change {
				case "source-phase-before":
					g.PhaseOut(source.ID())
				case "source-cycle-before":
					g.PhaseIn(g.PhaseOut(source.ID()))
				case "source-return-before", "source-exit-before":
					g.RemoveFromBattlefield(source)
					if change == "source-return-before" {
						g.PutOnBattlefield(card, a.PlayerID())
					}
				case "target-phase-before":
					g.PhaseOut(target.ID())
				case "target-cycle-before":
					g.PhaseIn(g.PhaseOut(target.ID()))
				case "target-return-before":
					g.RemoveFromBattlefield(target)
					target = g.PutOnBattlefield(targetCard, b.PlayerID())
				case "shroud":
					g.MutablePermanent(target.ID()).GrantBaseAttr(core.Shroud)
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				}
				pending := g.Clone()
				pending.ResolveStack()
				if g.FindPermanentIncludingPhased(target.ID()).HasSubType("Forest") {
					t.Fatal("clone changed original")
				}
				g.ResolveStack()
				if change == "target-phase-before" {
					g.PhaseIn([]uuid.UUID{target.ID()})
				}
				active := change != "source-return-before" && change != "source-exit-before" && change != "target-phase-before" && change != "target-return-before" && change != "shroud"
				check := func(game *mage.Game, want bool) {
					t.Helper()
					game.ApplyContinuousEffects()
					p := game.FindPermanentIncludingPhased(target.ID())
					if p.HasSubType("Forest") != want || p.HasSubType("Island") == want || p.HasSubType("Desert") == want || !p.Card.HasSuperType(core.SuperSnow) || p.Card.HasSuperType(core.SuperBasic) || !p.HasType(core.TypeLand) {
						t.Fatalf("wrong land types or supertypes: %v", p.SubTypes())
					}
				}
				check(g, active)
				resolved := g.Clone()
				switch change {
				case "source-phase-after":
					g.PhaseOut(source.ID())
				case "source-cycle-after":
					g.PhaseIn(g.PhaseOut(source.ID()))
				case "source-return-after":
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
					g.PutOnBattlefield(card, a.PlayerID())
					active = false
				case "source-control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "source-abilities":
					g.GetEffects().Add(mage.RemoveAllAbilities(source.ID(), core.EndOfTurn))
				case "source-untap":
					g.UntapPermanent(g.FindPermanent(source.ID()))
				case "target-cycle-after":
					g.PhaseIn(g.PhaseOut(target.ID()))
				case "target-return-after":
					g.RemoveFromBattlefield(g.FindPermanent(target.ID()))
					target = g.PutOnBattlefield(targetCard, b.PlayerID())
					active = false
				case "target-phased-source-exit":
					ids := g.PhaseOut(target.ID())
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
					g.PhaseIn(ids)
					active = false
				case "second-instance":
					other := g.PutOnBattlefield(factory(), a.PlayerID())
					g.RemoveFromBattlefield(other)
				case "cleanup":
					g.GetEffects().RemoveEndOfTurn()
				}
				check(g, active)
				if change == "source-return-after" {
					check(resolved, true)
				}
				if active {
					g.UntapPermanent(g.FindPermanent(target.ID()))
					if err := g.ActivateAbilityByIndex(b.PlayerID(), target.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
					if b.ManaPool().Count(core.Green) != 1 || b.ManaPool().Count(core.Blue) != 0 {
						t.Fatal("wrong intrinsic mana")
					}
				}
			})
		}
	}
}

func TestRONSourceLandTypeInvalid(t *testing.T) {
	valid := `CreatePermanentLayerEffectUntil(Ref_TargetPermanent,[SetLandType(Forest)],UntilPermanentLeavesBattlefield(ThisPermanent))`
	for _, action := range []string{
		strings.ReplaceAll(valid, "Forest", "Forest()"), strings.ReplaceAll(valid, "Forest", "Desert"), strings.ReplaceAll(valid, "SetLandType(Forest)", "SetLandType"),
		strings.ReplaceAll(valid, "SetLandType(Forest)", "SetLandType(Forest),AdjustPT(1,1)"), strings.ReplaceAll(valid, "SetLandType(Forest)", "AdjustPT(1,1)"),
		strings.ReplaceAll(valid, "[SetLandType(Forest)]", "[]"), strings.ReplaceAll(valid, "Ref_TargetPermanent", "Ref_TargetPlayer"),
		strings.ReplaceAll(valid, "ThisPermanent)", "ThisPermanent())"), strings.ReplaceAll(valid, "ThisPermanent)", "HostPermanent)"), strings.ReplaceAll(valid, "ThisPermanent)", "Ref_TargetPermanent)"), strings.ReplaceAll(valid, "UntilPermanentLeavesBattlefield(ThisPermanent)", "UntilPermanentLeavesBattlefield()"),
	} {
		src := fmt.Sprintf(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Land))],ActionList([%s])))])`, action)
		nodes, err := Parse(strings.NewReader(src), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("invalid shape accepted", action)
		}
	}
	nodes, err := Parse(strings.NewReader(`CreatePermanentLayerEffectUntil(ThisPermanent,[SetLandType(Forest)],UntilPermanentLeavesBattlefield(ThisPermanent))`), "context.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, ctx := range []scope{{}, {permanent: true}, {permanent: true, resolving: true, enteringSource: true}, {permanent: true, resolving: true, drawReplacementBody: true}} {
		if _, err := compileTemporaryLayer(nodes[0], ctx); err == nil {
			t.Fatal("invalid context accepted", ctx)
		}
	}
}

func TestRONSourceLandTypeDelayedAndOverlap(t *testing.T) {
	for _, landType := range []string{"Plains", "Island", "Swamp", "Mountain", "Forest"} {
		t.Run(landType, func(t *testing.T) {
			body := `CreatePermanentLayerEffectUntil(Ref_TargetPermanent,[SetLandType(` + landType + `)],UntilPermanentLeavesBattlefield(ThisPermanent))`
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Land))],ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+body+`]))])))`)
			first, second := factory(), factory()
			if first.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == second.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
				t.Fatal("shared target declarations")
			}
			g, a, b := ronGame()
			source := g.PutOnBattlefield(first, a.PlayerID())
			target := g.PutOnBattlefield(mage.NewLand("Land", mage.WithSubTypes("Desert")), b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			original := g.CaptureDamageSource(source.ID())
			originalTarget := g.CaptureDamageSource(target.ID())
			for range 2 {
				reads := g.WaitingDelayedPermanentSources()
				sourceRead, targetRead := false, false
				for _, read := range reads {
					sourceRead = sourceRead || read.SameObject(original)
					targetRead = targetRead || read.SameObject(originalTarget)
				}
				if len(reads) != 2 || !sourceRead || !targetRead || target.HasSubType(landType) || len(g.DelayedTriggers()) != 1 {
					t.Fatalf("waiting query changed duration or source: %d reads, source %v, target %v", len(reads), sourceRead, targetRead)
				}
			}
			g.PhaseOut(source.ID())
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			pending := g.Clone()
			g.ResolveStack()
			if !g.FindPermanent(target.ID()).HasSubType(landType) || pending.FindPermanent(target.ID()).HasSubType(landType) {
				t.Fatal("delayed timing or phased source")
			}
			pending.ResolveStack()
			g.PhaseIn([]uuid.UUID{source.ID()})
			g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			if !g.FindPermanent(target.ID()).HasSubType("Desert") || !pending.FindPermanent(target.ID()).HasSubType(landType) {
				t.Fatal("delayed clone lifetime")
			}
		})
	}
	g, a, b := ronGame()
	makeSource := func(landType string) *mage.Permanent {
		card := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Land))],ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent,[SetLandType(`+landType+`)],UntilPermanentLeavesBattlefield(ThisPermanent))])))`)()
		return g.PutOnBattlefield(card, a.PlayerID())
	}
	first, second := makeSource("Forest"), makeSource("Swamp")
	target := g.PutOnBattlefield(mage.NewLand("Land", mage.WithSubTypes("Desert")), b.PlayerID())
	for _, source := range []*mage.Permanent{first, second} {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
	}
	if !g.FindPermanent(target.ID()).HasSubType("Swamp") {
		t.Fatal("later timestamp lost")
	}
	g.RemoveFromBattlefield(second)
	if !g.FindPermanent(target.ID()).HasSubType("Forest") {
		t.Fatal("independent earlier duration lost")
	}
	g.RemoveFromBattlefield(first)
	if !g.FindPermanent(target.ID()).HasSubType("Desert") {
		t.Fatal("printed type not restored")
	}
}
