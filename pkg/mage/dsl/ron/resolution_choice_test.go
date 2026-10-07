package ron

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronResolutionChooser struct {
	*mage.BasePlayer
	choice, prompts int
}

func (p *ronResolutionChooser) ChooseMode([]string, string) int { p.prompts++; return p.choice }

func TestRONResolutionChoiceArmorer(t *testing.T) {
	for _, mode := range []int{0, 1} {
		for _, change := range []string{"none", "source-left", "control", "phase", "phase-in", "return", "shroud", "copy", "clone"} {
			t.Run(fmt.Sprintf("%d/%s", mode, change), func(t *testing.T) {
				a := &ronResolutionChooser{BasePlayer: mage.NewBasePlayer("A"), choice: mode}
				b := &ronResolutionChooser{BasePlayer: mage.NewBasePlayer("B"), choice: 1 - mode}
				g := mage.NewGame(a, b)
				g.SetStep(core.PrecombatMain)
				card := repositoryCard(t, "FEM", "Dwarven Armorer")
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
					t.Fatal("unpaid cast")
				}
				a.ManaPool().Add(core.Red, 1)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				source := g.FindPermanent(card.ID())
				if source.CurrentPower(g) != 0 || source.CurrentToughness(g) != 2 || !source.HasSubType("Dwarf") {
					t.Fatal("wrong metadata")
				}
				targetCard := mage.NewCreature("Target", "", 2, 3)
				targetCard.SetOwner(b.PlayerID())
				target := g.PutOnBattlefield(targetCard, b.PlayerID())
				discard := mage.NewLand("Payment")
				discard.SetOwner(a.PlayerID())
				a.AddToHand(discard)
				a.ManaPool().Add(core.Red, 1)
				activate := func(id uuid.UUID) error {
					return g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{id})
				}
				if activate(target.ID()) == nil || len(a.Hand()) != 1 || a.ManaPool().TotalMana() != 1 || g.FindPermanent(source.ID()).Tapped {
					t.Fatal("sickness rollback")
				}
				g.MutablePermanent(source.ID()).RevokeBaseAttr(core.AttrSummonSick)
				if activate(b.PlayerID()) == nil || len(a.Hand()) != 1 || a.ManaPool().TotalMana() != 1 || g.StackSize() != 0 {
					t.Fatal("illegal target rollback")
				}
				if err := activate(target.ID()); err != nil {
					t.Fatal(err)
				}
				if a.prompts != 0 || len(a.Hand()) != 0 || len(a.Graveyard()) != 1 || a.ManaPool().TotalMana() != 0 || !g.FindPermanent(source.ID()).Tapped || target.CurrentPower(g) != 2 {
					t.Fatal("choice or payment timing")
				}
				switch change {
				case "source-left":
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "phase":
					g.PhaseOut(target.ID())
				case "phase-in":
					g.PhaseIn(g.PhaseOut(target.ID()))
				case "return":
					g.RemoveFromBattlefield(target)
					target = g.PutOnBattlefield(targetCard, b.PlayerID())
				case "shroud":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.GrantKeywordModification(mage.This, core.Shroud)), target.ID(), b.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				}
				clone := g.Clone()
				g.ResolveStack()
				p := g.FindPermanentIncludingPhased(target.ID())
				wantPower, wantToughness, wantPrompts := 2+mode, 4-mode, 1
				if change == "phase" || change == "return" || change == "shroud" {
					wantPower, wantToughness, wantPrompts = 2, 3, 0
				}
				if change == "copy" {
					wantPower, wantToughness = 3, 4
					if b.prompts != 1 {
						t.Fatal("copy did not choose")
					}
				}
				if p.CurrentPower(g) != wantPower || p.CurrentToughness(g) != wantToughness || a.prompts != wantPrompts {
					t.Fatalf("wrong branch: %d/%d prompts %d", p.CurrentPower(g), p.CurrentToughness(g), a.prompts)
				}
				if clone.FindPermanentIncludingPhased(target.ID()).CurrentPower(clone) != 2 {
					t.Fatal("clone shares counters")
				}
				if change == "clone" {
					clone.ResolveStack()
					if clone.FindPermanent(target.ID()).CurrentToughness(clone) != 4 {
						t.Fatal("clone lost choice frame")
					}
				}
				g.GetEffects().RemoveEndOfTurn()
				g.ApplyContinuousEffects()
				p = g.FindPermanentIncludingPhased(p.ID())
				if p.CurrentPower(g) != wantPower || p.CurrentToughness(g) != wantToughness {
					t.Fatal("counters expired")
				}
			})
		}
	}
}

