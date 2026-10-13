package ron

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONIndefiniteTypeCasting(t *testing.T) {
	for _, tc := range []struct{ set, name, mana string }{
		{"ATQ", "Ashnod's Transmogrant", "{1}"},
		{"FEM", "Thelonite Monk", "{2}{G}{G}"},
	} {
		g, a, _ := ronGame()
		card := repositoryCard(t, tc.set, tc.name)
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
			t.Fatal("unpaid spell did not roll back")
		}
		if tc.set == "FEM" {
			a.ManaPool().Add(core.Green, 2)
		}
		a.ManaPool().Add(core.Colorless, core.ParseManaCost(tc.mana).Generic)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		p := g.FindPermanent(card.ID())
		if p == nil || a.ManaPool().TotalMana() != 0 {
			t.Fatal("wrong paid casting")
		}
		if tc.set == "FEM" && (p.CurrentPower(g) != 1 || p.CurrentToughness(g) != 2 || !p.HasSubType("Monk")) {
			t.Fatal("wrong Monk characteristics")
		}
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{p.ID()}); err == nil || g.StackSize() != 0 || g.FindPermanent(p.ID()).Tapped {
			t.Fatal("invalid activation spent costs")
		}
		other := repositoryCard(t, tc.set, tc.name)
		if card.Abilities()[0] == other.Abilities()[0] {
			t.Fatal("factory shared activation")
		}
	}
}

func TestRONIndefiniteTypeLegalityAndReplacement(t *testing.T) {
	for _, scenario := range []string{"becomes-artifact", "counter-prevented", "counter-doubled", "copy"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			p := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Ashnod's Transmogrant"), a.PlayerID())
			q := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 3), b.PlayerID())
			r := g.PutOnBattlefield(mage.NewCreature("Second", "", 3, 4), b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{q.ID()}); err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "becomes-artifact":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.AddCardType(mage.This, core.TypeArtifact)), q.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "counter-prevented":
				g.AddReplacementEffect(&counterPlacementReplacement{source: p.ID(), prevent: true})
			case "counter-doubled":
				g.AddCounterDoubler(r.ID(), core.P1P1, mage.AnyPermanent)
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			g.ResolveStack()
			want := int(1)
			if scenario == "counter-prevented" || scenario == "becomes-artifact" {
				want = 0
			}
			if scenario == "counter-doubled" {
				want = 2
			}
			if q = g.FindPermanent(q.ID()); q.Counters[core.P1P1] != want || !q.HasType(core.TypeArtifact) || g.FindPermanent(r.ID()).HasType(core.TypeArtifact) {
				t.Fatalf("wrong legality, counter replacement or captured recipient: counters=%d want=%d artifact=%t second=%t", q.Counters[core.P1P1], want, q.HasType(core.TypeArtifact), g.FindPermanent(r.ID()).HasType(core.TypeArtifact))
			}
		})
	}
}

