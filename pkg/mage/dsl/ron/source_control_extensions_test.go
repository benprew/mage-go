package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONSourceControlExtensionsOldMan(t *testing.T) {
	for _, change := range []string{"none", "power-before", "power-after", "source-power", "source-control", "untap", "target-phase", "source-phase", "return", "copy", "clone", "ability"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "ARN", "Old Man of the Sea")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			victimCard := mage.NewCreature("Victim", "", 2, 8)
			victimCard.SetOwner(b.PlayerID())
			victim := g.PutOnBattlefield(victimCard, b.PlayerID())
			tooLarge := g.PutOnBattlefield(mage.NewCreature("Too large", "", 3, 8), b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{tooLarge.ID()}); err == nil || source.Tapped || g.StackSize() != 0 {
				t.Fatal("illegal target consumed tap")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{victim.ID()}); err != nil {
				t.Fatal(err)
			}
			if change == "power-before" {
				victim.AddCounter(core.P1P1, 1)
			}
			if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			clone := g.Clone()
			g.ResolveStack()
			source = g.MutablePermanent(source.ID())
			victim = g.MutablePermanent(victim.ID())
			want := a.PlayerID()
			if change == "power-before" {
				want = b.PlayerID()
			}
			if g.FindPermanent(victim.ID()).ControllerID() != want {
				t.Fatal("resolution legality")
			}
			switch change {
			case "power-after":
				victim.AddCounter(core.P1P1, 1)
				want = b.PlayerID()
			case "source-power":
				source.AddCounter(core.M1M1, 1)
				want = b.PlayerID()
			case "source-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "untap":
				g.UntapPermanent(source)
				g.TapPermanent(source)
				want = b.PlayerID()
			case "target-phase":
				g.PhaseIn(g.PhaseOut(victim.ID()))
				want = b.PlayerID()
			case "source-phase":
				g.PhaseIn(g.PhaseOut(source.ID()))
				want = b.PlayerID()
			case "return":
				g.RemoveFromBattlefield(victim)
				victim = g.PutOnBattlefield(victimCard, b.PlayerID())
				want = b.PlayerID()
			case "ability":
				g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
			}
			g.ApplyContinuousEffects()
			if g.FindPermanent(victim.ID()).ControllerID() != want {
				t.Fatal("duration endpoint", change)
			}
			if change == "none" && !g.FindPermanent(source.ID()).HasAttr(core.AttrMayNotUntap) {
				t.Fatal("missing optional untap")
			}
			clone.ResolveStack()
			cloneWant := a.PlayerID()
			if change == "power-before" {
				cloneWant = b.PlayerID()
			}
			if clone.FindPermanent(victim.ID()).ControllerID() != cloneWant {
				t.Fatal("pending clone lost independent duration")
			}
		})
	}
}

func TestRONSourceControlExtensionsWretched(t *testing.T) {
	for _, change := range []string{"none", "remove-combat", "source-control", "source-phase", "source-return", "target-cycle", "target-return", "ability", "copy", "clone"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", "The Wretched")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			other := g.PutOnBattlefield(mage.NewCreature("Other attacker", "", 2, 8), a.PlayerID())
			other.RevokeBaseAttr(core.AttrSummonSick)
			firstCard := mage.NewCreature("First blocker", "", 1, 8)
			firstCard.SetOwner(b.PlayerID())
			first := g.PutOnBattlefield(firstCard, b.PlayerID())
			second := g.PutOnBattlefield(mage.NewCreature("Second blocker", "", 1, 8), b.PlayerID())
			unrelated := g.PutOnBattlefield(mage.NewCreature("Unrelated blocker", "", 1, 8), b.PlayerID())
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{source.ID(), other.ID()})
			g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: first.ID(), AttackerID: source.ID()}, {BlockerID: second.ID(), AttackerID: source.ID()}, {BlockerID: unrelated.ID(), AttackerID: other.ID()}})
			g.FireEvent(core.GameEvent{Type: core.EvtEndOfCombat, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("end combat trigger missing")
			}
			if change == "remove-combat" {
				g.RemoveFromCombat(first.ID())
			}
			if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			pending := g.Clone()
			g.ResolveStack()
			for _, p := range []*mage.Permanent{first, second} {
				want := a.PlayerID()
				if change == "remove-combat" && p.ID() == first.ID() {
					want = b.PlayerID()
				}
				if g.FindPermanent(p.ID()).ControllerID() != want {
					t.Fatal("captured combat membership", p.Card.Name())
				}
			}
			if unrelated.ControllerID() != b.PlayerID() {
				t.Fatal("unrelated blocker captured")
			}
			clone := g.Clone()
			wantFirst, wantSecond := a.PlayerID(), a.PlayerID()
			if change == "remove-combat" {
				wantFirst = b.PlayerID()
			}
			switch change {
			case "source-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.EndOfTurn})
				wantFirst = b.PlayerID()
				wantSecond = b.PlayerID()
			case "source-phase":
				g.PhaseIn(g.PhaseOut(source.ID()))
				wantFirst = b.PlayerID()
				wantSecond = b.PlayerID()
			case "source-return":
				g.RemoveFromBattlefield(source)
				g.PutOnBattlefield(card, a.PlayerID())
				wantFirst = b.PlayerID()
				wantSecond = b.PlayerID()
			case "target-cycle":
				g.PhaseIn(g.PhaseOut(first.ID()))
			case "target-return":
				g.RemoveFromBattlefield(first)
				first = g.PutOnBattlefield(firstCard, b.PlayerID())
				wantFirst = b.PlayerID()
			case "ability":
				g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
			}
			g.RemoveFromCombat(second.ID())
			g.ApplyContinuousEffects()
			if g.FindPermanent(first.ID()).ControllerID() != wantFirst || g.FindPermanent(second.ID()).ControllerID() != wantSecond {
				t.Fatal("plural duration endpoint")
			}
			pending.ResolveStack()
			if clone.FindPermanent(second.ID()).ControllerID() != a.PlayerID() || pending.FindPermanent(second.ID()).ControllerID() != a.PlayerID() {
				t.Fatal("clone shared membership or expiration")
			}
		})
	}
}

