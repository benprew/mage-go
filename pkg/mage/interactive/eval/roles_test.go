package eval

import (
	"testing"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// ── ClassifyPermanent ────────────────────────────────────────────────────────

func TestClassifyPermanent_Land(t *testing.T) {
	land := mage.NewLand("Forest")
	land.SetOwner(uuid.New())
	perm := mage.NewPermanent(land, land.Owner())
	if role := ClassifyPermanent(perm); role != RoleMana {
		t.Errorf("ClassifyPermanent(land) = %v, want RoleMana", role)
	}
}

func TestClassifyPermanent_ManaCreature(t *testing.T) {
	owner := uuid.New()
	perm := makePerm("Llanowar Elves", "{G}", 1, 1, owner, mage.WithManaAbility(core.Green))
	if role := ClassifyPermanent(perm); role != RoleMana {
		t.Errorf("ClassifyPermanent(mana elf) = %v, want RoleMana", role)
	}
}

func TestClassifyPermanent_ManaArtifact(t *testing.T) {
	art := mage.NewArtifact("Sol Ring", "{1}", mage.WithManaAbility(core.Colorless))
	art.SetOwner(uuid.New())
	perm := mage.NewPermanent(art, art.Owner())
	if role := ClassifyPermanent(perm); role != RoleMana {
		t.Errorf("ClassifyPermanent(mana artifact) = %v, want RoleMana", role)
	}
}

func TestClassifyPermanent_BigCreature_Threat(t *testing.T) {
	owner := uuid.New()
	perm := makePerm("Hill Giant", "{3}{R}", 3, 3, owner)
	if role := ClassifyPermanent(perm); role != RoleThreat {
		t.Errorf("ClassifyPermanent(3/3) = %v, want RoleThreat", role)
	}
}

func TestClassifyPermanent_EvasiveCreature_Threat(t *testing.T) {
	owner := uuid.New()
	perm := makePerm("Air Elemental", "{3}{U}{U}", 4, 4, owner, mage.WithKeyword(core.Flying))
	if role := ClassifyPermanent(perm); role != RoleThreat {
		t.Errorf("ClassifyPermanent(4/4 flyer) = %v, want RoleThreat", role)
	}
}

func TestClassifyPermanent_SmallEvasive_Threat(t *testing.T) {
	owner := uuid.New()
	perm := makePerm("Flying Men", "{U}", 1, 1, owner, mage.WithKeyword(core.Flying))
	if role := ClassifyPermanent(perm); role != RoleThreat {
		t.Errorf("ClassifyPermanent(1/1 flyer) = %v, want RoleThreat", role)
	}
}

func TestClassifyPermanent_VanillaSmall_Threat(t *testing.T) {
	owner := uuid.New()
	perm := makePerm("Grizzly Bears", "{1}{G}", 2, 2, owner)
	if role := ClassifyPermanent(perm); role != RoleThreat {
		t.Errorf("ClassifyPermanent(2/2 vanilla) = %v, want RoleThreat", role)
	}
}

func TestClassifyPermanent_Wall_Defense(t *testing.T) {
	owner := uuid.New()
	perm := makePerm("Wall of Stone", "{1}{R}{R}", 0, 8, owner, mage.WithKeyword(core.Defender))
	if role := ClassifyPermanent(perm); role != RoleDefense {
		t.Errorf("ClassifyPermanent(0/8 defender) = %v, want RoleDefense", role)
	}
}

func TestClassifyPermanent_HighToughnessLowPower_Defense(t *testing.T) {
	owner := uuid.New()
	// 1/5: toughness(5) > power(1)*2 → defense
	perm := makePerm("Horned Turtle", "{2}{U}", 1, 5, owner)
	if role := ClassifyPermanent(perm); role != RoleDefense {
		t.Errorf("ClassifyPermanent(1/5) = %v, want RoleDefense", role)
	}
}

func TestClassifyPermanent_BalancedPT_NotDefense(t *testing.T) {
	owner := uuid.New()
	// 2/3: toughness(3) is NOT > power(2)*2=4 → Threat
	perm := makePerm("Squire", "{1}{W}", 2, 3, owner)
	if role := ClassifyPermanent(perm); role != RoleThreat {
		t.Errorf("ClassifyPermanent(2/3) = %v, want RoleThreat", role)
	}
}

func TestClassifyPermanent_Pinger_Utility(t *testing.T) {
	owner := uuid.New()
	perm := makePerm("Prodigal Sorcerer", "{2}{U}", 1, 1, owner,
		mage.WithActivatedAbility(mage.DealDamage(mage.Fixed(1)), mage.Tap(),
			mage.WithTarget(mage.TargetDamageAnyTarget())),
	)
	if role := ClassifyPermanent(perm); role != RoleUtility {
		t.Errorf("ClassifyPermanent(pinger) = %v, want RoleUtility", role)
	}
}

func TestClassifyPermanent_NonCreatureArtifact_Utility(t *testing.T) {
	art := mage.NewArtifact("Disrupting Scepter", "{3}")
	art.SetOwner(uuid.New())
	perm := mage.NewPermanent(art, art.Owner())
	if role := ClassifyPermanent(perm); role != RoleUtility {
		t.Errorf("ClassifyPermanent(non-mana artifact) = %v, want RoleUtility", role)
	}
}

func TestClassifyPermanent_Enchantment_Utility(t *testing.T) {
	ench := mage.NewEnchantment("Glorious Anthem", "{1}{W}{W}")
	ench.SetOwner(uuid.New())
	perm := mage.NewPermanent(ench, ench.Owner())
	if role := ClassifyPermanent(perm); role != RoleUtility {
		t.Errorf("ClassifyPermanent(enchantment) = %v, want RoleUtility", role)
	}
}

// ── hasEvasion ───────────────────────────────────────────────────────────────

func TestHasEvasion_Flying(t *testing.T) {
	perm := makePerm("Bird", "{U}", 1, 1, uuid.New(), mage.WithKeyword(core.Flying))
	if !hasEvasion(perm) {
		t.Error("hasEvasion(flying) should be true")
	}
}

func TestHasEvasion_None(t *testing.T) {
	perm := makePerm("Bear", "{1}{G}", 2, 2, uuid.New())
	if hasEvasion(perm) {
		t.Error("hasEvasion(vanilla) should be false")
	}
}

func TestHasEvasion_Trample(t *testing.T) {
	perm := makePerm("Trampler", "{2}{G}", 3, 3, uuid.New(), mage.WithKeyword(core.Trample))
	if !hasEvasion(perm) {
		t.Error("hasEvasion(trample) should be true")
	}
}

// ── boardDiversity ───────────────────────────────────────────────────────────

func TestBoardDiversity_Empty(t *testing.T) {
	roles := map[PermanentRole]int{}
	if got := boardDiversity(roles); got != 0 {
		t.Errorf("boardDiversity(empty) = %d, want 0", got)
	}
}

func TestBoardDiversity_SingleRole(t *testing.T) {
	roles := map[PermanentRole]int{RoleThreat: 3}
	if got := boardDiversity(roles); got != 0 {
		t.Errorf("boardDiversity(1 role) = %d, want 0", got)
	}
}

func TestBoardDiversity_TwoRoles(t *testing.T) {
	roles := map[PermanentRole]int{RoleThreat: 2, RoleMana: 3}
	if got := boardDiversity(roles); got != 2 {
		t.Errorf("boardDiversity(2 roles) = %d, want 2", got)
	}
}

func TestBoardDiversity_ThreeRoles(t *testing.T) {
	roles := map[PermanentRole]int{RoleThreat: 1, RoleMana: 2, RoleUtility: 1}
	if got := boardDiversity(roles); got != 4 {
		t.Errorf("boardDiversity(3 roles) = %d, want 4", got)
	}
}

// ── PermanentRole.String ────────────────────────────────────────────────────

func TestPermanentRole_String(t *testing.T) {
	tests := []struct {
		role PermanentRole
		want string
	}{
		{RoleThreat, "Threat"},
		{RoleUtility, "Utility"},
		{RoleEngine, "Engine"},
		{RoleMana, "Mana"},
		{RoleDefense, "Defense"},
		{PermanentRole(99), "Unknown"},
	}
	for _, tt := range tests {
		if got := tt.role.String(); got != tt.want {
			t.Errorf("PermanentRole(%d).String() = %q, want %q", tt.role, got, tt.want)
		}
	}
}

// ── countRoles ──────────────────────────────────────────────────────────────

func TestCountRoles(t *testing.T) {
	owner := uuid.New()
	perms := []*mage.Permanent{
		makePerm("Bear", "{1}{G}", 2, 2, owner),
		makePerm("Bear2", "{1}{G}", 2, 2, owner),
		makePerm("Wall", "{1}{W}", 0, 5, owner, mage.WithKeyword(core.Defender)),
	}
	roles := countRoles(perms, func(p *mage.Permanent) bool { return p.Controller == owner })
	if roles[RoleThreat] != 2 {
		t.Errorf("countRoles threats = %d, want 2", roles[RoleThreat])
	}
	if roles[RoleDefense] != 1 {
		t.Errorf("countRoles defense = %d, want 1", roles[RoleDefense])
	}
}
