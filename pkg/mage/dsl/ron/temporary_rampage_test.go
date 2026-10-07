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

func TestRONTemporaryRampageCompleteRecords(t *testing.T) {
	for _, name := range []string{"Gabriel Angelfire", "Rapid Fire"} {
		t.Run(name, func(t *testing.T) {
			a := &ronResolutionChooser{BasePlayer: mage.NewBasePlayer("A"), choice: 3}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			c := repositoryCard(t, "LEG", name)
			if err := mage.ValidateCardDefinition(name, c); err != nil {
				t.Fatal(err)
			}
			c.SetOwner(a.PlayerID())
			a.AddToHand(c)
			target := g.PutOnBattlefield(mage.NewCreature("Target", "", 2, 6), a.PlayerID())
			targets := []uuid.UUID{target.ID()}
			if name == "Gabriel Angelfire" {
				targets = nil
			}
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), targets, 0); err == nil {
				t.Fatal("unpaid spell")
			}
			if len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("failed cast changed state")
			}
			a.ManaPool().Add(core.White, 2)
			a.ManaPool().Add(core.Green, 2)
			a.ManaPool().Add(core.Colorless, 3)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), targets, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			bonus := 2
			if name == "Gabriel Angelfire" {
				target = g.FindPermanent(c.ID())
				bonus = 3
				g.RunStepWithPriority(core.Upkeep)
				if a.prompts != 1 {
					t.Fatal("upkeep did not choose ability")
				}
			}
			if !mage.HasRampage.Match(g.FindPermanent(target.ID()), g) {
				t.Fatal("missing rampage grant")
			}
			base := target.CurrentPower(g)
			g.GetCombat().AddAttacker(target.ID(), b.PlayerID())
			for range 2 {
				g.PutOnBattlefieldBlocking(mage.NewCreature("Blocker", "", 1, 1), b.PlayerID(), target.ID())
			}
			g.PutTriggersOnStack()
			g.ResolveStack()
			if g.FindPermanent(target.ID()).CurrentPower(g) != base+bonus {
				t.Fatal("wrong rampage bonus")
			}
			g.RunStepWithPriority(core.Cleanup)
			if g.FindPermanent(target.ID()).CurrentPower(g) != base {
				t.Fatal("bonus survived cleanup")
			}
			if mage.HasRampage.Match(g.FindPermanent(target.ID()), g) != (name == "Gabriel Angelfire") {
				t.Fatal("wrong grant duration")
			}
		})
	}
}

