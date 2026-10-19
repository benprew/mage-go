package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func spellGrantFixture(t *testing.T, form string) mage.Card {
	t.Helper()
	ability := `Activated(And([PayMana([ManaCostGeneric(1)]),TapPermanent(ThisPermanent)]),Targeted([TargetSpell(AnySpell)],ActionList([CounterSpell(Ref_TargetSpell),PlayerAction(You,GainLife(Integer(2)))])))`
	rule := `PermanentLayerEffect(ThisPermanent,[AddAbility([` + ability + `])])`
	if form == "host" {
		rule = `EnchantPermanent(IsCardtype(Land)),PermanentLayerEffect(HostPermanent,[AddAbility([` + ability + `])])`
	}
	if form == "each" {
		rule = `EachPermanentLayerEffect(IsCardtype(Land),[AddAbility([` + ability + `])])`
	}
	if form == "retained" {
		rule = `Activated(PayMana([]),ActionList([CreatePermanentLayerEffect(ThisPermanent,[AddAbility([` + ability + `])])]))`
	}
	kind := "Artifact"
	if form == "host" {
		kind = "Enchantment"
	}
	subtypes := "[]"
	if form == "host" {
		subtypes = "[Aura]"
	}
	node := nameScopeNode(t, `Card(Name:"Grant fixture",Typeline:(Supertypes:[],Cardtypes:[`+kind+`],Subtypes:`+subtypes+`),Rules:[`+rule+`])`)
	factory, err := Compile(node)
	if err != nil {
		t.Fatal(err)
	}
	return factory()
}

func TestRONTargetedSpellGrantGameplay(t *testing.T) {
	for _, form := range []string{"source", "host", "each", "retained"} {
		for _, change := range []string{"none", "donor-left", "source-left", "source-return", "source-phase", "source-control", "target-left", "target-recast", "uncounterable", "copy"} {
			t.Run(form+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				donor := g.PutOnBattlefield(spellGrantFixture(t, form), a.PlayerID())
				recipient := donor
				actor := a
				if form == "host" || form == "each" {
					recipient = g.PutOnBattlefield(mage.NewLand("Recipient"), b.PlayerID())
					actor = b
					if form == "host" {
						g.Attach(donor.ID(), recipient.ID())
					}
				}
				index := 0
				if form == "source" {
					index = 1
				}
				if form == "retained" {
					if err := g.ActivateAbilityByIndex(actor.PlayerID(), recipient.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
					g.ResolveStack()
					index = 1
				}
				g.ApplyContinuousEffects()
				victimCard := repositoryCard(t, "LEB", "Sol Ring")
				if change == "target-recast" {
					victimCard = repositoryCard(t, "LEB", "Dark Ritual")
				}
				if change == "uncounterable" {
					victimCard = mage.NewArtifact("Uncounterable fixture", "", mage.WithUncounterable())
				}
				victim := castRONSpell(t, g, a, victimCard)
				actor.ManaPool().Clear()
				if err := g.ActivateAbilityByIndex(actor.PlayerID(), recipient.ID(), index, []uuid.UUID{victim.ID}); err == nil || g.FindPermanent(recipient.ID()).Tapped || g.StackSize() != 1 {
					t.Fatal("failed payment changed the source or stack")
				}
				actor.ManaPool().Add(core.Colorless, 1)
				if err := g.ActivateAbilityByIndex(actor.PlayerID(), recipient.ID(), index, []uuid.UUID{recipient.ID()}); err == nil || actor.ManaPool().TotalMana() != 1 || g.FindPermanent(recipient.ID()).Tapped {
					t.Fatal("illegal target consumed payment")
				}
				if err := g.ActivateAbilityByIndex(actor.PlayerID(), recipient.ID(), index, []uuid.UUID{victim.ID}); err != nil {
					t.Fatal(err)
				}
				if !g.FindPermanent(recipient.ID()).Tapped || actor.ManaPool().TotalMana() != 0 || actor.Life() != 20 || g.StackSize() != 2 {
					t.Fatal("wrong announcement or resolution timing")
				}
				switch change {
				case "donor-left":
					g.RemoveFromBattlefield(g.FindPermanent(donor.ID()))
				case "source-left":
					g.RemoveFromBattlefield(g.FindPermanent(recipient.ID()))
				case "source-return":
					card := g.FindPermanent(recipient.ID()).Card
					g.RemoveFromBattlefield(g.FindPermanent(recipient.ID()))
					g.PutOnBattlefield(card, actor.PlayerID())
				case "source-phase":
					g.PhaseOut(recipient.ID())
				case "source-control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: recipient.ID(), TargetID: recipient.ID(), ControllerID: g.GetOpponent(actor.PlayerID()).PlayerID(), Duration: core.Indefinite})
				case "target-left":
					g.CounterStackObject(victim.ID)
				case "target-recast":
					g.CounterStackObject(victim.ID)
					a.RemoveFromGraveyard(victim.Card.ID())
					castRONSpell(t, g, a, victim.Card)
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), actor.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				}
				clone := g.Clone()
				clone.ResolveStack()
				if actor.Life() != 20 {
					t.Fatal("clone changed original life")
				}
				g.ResolveStack()
				want := 22
				if change == "target-left" || change == "target-recast" {
					want = 20
				}
				if change == "uncounterable" {
					if g.FindPermanent(victim.SourceID) == nil {
						t.Fatal("the legal surviving spell did not resolve")
					}
				}
				if actor.Life() != want || clone.GetPlayer(actor.PlayerID()).Life() != want || g.StackSize() != 0 {
					t.Fatalf("life = %d; want %d", actor.Life(), want)
				}
			})
		}
	}
}