func TestRONResolutionChoiceAvenger(t *testing.T) {
	for mode, keyword := range []core.Attr{core.Banding, core.Flying, core.FirstStrike, core.Trample} {
		for _, change := range []string{"none", "phase", "phase-in", "return", "control", "copy", "remove-abilities"} {
			t.Run(fmt.Sprintf("%d/%s", mode, change), func(t *testing.T) {
				a := &ronResolutionChooser{BasePlayer: mage.NewBasePlayer("A"), choice: mode}
				b := &ronResolutionChooser{BasePlayer: mage.NewBasePlayer("B"), choice: (mode + 1) % 4}
				g := mage.NewGame(a, b)
				g.SetStep(core.PrecombatMain)
				card := repositoryCard(t, "ATQ", "Urza's Avenger")
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
					t.Fatal("unpaid cast")
				}
				a.ManaPool().Add(core.Colorless, 6)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				p := g.FindPermanent(card.ID())
				if !p.HasType(core.TypeArtifact) || !p.HasSubType("Shapeshifter") || p.CurrentPower(g) != 4 || p.CurrentToughness(g) != 4 {
					t.Fatal("wrong metadata")
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if a.prompts != 0 || p.CurrentPower(g) != 4 {
					t.Fatal("choice occurred at activation")
				}
				switch change {
				case "phase":
					g.PhaseOut(p.ID())
				case "phase-in":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "return":
					g.RemoveFromBattlefield(p)
					p = g.PutOnBattlefield(card, a.PlayerID())
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				case "remove-abilities":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), p.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				}
				clone := g.Clone()
				g.ResolveStack()
				p = g.FindPermanentIncludingPhased(p.ID())
				want := 3
				if change == "phase" || change == "return" {
					want = 4
				}
				if change == "copy" {
					want = 2
					if b.prompts != 1 || !p.HasKeyword([]core.Attr{core.Banding, core.Flying, core.FirstStrike, core.Trample}[(mode+1)%4]) {
						t.Fatal("copy choice lost")
					}
				}
				if p.CurrentPower(g) != want || p.CurrentToughness(g) != want || p.HasKeyword(keyword) != (want < 4) || a.prompts != 1 {
					t.Fatal("wrong chosen ability, identity or controller")
				}
				if clone.FindPermanentIncludingPhased(p.ID()).CurrentPower(clone) != 4 {
					t.Fatal("clone shared choice")
				}
				g.GetEffects().RemoveEndOfTurn()
				g.ApplyContinuousEffects()
				p = g.FindPermanentIncludingPhased(p.ID())
				if p.CurrentPower(g) != 4 || p.HasKeyword(keyword) {
					t.Fatal("choice did not expire")
				}
			})
		}
	}
}