func TestRONTemporaryRampageRapidFireLifecycle(t *testing.T) {
	for _, change := range []string{"none", "existing", "gained", "lost", "returned", "phased", "phase-cycle", "shroud", "copy", "installed-return", "installed-phase"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			opts := []mage.CardOption{}
			if change == "existing" || change == "lost" {
				opts = append(opts, mage.WithAbility(mage.RampageTrigger(5)))
			}
			card := mage.NewCreature("Target", "", 2, 6, opts...)
			card.SetOwner(b.PlayerID())
			target := g.PutOnBattlefield(card, b.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 6), a.PlayerID())
			spell := repositoryCard(t, "LEG", "Rapid Fire")
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			a.ManaPool().Add(core.White, 1)
			a.ManaPool().Add(core.Colorless, 3)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{target.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			if target.HasKeyword(core.FirstStrike) {
				t.Fatal("grant started at announcement")
			}
			clone := g.Clone()
			returnTarget := func() {
				g.BouncePermanentToHand(g.FindPermanent(target.ID()))
				b.RemoveFromHand(card.ID())
				target = g.PutOnBattlefield(card, b.PlayerID())
			}
			switch change {
			case "gained":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.TriggeredAbilityModification(mage.This, mage.WhenEvent(mage.BecomesBlocked(mage.This), mage.RampageEffect(5)).(*mage.GenericTriggered))), target.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "lost":
				g.AddContinuousEffect(mage.RemoveAllAbilities(target.ID(), core.EndOfTurn))
			case "returned":
				returnTarget()
			case "phased":
				g.PhaseOut(target.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(target.ID()))
			case "shroud":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.GrantKeywordModification(mage.This, core.Shroud)), target.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "copy":
				phased := g.PhaseOut(target.ID())
				copied := g.CopySpellOnStack(spell.ID(), b.PlayerID(), true)
				if copied == nil || len(copied.Targets) != 1 || copied.Targets[0] != other.ID() {
					t.Fatal("copy failed to retarget")
				}
				g.PhaseIn(phased)
			}
			g.ResolveStack()
			if change == "installed-return" {
				returnTarget()
			}
			if change == "installed-phase" {
				g.PhaseIn(g.PhaseOut(target.ID()))
			}
			g.PhaseIn([]uuid.UUID{target.ID()})
			granted := change != "returned" && change != "phased" && change != "shroud" && change != "installed-return"
			p := g.FindPermanent(target.ID())
			if p.HasKeyword(core.FirstStrike) != granted || mage.HasRampage.Match(p, g) != granted {
				t.Fatal("wrong subject or ability grant")
			}
			if mage.HasRampage.Match(g.FindPermanent(other.ID()), g) != (change == "copy") {
				t.Fatal("copy used wrong target")
			}
			if clone.FindPermanent(card.ID()).HasKeyword(core.FirstStrike) {
				t.Fatal("clone shared layer state")
			}
			clone.ResolveStack()
			if !clone.FindPermanent(card.ID()).HasKeyword(core.FirstStrike) {
				t.Fatal("clone lost pending spell")
			}
			if granted {
				g.GetCombat().AddAttacker(p.ID(), a.PlayerID())
				for range 2 {
					g.PutOnBattlefieldBlocking(mage.NewCreature("Blocker", "", 1, 1), a.PlayerID(), p.ID())
				}
				g.PutTriggersOnStack()
				if g.StackSize() != 1 {
					t.Fatalf("rampage instances=%d", g.StackSize())
				}
				g.ResolveStack()
				bonus := 2
				if change == "existing" || change == "gained" {
					bonus = 5
				}
				if g.FindPermanent(p.ID()).CurrentPower(g) != 2+bonus {
					t.Fatal("condition used announcement abilities")
				}
			}
		})
	}
}

func TestRONTemporaryRampageGabrielChoices(t *testing.T) {
	for choice := range 4 {
		for _, change := range []string{"none", "control", "return", "phase", "phase-cycle", "copy"} {
			t.Run(fmt.Sprintf("%d/%s", choice, change), func(t *testing.T) {
				a := &ronResolutionChooser{BasePlayer: mage.NewBasePlayer("A"), choice: choice}
				b := &ronResolutionChooser{BasePlayer: mage.NewBasePlayer("B"), choice: (choice + 1) % 4}
				g := mage.NewGame(a, b)
				card := repositoryCard(t, "LEG", "Gabriel Angelfire")
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
				g.PutTriggersOnStack()
				if g.StackSize() != 1 || a.prompts != 0 {
					t.Fatal("upkeep choice timing")
				}
				switch change {
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "return":
					g.BouncePermanentToHand(p)
					a.RemoveFromHand(card.ID())
					p = g.PutOnBattlefield(card, a.PlayerID())
				case "phase":
					g.PhaseOut(p.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				}
				g.ResolveStack()
				g.PhaseIn([]uuid.UUID{p.ID()})
				has := func(choice int) bool {
					p := g.FindPermanent(card.ID())
					if choice == 3 {
						return mage.HasRampage.Match(p, g)
					}
					return p.HasKeyword([]core.Attr{core.Flying, core.FirstStrike, core.Trample}[choice])
				}
				if a.prompts != 1 || has(choice) != (change != "return" && change != "phase") {
					t.Fatal("choice controller or source identity")
				}
				if change == "copy" && (b.prompts != 1 || !has((choice+1)%4)) {
					t.Fatal("copied trigger shared choice")
				}
				g.SetActivePlayerIndex(1)
				g.RunStepWithPriority(core.Upkeep)
				if change != "return" && change != "phase" && !has(choice) {
					t.Fatal("grant expired at opponent upkeep")
				}
				g.GetEffects().RemoveUntilYourNextTurn(g, a.PlayerID())
				g.ApplyContinuousEffects()
				if has(choice) {
					t.Fatal("grant survived saved player's next upkeep")
				}
			})
		}
	}
}

func TestRONTemporaryRampageInvalid(t *testing.T) {
	for _, ability := range []string{`Rampage`, `Rampage()`, `Rampage(-1)`, `Rampage(Integer(2))`, `Rampage(ValueX)`, `Rampage(2,3)`, `AnyRampage`, `AnyRampage()`} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreatePermanentLayerEffectUntil(ThisPermanent,[AddAbility([` + ability + `])],UntilEndOfTurn)]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid-rampage.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid ability accepted", ability)
		}
	}
	for _, body := range []string{
		`ChooseAnAbility([Rampage(2),Rampage(2)]),CreatePermanentLayerEffectUntil(ThisPermanent,[AddAbilityVariable(TheChosenAbility)],UntilEndOfTurn)`,
		`CreatePermanentLayerEffectUntil(Ref_TargetPermanent,[AddAbility([Rampage(2)])],UntilEndOfTurn)`,
		`If(PermanentPassesFilter(ThisPermanent,DoesntHaveAbility(AnyRampage())),[GainLife(Integer(1))])`,
		`ChooseAnAbility([Flying,Rampage(2)]),CreatePermanentLayerEffectUntil(ThisPermanent,[AddAbilityVariable(TheChosenAbility)],UntilEndOfTurn),CreatePermanentLayerEffectUntil(ThisPermanent,[AddAbilityVariable(TheChosenAbility)],UntilEndOfTurn)`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([` + body + `]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid-rampage.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid binding or choice accepted", body)
		}
	}
}

