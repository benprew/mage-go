package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONNoncolorTargetQueryGameplay(t *testing.T) {
	for _, name := range []string{"Stone Rain", "Ice Storm", "Sinkhole", "Shatter"} {
		for _, state := range []string{"matching", "other", "everything", "exempt", "phase", "return", "copy"} {
			t.Run(name+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				card := mage.NewLand("Subject")
				if name == "Shatter" {
					card = mage.NewArtifact("Subject", "")
				}
				card.SetOwner(b.PlayerID())
				victim := g.PutOnBattlefield(card, b.PlayerID())
				spell := repositoryCard(t, "LEA", name)
				obj := castRONSpell(t, g, a, spell, victim.ID())
				kind := core.TypeSorcery
				if name == "Shatter" {
					kind = core.TypeInstant
				}
				protection := mage.ProtectionFromCardType(kind)
				want := false
				switch state {
				case "other":
					protection = mage.ProtectionFromCardType(core.TypeCreature)
					want = true
				case "everything":
					protection = mage.ProtectionFromAll()
				case "exempt":
					protection.ExemptCardID = spell.ID()
					want = true
				case "phase":
					g.PhaseOut(victim.ID())
				case "return":
					g.RemoveFromBattlefield(victim)
					victim = g.PutOnBattlefield(card, b.PlayerID())
				case "copy":
					obj = g.CopyStackObjectDirect(obj, a.PlayerID(), false)
				}
				g.FindPermanentIncludingPhased(victim.ID()).RuntimeAbilities = append(g.FindPermanentIncludingPhased(victim.ID()).RuntimeAbilities, protection)
				instructions, ok := mage.InspectDirectDestruction(obj.Effects[0])
				if !ok {
					t.Fatal("destruction lowering lost")
				}
				plan, ok := mage.InspectPermanentTarget(instructions[0].Subject)
				if !ok {
					t.Fatal("target lowering lost")
				}
				for _, game := range []*mage.Game{g, g.Clone()} {
					got, known := plan.Select(game, game.StackPeek())
					if !known || (len(got) == 1) != want {
						t.Fatalf("got %v known %v want %v", got, known, want)
					}
					game.ResolveStack()
					if (game.FindPermanentIncludingPhased(victim.ID()) == nil) != want {
						t.Fatal("native resolution differs")
					}
				}
			})
		}
	}
}

func TestRONNoncolorTargetQuerySourceInformation(t *testing.T) {
	for _, state := range []string{"live", "departed", "returned", "phased", "copy", "face-down"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			victim := g.PutOnBattlefield(mage.NewLand("Subject"), b.PlayerID())
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([ManaCostGeneric(0)]),Targeted([TargetPermanent(AnyPermanent)],ActionList([DestroyPermanent(Ref_TargetPermanent)])))`)
			card := factory()
			source := g.PutOnBattlefield(card, a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
				t.Fatal(err)
			}
			obj := g.StackPeek()
			switch state {
			case "departed", "returned":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				if state == "returned" {
					g.PutOnBattlefield(card, b.PlayerID()).FaceDown = true
				}
			case "phased":
				g.PhaseOut(source.ID())
			case "copy":
				obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
			case "face-down":
				g.FindPermanent(source.ID()).FaceDown = true
			}
			p := g.FindPermanent(victim.ID())
			p.RuntimeAbilities = append(p.RuntimeAbilities, mage.ProtectionFromCardType(core.TypeArtifact))
			instructions, ok := mage.InspectDirectDestruction(obj.Effects[0])
			if !ok {
				t.Fatal("missing destruction")
			}
			plan, ok := mage.InspectPermanentTarget(instructions[0].Subject)
			if !ok {
				t.Fatal("missing target")
			}
			for _, game := range []*mage.Game{g, g.Clone()} {
				got, known := plan.Select(game, game.StackPeek())
				want := state == "face-down"
				if !known || (len(got) == 1) != want {
					t.Fatalf("got %v known %v want %v", got, known, want)
				}
				game.ResolveStack()
				if (game.FindPermanent(victim.ID()) == nil) != want {
					t.Fatal("source information differs")
				}
			}
		})
	}
}

func TestRONNoncolorTargetQueryDeclarations(t *testing.T) {
	for _, syntax := range []string{`AnyPermanent(AnyPermanent)`, `IsCardtype(Land())`, `IsNonCardtype("Creature")`, `And([])`, `Or([IsCardtype(Land),IsCardtype(Unknown)])`, `IsCardtype(Land,Creature)`} {
		node := nameScopeNode(t, `Card(Name:"Invalid target",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(`+syntax+`)],ActionList([DestroyPermanent(Ref_TargetPermanent)])))])`)
		if _, err := Compile(node); err == nil {
			t.Fatalf("accepted %s", syntax)
		}
	}
}
