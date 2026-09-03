package app

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeCaregiverStore is a deterministic in-memory CaregiverStore test double.
type fakeCaregiverStore struct {
	byID map[uuid.UUID]*Caregiver
}

func newFakeCaregiverStore() *fakeCaregiverStore {
	return &fakeCaregiverStore{byID: map[uuid.UUID]*Caregiver{}}
}

func (f *fakeCaregiverStore) List(ctx context.Context, q, status string, unresolvedLink, incomplete, excludePending bool, page, pageSize int) ([]Caregiver, int64, error) {
	var out []Caregiver
	for _, c := range f.byID {
		out = append(out, *c)
	}
	return out, int64(len(out)), nil
}

func (f *fakeCaregiverStore) GetByID(ctx context.Context, id uuid.UUID) (*Caregiver, error) {
	c, ok := f.byID[id]
	if !ok {
		return nil, ErrCaregiverNotFound
	}
	copyC := *c
	return &copyC, nil
}

func (f *fakeCaregiverStore) Create(ctx context.Context, c *Caregiver) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	copyC := *c
	f.byID[c.ID] = &copyC
	return nil
}

func (f *fakeCaregiverStore) Update(ctx context.Context, c *Caregiver) error {
	if _, ok := f.byID[c.ID]; !ok {
		return ErrCaregiverNotFound
	}
	copyC := *c
	f.byID[c.ID] = &copyC
	return nil
}

func (f *fakeCaregiverStore) Delete(ctx context.Context, id uuid.UUID) error {
	delete(f.byID, id)
	return nil
}

func TestCaregiverService_Create_RequiresName(t *testing.T) {
	svc := NewCaregiverService(newFakeCaregiverStore(), nil, nil, nil)

	_, err := svc.Create(context.Background(), CreateCaregiverInput{Contact: "0912-000-000"})

	assert.ErrorIs(t, err, ErrCaregiverNameRequired)
}

func TestCaregiverService_Create_Succeeds(t *testing.T) {
	svc := NewCaregiverService(newFakeCaregiverStore(), nil, nil, nil)
	siteID := uuid.New()

	c, err := svc.Create(context.Background(), CreateCaregiverInput{SiteID: &siteID, Name: "陳小華", Type: CaregiverTypeCaseManager, Contact: "0912-000-000"})

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, c.ID)
	assert.Equal(t, "陳小華", c.Name)
	assert.Equal(t, &siteID, c.SiteID)
}

func TestCaregiverService_Create_RequiresValidType(t *testing.T) {
	svc := NewCaregiverService(newFakeCaregiverStore(), nil, nil, nil)

	_, err := svc.Create(context.Background(), CreateCaregiverInput{Name: "陳小華", Type: "居服員"})

	assert.ErrorIs(t, err, ErrCaregiverTypeInvalid)
}

func TestCaregiverService_Create_NormalizesStatus(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "空字串預設 active", input: "", want: "active"},
		{name: "非法值預設 active", input: "已離職", want: "active"},
		{name: "接受 inactive", input: "inactive", want: "inactive"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewCaregiverService(newFakeCaregiverStore(), nil, nil, nil)

			c, err := svc.Create(context.Background(), CreateCaregiverInput{Name: "陳小華", Type: CaregiverTypeCaseManager, Status: tt.input})

			require.NoError(t, err)
			assert.Equal(t, tt.want, c.Status)
		})
	}
}

func TestCaregiverService_Update_NormalizesStatus(t *testing.T) {
	store := newFakeCaregiverStore()
	existing := Caregiver{ID: uuid.New(), Name: "王大明", Status: "active"}
	require.NoError(t, store.Create(context.Background(), &existing))
	svc := NewCaregiverService(store, nil, nil, nil)

	t.Run("接受合法狀態", func(t *testing.T) {
		c, err := svc.Update(context.Background(), existing.ID, UpdateCaregiverInput{Status: strPtr("inactive")})
		require.NoError(t, err)
		assert.Equal(t, "inactive", c.Status)
	})

	t.Run("非法狀態預設 active", func(t *testing.T) {
		c, err := svc.Update(context.Background(), existing.ID, UpdateCaregiverInput{Status: strPtr("已離職")})
		require.NoError(t, err)
		assert.Equal(t, "active", c.Status)
	})
}

func strPtr(v string) *string { return &v }

func TestCaregiverService_LinkSite_ClearsRawName(t *testing.T) {
	store := newFakeCaregiverStore()
	svc := NewCaregiverService(store, nil, nil, nil)

	existing := Caregiver{ID: uuid.New(), Name: "王大明", SiteNameRaw: "竹南日照單位"}
	require.NoError(t, store.Create(context.Background(), &existing))

	siteID := uuid.New()
	updated, err := svc.LinkSite(context.Background(), existing.ID, siteID)

	require.NoError(t, err)
	assert.Equal(t, &siteID, updated.SiteID)
	assert.Empty(t, updated.SiteNameRaw, "手動關聯單位後應清空原始單位名稱")
}
