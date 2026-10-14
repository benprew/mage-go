package ron

import (
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONEventTargetChooserAbyss(t *testing.T) {
	for _, index := range []int{0, 1} {
		for _, change := range []string{"none", "control", "return", "phase", "cycle", "shroud", "indestructible", "source-control", "source-return", "source-phase", "no-target", "copy", "source-phased-before", "invalid-choice", "hexproof"} {
			t.Run(fmt.Sprintf("%d/%s", index, change), func(t *testing.T) {
				g, a, b := opponentTargetGame()
				c := repositoryCard(t, "LEG", "The Abyss")
				c.SetOwner(a.PlayerID())
				var source *mage.Permanent
				if change == "none" {
					a.AddToHand(c)
					if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err == nil {
						t.Fatal("unpaid spell accepted")
					}
					a.ManaPool().Add(core.Black, 4)
					if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
						t.Fatal(err)
					}
					g.ResolveStack()
					source = g.FindPermanent(c.ID())
					if source == nil || a.ManaPool().TotalMana() != 0 {
						t.Fatal("paid enchantment did not enter")
					}
				} else {
					source = g.PutOnBattlefield(c, a.PlayerID())
				}
				players := []*ronTargetChooser{a, b}
				chooser := players[index]
				other := players[1-index]
				victimCard := mage.NewCreature("Victim", "", 2, 3)
				victim := g.PutOnBattlefield(victimCard, chooser.PlayerID())
				spare := g.PutOnBattlefield(mage.NewCreature("Spare", "", 2, 3), chooser.PlayerID())
				artifact := g.PutOnBattlefield(mage.NewCreature("Artifact", "", 2, 3, mage.WithCardType(core.TypeArtifact)), chooser.PlayerID())
				foreign := g.PutOnBattlefield(mage.NewCreature("Foreign", "", 2, 3), other.PlayerID())
				hidden := g.PutOnBattlefield(mage.NewCreature("Hidden", "", 2, 3), chooser.PlayerID())
				g.PhaseOut(hidden.ID())
				protected := g.PutOnBattlefield(mage.NewCreature("Protected", "", 2, 3, mage.WithAbility(mage.ProtectionFromColor(core.Black))), chooser.PlayerID())
				if change == "no-target" {
					g.PhaseOut(victim.ID())
					g.PhaseOut(spare.ID())
				}
				chooser.requested = []uuid.UUID{victim.ID()}
				if change == "invalid-choice" {
					chooser.requested = []uuid.UUID{artifact.ID()}
				}
				if change == "source-phased-before" {
					g.PhaseOut(source.ID())
				}
				if change == "hexproof" {
					g.MutablePermanent(victim.ID()).GrantBaseAttr(core.Hexproof)
				}
				g.SetActivePlayerIndex(index)
				g.SetStep(core.Upkeep)
				g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: chooser.PlayerID()})
				if g.StackSize() != 0 {
					t.Fatal("choice occurred before placement")
				}
				if change == "source-control" {
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
				if change == "source-return" {
					g.RemoveFromBattlefield(source)
					g.PutOnBattlefield(c, b.PlayerID())
				}
				if change == "source-phase" {
					g.PhaseOut(source.ID())
				}
				g.PutTriggersOnStack()
				if change == "no-target" || change == "source-phased-before" {
					if g.StackSize() != 0 || len(chooser.candidates) != 0 {
						t.Fatal("targetless trigger stacked")
					}
					return
				}
				if change == "invalid-choice" {
					if g.StackSize() != 0 || len(chooser.candidates) != 1 || g.FindPermanent(artifact.ID()) == nil {
						t.Fatal("illegal chooser response bypassed target rules")
					}
					return
				}
				if change == "hexproof" && index == 1 {
					if g.StackSize() != 0 || len(chooser.candidates) != 1 || slices.Contains(chooser.candidates[0], victim.ID()) {
						t.Fatal("chooser bypassed controller-relative hexproof")
					}
					return
				}
				if g.StackSize() != 1 || len(chooser.candidates) != 1 || len(other.candidates) != 0 || !slices.Equal(g.StackPeek().Targets, []uuid.UUID{victim.ID()}) {
					t.Fatal("wrong chooser or chosen target")
				}
				possible := chooser.candidates[0]
				if !slices.Contains(possible, spare.ID()) || slices.Contains(possible, artifact.ID()) || slices.Contains(possible, foreign.ID()) || slices.Contains(possible, hidden.ID()) || slices.Contains(possible, protected.ID()) {
					t.Fatal("wrong target legality")
				}
				switch change {
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: victim.ID(), ControllerID: other.PlayerID(), Duration: core.Indefinite})
				case "return":
					g.RemoveFromBattlefield(victim)
					g.PutOnBattlefield(victimCard, chooser.PlayerID())
				case "phase":
					g.PhaseOut(victim.ID())
				case "cycle":
					g.PhaseIn(g.PhaseOut(victim.ID()))
				case "shroud":
					g.MutablePermanent(victim.ID()).GrantBaseAttr(core.Shroud)
				case "indestructible":
					g.MutablePermanent(victim.ID()).GrantBaseAttr(core.Indestructible)
				case "copy":
					other.requested = []uuid.UUID{spare.ID()}
					copied := g.CopyStackObjectDirect(g.StackPeek(), other.PlayerID(), true)
					if copied == nil || !slices.Equal(copied.Targets, []uuid.UUID{spare.ID()}) || len(other.candidates) != 1 {
						t.Fatal("copy target or saved event changed")
					}
				}
				g.AddRegenerationShield(victim.ID())
				clone := g.Clone()
				clone.ResolveStack()
				if g.FindPermanentIncludingPhased(victim.ID()) == nil {
					t.Fatal("clone changed original")
				}
				g.ResolveStack()
				survives := change == "control" || change == "return" || change == "phase" || change == "shroud" || change == "indestructible"
				for _, game := range []*mage.Game{g, clone} {
					if change == "copy" && game.FindPermanent(spare.ID()) != nil {
						t.Fatal("copy did not destroy its own target")
					}
					if (game.FindPermanentIncludingPhased(victim.ID()) != nil) != survives {
						t.Fatal("target identity, legality or regeneration")
					}
				}
			})
		}
	}
}

