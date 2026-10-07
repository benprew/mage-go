package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONExtendedAnimationXenicLifetime(t *testing.T) {
	for _, change := range []string{"land", "creature", "shroud", "returned-before", "phased-before", "creature-before", "returned-after", "phase-cycle", "control", "source-left", "value-before", "value-after", "abilities"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Xenic Poltergeist"), a.PlayerID())
			g.MutablePermanent(source.ID()).RevokeBaseAttr(core.AttrSummonSick)
			card := mage.Card(mage.NewArtifact("Target", "{4}"))
			if change == "land" {
				card = mage.NewLand("Land")
			}
			if change == "creature" {
				card = mage.NewCreature("Creature", "{4}", 2, 3, mage.WithCardType(core.TypeArtifact))
			}
			card.SetOwner(b.PlayerID())
			target := g.PutOnBattlefield(card, b.PlayerID())
			if change == "shroud" {
				g.AddContinuousEffect(mage.TemporaryKeyword(target.ID(), core.Shroud))
			}
			err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()})
			if change == "land" || change == "creature" || change == "shroud" {
				if err == nil || g.FindPermanent(source.ID()).Tapped || g.StackSize() != 0 {
					t.Fatal("illegal target changed payment state")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			alter := func() {
				switch change {
				case "returned-before", "returned-after":
					g.RemoveFromBattlefield(g.FindPermanent(target.ID()))
					target = g.PutOnBattlefield(card, b.PlayerID())
				case "phased-before":
					g.PhaseOut(target.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(target.ID()))
				case "creature-before":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.AddCardType(mage.This, core.TypeCreature), mage.SetBasePowerToughness(mage.This, mage.Fixed(2), mage.Fixed(3))), target.ID(), b.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: target.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				case "source-left":
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				case "value-before", "value-after":
					g.MutablePermanent(target.ID()).Card.(*mage.BaseCard).CloneFrom(mage.NewArtifact("Seven", "{7}"))
					g.ApplyContinuousEffects()
				case "abilities":
					g.AddContinuousEffect(mage.RemoveAllAbilities(target.ID(), core.EndOfTurn))
				}
			}
			after := change == "returned-after" || change == "value-after" || change == "abilities"
			if !after {
				alter()
			}
			g.ResolveStack()
			if after {
				alter()
			}
			g.PhaseIn([]uuid.UUID{target.ID()})
			want := change != "returned-before" && change != "returned-after" && change != "phased-before" && change != "creature-before"
			if change != "creature-before" && g.FindPermanent(target.ID()).HasType(core.TypeCreature) != want {
				t.Fatal("target identity or legality")
			}
			if want {
				power := 4
				if change == "value-before" {
					power = 7
				}
				checkConditionalPT(t, g, target.ID(), power, power)
			}
		})
	}
}

func TestRONExtendedAnimationCopy(t *testing.T) {
	g, a, b := ronModalGame(0)
	card := repositoryCard(t, "ATQ", "Xenic Poltergeist")
	other := repositoryCard(t, "ATQ", "Xenic Poltergeist")
	if card.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == other.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
		t.Fatal("shared target declarations")
	}
	source := g.PutOnBattlefield(card, a.PlayerID())
	g.MutablePermanent(source.ID()).RevokeBaseAttr(core.AttrSummonSick)
	first := g.PutOnBattlefield(mage.NewArtifact("First", "{4}"), b.PlayerID())
	second := g.PutOnBattlefield(mage.NewArtifact("Second", "{7}"), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{first.ID()}); err != nil {
		t.Fatal(err)
	}
	a.targets = []uuid.UUID{second.ID()}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), true) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveTopOfStack()
	checkConditionalPT(t, g, second.ID(), 7, 7)
	if g.FindPermanent(first.ID()).HasType(core.TypeCreature) {
		t.Fatal("copy used original target")
	}
	g.ResolveStack()
	checkConditionalPT(t, g, first.ID(), 4, 4)
}