func TestRONTargetedSpellGrantInvalid(t *testing.T) {
	for _, body := range []string{
		`Targeted([],ActionList([GainLife(Integer(1))]))`,
		`Targeted([TargetSpell(AnySpell)],ActionList([]))`,
		`Targeted([TargetSpell(AnySpell)],ActionList([RegeneratePermanent(Ref_TargetPermanent)]))`,
		`Targeted([TargetSpell(AnySpell)],ActionList([TapPermanent(HostPermanent)]))`,
		`Targeted([TargetSpell(AnySpell)],ActionList([TapPermanent(Trigger_ThatCreature)]))`,
		`Targeted([TargetSpell(AnySpell)],ActionList([If(CostWasPaid,[GainLife(Integer(1))])]))`,
		`Targeted([TargetSpell(AnySpell)],Targeted([TargetSpell(AnySpell)],ActionList([CounterSpell(Ref_TargetSpell)])))`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[EachPermanentLayerEffect(AnyPermanent,[AddAbility([Activated(PayMana([]),` + body + `)])])])`
		node := nameScopeNode(t, src)
		if _, err := Compile(node); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestRONTargetedSpellGrantEquinoxBoundary(t *testing.T) {
	record := repositoryRecord(t, "LEG", "Equinox", "LEG.ron")
	query := record.Fields["Rules"].Args[1].Args[1].Args[0].Args[0].Args[0].Args[1].Args[1].Args[0].Args[0]
	if query.Name != "If" || query.Args[0].Args[1].Name != "WouldDestroyAPermanent" {
		t.Fatal("Equinox query shape changed")
	}
	_, err := Compile(record)
	if err == nil || (!strings.Contains(err.Error(), query.Position.String()) && !strings.Contains(err.Error(), query.Args[0].Args[1].Position.String())) {
		t.Fatalf("expected the remaining prospective destruction gap, got %v", err)
	}
}

func TestRONTargetedSpellGrantEquinoxProjection(t *testing.T) {
	record := repositoryRecord(t, "LEG", "Equinox", "LEG.ron")
	record.Fields["Name"] = Node{Kind: String, String: "Equinox grant fixture"}
	grant := &record.Fields["Rules"].Args[1].Args[1].Args[0].Args[0].Args[0]
	actions := &grant.Args[1].Args[1].Args[0]
	if len(actions.Args) != 1 || actions.Args[0].Name != "If" {
		t.Fatal("Equinox no longer has the expected prospective query")
	}
	// This fixture tests the grant before the destruction query is available.
	actions.Args = actions.Args[0].Args[1].Args
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		g, a, b := ronGame()
		land := g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), b.PlayerID())
		aura := castRONSpell(t, g, a, factory(), land.ID())
		g.ResolveStack()
		if g.FindPermanent(aura.SourceID) == nil {
			t.Fatal("Aura did not resolve")
		}
		victim := castRONSpell(t, g, a, repositoryCard(t, "LEB", "Sol Ring"))
		if err := g.ActivateAbilityByIndex(b.PlayerID(), land.ID(), 1, []uuid.UUID{victim.ID}); err != nil {
			t.Fatal(err)
		}
		if !g.FindPermanent(land.ID()).Tapped || g.FindPermanent(aura.SourceID).Tapped || g.StackSize() != 2 {
			t.Fatal("the grant did not use its recipient")
		}
		g.ResolveStack()
		if g.FindPermanent(victim.SourceID) != nil || len(a.Graveyard()) != 1 {
			t.Fatal("the targeted spell was not countered")
		}
	}
}
