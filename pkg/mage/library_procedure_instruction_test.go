package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestLibraryProcedureInstructionDeclaration(t *testing.T) {
	target := DeclarePlayerTarget("library")
	for _, instruction := range []LibraryProcedureInstruction{
		{Chooser: You, Library: target, Count: 3, Reorder: true},
		{Chooser: You, Library: target, Count: 5},
		{Chooser: You, Library: target, OptionalShuffle: true},
	} {
		got, ok := InspectLibraryProcedureInstruction(instruction.Effect())
		if !ok || got != instruction {
			t.Fatal("lost library operands")
		}
	}
	for _, instruction := range []LibraryProcedureInstruction{
		{}, {Chooser: You, Library: target, Count: -1},
		{Chooser: You, Library: target, Count: 1, OptionalShuffle: true},
		{Chooser: You, Library: target, Reorder: true, OptionalShuffle: true},
		{Chooser: (*PlayerTargetBinding)(nil), Library: target},
	} {
		if instruction.Effect() != nil {
			t.Fatal("accepted invalid declaration")
		}
	}
	for _, effect := range []Effect{nil, (*libraryInspectionEffect)(nil), (*mayShuffleLibraryEffect)(nil), LookAtLibraryTop(nil, You, 3, false), MayShuffleLibrary(You, nil), LookAtLibraryTopNumber(You, You, ValueAtResolution(XValue()), true), LookAtLibraryTop(You, You, -1, false), Sequence(LookAtLibraryTop(You, You, 1, false), destructionInspectionTrap{})} {
		if _, ok := InspectLibraryProcedureInstruction(effect); ok {
			t.Fatal("accepted incomplete procedure")
		}
	}
}

func TestLibraryProcedureInstructionOrder(t *testing.T) {
	for _, shuffle := range []bool{false, true} {
		for _, state := range []string{"alone", "prefix", "suffix", "unknown suffix", "all illegal", "unknown all illegal", "clone", "copy", "foreign", "opaque chooser", "illegal library", "illegal chooser", "bound chooser", "foreign chooser", "opaque library", "duplicate"} {
			t.Run(state+map[bool]string{false: "/look", true: "/shuffle"}[shuffle], func(t *testing.T) {
				g, p, obj, target := targetQueryFixture(t)
				library := DeclarePlayerTarget("library")
				obj.TargetSpecs = append(obj.TargetSpecs, library.targetSpec())
				obj.Targets = append(obj.Targets, obj.Controller)
				var chooser PlayerRef = You
				if state == "opaque chooser" {
					chooser = lifeInspectionPlayerTrap{}
				}
				if state == "bound chooser" || state == "illegal chooser" {
					chooser = library
				}
				if state == "foreign chooser" {
					chooser = DeclarePlayerTarget("foreign chooser")
				}
				if state == "foreign" {
					library = DeclarePlayerTarget("foreign")
				}
				var libraryRef PlayerRef = library
				if state == "opaque library" {
					libraryRef = lifeInspectionPlayerTrap{}
				}
				procedure := (LibraryProcedureInstruction{Chooser: chooser, Library: libraryRef, OptionalShuffle: shuffle}).Effect()
				obj.Effects = []Effect{procedure}
				want, known := false, true
				switch state {
				case "prefix":
					obj.Effects = []Effect{Sequence(procedure, Destroy(target))}
					known = false
				case "suffix", "clone", "copy":
					obj.Effects = []Effect{Sequence(Destroy(target), procedure)}
					want = true
				case "unknown suffix":
					obj.Effects = []Effect{Sequence(Destroy(target), procedure, destructionInspectionTrap{})}
					known = false
				case "all illegal", "unknown all illegal":
					p.GrantBaseAttr(core.Shroud)
					obj.Targets[1] = p.ID()
					obj.Effects = []Effect{Sequence(procedure, Destroy(target))}
					if state == "unknown all illegal" {
						obj.Effects = append(obj.Effects, destructionInspectionTrap{})
						known = false
					}
				case "illegal library":
					obj.Targets[1] = p.ID()
					obj.Effects = []Effect{Sequence(procedure, Destroy(target))}
					want = true
				case "illegal chooser":
					obj.Targets[1] = p.ID()
					obj.Effects = []Effect{Sequence(procedure, Destroy(target))}
					want = true
				case "foreign", "opaque chooser", "foreign chooser", "opaque library":
					known = false
				case "duplicate":
					obj.TargetSpecs = append(obj.TargetSpecs, library.targetSpec())
					obj.Targets = append(obj.Targets, obj.Controller)
					known = false
				}
				if state == "clone" {
					g = g.Clone()
					obj = g.StackPeek()
				}
				if state == "copy" {
					obj = g.CopyStackObjectDirect(obj, g.GetOpponent(obj.Controller).PlayerID(), false)
				}
				for range 2 {
					got, k := g.InspectSpellDestruction(obj, []PermanentVisit{{p.ID(), p.incarnationID}})
					if got != want || k != known {
						t.Fatalf("certainty=%v/%v want=%v/%v", got, k, want, known)
					}
				}
				if g.FindPermanent(p.ID()) == nil {
					t.Fatal("inspection destroyed permanent")
				}
			})
		}
	}
}

func TestLibraryProcedureInstructionPurity(t *testing.T) {
	for _, shuffle := range []bool{false, true} {
		g, _, obj, _ := targetQueryFixture(t)
		for i, player := range g.players {
			g.players[i] = &libraryProcedureTrapPlayer{librarySearchInspectionTrapPlayer{player.(*BasePlayer)}}
		}
		obj.Effects = []Effect{(LibraryProcedureInstruction{Chooser: You, Library: ActivePlayer, Count: map[bool]int{false: 3, true: 0}[shuffle], Reorder: !shuffle, OptionalShuffle: shuffle}).Effect()}
		for range 2 {
			if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
				t.Fatal("native procedure is unknown")
			}
		}
	}
}

type libraryProcedureTrapPlayer struct {
	librarySearchInspectionTrapPlayer
}

func (*libraryProcedureTrapPlayer) ChooseMayAbility(string) bool {
	panic("inspection made a shuffle choice")
}
func (*libraryProcedureTrapPlayer) ChooseLibraryTopOrder([]Card, GameReader) []uuid.UUID {
	panic("inspection ordered cards")
}
func (*libraryProcedureTrapPlayer) ObserveLibraryLook([]Card, GameReader) {
	panic("inspection revealed cards")
}
