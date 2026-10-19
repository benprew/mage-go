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

func TestRONSourcePresenceDurationScarwood(t *testing.T) {
	for _, change := range []string{"none", "paid", "short", "source-phase-before", "source-cycle-before", "source-return-before", "source-control-before", "abilities-before", "target-return-before", "target-phase-before", "target-cycle-before", "shroud", "source-phase-after", "source-cycle-after", "source-return-after", "source-control-after", "abilities-after", "target-cycle-after", "target-return-after", "target-phase-source-cycle", "copy", "second-instance"} {
		t.Run(change, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: false}
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: change == "paid"}
			g := mage.NewGame(a, b)
			card := repositoryCard(t, "DRK", "Scarwood Bandits")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			targetCard := mage.NewArtifact("Target", "")
			targetCard.SetOwner(b.PlayerID())
			target := g.PutOnBattlefield(targetCard, b.PlayerID())
			forest := g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), b.PlayerID())
			if !mage.HasLandwalkEvasion(source, b.PlayerID(), g) {
				t.Fatal("missing forestwalk")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{target.ID()}); err == nil || source.Tapped || !g.GetStack().IsEmpty() {
				t.Fatal("unpaid activation changed state")
			}
			a.ManaPool().Add(core.Colorless, 2)
			a.ManaPool().Add(core.Green, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{forest.ID()}); err == nil {
				t.Fatal("land accepted as artifact")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{target.ID()}); err != nil {
				t.Fatal(err)
			}
			if !source.Tapped || a.ManaPool().TotalMana() != 0 || b.offers != 0 || target.ControllerID() != b.PlayerID() {
				t.Fatal("announcement timing")
			}
			if change != "short" {
				b.ManaPool().Add(core.Colorless, 2)
			}
			switch change {
			case "source-phase-before":
				g.PhaseOut(source.ID())
			case "source-cycle-before":
				g.PhaseIn(g.PhaseOut(source.ID()))
			case "source-return-before":
				g.RemoveFromBattlefield(source)
				g.PutOnBattlefield(card, a.PlayerID())
			case "source-control-before":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "abilities-before":
				g.GetEffects().Add(mage.RemoveAllAbilities(source.ID(), core.EndOfTurn))
				g.ApplyContinuousEffects()
			case "target-return-before":
				g.RemoveFromBattlefield(target)
				target = g.PutOnBattlefield(targetCard, b.PlayerID())
			case "target-phase-before":
				g.PhaseOut(target.ID())
			case "target-cycle-before":
				g.PhaseIn(g.PhaseOut(target.ID()))
			case "shroud":
				g.MutablePermanent(target.ID()).GrantBaseAttr(core.Shroud)
			case "copy":
				g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
			}
			clone := g.Clone()
			clone.ResolveStack()
			if g.FindPermanentIncludingPhased(target.ID()).ControllerID() != b.PlayerID() {
				t.Fatal("clone resolved original")
			}
			g.ResolveStack()
			active := change != "paid" && change != "source-phase-before" && change != "source-return-before" && change != "target-return-before" && change != "target-phase-before" && change != "shroud"
			want := b.PlayerID()
			if active {
				want = a.PlayerID()
			}
			if g.FindPermanentIncludingPhased(target.ID()).ControllerID() != want {
				t.Fatal("wrong resolving controller")
			}
			switch change {
			case "source-phase-after":
				g.PhaseOut(source.ID())
				active = false
			case "source-cycle-after":
				g.PhaseIn(g.PhaseOut(source.ID()))
				active = false
			case "source-return-after":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				g.PutOnBattlefield(card, a.PlayerID())
				active = false
			case "source-control-after":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "abilities-after":
				g.GetEffects().Add(mage.RemoveAllAbilities(source.ID(), core.EndOfTurn))
			case "target-cycle-after":
				g.PhaseIn(g.PhaseOut(target.ID()))
			case "target-return-after":
				g.RemoveFromBattlefield(g.FindPermanent(target.ID()))
				target = g.PutOnBattlefield(targetCard, b.PlayerID())
				active = false
			case "target-phase-source-cycle":
				ids := g.PhaseOut(target.ID())
				g.PhaseIn(g.PhaseOut(source.ID()))
				g.PhaseIn(ids)
				active = false
			case "second-instance":
				other := g.PutOnBattlefield(repositoryCard(t, "DRK", "Scarwood Bandits"), a.PlayerID())
				g.PhaseOut(other.ID())
			}
			g.ApplyContinuousEffects()
			want = b.PlayerID()
			if active {
				want = a.PlayerID()
			}
			if g.FindPermanentIncludingPhased(target.ID()).ControllerID() != want {
				t.Fatal("wrong lifetime")
			}
			if change == "source-cycle-after" && clone.FindPermanent(target.ID()).ControllerID() != a.PlayerID() {
				t.Fatal("original expiration changed clone")
			}
			if change == "paid" && (b.offers != 1 || b.ManaPool().TotalMana() != 0) {
				t.Fatal("wrong opponent payment")
			}
		})
	}
}