func TestRONSourceControlExtensionsInvalid(t *testing.T) {
	for _, replacement := range []string{`PowerOfPermanent(HostPermanent)`, `PowerOfPermanent(ThisPermanent())`, `PowerOfPermanent(Ref_TargetPermanent)`, `PowerOfPermanent()`, `Integer(-1,2)`} {
		source := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:3),Rules:[Activated(TapPermanent(ThisPermanent),Targeted([TargetPermanent(PowerIs(LessThanOrEqualTo(PowerOfPermanent(ThisPermanent))))],ActionList([GainControlOfPermanentUntil(Ref_TargetPermanent,Or([ForAsLongAsPermanentRemainsTapped(ThisPermanent),UntilPermanentNoLongerPassesFilter(Ref_TargetPermanent,PowerIs(LessThanOrEqualTo(PowerOfPermanent(ThisPermanent))))]))])))])`
		source = strings.ReplaceAll(source, `PowerOfPermanent(ThisPermanent)`, replacement)
		nodes, err := Parse(strings.NewReader(source), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("invalid power source accepted", replacement)
		}
	}
	for _, duration := range []string{`UntilPermanentChangesControl(Opponent,ThisPermanent)`, `UntilPermanentChangesControl(You,HostPermanent)`, `ForAsLongAsPermanentRemainsTapped(ThisPermanent)`, `UntilEndOfTurn`, `UntilPermanentChangesControl(You,ThisPermanent())`} {
		source := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([GainControlOfEachPermanentUntil(IsCardtype(Creature),` + duration + `)]))])`
		nodes, err := Parse(strings.NewReader(source), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("invalid plural duration accepted", duration)
		}
	}
}

func TestRONSourceControlExtensionsCasting(t *testing.T) {
	for _, tc := range []struct {
		set, name, mana  string
		power, toughness int
	}{{"ARN", "Old Man of the Sea", "{1}{U}{U}", 2, 3}, {"LEG", "The Wretched", "{3}{B}{B}", 2, 5}} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, _ := ronGame()
			f, err := Compile(staticBaseRecord(t, tc.set, tc.name))
			if err != nil {
				t.Fatal(err)
			}
			card := f()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			if card.Abilities()[0] == f().Abilities()[0] {
				t.Fatal("factory shares abilities")
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || g.StackSize() != 0 {
				t.Fatal("unpaid cast succeeded")
			}
			cost := core.ParseManaCost(tc.mana)
			a.ManaPool().Add(core.Blue, cost.Blue)
			a.ManaPool().Add(core.Black, cost.Black)
			a.ManaPool().Add(core.Colorless, cost.Generic)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			source := g.FindPermanent(card.ID())
			if source == nil || source.CurrentPower(g) != tc.power || source.CurrentToughness(g) != tc.toughness || !source.HasAttr(core.AttrSummonSick) || a.ManaPool().TotalMana() != 0 {
				t.Fatal("paid casting or reconstruction failed")
			}
		})
	}
}

func TestRONSourceControlExtensionsRetargetAndUntap(t *testing.T) {
	a := &ronModePlayer{BasePlayer: mage.NewBasePlayer("A")}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	source := g.PutOnBattlefield(repositoryCard(t, "ARN", "Old Man of the Sea"), a.PlayerID())
	source.RevokeBaseAttr(core.AttrSummonSick)
	first := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 8), b.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second", "", 2, 8), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{first.ID()}); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	a.targets = []uuid.UUID{second.ID()}
	copied := g.CopyStackObjectDirect(original, a.PlayerID(), true)
	if copied == nil || copied.Targets[0] != second.ID() || original.Targets[0] != first.ID() {
		t.Fatal("copy lost target isolation")
	}
	g.ResolveStack()
	g.MutablePermanent(first.ID()).AddCounter(core.P1P1, 1)
	g.ApplyContinuousEffects()
	if g.FindPermanent(first.ID()).ControllerID() != b.PlayerID() || g.FindPermanent(second.ID()).ControllerID() != a.PlayerID() {
		t.Fatal("power endpoint shared across targets")
	}
	g.UntapPermanent(g.FindPermanent(source.ID()))
	if g.FindPermanent(second.ID()).ControllerID() != b.PlayerID() {
		t.Fatal("untap did not end copied duration")
	}
}