func TestRONEventTargetChooserInvalid(t *testing.T) {
	for _, target := range []string{
		`TargetPermanentOfPlayersChoice(AnyPermanent)`,
		`TargetPermanentOfPlayersChoice(AnyPermanent,Opponent)`,
		`TargetPermanentOfPlayersChoice(AnyPermanent,Trigger_ThatPlayer())`,
		`TargetPermanentOfPlayersChoice(AnyPermanent,Trigger_ThatPlayer,You)`,
		`TargetPermanentOfPlayersChoice(AnyPermanent,Ref_TargetPlayer)`,
	} {
		src := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([` + target + `],ActionList([TapPermanent(Ref_TargetPermanent)])))])`
		nodes, err := Parse(strings.NewReader(src), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("invalid chooser accepted", target)
		}
		src = `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[TriggerA(AtTheBeginningOfAPlayersUpkeep(AnyPlayer),Targeted([` + target + `],ActionList([TapPermanent(Ref_TargetPermanent)])))])`
		nodes, err = Parse(strings.NewReader(src), "bad-step.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("invalid bound chooser accepted", target)
		}
	}
}

func TestRONEventTargetChooserBindingsAndStrictLoad(t *testing.T) {
	data, err := fs.ReadFile(repositoryFixtures.files, "LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "The Abyss"`) {
			record = line
			break
		}
	}
	if record == "" {
		t.Fatal("missing record")
	}
	registry := mage.NewRegistry()
	if err := LoadInto(registry, fstest.MapFS{"abyss.ron": {Data: []byte(record)}}, "*.ron"); err != nil {
		t.Fatal(err)
	}
	g, a, b := opponentTargetGame()
	card, err := registry.CreateCard("The Abyss")
	if err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(card, a.PlayerID())
	own := g.PutOnBattlefield(mage.NewCreature("Own", "", 2, 3), a.PlayerID())
	other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 3), b.PlayerID())
	a.requested = []uuid.UUID{own.ID()}
	b.requested = []uuid.UUID{other.ID()}
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 || len(a.candidates) != 1 || len(b.candidates) != 1 {
		t.Fatal("pending events did not retain separate choosers")
	}
	clone := g.Clone()
	g.ResolveStack()
	if clone.FindPermanent(own.ID()) == nil || clone.FindPermanent(other.ID()) == nil {
		t.Fatal("resolution changed clone")
	}
	clone.ResolveStack()
	for _, game := range []*mage.Game{g, clone} {
		if game.FindPermanent(own.ID()) != nil || game.FindPermanent(other.ID()) != nil {
			t.Fatal("pending event bindings changed")
		}
	}
	for _, replacement := range []string{"Opponent", "Trigger_ThatPlayer()", "Ref_TargetPlayer"} {
		bad := strings.ReplaceAll(record, ", Trigger_ThatPlayer)", ", "+replacement+")")
		if bad == record {
			t.Fatal("fixture did not replace chooser")
		}
		if err := LoadInto(registry, fstest.MapFS{"bad.ron": {Data: []byte(bad)}}, "*.ron"); err == nil {
			t.Fatal("invalid collection accepted")
		}
		if !registry.CardRegistered("The Abyss") || len(registry.RegisteredCardNames()) != 1 {
			t.Fatal("failed collection changed registry")
		}
	}
}

func TestRONEventTargetChooserNumberedInstances(t *testing.T) {
	g, a, b := opponentTargetGame()
	factory := permanentScopeFactory(t, "Artifact", `TriggerA(AtTheBeginningOfAPlayersEndStep(AnyPlayer),Targeted([TargetPermanentOfPlayersChoice(And([IsCardtype(Creature),ControlledByAPlayer(SinglePlayer(Trigger_ThatPlayer))]),Trigger_ThatPlayer),TargetPermanentOfPlayersChoice(IsCardtype(Creature),Trigger_ThatPlayer)],ActionList([TapPermanent(Ref_TargetPermanent1),TapPermanent(Ref_TargetPermanent2)])))`)
	g.PutOnBattlefield(factory(), a.PlayerID())
	g.PutOnBattlefield(factory(), a.PlayerID())
	own := g.PutOnBattlefield(mage.NewCreature("Own", "", 2, 3), a.PlayerID())
	other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 3), b.PlayerID())
	b.queue = [][]uuid.UUID{{other.ID()}, {own.ID()}, {other.ID()}, {own.ID()}}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 || len(a.candidates) != 0 || len(b.candidates) != 4 {
		t.Fatal("numbered declarations or factory chooser bindings changed")
	}
	clone := g.Clone()
	g.ResolveStack()
	if clone.FindPermanent(own.ID()).Tapped || clone.FindPermanent(other.ID()).Tapped {
		t.Fatal("factory resolution changed clone")
	}
	clone.ResolveStack()
	for _, game := range []*mage.Game{g, clone} {
		if !game.FindPermanent(own.ID()).Tapped || !game.FindPermanent(other.ID()).Tapped {
			t.Fatal("numbered target group changed")
		}
	}
}