func TestRONResolutionChoiceInvalid(t *testing.T) {
	for _, body := range []string{
		`ChooseAnAction`, `ChooseAnAction()`, `ChooseAnAction([])`, `ChooseAnAction([[]])`,
		`ChooseAnAction([[DrawACard]])`, `ChooseAnAction([[Targeted([],ActionList([]))],[DrawACard]])`,
		`ChooseAnAbility`, `ChooseAnAbility([])`, `ChooseAnAbility([Flying])`, `ChooseAnAbility([Flying(),Trample])`,
		`ChooseAnAbility([Flying,Rampage(-1)]),CreatePermanentLayerEffectUntil(ThisPermanent,[AddAbilityVariable(TheChosenAbility)],UntilEndOfTurn)`,
		`ChooseAnAbility([Flying,Trample]),GainLife(Integer(1))`,
		`ChooseAnAbility([Flying,Flying]),CreatePermanentLayerEffectUntil(ThisPermanent,[AddAbilityVariable(TheChosenAbility)],UntilEndOfTurn)`,
		`ChooseAnAbility([Flying,Trample]),CreatePermanentLayerEffectUntil(ThisPermanent,[AddAbility([Flying])],UntilEndOfTurn)`,
		`CreatePermanentLayerEffectUntil(ThisPermanent,[AddAbilityVariable(TheChosenAbility)],UntilEndOfTurn)`,
		`ChooseAnAbility([Flying,Trample]),CreatePermanentLayerEffectUntil(ThisPermanent,[AddAbilityVariable(TheChosenAbility())],UntilEndOfTurn)`,
		`ChooseAnAbility([Flying,Trample]),CreatePermanentLayerEffectUntil(Ref_TargetPermanent,[AddAbilityVariable(TheChosenAbility)],UntilEndOfTurn)`,
		`ChooseAnAbility([Flying,Trample]),CreatePermanentLayerEffectUntil(ThisPermanent,[AddAbilityVariable(TheChosenAbility)],UntilEndOfTurn),CreatePermanentLayerEffectUntil(ThisPermanent,[AddAbilityVariable(TheChosenAbility)],UntilEndOfTurn)`,
	} {
		source := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([` + body + `]))])`
		nodes, err := Parse(strings.NewReader(source), "invalid-choice.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("invalid choice accepted", body)
		}
	}
}

func TestRONResolutionChoicePendingInstances(t *testing.T) {
	a := &ronResolutionChooser{BasePlayer: mage.NewBasePlayer("A"), choice: 0}
	b := &ronResolutionChooser{BasePlayer: mage.NewBasePlayer("B"), choice: 1}
	g := mage.NewGame(a, b)
	first := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Urza's Avenger"), a.PlayerID())
	second := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Urza's Avenger"), b.PlayerID())
	for _, p := range []*mage.Permanent{first, second} {
		if err := g.ActivateAbilityByIndex(p.ControllerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
	}
	clone := g.Clone()
	g.ResolveStack()
	if !g.FindPermanent(first.ID()).HasKeyword(core.Banding) || g.FindPermanent(first.ID()).HasKeyword(core.Flying) || !g.FindPermanent(second.ID()).HasKeyword(core.Flying) || g.FindPermanent(second.ID()).HasKeyword(core.Banding) {
		t.Fatal("instances share choices")
	}
	a.choice = 2
	if err := g.ActivateAbilityByIndex(a.PlayerID(), first.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	a.choice = 3
	g.ResolveStack()
	p := g.FindPermanent(first.ID())
	if p.CurrentPower(g) != 2 || !p.HasKeyword(core.Banding) || !p.HasKeyword(core.Trample) || p.HasKeyword(core.FirstStrike) || a.prompts != 2 {
		t.Fatal("repeated activation saved the early choice")
	}
	clone.ResolveStack()
	if clone.FindPermanent(first.ID()).CurrentPower(clone) != 3 || !clone.FindPermanent(first.ID()).HasKeyword(core.Banding) || clone.FindPermanent(first.ID()).HasKeyword(core.Trample) {
		t.Fatal("pending clone shares choice state")
	}
	g.RunStepWithPriority(core.Cleanup)
	if g.FindPermanent(first.ID()).CurrentPower(g) != 4 || g.FindPermanent(first.ID()).HasKeyword(core.Banding) || g.FindPermanent(second.ID()).HasKeyword(core.Flying) {
		t.Fatal("native cleanup did not end choices")
	}
}

func TestRONResolutionChoiceContexts(t *testing.T) {
	choice := `ChooseAnAbility([Flying,Trample]),CreatePermanentLayerEffectUntil(Ref_TargetPermanent,[AddAbilityVariable(TheChosenAbility)],UntilEndOfTurn)`
	body := `Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([` + choice + `]))`
	for _, rule := range []string{`SpellActions(` + body + `)`, `Activated(PayMana([]),` + body + `)`, `TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),` + body + `)`, `SpellActions(Modal_ChooseOne([` + body + `,ActionList([GainLife(Integer(1))])]))`} {
		kind := "Artifact"
		if strings.HasPrefix(rule, "SpellActions") {
			kind = "Instant"
		}
		factory := permanentScopeFactory(t, kind, rule)
		if card := factory(); mage.ValidateCardDefinition(card.Name(), card) != nil {
			t.Fatal("choice lost target binding")
		}
	}
	a := &ronResolutionChooser{BasePlayer: mage.NewBasePlayer("A"), choice: 1}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	factory := permanentScopeFactory(t, "Artifact", `TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([ChooseAnAbility([Flying,Trample]),CreatePermanentLayerEffectUntil(ThisPermanent,[AddAbilityVariable(TheChosenAbility)],UntilNextUpkeep(You))]))`)
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if a.prompts != 0 || g.StackSize() != 1 {
		t.Fatal("trigger choice was early")
	}
	g.ResolveStack()
	if !g.FindPermanent(p.ID()).HasKeyword(core.Trample) || g.FindPermanent(p.ID()).HasKeyword(core.Flying) {
		t.Fatal("trigger choice failed")
	}
	g.SetActivePlayerIndex(1)
	g.RunStepWithPriority(core.Upkeep)
	if !g.FindPermanent(p.ID()).HasKeyword(core.Trample) {
		t.Fatal("choice expired at opponent upkeep")
	}
	g.SetActivePlayerIndex(0)
	g.SetStep(core.Upkeep)
	g.GetEffects().RemoveEndOfTurn()
	// Native upkeep ends the old duration before it places the new trigger.
	a.choice = 0
	g.RunStepWithPriority(core.Upkeep)
	if !g.FindPermanent(p.ID()).HasKeyword(core.Flying) || g.FindPermanent(p.ID()).HasKeyword(core.Trample) {
		t.Fatal("next upkeep did not replace the expired choice")
	}
}

func TestRONResolutionChoiceInventoryAndAtomicLoad(t *testing.T) {
	files := fstest.MapFS{}
	for _, tc := range []struct{ set, name string }{{"FEM", "Dwarven Armorer"}, {"ATQ", "Urza's Avenger"}} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + tc.set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.Contains(line, `Name: "`+tc.name+`"`) {
				files[tc.set+".ron"] = &fstest.MapFile{Data: []byte(line)}
			}
		}
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Dwarven Armorer", "Urza's Avenger"} {
		first, err := registry.CreateCard(name)
		if err != nil {
			t.Fatal(err)
		}
		second, err := registry.CreateCard(name)
		if err != nil {
			t.Fatal(err)
		}
		if first.ID() == second.ID() || first.Abilities()[0] == second.Abilities()[0] || mage.ValidateCardDefinition(name, first) != nil {
			t.Fatal("invalid or shared factory")
		}
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 2 || len(report.Issues) != 0 {
		t.Fatal("valid choice records rejected")
	}
	roles := map[string]string{"ChooseAnAction": "action", "ChooseAnAbility": "action", "TheChosenAbility": "ability_reference", "PTCounter": "permanent_counter_type"}
	for name, role := range roles {
		found := false
		for _, signature := range report.Signatures {
			if signature.Name == name && signature.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatal("missing inventory role", name, role)
		}
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([ChooseAnAbility([Flying,Rampage(-1)]),CreatePermanentLayerEffectUntil(ThisPermanent,[AddAbilityVariable(TheChosenAbility)],UntilEndOfTurn)]))])`)}
	if err := LoadInto(registry, files, "*.ron"); err == nil || !registry.CardRegistered("Dwarven Armorer") || !registry.CardRegistered("Urza's Avenger") || registry.CardRegistered("Bad") {
		t.Fatal("invalid collection replaced registry")
	}
	if _, err := Compile(chosenDiscardRecord(t, "LEG", "Gabriel Angelfire")); err != nil {
		t.Fatal("complete rampage choice rejected", err)
	}
}
