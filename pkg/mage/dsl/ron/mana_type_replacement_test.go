package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONManaTypeReplacementCompleteRecords(t *testing.T) {
	for _, name := range []string{"Deep Water", "Quarum Trench Gnomes"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronGame()
			set := "DRK"
			if name == "Quarum Trench Gnomes" {
				set = "LEG"
			}
			card := repositoryCard(t, set, name)
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			land := g.PutOnBattlefield(mage.NewLand("Mixed Plains", mage.WithSubTypes("Plains"), mage.WithMultiManaAbility(mage.ManaProduction{Color: core.White, Amount: 2}, mage.ManaProduction{Color: core.Green, Amount: 1})), a.PlayerID())
			targets := []uuid.UUID(nil)
			if name == "Quarum Trench Gnomes" {
				targets = []uuid.UUID{land.ID()}
			} else {
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil {
					t.Fatal("unpaid activation")
				}
				if g.StackSize() != 0 || land.Tapped {
					t.Fatal("failed payment changed state")
				}
				a.ManaPool().Add(core.Blue, 1)
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, targets); err != nil {
				t.Fatal(err)
			}
			if g.StackSize() != 1 {
				t.Fatal("replacement activation must use stack")
			}
			pending := g.Clone()
			g.ResolveStack()
			pending.ResolveStack()
			g.BouncePermanentToHand(g.FindPermanent(source.ID()))
			if err := g.TapForMana(a.PlayerID(), land.ID()); err != nil {
				t.Fatal(err)
			}
			if name == "Deep Water" {
				if a.ManaPool().Count(core.Blue) != 3 || a.ManaPool().Count(core.Green) != 0 {
					t.Fatal("wrong all-type production")
				}
			} else {
				if a.ManaPool().Count(core.Colorless) != 2 || a.ManaPool().Count(core.Green) != 1 {
					t.Fatal("must replace only white mana")
				}
			}
			other := g.PutOnBattlefield(mage.NewLand("Other", mage.WithManaAbility(core.White)), b.PlayerID())
			if err := g.TapForMana(b.PlayerID(), other.ID()); err != nil {
				t.Fatal(err)
			}
			if b.ManaPool().Count(core.White) != 1 {
				t.Fatal("wrong player or subject")
			}
			if pending.FindPermanent(land.ID()).Tapped {
				t.Fatal("clone shared tap state")
			}
			if err := pending.TapForMana(a.PlayerID(), land.ID()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRONManaTypeReplacementInvalid(t *testing.T) {
	event := `PlayerTappingAPermanentWouldProduceMana(You,And([IsCardtype(Land),ControlledByAPlayer(SinglePlayer(You))]))`
	for _, action := range []string{
		`CreateReplaceWouldProduceManaUntil`,
		`CreateReplaceWouldProduceManaUntil(` + event + `,[],UntilEndOfTurn)`,
		`CreateReplaceWouldProduceManaUntil(` + event + `,[WouldProduceMana_ReplaceType(AnyManaColor)],UntilEndOfTurn)`,
		`CreateReplaceWouldProduceManaUntil(` + event + `,[WouldProduceMana_ReplaceType(ManaProduceU),WouldProduceMana_ReplaceType(ManaProduceR)],UntilEndOfTurn)`,
		`CreateReplaceWouldProduceManaUntil(TappingPermanentWouldProduceMana(Ref_TargetPermanent),[WouldProduceMana_ReplaceType(ManaProduceU)],UntilEndOfTurn)`,
		`CreateReplaceWouldProduceManaUntil(` + event + `,[WouldProduceMana_ReplaceType(ManaProduceU)],UntilPermanentLeavesBattlefield(ThisPermanent))`,
		`CreateReplaceWouldProduceManaUntil(PlayerTappingAPermanentWouldProduceMana(You,IsCardtype(Land)),[WouldProduceMana_ReplaceType(ManaProduceU)],UntilEndOfTurn)`,
		`CreateReplaceWouldProduceManaUntil(` + event + `,[WouldProduceMana_ReplaceTypeOfType(ManaProduceW,ManaProduceC)],UntilEndOfTurn)`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([` + action + `]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", action)
		}
	}
}

func TestRONManaTypeReplacementLifecycle(t *testing.T) {
	for _, name := range []string{"Deep Water", "Quarum Trench Gnomes"} {
		for _, change := range []string{"illegal-type", "shroud", "return-before", "phase-before", "phase-cycle", "control", "copy", "installed-return", "cleanup", "removed"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				set := "DRK"
				if name == "Quarum Trench Gnomes" {
					set = "LEG"
				}
				source := g.PutOnBattlefield(repositoryCard(t, set, name), a.PlayerID())
				source.RevokeBaseAttr(core.AttrSummonSick)
				card := mage.NewLand("Plains", mage.WithSubTypes("Plains"), mage.WithManaAbility(core.White))
				card.SetOwner(a.PlayerID())
				land := g.PutOnBattlefield(card, a.PlayerID())
				targets := []uuid.UUID(nil)
				if name == "Quarum Trench Gnomes" {
					targets = []uuid.UUID{land.ID()}
				} else {
					a.ManaPool().Add(core.Blue, 1)
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, targets); err != nil {
					t.Fatal(err)
				}
				replaced := true
				player := a.PlayerID()
				switch change {
				case "illegal-type":
					g.AddContinuousEffect(mage.BecomesBasicLandTargetEffect(land.ID(), core.EndOfTurn, "Forest"))
					replaced = name == "Deep Water"
				case "shroud":
					g.MutablePermanent(land.ID()).GrantBaseAttr(core.Shroud)
					replaced = name == "Deep Water"
				case "return-before":
					g.BouncePermanentToHand(g.FindPermanent(land.ID()))
					a.RemoveFromHand(card.ID())
					land = g.PutOnBattlefield(card, a.PlayerID())
					replaced = name == "Deep Water"
				case "phase-before":
					g.PhaseOut(land.ID())
					g.ResolveStack()
					g.PhaseIn([]uuid.UUID{land.ID()})
					replaced = name == "Deep Water"
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(land.ID()))
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: land.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					player = b.PlayerID()
					replaced = name == "Quarum Trench Gnomes"
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				}
				g.ResolveStack()
				switch change {
				case "installed-return":
					g.BouncePermanentToHand(g.FindPermanent(land.ID()))
					a.RemoveFromHand(card.ID())
					land = g.PutOnBattlefield(card, a.PlayerID())
					replaced = name == "Deep Water"
				case "cleanup":
					g.RunStepWithPriority(core.Cleanup)
					replaced = name == "Quarum Trench Gnomes"
				case "removed":
					g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.EndOfTurn))
					g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				}
				if err := g.TapForMana(player, land.ID()); err != nil {
					t.Fatal(err)
				}
				want := core.White
				if change == "illegal-type" {
					want = core.Green
				}
				if replaced {
					want = core.Blue
					if name == "Quarum Trench Gnomes" {
						want = core.Colorless
					}
				}
				if g.GetPlayer(player).ManaPool().Count(want) != 1 {
					t.Fatalf("pool = %v, want %v", g.GetPlayer(player).ManaPool(), want)
				}
			})
		}
	}
}