func TestRONExtendedAnimationZeroManaValue(t *testing.T) {
	g, a, b := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Xenic Poltergeist"), a.PlayerID())
	g.MutablePermanent(source.ID()).RevokeBaseAttr(core.AttrSummonSick)
	card := mage.NewArtifact("Zero", "{0}")
	card.SetOwner(b.PlayerID())
	target := g.PutOnBattlefield(card, b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(target.ID()) != nil || len(b.Graveyard()) != 1 || b.Graveyard()[0].ID() != card.ID() {
		t.Fatal("zero toughness did not use state-based graveyard movement")
	}
}

func TestRONExtendedAnimationDruidMembership(t *testing.T) {
	for _, change := range []string{"control-before", "control-after", "phase-before", "phase-cycle", "return-after", "type-after"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(repositoryCard(t, "FEM", "Thelonite Druid"), a.PlayerID())
			g.MutablePermanent(source.ID()).RevokeBaseAttr(core.AttrSummonSick)
			card := mage.NewLand("Forest", mage.WithSubTypes("Forest"))
			target := g.PutOnBattlefield(card, a.PlayerID())
			a.ManaPool().Add(core.Green, 2)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			alter := func() {
				switch change {
				case "control-before", "control-after":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: target.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "phase-before":
					g.PhaseOut(target.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(target.ID()))
				case "return-after":
					g.RemoveFromBattlefield(g.FindPermanent(target.ID()))
					target = g.PutOnBattlefield(card, a.PlayerID())
				case "type-after":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.SetSubtypeFamily(mage.This, mage.SubtypeLand, "Island")), target.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				}
			}
			after := change == "control-after" || change == "return-after" || change == "type-after"
			if !after {
				alter()
			}
			g.ResolveStack()
			if after {
				alter()
			}
			g.PhaseIn([]uuid.UUID{target.ID()})
			want := change == "control-after" || change == "phase-cycle" || change == "type-after"
			if g.FindPermanent(target.ID()).HasType(core.TypeCreature) != want {
				t.Fatal("wrong resolution membership")
			}
			if want {
				checkConditionalPT(t, g, target.ID(), 2, 3)
			}
		})
	}
}

func TestRONExtendedAnimationStrictLoad(t *testing.T) {
	files := fstest.MapFS{}
	for _, set := range []string{"FEM", "ATQ"} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.Contains(line, `Name: "Thelonite Druid"`) || strings.Contains(line, `Name: "Xenic Poltergeist"`) {
				files[set+".ron"] = &fstest.MapFile{Data: []byte(line)}
			}
		}
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	inventory, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	roles := map[string]bool{}
	for _, sig := range inventory.Signatures {
		if sig.Name == "ManaValueOfPermanent" && sig.Role == "value" {
			roles["value"] = true
		}
		if (sig.Name == "Artifact" || sig.Name == "Creature") && sig.Role == "card_type" {
			roles[sig.Name] = true
		}
	}
	if !roles["value"] || !roles["Artifact"] || !roles["Creature"] {
		t.Fatal("animation inventory lost nested roles", roles)
	}
	for _, name := range []string{"Thelonite Druid", "Xenic Poltergeist"} {
		card, err := registry.CreateCard(name)
		if err != nil {
			t.Fatal(err)
		}
		if err := mage.ValidateCardDefinition(name, card); err != nil {
			t.Fatal(err)
		}
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Unsupported",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Unsupported])`)}
	if err := LoadInto(registry, files, "*.ron"); err == nil {
		t.Fatal("partial registry accepted")
	}
	if _, err := registry.CreateCard("Xenic Poltergeist"); err != nil {
		t.Fatal("failed load changed registry")
	}
}