func TestRONSourcePresenceDurationContexts(t *testing.T) {
	for _, rule := range []string{
		`TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([GainControlOfPermanentUntil(ThisPermanent,UntilPermanentLeavesBattlefield(ThisPermanent))]))`,
		`Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([GainControlOfPermanentUntil(ThisPermanent,UntilPermanentLeavesBattlefield(ThisPermanent))]))]))`,
	} {
		t.Run(rule, func(t *testing.T) {
			g, a, b := ronGame()
			card := permanentScopeFactory(t, "Artifact", rule)()
			source := g.PutOnBattlefield(card, a.PlayerID())
			if strings.HasPrefix(rule, "Activated") {
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				original := g.CaptureDamageSource(source.ID())
				for range 2 {
					reads := g.WaitingDelayedPermanentSources()
					if len(reads) != 1 || !reads[0].SameObject(original) || g.FindPermanent(source.ID()).ControllerID() != a.PlayerID() || len(g.DelayedTriggers()) != 1 {
						t.Fatal("waiting source query changed state or identity")
					}
				}
			}
			if strings.HasPrefix(rule, "TriggerA") {
				g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			} else {
				g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			}
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			g.ResolveStack()
			want := a.PlayerID()
			if g.FindPermanent(source.ID()).ControllerID() != want {
				t.Fatal("wrong scoped controller")
			}
		})
	}
}

func TestRONSourcePresenceDurationInvalidAndLoading(t *testing.T) {
	for _, duration := range []string{`UntilPermanentLeavesBattlefield`, `UntilPermanentLeavesBattlefield()`, `UntilPermanentLeavesBattlefield(ThisPermanent,You)`, `UntilPermanentLeavesBattlefield(ThisPermanent())`, `UntilPermanentLeavesBattlefield(Ref_TargetPermanent)`, `UntilPermanentLeavesBattlefield(HostPermanent)`} {
		text := fmt.Sprintf(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),Targeted([TargetPermanent(AnyPermanent)],ActionList([GainControlOfPermanentUntil(Ref_TargetPermanent,%s)])))])`, duration)
		nodes, err := Parse(strings.NewReader(text), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid duration accepted", duration)
		}
	}
	record := staticBaseRecord(t, "DRK", "Scarwood Bandits")
	if _, err := Compile(record); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ set, name string }{{"ARN", "Oubliette"}} {
		if _, err := Compile(staticBaseRecord(t, tc.set, tc.name)); err != nil {
			t.Fatal("supported phasing duration rejected", tc.name, err)
		}
	}
	fs := fstest.MapFS{"valid.ron": &fstest.MapFile{Data: []byte(`Card(Name:"Valid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([GainControlOfPermanentUntil(ThisPermanent,UntilPermanentLeavesBattlefield(ThisPermanent))]))])`)}}
	registry := mage.NewRegistry()
	if err := LoadInto(registry, fs, "*.ron"); err != nil {
		t.Fatal(err)
	}
	fs["invalid.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([GainControlOfPermanentUntil(ThisPermanent,UntilPermanentLeavesBattlefield(ThisPermanent))]))])`)}
	if err := LoadInto(registry, fs, "*.ron"); err == nil {
		t.Fatal("partial collection accepted")
	}
	if !registry.CardRegistered("Valid") {
		t.Fatal("failed load changed registry")
	}
}

func TestRONSourcePresenceDurationCastingAndBoundary(t *testing.T) {
	g, a, _ := ronGame()
	first := repositoryCard(t, "DRK", "Scarwood Bandits")
	second := repositoryCard(t, "DRK", "Scarwood Bandits")
	first.SetOwner(a.PlayerID())
	if first.Abilities()[1].(*mage.ActionDefinition).Targets()[0] == second.Abilities()[1].(*mage.ActionDefinition).Targets()[0] {
		t.Fatal("factories share target declarations")
	}
	a.AddToHand(first)
	if err := g.CastSpellByID(a.PlayerID(), first.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || !g.GetStack().IsEmpty() {
		t.Fatal("unpaid casting changed state")
	}
	a.ManaPool().Add(core.Colorless, 2)
	a.ManaPool().Add(core.Green, 2)
	if err := g.CastSpellByID(a.PlayerID(), first.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(first.ID()) != nil || a.ManaPool().TotalMana() != 0 {
		t.Fatal("cast timing or mana")
	}
	g.ResolveStack()
	if p := g.FindPermanent(first.ID()); p == nil || p.CurrentPower(g) != 2 || p.CurrentToughness(g) != 2 || !p.HasKeyword(core.Forestwalk) {
		t.Fatal("wrong full-card definition")
	}
	nodes, err := Parse(strings.NewReader(`GainControlOfPermanentUntil(ThisPermanent,UntilPermanentLeavesBattlefield(ThisPermanent))`), "context.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, ctx := range []scope{{permanent: true, enteringSource: true}, {permanent: true, drawReplacementBody: true}, {}} {
		if _, err := compileSourceControl(nodes[0], ctx); err == nil {
			t.Fatal("invalid execution context accepted")
		}
	}
	invalid := `Card(Name:"Bad layer",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreatePermanentLayerEffectUntil(ThisPermanent,[AdjustPT(1,1)],UntilPermanentLeavesBattlefield(ThisPermanent))]))])`
	nodes, err = Parse(strings.NewReader(invalid), "boundary.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Compile(nodes[0]); err == nil {
		t.Fatal("control lifetime leaked into layer syntax")
	}
}
