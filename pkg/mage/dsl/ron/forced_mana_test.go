package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronForcedManaPlayer struct {
	mage.Player
	modes int
	color core.Color
}

func (p *ronForcedManaPlayer) ChooseMode(options []string, _ string) int {
	p.modes++
	return len(options) - 1
}
func (p *ronForcedManaPlayer) ChooseManaColor(string) core.Color { return p.color }

func TestRONForcedManaRecords(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			a := &ronForcedManaPlayer{Player: mage.NewBasePlayer("A"), color: core.White}
			b := &ronForcedManaPlayer{Player: mage.NewBasePlayer("B"), color: core.Green}
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			land := g.PutOnBattlefield(mage.NewLand("Choice", mage.WithManaAbility(core.Red), mage.WithManaAbility(core.AnyColor)), b.PlayerID())
			tapped := g.PutOnBattlefield(mage.NewLand("Tapped", mage.WithManaAbility(core.Blue)), b.PlayerID())
			g.TapPermanent(tapped)
			artifact := g.PutOnBattlefield(mage.NewArtifact("Artifact", "", mage.WithManaAbility(core.Red)), b.PlayerID())
			own := g.PutOnBattlefield(mage.NewLand("Own", mage.WithManaAbility(core.White)), a.PlayerID())
			b.ManaPool().AddRestricted(core.Colorless, 2, core.ArtifactSpellsOnly{})
			card := repositoryCard(t, set, "Drain Power")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err == nil {
				t.Fatal("unpaid cast succeeded")
			}
			a.ManaPool().Add(core.Blue, 2)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
				t.Fatal(err)
			}
			if land.Tapped || b.modes != 0 || b.ManaPool().TotalMana() != 2 {
				t.Fatal("activation ran before resolution")
			}
			clone := g.Clone()
			g.ResolveStack()
			if !g.FindPermanent(land.ID()).Tapped || g.FindPermanent(artifact.ID()).Tapped || g.FindPermanent(own.ID()).Tapped || b.ManaPool().TotalMana() != 0 || a.ManaPool().Count(core.Green) != 1 || a.ManaPool().Count(core.Colorless) != 2 || a.modes != 0 || b.modes != 1 {
				t.Fatal("wrong activator, selection or transfer")
			}
			if a.ManaPool().CanPay(core.ManaCost{Generic: 3}, nil) || a.ManaPool().TotalProducedThisTurn() != 2 {
				t.Fatal("transfer lost restrictions or produced new mana")
			}
			if clone.FindPermanent(land.ID()).Tapped || clone.GetPlayer(b.PlayerID()).ManaPool().TotalMana() != 2 {
				t.Fatal("resolution changed pending clone")
			}
			clone.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 3 || !clone.FindPermanent(land.ID()).Tapped {
				t.Fatal("pending clone did not resolve independently")
			}
		})
	}
}

