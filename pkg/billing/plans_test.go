package billing

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------- PlanID ----------

func TestValidPlanID(t *testing.T) {
	tests := []struct {
		id   PlanID
		want bool
	}{
		{PlanFree, true},
		{PlanStarter, true},
		{PlanPro, true},
		{PlanEnterprise, true},
		{"unknown", false},
		{"", false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, ValidPlanID(tt.id), "ValidPlanID(%q)", tt.id)
	}
}

func TestAllPlanIDs(t *testing.T) {
	ids := AllPlanIDs()
	assert.Equal(t, []PlanID{PlanFree, PlanStarter, PlanPro, PlanEnterprise}, ids)
}

// ---------- DefaultPlans ----------

func TestDefaultPlans(t *testing.T) {
	plans := DefaultPlans()
	require.Len(t, plans, 4)

	// Free tier has expected limits.
	free := plans[PlanFree]
	require.NotNil(t, free)
	assert.Equal(t, "Free", free.Name)
	assert.Equal(t, int64(0), free.PriceMonthly)
	assert.Equal(t, 1, free.Limits.MaxAgents)
	assert.Equal(t, int64(500), free.Limits.MaxMessagesPerMonth)
	assert.False(t, free.Limits.CustomSkills)
	assert.Equal(t, "none", free.Limits.APIAccess)
	assert.True(t, free.Active)

	// Starter pricing.
	starter := plans[PlanStarter]
	assert.Equal(t, int64(900), starter.PriceMonthly)
	assert.Equal(t, int64(8640), starter.PriceYearly)
	assert.Equal(t, 3, starter.Limits.MaxAgents)

	// Pro has custom skills.
	pro := plans[PlanPro]
	assert.True(t, pro.Limits.CustomSkills)
	assert.Equal(t, "full", pro.Limits.APIAccess)
	assert.Equal(t, 10, pro.Limits.MaxAgents)

	// Enterprise is unlimited.
	ent := plans[PlanEnterprise]
	assert.Equal(t, 0, ent.Limits.MaxAgents)             // 0 = unlimited
	assert.Equal(t, int64(0), ent.Limits.MaxMessagesPerMonth) // unlimited
	assert.True(t, ent.Limits.CustomSkills)
}

func TestDefaultPlansAllActive(t *testing.T) {
	for id, p := range DefaultPlans() {
		assert.True(t, p.Active, "plan %s should be active", id)
	}
}

func TestDefaultPlansModelsSet(t *testing.T) {
	plans := DefaultPlans()
	// Free tier restricts models.
	assert.NotEmpty(t, plans[PlanFree].Limits.AllowedModels)
	// Enterprise allows all.
	assert.Nil(t, plans[PlanEnterprise].Limits.AllowedModels)
}

// ---------- Catalogue ----------

func TestCatalogueDefault(t *testing.T) {
	c := NewCatalogue(nil)
	assert.NotNil(t, c.Get(PlanFree))
	assert.NotNil(t, c.Get(PlanStarter))
	assert.NotNil(t, c.Get(PlanPro))
	assert.NotNil(t, c.Get(PlanEnterprise))
}

func TestCatalogueGetNotFound(t *testing.T) {
	c := NewCatalogue(nil)
	assert.Nil(t, c.Get("nonexistent"))
}

func TestCatalogueGetReturnsCopy(t *testing.T) {
	c := NewCatalogue(nil)
	p1 := c.Get(PlanFree)
	p1.Name = "Modified"
	p2 := c.Get(PlanFree)
	assert.Equal(t, "Free", p2.Name, "catalogue should return copies")
}

func TestCatalogueList(t *testing.T) {
	c := NewCatalogue(nil)
	plans := c.List()
	require.Len(t, plans, 4)
	// Should be sorted by SortOrder.
	assert.Equal(t, PlanFree, plans[0].ID)
	assert.Equal(t, PlanStarter, plans[1].ID)
	assert.Equal(t, PlanPro, plans[2].ID)
	assert.Equal(t, PlanEnterprise, plans[3].ID)
}

func TestCatalogueListActive(t *testing.T) {
	plans := DefaultPlans()
	plans[PlanStarter].Active = false
	c := NewCatalogue(plans)
	active := c.ListActive()
	assert.Len(t, active, 3)
	for _, p := range active {
		assert.NotEqual(t, PlanStarter, p.ID)
	}
}

func TestCatalogueReplace(t *testing.T) {
	c := NewCatalogue(nil)
	require.Len(t, c.List(), 4)

	// Replace with just one plan.
	c.Replace(map[PlanID]*Plan{
		PlanFree: {ID: PlanFree, Name: "Only Free", Active: true},
	})
	assert.Len(t, c.List(), 1)
	assert.Equal(t, "Only Free", c.Get(PlanFree).Name)
}