func TestRONExtendedAnimationCards(t *testing.T) {
	for _, name := range []string{"Thelonite Druid", "Xenic Poltergeist"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronGame()
			set := "FEM"
			if name == "Xenic Poltergeist" {
				set = "ATQ"
			}
			card := repositoryCard(t, set, name)
			if err := mage.ValidateCardDefinition(name, card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid casting")
			}
			a.ManaPool().Add(core.Colorless, 2)
			a.ManaPool().Add(core.Green, 1)
			a.ManaPool().Add(core.Black, 2)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			source := g.FindPermanent(card.ID())
			targetCard := mage.Card(mage.NewLand("Forest", mage.WithSubTypes("Forest")))
			if name == "Xenic Poltergeist" {
				targetCard = mage.NewArtifact("Artifact", "{4}", mage.WithCardType(core.TypeLand), mage.WithCardType(core.TypeEnchantment), mage.WithSubTypes("Forest"))
			}
			target := g.PutOnBattlefield(targetCard, a.PlayerID())
			var targets []uuid.UUID
			if name == "Xenic Poltergeist" {
				targets = []uuid.UUID{target.ID()}
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, targets); err == nil || g.FindPermanent(source.ID()).Tapped {
				t.Fatal("summoning sickness payment")
			}
			g.MutablePermanent(source.ID()).RevokeBaseAttr(core.AttrSummonSick)
			a.ManaPool().Clear()
			if name == "Thelonite Druid" {
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, targets); err == nil || g.FindPermanent(source.ID()).Tapped || g.StackSize() != 0 {
					t.Fatal("failed mana payment changed state")
				}
				a.ManaPool().Add(core.Colorless, 1)
				a.ManaPool().Add(core.Green, 1)
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, targets); err != nil {
				t.Fatal(err)
			}
			if name == "Thelonite Druid" && g.FindPermanent(source.ID()) != nil {
				t.Fatal("source could not pay its own sacrifice")
			}
			if target.HasType(core.TypeCreature) {
				t.Fatal("animation before resolution")
			}
			clone := g.Clone()
			g.ResolveStack()
			power, toughness := 2, 3
			if name == "Xenic Poltergeist" {
				power, toughness = 4, 4
			}
			checkConditionalPT(t, g, target.ID(), power, toughness)
			if !g.FindPermanent(target.ID()).HasType(core.TypeCreature) || !g.FindPermanent(target.ID()).HasType(core.TypeLand) || !g.FindPermanent(target.ID()).HasSubType("Forest") {
				t.Fatal("prior types or subtypes lost")
			}
			if name == "Xenic Poltergeist" && !g.FindPermanent(target.ID()).HasType(core.TypeEnchantment) {
				t.Fatal("artifact creature exception lost enchantment")
			}
			if clone.FindPermanent(target.ID()).HasType(core.TypeCreature) {
				t.Fatal("clone changed before resolution")
			}
			clone.ResolveStack()
			checkConditionalPT(t, clone, target.ID(), power, toughness)
			late := g.PutOnBattlefield(mage.NewLand("Late Forest", mage.WithSubTypes("Forest")), a.PlayerID())
			if late.HasType(core.TypeCreature) {
				t.Fatal("later entry inherited animation")
			}
			g.MutablePermanent(target.ID()).RevokeBaseAttr(core.AttrSummonSick)
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{target.ID()})
			g.ExecuteCombatDamage()
			if b.Life() != 20-power {
				t.Fatal("native animated combat damage", b.Life())
			}
			g.RunStepWithPriority(core.Cleanup)
			if name == "Xenic Poltergeist" {
				if !g.FindPermanent(target.ID()).HasType(core.TypeCreature) {
					t.Fatal("expired before upkeep")
				}
				g.SetActivePlayerIndex(1)
				g.RunStepWithPriority(core.Upkeep)
				if !g.FindPermanent(target.ID()).HasType(core.TypeCreature) {
					t.Fatal("expired at other upkeep")
				}
				g.SetActivePlayerIndex(0)
				g.RunStepWithPriority(core.Upkeep)
			}
			if g.FindPermanent(target.ID()).HasType(core.TypeCreature) {
				t.Fatal("duration did not expire")
			}
			if !clone.FindPermanent(target.ID()).HasType(core.TypeCreature) {
				t.Fatal("clone expiry isolation")
			}
		})
	}
}

func TestRONExtendedAnimationInvalid(t *testing.T) {
	for _, action := range []string{
		`CreatePermanentLayerEffectUntil(ThisPermanent,[SetCardtypes([Creature])],UntilEndOfTurn)`,
		`CreatePermanentLayerEffectUntil(ThisPermanent,[SetCardtypes([Artifact,Creature,Land])],UntilEndOfTurn)`,
		`CreatePermanentLayerEffectUntil(ThisPermanent,[SetCardtypes([Artifact,Artifact])],UntilEndOfTurn)`,
		`CreatePermanentLayerEffectUntil(ThisPermanent,[SetPowerAndToughnessBoth(ManaValueOfPermanent(Ref_TargetPermanent))],UntilEndOfTurn)`,
		`CreatePermanentLayerEffectUntil(ThisPermanent,[SetPowerAndToughnessBoth()],UntilEndOfTurn)`,
		`CreateEachPermanentLayerEffectUntil(IsCardtype(Land),[SetPT(PT(2)),AddCardtype(Creature)],UntilEndOfTurn)`,
		`CreateEachPermanentLayerEffectUntil(IsCardtype(Land),[SetPT(PT(2,3)),AddCardtype(Instant)],UntilEndOfTurn)`,
		`CreateEachPermanentLayerEffectUntil(IsCardtype(Land),[SetPT(PT(2,3)),AddCardtype(Creature)],UntilNextUpkeep(You))`,
	} {
		source := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([` + action + `]))])`
		nodes, err := Parse(strings.NewReader(source), "invalid-animation.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted invalid declaration: %s", action)
		}
	}
}