func TestRONIndefiniteTypeAllBasicTypes(t *testing.T) {
	for i, landType := range []string{"Plains", "Island", "Swamp", "Mountain", "Forest"} {
		g, a, b := ronGame()
		factory := permanentScopeFactory(t, "Artifact", fmt.Sprintf(`Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Land))],ActionList([CreatePermanentLayerEffect(Ref_TargetPermanent,[SetLandType(%s)])])))`, landType))
		p := g.PutOnBattlefield(factory(), a.PlayerID())
		q := g.PutOnBattlefield(mage.NewCreature("Land creature", "", 2, 4, mage.WithCardType(core.TypeLand), mage.WithSubTypes("Bird", "Desert"), mage.WithSuperTypes(core.SuperLegendary), mage.WithKeyword(core.Flying)), b.PlayerID())
		q.RevokeBaseAttr(core.AttrSummonSick)
		if err := mage.ApplyEffect(g, mage.Until(core.Indefinite, mage.GrantKeywordModification(mage.This, core.Vigilance)), q.ID(), b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{q.ID()}); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		q = g.FindPermanent(q.ID())
		if !q.HasType(core.TypeCreature) || !q.HasType(core.TypeLand) || !q.HasSubType("Bird") || !q.HasSubType(landType) || q.HasSubType("Desert") || !q.Card.HasSuperType(core.SuperLegendary) || q.HasKeyword(core.Flying) || !q.HasKeyword(core.Vigilance) || q.CurrentPower(g) != 2 || q.CurrentToughness(g) != 4 {
			t.Fatal("basic type set lost other characteristics or retained printed ability")
		}
		if err := g.ActivateAbilityByIndex(b.PlayerID(), q.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		if b.ManaPool().Count(core.Color(i+1)) != 1 {
			t.Fatal("wrong intrinsic mana color")
		}
	}
}

func TestRONIndefiniteTypeInventory(t *testing.T) {
	src := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreatePermanentLayerEffect(ThisPermanent,[SetLandType(Forest),AddCardtype(Artifact),UnsupportedModification])]))])`
	nodes, err := Parse(strings.NewReader(src), "inventory.ron")
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string]*InventorySignature{}
	walkInventory(nodes[0], "Inventory", "Card", "card", "collection", true, entries)
	for name, role := range map[string]string{"ThisPermanent": "permanent_reference", "SetLandType": "resolving_modification", "Forest": "land_subtype", "AddCardtype": "resolving_modification", "Artifact": "card_type"} {
		found := false
		for _, entry := range entries {
			if entry.Name == name && entry.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing %s role %s", name, role)
		}
	}
}

func TestRONIndefiniteTypeInstanceBindings(t *testing.T) {
	g, a, b := ronGame()
	first := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Ashnod's Transmogrant"), a.PlayerID())
	second := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Ashnod's Transmogrant"), a.PlayerID())
	q := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 3), b.PlayerID())
	r := g.PutOnBattlefield(mage.NewCreature("Second", "", 3, 4), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), first.ID(), 0, []uuid.UUID{q.ID()}); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), second.ID(), 0, []uuid.UUID{r.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	for _, id := range []uuid.UUID{q.ID(), r.ID()} {
		p := g.FindPermanent(id)
		if !p.HasType(core.TypeArtifact) || p.Counters[core.P1P1] != 1 {
			t.Fatal("bindings crossed card instances")
		}
	}
}

func TestRONIndefiniteTypeMonkSelfPayment(t *testing.T) {
	g, a, b := ronGame()
	p := g.PutOnBattlefield(repositoryCard(t, "FEM", "Thelonite Monk"), a.PlayerID())
	q := g.PutOnBattlefield(repositoryCard(t, "LEB", "Mountain"), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{q.ID()}); err == nil || g.FindPermanent(p.ID()) == nil || g.FindPermanent(p.ID()).Tapped || g.StackSize() != 0 {
		t.Fatal("summoning sickness did not roll back payment")
	}
	g.MutablePermanent(p.ID()).RevokeBaseAttr(core.AttrSummonSick)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{q.ID()}); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(p.ID()) != nil {
		t.Fatal("source could not pay its own sacrifice cost")
	}
	g.ResolveStack()
	if !g.FindPermanent(q.ID()).HasSubType("Forest") || g.FindPermanent(q.ID()).HasSubType("Mountain") {
		t.Fatal("source sacrifice ended the type change")
	}
}

func TestRONIndefiniteTypeRealCards(t *testing.T) {
	for _, monk := range []bool{false, true} {
		for _, change := range []string{"none", "return", "phase", "phase-cycle", "control", "source-leaves", "after-phase", "after-return"} {
			t.Run(boolName(monk)+"/"+change, func(t *testing.T) {
				a := &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("Alice")}
				b := mage.NewBasePlayer("Bob")
				g := mage.NewGame(a, b)
				g.SetTurn(1)
				g.SetStep(core.PrecombatMain)
				set, name := "ATQ", "Ashnod's Transmogrant"
				if monk {
					set, name = "FEM", "Thelonite Monk"
				}
				card := repositoryCard(t, set, name)
				if err := mage.ValidateCardDefinition(name, card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				source.RevokeBaseAttr(core.AttrSummonSick)
				var payment *mage.Permanent
				if monk {
					payment = g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), a.PlayerID())
					a.choice = payment.ID()
				}
				var victim mage.Card = mage.NewCreature("Victim", "{U}", 2, 3, mage.WithSubTypes("Bird"), mage.WithKeyword(core.Flying))
				if monk {
					victim = repositoryCard(t, "ATQ", "Mishra's Factory")
				}
				victim.SetOwner(b.PlayerID())
				target := g.PutOnBattlefield(victim, b.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{source.ID()}); err == nil {
					t.Fatal("invalid target accepted")
				}
				if g.FindPermanent(source.ID()) == nil || g.FindPermanent(source.ID()).Tapped || (monk && g.FindPermanent(payment.ID()) == nil) || g.StackSize() != 0 {
					t.Fatal("invalid target spent costs")
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
					t.Fatal(err)
				}
				if monk {
					if g.FindPermanent(payment.ID()) != nil || !g.FindPermanent(source.ID()).Tapped {
						t.Fatal("missing sacrifice or tap payment")
					}
				} else if g.FindPermanent(source.ID()) != nil || len(a.Graveyard()) != 1 {
					t.Fatal("source sacrifice missing")
				}
				if target.HasType(core.TypeArtifact) || target.Counters[core.P1P1] != 0 || target.HasSubType("Forest") {
					t.Fatal("type effect ran at announcement")
				}
				clone := g.Clone()
				switch change {
				case "return":
					g.RemoveFromBattlefield(target)
					g.PutOnBattlefield(victim, b.PlayerID())
				case "phase":
					g.PhaseOut(target.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(target.ID()))
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: target.ID(), TargetID: target.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				case "source-leaves":
					if monk {
						g.RemoveFromBattlefield(source)
					}
				}
				g.ResolveStack()
				g.PhaseIn([]uuid.UUID{target.ID()})
				changed := change != "return" && change != "phase"
				assertIndefiniteType(t, g, target.ID(), monk, changed)
				clone.ResolveStack()
				assertIndefiniteType(t, clone, target.ID(), monk, true)
				if change == "after-phase" {
					g.PhaseIn(g.PhaseOut(target.ID()))
				}
				if change == "after-return" {
					g.RemoveFromBattlefield(g.FindPermanent(target.ID()))
					g.PutOnBattlefield(victim, b.PlayerID())
					changed = false
				}
				g.GetEffects().RemoveEndOfTurn()
				g.GetEffects().RemoveEndOfCombat()
				g.ApplyContinuousEffects()
				assertIndefiniteType(t, g, target.ID(), monk, changed)
				assertIndefiniteType(t, clone, target.ID(), monk, true)
			})
		}
	}
}

func assertIndefiniteType(t *testing.T, g *mage.Game, id uuid.UUID, monk, changed bool) {
	t.Helper()
	p := g.FindPermanent(id)
	if p == nil {
		t.Fatal("missing recipient")
	}
	if monk {
		if !p.HasType(core.TypeLand) || p.HasSubType("Forest") != changed || p.Card.HasSuperType(core.SuperBasic) {
			t.Fatal("wrong land types or supertype")
		}
		if changed {
			if p.HasSubType("Assembly-Worker") || len(p.RuntimeAbilities) != 1 {
				t.Fatal("printed Factory abilities survived")
			}
			controller := p.ControllerID()
			g.MutablePermanent(id).Tapped = false
			if err := g.ActivateAbilityByIndex(controller, id, 0, nil); err != nil {
				t.Fatal("intrinsic mana unavailable", err)
			}
			if g.GetPlayer(controller).ManaPool().Count(core.Green) == 0 || g.StackSize() != 0 {
				t.Fatal("Forest mana did not resolve immediately")
			}
		}
	} else {
		count := int(0)
		if changed {
			count = 1
		}
		if p.HasType(core.TypeArtifact) != changed || !p.HasType(core.TypeCreature) || !p.HasSubType("Bird") || !p.HasKeyword(core.Flying) || p.Counters[core.P1P1] != count || p.CurrentPower(g) != 2+count || p.CurrentToughness(g) != 3+count {
			t.Fatal("artifact addition lost characteristics or counter")
		}
	}
}

func TestRONIndefiniteTypeInvalid(t *testing.T) {
	for _, action := range []string{
		`CreatePermanentLayerEffect(ThisPermanent, [])`,
		`CreatePermanentLayerEffect(ThisPermanent, [AddCardtype(Land), SetLandType(Forest)])`,
		`CreatePermanentLayerEffect(ThisPermanent, [AddCardtype(Instant)])`,
		`CreatePermanentLayerEffect(ThisPermanent, [AddCardtype(Artifact())])`,
		`CreatePermanentLayerEffect(ThisPermanent, [SetLandType(Desert)])`,
		`CreatePermanentLayerEffect(ThisPermanent, [SetLandType(Forest())])`,
		`CreatePermanentLayerEffect(ThisPermanent, [SetLandType("Forest")])`,
		`CreatePermanentLayerEffect(ThisPermanent, [SetLandType(Forest, Island)])`,
		`CreatePermanentLayerEffect(Ref_TargetPermanent, [AddCardtype(Artifact)])`,
		`CreatePermanentLayerEffect(Trigger_ThatOtherCreature, [AddCardtype(Artifact)])`,
		`CreatePermanentLayerEffect(ThisPermanent, [AddAbility([Activated(PayMana([]), ActionList([]))])])`,
		`CreatePermanentLayerEffect(ThisPermanent, [AddColor(SimpleColorList([Green]))])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+action+`]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", action)
		}
	}
}