func TestCatalogueMerge(t *testing.T) {
	c := NewCatalogue(nil)
	c.Merge(map[PlanID]*Plan{
		PlanFree: {ID: PlanFree, Name: "Updated Free", SortOrder: 0, Active: true},
	})
	assert.Equal(t, "Updated Free", c.Get(PlanFree).Name)
	// Other plans untouched.
	assert.Equal(t, "Starter", c.Get(PlanStarter).Name)
}

// ---------- Limit helpers ----------

func TestIsUnlimited(t *testing.T) {
	assert.True(t, IsUnlimited(0))
	assert.False(t, IsUnlimited(1))
	assert.False(t, IsUnlimited(-1))
}

func TestIsUnlimitedInt(t *testing.T) {
	assert.True(t, IsUnlimitedInt(0))
	assert.False(t, IsUnlimitedInt(5))
}

func TestCheckLimit(t *testing.T) {
	// Under limit.
	assert.NoError(t, CheckLimit("agents", 2, 10))
	// At limit.
	assert.Error(t, CheckLimit("agents", 10, 10))
	// Over limit.
	assert.Error(t, CheckLimit("agents", 15, 10))
	// Unlimited.
	assert.NoError(t, CheckLimit("agents", 999999, 0))
}

func TestCheckLimitInt(t *testing.T) {
	assert.NoError(t, CheckLimitInt("integrations", 3, 5))
	assert.Error(t, CheckLimitInt("integrations", 5, 5))
	assert.NoError(t, CheckLimitInt("integrations", 100, 0))
}

func TestCheckLimitErrorMessage(t *testing.T) {
	err := CheckLimit("messages", 500, 500)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "messages limit reached")
	assert.Contains(t, err.Error(), "500/500")
}

// ---------- JSON ----------

func TestPlanFromJSON(t *testing.T) {
	p := &Plan{
		ID:           PlanPro,
		Name:         "Pro",
		PriceMonthly: 2900,
		Limits:       PlanLimits{MaxAgents: 10, CustomSkills: true},
		Active:       true,
	}
	data, err := json.Marshal(p)
	require.NoError(t, err)

	got, err := PlanFromJSON(data)
	require.NoError(t, err)
	assert.Equal(t, PlanPro, got.ID)
	assert.Equal(t, "Pro", got.Name)
	assert.Equal(t, 10, got.Limits.MaxAgents)
	assert.True(t, got.Limits.CustomSkills)
}

func TestPlanFromJSONInvalidID(t *testing.T) {
	data := []byte(`{"id":"bogus","name":"Bogus"}`)
	_, err := PlanFromJSON(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid plan ID")
}

func TestPlanFromJSONInvalidJSON(t *testing.T) {
	_, err := PlanFromJSON([]byte(`{bad json`))
	assert.Error(t, err)
}

func TestPlansFromJSON(t *testing.T) {
	input := map[string]*Plan{
		"free":    {ID: PlanFree, Name: "Free"},
		"starter": {ID: PlanStarter, Name: "Starter"},
	}
	data, err := json.Marshal(input)
	require.NoError(t, err)

	plans, err := PlansFromJSON(data)
	require.NoError(t, err)
	assert.Len(t, plans, 2)
	assert.Equal(t, "Free", plans[PlanFree].Name)
}

func TestPlansFromJSONInvalidKey(t *testing.T) {
	data := []byte(`{"invalid":{"id":"invalid","name":"bad"}}`)
	_, err := PlansFromJSON(data)
	assert.Error(t, err)
}

func TestPlansFromJSONBadJSON(t *testing.T) {
	_, err := PlansFromJSON([]byte(`not json`))
	assert.Error(t, err)
}

func TestPlanJSONRoundTrip(t *testing.T) {
	original := DefaultPlans()
	data, err := json.Marshal(original)
	require.NoError(t, err)

	got, err := PlansFromJSON(data)
	require.NoError(t, err)
	assert.Len(t, got, 4)
	assert.Equal(t, original[PlanFree].Limits.MaxAgents, got[PlanFree].Limits.MaxAgents)
}

// ---------- Sort stability ----------

func TestSortPlansStable(t *testing.T) {
	plans := []*Plan{
		{ID: PlanEnterprise, Name: "Enterprise", SortOrder: 3},
		{ID: PlanFree, Name: "Free", SortOrder: 0},
		{ID: PlanPro, Name: "Pro", SortOrder: 2},
		{ID: PlanStarter, Name: "Starter", SortOrder: 1},
	}
	sortPlans(plans)
	assert.Equal(t, PlanFree, plans[0].ID)
	assert.Equal(t, PlanStarter, plans[1].ID)
	assert.Equal(t, PlanPro, plans[2].ID)
	assert.Equal(t, PlanEnterprise, plans[3].ID)
}

func TestSortPlansSameOrder(t *testing.T) {
	// Same sort order → sort by name.
	plans := []*Plan{
		{ID: "b", Name: "Bravo", SortOrder: 0},
		{ID: "a", Name: "Alpha", SortOrder: 0},
	}
	sortPlans(plans)
	assert.Equal(t, "Alpha", plans[0].Name)
	assert.Equal(t, "Bravo", plans[1].Name)
}
