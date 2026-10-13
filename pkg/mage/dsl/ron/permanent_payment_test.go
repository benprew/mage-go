package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const ronPermanentPayment = `ForEachPermanentAnyPlayerMayCost(IsCardtype(Land),AnyPlayer,PayLife(Integer(1)))`
const ronUnpaidDestroy = `DestroyEachPermanent(CostWasntPaidForIt)`

func TestRONPermanentPaymentCleansing(t *testing.T) {
	for _, state := range []string{"none", "controller", "opponent", "short", "phase", "regenerate", "indestructible", "copy", "clone"} {
		t.Run(state, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: state == "controller" || state == "short"}
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: state == "opponent"}
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			lands := make([]*mage.Permanent, 0, 2)
			for range 2 {
				c := mage.NewLand("Land")
				c.SetOwner(b.PlayerID())
				lands = append(lands, g.PutOnBattlefield(c, b.PlayerID()))
			}
			other := g.PutOnBattlefield(mage.NewArtifact("Other", ""), a.PlayerID())
			if state == "short" {
				a.SetLife(1)
			}
			if state == "phase" {
				g.PhaseOut(lands[0].ID())
			}
			if state == "regenerate" {
				g.AddRegenerationShield(lands[0].ID())
			}
			if state == "indestructible" {
				g.MutablePermanent(lands[0].ID()).GrantBaseAttr(core.Indestructible)
			}
			card := repositoryCard(t, "DRK", "Cleansing")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid spell cast")
			}
			a.ManaPool().Add(core.White, 3)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if a.offers+b.offers != 0 {
				t.Fatal("offered before resolution")
			}
			if state == "copy" {
				g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false)
			}
			if state == "clone" {
				clone := g.Clone()
				clone.ResolveStack()
				if g.FindPermanent(lands[0].ID()) == nil || clone.FindPermanent(lands[0].ID()) != nil {
					t.Fatal("clone changed original")
				}
			}
			g.ResolveStack()
			remaining := 0
			for _, p := range lands {
				if g.FindPermanentIncludingPhased(p.ID()) != nil {
					remaining++
				}
			}
			want := 0
			if state == "controller" || state == "opponent" {
				want = 2
			}
			if state == "short" || state == "phase" || state == "regenerate" || state == "indestructible" {
				want = 1
			}
			if remaining != want || g.FindPermanent(other.ID()) == nil {
				t.Fatalf("remaining=%d want=%d", remaining, want)
			}
			if state == "controller" && (a.Life() != 18 || b.Life() != 20 || b.offers != 0) {
				t.Fatal("controller payments")
			}
			if state == "opponent" && (a.Life() != 20 || b.Life() != 18) {
				t.Fatal("opponent payments")
			}
			if state == "short" && (a.Life() != 0 || a.offers != 1) {
				t.Fatal("life affordability was not rechecked per subject")
			}
		})
	}
}

func TestRONPermanentPaymentInvalidAndLoading(t *testing.T) {
	for _, body := range []string{
		ronUnpaidDestroy,
		strings.Replace(ronPermanentPayment, "AnyPlayer", "Opponent", 1) + `,` + ronUnpaidDestroy,
		strings.Replace(ronPermanentPayment, "Integer(1)", "Integer(-1)", 1) + `,` + ronUnpaidDestroy,
		strings.Replace(ronPermanentPayment, "Integer(1)", "ValueX", 1) + `,` + ronUnpaidDestroy,
		strings.Replace(ronPermanentPayment, "PayLife(Integer(1))", "PayMana([])", 1) + `,` + ronUnpaidDestroy,
		ronPermanentPayment + `,DestroyEachPermanent(CostWasntPaidForIt(You))`,
		`If(PermanentPassesFilter(ThisPermanent,IsUntapped),[` + ronPermanentPayment + `]),` + ronUnpaidDestroy,
	} {
		src := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([` + body + `]))])`
		nodes, err := Parse(strings.NewReader(src), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid declaration accepted", body)
		}
	}
	src := `Card(Name:"Offer",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([` + ronPermanentPayment + `,` + ronUnpaidDestroy + `]))])`
	files := fstest.MapFS{"good.ron": {Data: []byte(src)}}
	if _, err := LoadFS(files, "*.ron"); err != nil {
		t.Fatal(err)
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(src, "Offer", "Bad", 1) + `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(` + ronUnpaidDestroy + `)])`)}
	if reg, err := LoadFS(files, "*.ron"); err == nil || reg != nil {
		t.Fatal("invalid collection published")
	}
}

func TestRONPermanentPaymentContextsAndScope(t *testing.T) {
	for _, context := range []string{"trigger", "delay", "saved-delay", "nested"} {
		t.Run(context, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: false}
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: false}
			g := mage.NewGame(a, b)
			body := ronPermanentPayment + `,` + ronUnpaidDestroy
			if context == "saved-delay" {
				body = ronPermanentPayment + `,CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([` + ronUnpaidDestroy + `]))`
			}
			if context == "nested" {
				body = ronPermanentPayment + `,If(PermanentPassesFilter(ThisPermanent,IsUntapped),[ForEachPermanentAnyPlayerMayCost(IsCardtype(Artifact),AnyPlayer,PayLife(Integer(0))),DestroyEachPermanent(CostWasntPaidForIt)]),` + ronUnpaidDestroy
			}
			rule := `Activated(PayMana([]),ActionList([` + body + `]))`
			if context == "trigger" {
				rule = `TriggerA(AtTheBeginningOfAPlayersUpkeep(AnyPlayer),ActionList([` + body + `]))`
			}
			if context == "delay" {
				rule = `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([` + body + `]))]))`
			}
			factory := permanentScopeFactory(t, "Artifact", rule)
			source := g.PutOnBattlefield(factory(), a.PlayerID())
			land := g.PutOnBattlefield(mage.NewLand("Land"), b.PlayerID())
			if context == "trigger" {
				g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
			} else if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if a.offers+b.offers != 0 {
				t.Fatal("payment at announcement")
			}
			g.ResolveStack()
			if context == "delay" || context == "saved-delay" {
				g.RemoveFromBattlefield(source)
				later := g.PutOnBattlefield(mage.NewLand("Later"), b.PlayerID())
				offers := a.offers + b.offers
				for range 3 {
					g.WaitingDelayedPermanentSources()
				}
				if a.offers+b.offers != offers {
					t.Fatal("waiting query offered payment")
				}
				clone := g.Clone()
				clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				clone.ResolveStack()
				if g.FindPermanent(land.ID()) == nil || clone.FindPermanent(land.ID()) != nil {
					t.Fatal("delayed clone frame")
				}
				g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				g.ResolveStack()
				if (g.FindPermanent(later.ID()) != nil) != (context == "saved-delay") {
					t.Fatal("saved result confused with live collection")
				}
			}
			if g.FindPermanent(land.ID()) != nil {
				t.Fatal("context lost unpaid binding")
			}
		})
	}
}

func TestRONPermanentPaymentInventory(t *testing.T) {
	source := `Card(Name:"Offer",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([` + ronPermanentPayment + `,` + ronUnpaidDestroy + `]))])`
	report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(source)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range report.Signatures {
		if entry.Name == "PayLife" && entry.Role == "resolution_cost" {
			found = true
		}
	}
	if !found {
		t.Fatal("per-permanent cost lost its inventory role")
	}
}
