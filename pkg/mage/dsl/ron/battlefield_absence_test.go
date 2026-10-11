package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func absenceCardRule(t *testing.T, set, name string) (Node, Node) {
	t.Helper()
	record := staticBaseRecord(t, set, name)
	for _, rule := range record.Fields["Rules"].Args {
		if rule.Name == "TriggerA" && rule.Args[0].Name == "WhenNoPermanentsAreOnTheBattlefield" {
			return record, rule
		}
	}
	t.Fatal("missing absence rule", name)
	return Node{}, Node{}
}

func TestRONBattlefieldAbsenceTargets(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `TriggerA(WhenNoPermanentsAreOnTheBattlefield(IsCardtype(Land)),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([DestroyPermanent(Ref_TargetPermanent),GainLife(Integer(1))])))`)
	for _, scenario := range []string{"legal", "returned", "phased", "shroud", "no-target"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := opponentTargetGame()
			card := mage.NewCreature("Victim", "", 1, 1)
			card.SetOwner(b.PlayerID())
			victim := g.PutOnBattlefield(card, b.PlayerID())
			a.requested = []uuid.UUID{victim.ID()}
			if scenario == "no-target" {
				g.PhaseOut(victim.ID())
			}
			g.PutOnBattlefield(factory(), a.PlayerID())
			if len(a.candidates) != 0 {
				t.Fatal("detection chose targets before placement")
			}
			g.PutTriggersOnStack()
			if scenario == "no-target" {
				if g.StackSize() != 0 {
					t.Fatal("trigger without legal targets was stacked")
				}
				g.PhaseIn([]uuid.UUID{victim.ID()})
				g.CheckStateTriggers()
				g.PutTriggersOnStack()
			}
			if g.StackSize() != 1 || len(a.candidates) != 1 {
				t.Fatal("placement did not select a legal target")
			}
			g.PutOnBattlefield(mage.NewLand("Late land"), b.PlayerID())
			switch scenario {
			case "returned":
				g.BouncePermanentToHand(victim)
				b.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, b.PlayerID())
			case "phased":
				g.PhaseOut(victim.ID())
			case "shroud":
				g.MutablePermanent(victim.ID()).GrantBaseAttr(core.Shroud)
			}
			g.ResolveTopOfStack()
			legal := scenario == "legal" || scenario == "no-target"
			if (g.FindPermanentIncludingPhased(victim.ID()) == nil) != legal || (a.Life() == 21) != legal {
				t.Fatal("resolution lost target legality or rechecked absence")
			}
		})
	}
}