func TestRONManaTypeReplacementGnomesTargetPayment(t *testing.T) {
	g, a, b := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "LEG", "Quarum Trench Gnomes"), a.PlayerID())
	land := g.PutOnBattlefield(mage.NewLand("Land", mage.WithManaAbility(core.White)), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{land.ID()}); err == nil || source.Tapped {
		t.Fatal("illegal target paid cost")
	}
	plains := g.PutOnBattlefield(mage.NewLand("Plains", mage.WithSubTypes("Plains"), mage.WithManaAbility(core.White)), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{plains.ID()}); err == nil || source.Tapped {
		t.Fatal("summoning sick source paid tap")
	}
	source.RevokeBaseAttr(core.AttrSummonSick)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{plains.ID()}); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{plains.ID()}); err == nil {
		t.Fatal("tapped source activated")
	}
	g.ResolveStack()
	if err := g.AutoTapForCost(b.PlayerID(), core.ManaCost{Generic: 1}); err != nil {
		t.Fatal(err)
	}
	if b.ManaPool().Count(core.Colorless) != 1 {
		t.Fatal("opponent target was not replaced")
	}
}

func TestRONManaTypeReplacementCopyAndInstances(t *testing.T) {
	g, a, b := ronModalGame(0)
	source := g.PutOnBattlefield(repositoryCard(t, "LEG", "Quarum Trench Gnomes"), a.PlayerID())
	source.RevokeBaseAttr(core.AttrSummonSick)
	first := g.PutOnBattlefield(mage.NewLand("First", mage.WithSubTypes("Plains"), mage.WithManaAbility(core.White)), a.PlayerID())
	second := g.PutOnBattlefield(mage.NewLand("Second", mage.WithSubTypes("Plains"), mage.WithManaAbility(core.White)), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{first.ID()}); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	pending := g.Clone()
	a.targets = []uuid.UUID{second.ID()}
	copied := g.CopyStackObjectDirect(original, a.PlayerID(), true)
	if copied == nil || copied.Targets[0] != second.ID() || original.Targets[0] != first.ID() {
		t.Fatal("copy did not keep independent target")
	}
	g.ResolveStack()
	pending.ResolveStack()
	installed := g.Clone()
	for _, p := range []*mage.Permanent{first, second} {
		if err := g.TapForMana(p.ControllerID(), p.ID()); err != nil {
			t.Fatal(err)
		}
	}
	if a.ManaPool().Count(core.Colorless) != 1 || b.ManaPool().Count(core.Colorless) != 1 {
		t.Fatal("copy and original shared subject")
	}
	if err := pending.TapForMana(b.PlayerID(), second.ID()); err != nil {
		t.Fatal(err)
	}
	if pending.GetPlayer(b.PlayerID()).ManaPool().Count(core.White) != 1 {
		t.Fatal("pending clone inherited copy")
	}
	g.BouncePermanentToHand(g.FindPermanent(second.ID()))
	if err := installed.TapForMana(b.PlayerID(), second.ID()); err != nil {
		t.Fatal(err)
	}
	if installed.GetPlayer(b.PlayerID()).ManaPool().Count(core.Colorless) != 1 {
		t.Fatal("installed clone shared duration")
	}
	firstCard := repositoryCard(t, "LEG", "Quarum Trench Gnomes")
	fresh := rebuildRONDefinition(t, firstCard)
	if firstCard.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == fresh.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
		t.Fatal("definition rebuild shared target")
	}
}

func TestRONManaTypeReplacementInventory(t *testing.T) {
	src := `Card(Name:"Types",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreateReplaceWouldProduceManaUntil(PlayerTappingAPermanentWouldProduceMana(You,ControlledByAPlayer(SinglePlayer(You))),[WouldProduceMana_ReplaceType(ManaProduceU)],UntilEndOfTurn)]))])`
	report, err := InspectFS(fstest.MapFS{"types.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"PlayerTappingAPermanentWouldProduceMana": "mana_replacement_event", "WouldProduceMana_ReplaceType": "mana_replacement_operation", "ManaProduceU": "mana_type"} {
		found := false
		for _, signature := range report.Signatures {
			if signature.Name == name && signature.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing %s role for %s", role, name)
		}
	}
}