func TestRONForcedManaInvalid(t *testing.T) {
	valid := `ActivateAManaAbilityOfEachPermanentAndLoseUnspentMana(And([IsCardtype(Land),ControlledByAPlayer(SinglePlayer(ActionPlayer))]))`
	for _, action := range []string{
		valid,
		`PlayerAction(You,ActivateAManaAbilityOfEachPermanentAndLoseUnspentMana())`,
		`PlayerAction(You,` + strings.Replace(valid, "Land", "Creature", 1) + `)`,
		`PlayerAction(You,` + strings.Replace(valid, "ActionPlayer", "You", 1) + `)`,
		`PlayerAction(You,` + strings.Replace(valid, "And([", "Or([", 1) + `)`,
		`PlayerAction(You,` + strings.Replace(valid, "IsCardtype(Land)", "IsCardtype(Land),IsTapped", 1) + `)`,
		`PlayerAction(Ref_TargetPlayer,` + valid + `)`,
	} {
		source := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([` + action + `,AddMana(TheManaLostThisWay)]))])`
		if _, err := Compile(nameScopeNode(t, source)); err == nil {
			t.Fatal("accepted", action)
		}
	}
	for _, rule := range []string{
		`AsPermanentEnters(ThisPermanent,[PlayerAction(You,` + valid + `)])`,
		`ReplaceWouldDraw(APlayerWouldDrawExceptFirstDrawStepDraw(AnyPlayer),[PlayerAction(You,` + valid + `)])`,
	} {
		if _, err := Compile(nameScopeNode(t, `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`)); err == nil {
			t.Fatal("accepted", rule)
		}
	}
}

func TestRONForcedManaCopiesAndInstances(t *testing.T) {
	g, a, b := ronGame()
	g.PutOnBattlefield(mage.NewLand("A land", mage.WithManaAbility(core.White)), a.PlayerID())
	g.PutOnBattlefield(mage.NewLand("B land", mage.WithManaAbility(core.Red)), b.PlayerID())
	first := repositoryCard(t, "LEA", "Drain Power")
	second := repositoryCard(t, "LEB", "Drain Power")
	for i, card := range []mage.Card{first, second} {
		a.ManaPool().Add(core.Blue, 2)
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		target := b.PlayerID()
		if i == 1 {
			target = a.PlayerID()
		}
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target}, 0); err != nil {
			t.Fatal(err)
		}
		if i == 0 && g.CopySpellOnStack(first.ID(), b.PlayerID(), false) == nil {
			t.Fatal("spell copy failed")
		}
		g.ResolveStack()
	}
	if a.ManaPool().Count(core.White) != 1 || a.ManaPool().Count(core.Red) != 1 || b.ManaPool().TotalMana() != 0 || a.ManaPool().TotalProducedThisTurn() != 5 || b.ManaPool().TotalProducedThisTurn() != 1 {
		t.Fatal("copies or separate factories mixed mana results or activators")
	}
}

func TestRONForcedManaPendingSource(t *testing.T) {
	for _, change := range []string{"leave", "phase", "control"} {
		g, a, b := ronGame()
		rule := `Activated(PayMana([]),Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerAction(Ref_TargetPlayer,ActivateAManaAbilityOfEachPermanentAndLoseUnspentMana(And([IsCardtype(Land),ControlledByAPlayer(SinglePlayer(ActionPlayer))]))),AddMana(TheManaLostThisWay)])))`
		card := permanentScopeFactory(t, "Artifact", rule)()
		card.SetOwner(a.PlayerID())
		source := g.PutOnBattlefield(card, a.PlayerID())
		g.PutOnBattlefield(mage.NewLand("B land", mage.WithManaAbility(core.Red)), b.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		switch change {
		case "leave":
			g.RemoveFromBattlefield(source)
		case "phase":
			g.PhaseOut(source.ID())
		case "control":
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			g.ApplyContinuousEffects()
		}
		g.ResolveStack()
		if a.ManaPool().Count(core.Red) != 1 || b.ManaPool().TotalMana() != 0 {
			t.Fatal(change, "changed pending player or result")
		}
	}
}

func TestRONForcedManaResultScope(t *testing.T) {
	producer := `PlayerAction(You,ActivateAManaAbilityOfEachPermanentAndLoseUnspentMana(And([IsCardtype(Land),ControlledByAPlayer(SinglePlayer(ActionPlayer))])))`
	for _, body := range []string{
		producer,
		`PlayerActions(You,[` + producer + `,AddMana(TheManaLostThisWay)])`,
	} {
		g, a, _ := ronGame()
		card := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([`+body+`]))`)()
		source := g.PutOnBattlefield(card, a.PlayerID())
		a.ManaPool().Add(core.Blue, 2)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		want := 0
		if strings.HasPrefix(body, "PlayerActions") {
			want = 2
		}
		if a.ManaPool().TotalMana() != want {
			t.Fatal("no-land loss or local result transfer failed")
		}
	}
	for _, body := range []string{
		`PlayerActions(You,[` + producer + `]),AddMana(TheManaLostThisWay)`,
		`MayAction(` + producer + `),AddMana(TheManaLostThisWay)`,
		`AddMana(TheManaLostThisWay),` + producer,
	} {
		if _, err := Compile(nameScopeNode(t, `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([`+body+`]))])`)); err == nil {
			t.Fatal("accepted out-of-scope result", body)
		}
	}
}