func TestRONBattlefieldAbsencePhaseAndLayers(t *testing.T) {
	for _, scenario := range []string{"source", "member", "pending-source", "restore-abilities", "type-layer", "nested-filter"} {
		t.Run(scenario, func(t *testing.T) {
			record, rule := absenceCardRule(t, "DRK", "Mana Vortex")
			if scenario == "nested-filter" {
				rule.Args[0].Args[0] = nameScopeNode(t, `And([IsNonToken,Or([IsCardtype(Land),IsCardtype(Creature)])])`)
			}
			factory := staticBaseSubrule(t, record, rule)
			g, a, b := ronGame()
			land := g.PutOnBattlefield(mage.NewLand("Land"), b.PlayerID())
			source := g.PutOnBattlefield(factory(), a.PlayerID())
			switch scenario {
			case "source":
				group := g.PhaseOut(source.ID())
				g.BouncePermanentToHand(land)
				g.CheckStateTriggers()
				g.PutTriggersOnStack()
				if g.StackSize() != 0 {
					t.Fatal("phased source detected absence")
				}
				g.PhaseIn(group)
			case "member":
				group := g.PhaseOut(land.ID())
				g.CheckStateTriggers()
				g.PutTriggersOnStack()
				g.PhaseIn(group)
			case "pending-source":
				g.BouncePermanentToHand(land)
				g.PutTriggersOnStack()
				group := g.PhaseOut(source.ID())
				g.ResolveTopOfStack()
				if g.FindPermanentIncludingPhased(source.ID()) == nil {
					t.Fatal("pending sacrifice moved phased source")
				}
				g.PhaseIn(group)
			case "restore-abilities":
				g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.EndOfTurn))
				g.ApplyContinuousEffects()
				g.BouncePermanentToHand(land)
				g.CheckStateTriggers()
				g.PutTriggersOnStack()
				if g.StackSize() != 0 {
					t.Fatal("removed source detected absence")
				}
				g.PutOnBattlefield(mage.NewLand("Cleanup land"), b.PlayerID())
				g.RunStepWithPriority(core.Cleanup)
				if g.FindPermanent(source.ID()) == nil {
					t.Fatal("cleanup ignored a matching land")
				}
				if err := mage.ReturnEachToHand(mage.MatchingPermanents(mage.IsLand)).Apply(&mage.EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
					t.Fatal(err)
				}
			case "type-layer":
				g.MutablePermanent(land.ID()).RevokeBaseAttr(core.AttrIsLand)
				g.MutablePermanent(land.ID()).GrantBaseAttr(core.AttrIsArtifact)
				if err := mage.Until(core.EndOfTurn, mage.AddCardType(mage.This, core.TypeLand)).Apply(&mage.EffectContext{Game: g, SourceID: land.ID(), Controller: b.PlayerID()}); err != nil {
					t.Fatal(err)
				}
				g.ApplyContinuousEffects()
				g.CheckStateTriggers()
				g.PutTriggersOnStack()
				if g.StackSize() != 0 {
					t.Fatal("condition ignored the type layer")
				}
				g.BouncePermanentToHand(land)
			case "nested-filter":
				g.PutOnBattlefield(mage.NewToken("Land token", 0, 0, []core.CardType{core.TypeLand}, nil), a.PlayerID())
				creature := g.PutOnBattlefield(mage.NewCreature("Creature", "", 1, 1), b.PlayerID())
				g.BouncePermanentToHand(land)
				g.PutTriggersOnStack()
				if g.StackSize() != 0 {
					t.Fatal("nested disjunction ignored creature")
				}
				g.BouncePermanentToHand(creature)
			}
			g.CheckStateTriggers()
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("restored or layered state did not trigger")
			}
			g.ResolveStack()
			if g.FindPermanent(source.ID()) != nil {
				t.Fatal("absence trigger did not sacrifice source")
			}
		})
	}
}

func TestRONBattlefieldAbsenceRealRules(t *testing.T) {
	for _, card := range []struct{ set, name, kind string }{
		{"DRK", "Mana Vortex", "Land"},
		{"ARN", "Drop of Honey", "Creature"},
	} {
		t.Run(card.name, func(t *testing.T) {
			record, rule := absenceCardRule(t, card.set, card.name)
			factory := staticBaseSubrule(t, record, rule)
			for _, scenario := range []string{"own", "opponent", "token", "phased", "type-loss", "control", "source-matches"} {
				t.Run(scenario, func(t *testing.T) {
					g, a, b := ronGame()
					controller := a.PlayerID()
					if scenario == "opponent" || scenario == "control" {
						controller = b.PlayerID()
					}
					var subject mage.Card = mage.NewLand("Land")
					attr := core.AttrIsLand
					kind := core.TypeLand
					if card.kind == "Creature" {
						subject = mage.NewCreature("Creature", "", 1, 1)
						attr, kind = core.AttrIsCreature, core.TypeCreature
					}
					if scenario == "token" {
						subject = mage.NewToken("Token", 1, 1, []core.CardType{kind}, nil)
					}
					match := g.PutOnBattlefield(subject, controller)
					sourceCard := factory()
					sourceCard.SetOwner(b.PlayerID())
					source := g.PutOnBattlefield(sourceCard, a.PlayerID())
					switch scenario {
					case "phased":
						g.PhaseOut(match.ID())
					case "type-loss":
						g.MutablePermanent(match.ID()).RevokeBaseAttr(attr)
						g.MutablePermanent(match.ID()).GrantBaseAttr(core.AttrIsArtifact)
					case "control":
						g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: match.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
					case "source-matches":
						g.MutablePermanent(source.ID()).GrantBaseAttr(attr)
						g.BouncePermanentToHand(match)
					}
					for range 3 {
						g.CheckStateTriggers()
					}
					g.PutTriggersOnStack()
					want := 0
					if scenario == "phased" || scenario == "type-loss" {
						want = 1
					}
					if g.StackSize() != want || g.FindPermanent(source.ID()) == nil {
						t.Fatalf("global detection: stack %d, want %d", g.StackSize(), want)
					}
					g.ResolveStack()
					if (g.FindPermanent(source.ID()) == nil) != (want == 1) {
						t.Fatal("wrong sacrifice outcome")
					}
					if want == 1 && len(b.Graveyard()) != 1 {
						t.Fatal("sacrifice lost owner destination")
					}
				})
			}
		})
	}
}