func TestRONSourceControlExtensionsOptionalUntap(t *testing.T) {
	for _, change := range []string{"decline", "accept", "stun", "gone-target"} {
		t.Run(change, func(t *testing.T) {
			a := &ronStoragePlayer{ronCounterPayer: &ronCounterPayer{BasePlayer: mage.NewBasePlayer("A")}}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			source := g.PutOnBattlefield(repositoryCard(t, "ARN", "Old Man of the Sea"), a.PlayerID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 2, 8), b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{victim.ID()}); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			a.untap = change == "accept" || change == "stun"
			if change == "stun" {
				g.MutablePermanent(source.ID()).AddCounter(core.Stun, 1)
			}
			if change == "gone-target" {
				g.RemoveFromBattlefield(g.FindPermanent(victim.ID()))
			}
			g.RunStepWithPriority(core.Untap)
			if g.FindPermanent(source.ID()).Tapped != (change != "accept") {
				t.Fatal("optional untap permission or stun")
			}
			if p := g.FindPermanent(victim.ID()); p != nil {
				want := a.PlayerID()
				if change == "accept" {
					want = b.PlayerID()
				}
				if p.ControllerID() != want {
					t.Fatal("untap endpoint")
				}
			}
		})
	}
}

func TestRONSourceControlExtensionsInventory(t *testing.T) {
	source := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:3),Rules:[Activated(PayMana([]),ActionList([Unknown,GainControlOfEachPermanentUntil(IsBlockingAttacker(ThisPermanent),UntilPermanentChangesControl(You,ThisPermanent)),GainControlOfPermanentUntil(ThisPermanent,Or([ForAsLongAsPermanentRemainsTapped(ThisPermanent),UntilPermanentNoLongerPassesFilter(ThisPermanent,PowerIs(LessThanOrEqualTo(PowerOfPermanent(ThisPermanent))))]))]))])`
	report, err := InspectFS(fstest.MapFS{"nested.ron": {Data: []byte(source)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 0 || len(report.Issues) != 1 {
		t.Fatal("unsupported parent accepted")
	}
	roles := map[string]bool{}
	for _, sig := range report.Signatures {
		roles[sig.Role+"/"+sig.Name] = true
	}
	for _, key := range []string{"action/GainControlOfEachPermanentUntil", "permanent_collection/IsBlockingAttacker", "source_control_duration/UntilPermanentNoLongerPassesFilter", "permanent_filter/PowerIs", "power_comparison/LessThanOrEqualTo", "value/PowerOfPermanent", "permanent_reference/ThisPermanent"} {
		if !roles[key] {
			t.Fatal("missing inventory role", key)
		}
	}
}

func TestRONSourceControlExtensionsStrictDeclarations(t *testing.T) {
	endpoints := `Or([ForAsLongAsPermanentRemainsTapped(ThisPermanent),UntilPermanentNoLongerPassesFilter(Ref_TargetPermanent,PowerIs(LessThanOrEqualTo(PowerOfPermanent(ThisPermanent))))])`
	for _, duration := range []string{
		strings.Replace(endpoints, `Ref_TargetPermanent`, `ThisPermanent`, 1),
		strings.Replace(endpoints, `LessThanOrEqualTo`, `LessThan`, 1),
		strings.Replace(endpoints, `ForAsLongAsPermanentRemainsTapped(ThisPermanent)`, `ForAsLongAsPermanentRemainsTapped(ThisPermanent())`, 1),
		`Or([UntilPermanentNoLongerPassesFilter(Ref_TargetPermanent,PowerIs(LessThanOrEqualTo(PowerOfPermanent(ThisPermanent))))])`,
		`Or([UntilPermanentNoLongerPassesFilter(Ref_TargetPermanent,PowerIs(LessThanOrEqualTo(PowerOfPermanent(ThisPermanent)))),UntilPermanentNoLongerPassesFilter(Ref_TargetPermanent,PowerIs(LessThanOrEqualTo(PowerOfPermanent(ThisPermanent))))])`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),Targeted([TargetPermanent(AnyPermanent)],ActionList([GainControlOfPermanentUntil(Ref_TargetPermanent,` + duration + `)])))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("invalid endpoint accepted", duration)
		}
	}
	action := `GainControlOfEachPermanentUntil(IsCardtype(Creature),UntilPermanentChangesControl(You,ThisPermanent))`
	for _, src := range []string{
		`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([` + action + `]))])`,
		`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[AsPermanentEnters(ThisPermanent,[` + action + `])])`,
		`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(` + action + `,ActionList([]))])`,
	} {
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("invalid context accepted", src)
		}
	}
}
