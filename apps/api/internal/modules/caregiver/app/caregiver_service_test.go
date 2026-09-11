package app

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeCaregiverStore is a deterministic in-memory CaregiverStore test double.
type fakeCaregiverStore struct {
	byID      map[uuid.UUID]*Caregiver
	listErr   error
	deleteErr error
	createErr error
}

func newFakeCaregiverStore() *fakeCaregiverStore {
	return &fakeCaregiverStore{byID: map[uuid.UUID]*Caregiver{}}
}

func (f *fakeCaregiverStore) List(ctx context.Context, q, status string, pending, excludePending bool, page, pageSize int) ([]Caregiver, int64, error) {
	if f.listErr != nil {
		return nil, 0, f.listErr
	}
	var out []Caregiver
	for _, c := range f.byID {
		isPending := strings.TrimSpace(c.Name) == "" || strings.TrimSpace(c.Type) == ""
		if pending && !isPending {
			continue
		}
		if excludePending && isPending {
			continue
		}
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
	if f.createErr != nil {
		return f.createErr
	}
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
	if f.deleteErr != nil {
		return f.deleteErr
	}
	delete(f.byID, id)
	return nil
}

func TestCaregiverService_Create_RequiresName(t *testing.T) {
	svc := NewCaregiverService(newFakeCaregiverStore(), nil, nil)

	_, err := svc.Create(context.Background(), CreateCaregiverInput{Contact: "0912-000-000"})

	assert.ErrorIs(t, err, ErrCaregiverNameRequired)
}

func TestCaregiverService_Create_Succeeds(t *testing.T) {
	svc := NewCaregiverService(newFakeCaregiverStore(), nil, nil)

	c, err := svc.Create(context.Background(), CreateCaregiverInput{SiteName: "竹南日照據點", Name: "陳小華", Type: CaregiverTypeCaseManager, Contact: "0912-000-000"})

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, c.ID)
	assert.Equal(t, "陳小華", c.Name)
	assert.Equal(t, "竹南日照據點", c.SiteName)
}

func TestCaregiverService_Create_RequiresValidType(t *testing.T) {
	svc := NewCaregiverService(newFakeCaregiverStore(), nil, nil)

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
		{name: "接受 inactive", input: "inactive", want: "inactive"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewCaregiverService(newFakeCaregiverStore(), nil, nil)

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
	svc := NewCaregiverService(store, nil, nil)

	t.Run("接受合法狀態", func(t *testing.T) {
		c, err := svc.Update(context.Background(), existing.ID, UpdateCaregiverInput{Status: strPtr("inactive")})
		require.NoError(t, err)
		assert.Equal(t, "inactive", c.Status)
	})

	t.Run("拒絕非法狀態", func(t *testing.T) {
		_, err := svc.Update(context.Background(), existing.ID, UpdateCaregiverInput{Status: strPtr("已離職")})
		assert.ErrorIs(t, err, ErrCaregiverStatusInvalid)
	})
}

func strPtr(v string) *string { return &v }

func TestCaregiverService_Update_SetsSiteName(t *testing.T) {
	store := newFakeCaregiverStore()
	existing := Caregiver{ID: uuid.New(), Name: "王大明"}
	require.NoError(t, store.Create(context.Background(), &existing))
	svc := NewCaregiverService(store, nil, nil)

	updated, err := svc.Update(context.Background(), existing.ID, UpdateCaregiverInput{SiteName: strPtr("新據點")})

	require.NoError(t, err)
	assert.Equal(t, "新據點", updated.SiteName)
}

// auditRepo 未設定（nil）時，Delete 仍必須先確認資源存在；否則刪除不存在的 ID
// 會直接呼叫 store.Delete 並被當成成功的 204，使用者拿不到 404。
func TestCaregiverService_Delete_ReturnsNotFound_WhenMissing_EvenWithoutAuditRepo(t *testing.T) {
	store := newFakeCaregiverStore()
	svc := NewCaregiverService(store, nil, nil)

	err := svc.Delete(context.Background(), uuid.New())

	assert.ErrorIs(t, err, ErrCaregiverNotFound)
}

func TestCaregiverService_Delete_Succeeds_WithoutAuditRepo(t *testing.T) {
	store := newFakeCaregiverStore()
	existing := Caregiver{ID: uuid.New(), Name: "王大明", Type: CaregiverTypeCaseManager}
	require.NoError(t, store.Create(context.Background(), &existing))
	svc := NewCaregiverService(store, nil, nil)

	err := svc.Delete(context.Background(), existing.ID)

	require.NoError(t, err)
	_, ok := store.byID[existing.ID]
	assert.False(t, ok)
}

// 仍被個案關聯時，repo 層會回傳 ErrCaregiverInUse；service 層需原樣往上拋，
// 由 transport 層映射為 409，而不是吞掉或包裝成其他錯誤。
func TestCaregiverService_Delete_PropagatesInUseError(t *testing.T) {
	store := newFakeCaregiverStore()
	existing := Caregiver{ID: uuid.New(), Name: "王大明", Type: CaregiverTypeCaseManager}
	require.NoError(t, store.Create(context.Background(), &existing))
	store.deleteErr = ErrCaregiverInUse
	svc := NewCaregiverService(store, nil, nil)

	err := svc.Delete(context.Background(), existing.ID)

	assert.ErrorIs(t, err, ErrCaregiverInUse)
}