func TestRONTemporaryRampageStrictLoad(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var records []string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Gabriel Angelfire"`) || strings.Contains(line, `Name: "Rapid Fire"`) {
			records = append(records, line)
		}
	}
	if len(records) != 2 {
		t.Fatal("missing complete records")
	}
	files := fstest.MapFS{"cards.ron": {Data: []byte(strings.Join(records, "\n"))}}
	registry := mage.NewRegistry()
	if err := LoadInto(registry, files, "*.ron"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Gabriel Angelfire", "Rapid Fire"} {
		first, err := registry.CreateCard(name)
		if err != nil {
			t.Fatal(err)
		}
		second, _ := registry.CreateCard(name)
		if first.Abilities()[0] == second.Abilities()[0] || mage.ValidateCardDefinition(name, first) != nil {
			t.Fatal("shared or invalid factory")
		}
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreatePermanentLayerEffectUntil(ThisPermanent,[AddAbility([Rampage(-1)])],UntilEndOfTurn)]))])`)}
	if err := LoadInto(registry, files, "*.ron"); err == nil || !registry.CardRegistered("Rapid Fire") || registry.CardRegistered("Bad") {
		t.Fatal("partial collection installed")
	}
}

func TestRONTemporaryRampageTimingAndShapes(t *testing.T) {
	g, a, b := ronGame()
	target := g.PutOnBattlefield(mage.NewCreature("Target", "", 2, 6), b.PlayerID())
	spell := repositoryCard(t, "LEG", "Rapid Fire")
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	a.ManaPool().Add(core.White, 1)
	a.ManaPool().Add(core.Colorless, 3)
	g.SetStep(core.DeclareBlockers)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{target.ID()}, 0); err == nil || a.ManaPool().TotalMana() != 4 || len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("late cast or payment rollback")
	}
	g.SetStep(core.PrecombatMain)
	land := g.PutOnBattlefield(mage.NewLand("Land"), b.PlayerID())
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{land.ID()}, 0); err == nil || a.ManaPool().TotalMana() != 4 {
		t.Fatal("illegal target paid cost")
	}
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreatePermanentLayerEffectUntil(ThisPermanent,[AddAbility([Rampage(0),Rampage(2)])],UntilEndOfTurn)]))`)
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.GetCombat().AddAttacker(p.ID(), b.PlayerID())
	for range 2 {
		g.PutOnBattlefieldBlocking(mage.NewCreature("Blocker", "", 1, 1), b.PlayerID(), p.ID())
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("rampage instances merged")
	}
	src := `Card(Name:"Choice",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([ChooseAnAbility([Rampage(0),Rampage(2)]),CreatePermanentLayerEffectUntil(ThisPermanent,[AddAbilityVariable(TheChosenAbility)],UntilEndOfTurn)]))])`
	report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal("different rampage amounts rejected", err)
	}
	for _, sig := range report.Signatures {
		if sig.Name == "0" && sig.Role == "integer_literal" {
			return
		}
	}
	for _, sig := range report.Signatures {
		if sig.Kind == "integer" && sig.Role == "integer_literal" {
			return
		}
	}
	t.Fatal("nested rampage amount missing semantic role")
}