func TestRONBattlefieldAbsencePending(t *testing.T) {
	for _, scenario := range []string{"late-land", "counter", "source-phased", "removed", "removed-after", "control-after", "departed", "returned", "phase-cycle", "copy", "copy-opponent", "clone", "exile", "indestructible", "two"} {
		t.Run(scenario, func(t *testing.T) {
			record, rule := absenceCardRule(t, "DRK", "Mana Vortex")
			factory := staticBaseSubrule(t, record, rule)
			g, a, b := ronGame()
			land := g.PutOnBattlefield(mage.NewLand("Land"), b.PlayerID())
			card := factory()
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			if scenario == "source-phased" {
				g.PhaseOut(source.ID())
			}
			if scenario == "removed" {
				g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.EndOfTurn))
				g.ApplyContinuousEffects()
			}
			g.BouncePermanentToHand(land)
			g.CheckStateTriggers()
			g.CheckStateTriggers()
			g.PutTriggersOnStack()
			if scenario == "source-phased" || scenario == "removed" {
				if g.StackSize() != 0 {
					t.Fatal("inactive source detected absence")
				}
				return
			}
			g.CheckStateTriggers()
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("pending or active trigger repeated")
			}
			switch scenario {
			case "late-land":
				g.PutOnBattlefield(mage.NewLand("Late land"), b.PlayerID())
			case "counter":
				g.CounterStackObject(g.StackPeek().ID)
				g.CheckStateTriggers()
				g.PutTriggersOnStack()
				if g.StackSize() != 1 {
					t.Fatal("countered trigger did not rearm")
				}
			case "control-after":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				g.PutOnBattlefield(mage.NewLand("Late land"), b.PlayerID())
			case "departed", "returned":
				g.BouncePermanentToHand(source)
				if scenario == "returned" {
					g.PutOnBattlefield(mage.NewLand("Late land"), b.PlayerID())
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, a.PlayerID())
				}
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(source.ID()))
			case "removed-after":
				g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.EndOfTurn))
				g.ApplyContinuousEffects()
			case "copy", "copy-opponent":
				controller := a.PlayerID()
				if scenario == "copy-opponent" {
					controller = b.PlayerID()
				}
				if g.CopyStackObjectDirect(g.StackPeek(), controller, false) == nil {
					t.Fatal("copy failed")
				}
				g.ResolveTopOfStack()
				if (g.FindPermanent(source.ID()) == nil) != (scenario == "copy") {
					t.Fatal("copy lost sacrifice permission")
				}
			case "clone":
				clone := g.Clone()
				clone.PutOnBattlefield(mage.NewLand("Clone land"), b.PlayerID())
				clone.ResolveStack()
				if clone.FindPermanent(source.ID()) != nil || g.FindPermanent(source.ID()) == nil {
					t.Fatal("clone changed original or rechecked condition")
				}
			case "exile":
				g.AddExileIfWouldGoToGraveyardThisTurn(source.ID(), b.PlayerID())
			case "indestructible":
				g.MutablePermanent(source.ID()).GrantBaseAttr(core.Indestructible)
				g.AddRegenerationShield(source.ID())
			case "two":
				other := g.PutOnBattlefield(factory(), b.PlayerID())
				g.CheckStateTriggers()
				g.PutTriggersOnStack()
				if g.StackSize() != 2 {
					t.Fatal("source declarations share arming")
				}
				g.ResolveTopOfStack()
				if g.FindPermanent(other.ID()) != nil || g.FindPermanent(source.ID()) == nil {
					t.Fatal("wrong instance sacrificed")
				}
			}
			g.ResolveStack()
			survives := scenario == "control-after" || scenario == "returned"
			if (g.FindPermanent(source.ID()) != nil) != survives {
				t.Fatal("pending controller or source visit changed")
			}
			if scenario == "exile" && g.FindExiledCard(source.ID()) == nil {
				t.Fatal("sacrifice replacement bypassed")
			}
		})
	}
}

func TestRONBattlefieldAbsenceInvalid(t *testing.T) {
	for _, event := range []string{
		`WhenNoPermanentsAreOnTheBattlefield`,
		`WhenNoPermanentsAreOnTheBattlefield()`,
		`WhenNoPermanentsAreOnTheBattlefield(IsCardtype(Land),IsCardtype(Creature))`,
	} {
		assertConditionalRejected(t, `TriggerA(`+event+`,ActionList([SacrificePermanent(ThisPermanent)]))`)
	}
	for _, filter := range []string{
		`IsCardtype`, `IsCardtype()`, `IsCardtype(Land,Creature)`, `IsCardtype(Land())`,
		`IsCardtype(Unknown)`, `And([])`, `Or([])`, `And([IsNonToken,Or([])])`,
		`SinglePermanent(ThisPermanent)`, `Other(ThisPermanent)`, `ControlledByPlayer(You)`,
		`SinglePlayer(You)`, `Trigger_ThatPermanent`, `Ref_TargetPermanent`,
	} {
		assertConditionalRejected(t, `TriggerA(WhenNoPermanentsAreOnTheBattlefield(`+filter+`),ActionList([SacrificePermanent(ThisPermanent)]))`)
	}
	for _, body := range []string{`DrawNumberCards(Trigger_AmountOfDamageDealt)`, `PlayerActions(Trigger_ThatPlayer,[DrawACard])`, `SacrificePermanent(Trigger_ThatPermanent)`, `ControllersSacrificeEachPermanent(Trigger_ThosePermanents)`} {
		assertConditionalRejected(t, `TriggerA(WhenNoPermanentsAreOnTheBattlefield(IsCardtype(Land)),ActionList([`+body+`]))`)
	}
	assertConditionalRejected(t, `FromStack(TriggerA(WhenNoPermanentsAreOnTheBattlefield(IsCardtype(Land)),ActionList([SacrificePermanent(ThisPermanent)])))`)
}

func TestRONBattlefieldAbsenceRecordsAndInventory(t *testing.T) {
	for _, card := range []struct{ set, name, blocker string }{
		{"DRK", "Mana Vortex", ""},
		{"ARN", "Drop of Honey", ""},
	} {
		t.Run(card.name, func(t *testing.T) {
			record, rule := absenceCardRule(t, card.set, card.name)
			if _, err := Compile(record); (card.blocker == "" && err != nil) || (card.blocker != "" && (err == nil || !strings.Contains(err.Error(), card.blocker))) {
				t.Fatal("complete-card blocker changed", err)
			}
			factory := staticBaseSubrule(t, record, rule)
			first, second := factory(), factory()
			if first.Abilities()[0].AbilityID() == second.Abilities()[0].AbilityID() {
				t.Fatal("shared state declaration")
			}
		})
	}
	text := `Card(Name:"Observer",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[TriggerA(WhenNoPermanentsAreOnTheBattlefield(And([IsNonToken,Or([IsCardtype(Land),IsCardtype(Creature)])])),ActionList([SacrificePermanent(ThisPermanent)]))])`
	files := fstest.MapFS{"a.ron": {Data: []byte(text)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal("inventory", err)
	}
	found := false
	for _, sig := range report.Signatures {
		if sig.Name == "IsCardtype" && sig.Role == "permanent_filter" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing permanent filter inventory role")
	}
	files["b.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(text, "Observer", "Invalid", 1))}
	files["b.ron"].Data = []byte(strings.Replace(string(files["b.ron"].Data), "IsNonToken", "Trigger_ThatPermanent", 1))
	if partial, err := LoadFS(files, "*.ron"); err == nil || partial != nil {
		t.Fatal("invalid collection published")
	}
	if LoadInto(registry, files, "*.ron") == nil {
		t.Fatal("invalid replacement accepted")
	}
	if _, err := registry.CreateCard("Observer"); err != nil {
		t.Fatal("failed replacement erased registry")
	}
}
