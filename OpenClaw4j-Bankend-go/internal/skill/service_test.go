package skill

import (
	"context"
	"testing"
	"time"

	"github.com/seaskyland/openclaw4j-backend-go/internal/dao"
)

func TestServiceCreatesEditsAndPublishesVersions(t *testing.T) {
	store := newMemoryStore()
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	service := NewService(store, func() time.Time { return now })
	code, err := service.Create(context.Background(), "ws_1", "acct_1", Input{Name: "weather"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	created, err := service.Get(context.Background(), "ws_1", code, "", false)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if created.Status != statusDraft || created.Version != "1" {
		t.Fatalf("created = %#v", created)
	}
	if err := service.Publish(context.Background(), "ws_1", "acct_1", code); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if err := service.Update(context.Background(), "ws_1", "acct_1", Input{SkillCode: code, Name: "weather-v2"}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	updated, err := service.Get(context.Background(), "ws_1", code, "", false)
	if err != nil {
		t.Fatalf("Get() after update error = %v", err)
	}
	if updated.Status != statusPublishedEditing || updated.Version != "2" || updated.Name != "weather-v2" {
		t.Fatalf("updated = %#v", updated)
	}
	if store.versions[code+":2"].Status != statusDraft {
		t.Fatalf("new version status = %d, want draft", store.versions[code+":2"].Status)
	}
}

type memoryStore struct {
	skills   map[string]dao.Skill
	versions map[string]dao.SkillVersion
}

func newMemoryStore() *memoryStore {
	return &memoryStore{skills: map[string]dao.Skill{}, versions: map[string]dao.SkillVersion{}}
}
func skillKey(code, workspace string) string { return workspace + ":" + code }
func versionKey(code, version string) string { return code + ":" + version }
func (s *memoryStore) Create(_ context.Context, value dao.Skill) error {
	s.skills[skillKey(value.SkillCode, value.WorkspaceID)] = value
	return nil
}
func (s *memoryStore) CreateVersion(_ context.Context, value dao.SkillVersion) error {
	s.versions[versionKey(value.SkillCode, value.Version)] = value
	return nil
}
func (s *memoryStore) Find(_ context.Context, code, workspace string) (*dao.Skill, error) {
	value, ok := s.skills[skillKey(code, workspace)]
	if !ok || value.Status == statusDeleted {
		return nil, dao.ErrNotFound
	}
	return &value, nil
}
func (s *memoryStore) FindVersion(_ context.Context, code, _ string, version string) (*dao.SkillVersion, error) {
	value, ok := s.versions[versionKey(code, version)]
	if !ok || value.Status == statusDeleted {
		return nil, dao.ErrNotFound
	}
	return &value, nil
}
func (s *memoryStore) FindLatestVersion(_ context.Context, code, _ string) (*dao.SkillVersion, error) {
	var latest *dao.SkillVersion
	for _, value := range s.versions {
		if value.SkillCode == code && value.Status != statusDeleted && (latest == nil || value.Version > latest.Version) {
			copy := value
			latest = &copy
		}
	}
	if latest == nil {
		return nil, dao.ErrNotFound
	}
	return latest, nil
}
func (s *memoryStore) List(_ context.Context, workspace, _ string, status int16) ([]dao.Skill, error) {
	result := []dao.Skill{}
	for _, value := range s.skills {
		if value.WorkspaceID == workspace && value.Status != statusDeleted && (status < 0 || value.Status == status) {
			result = append(result, value)
		}
	}
	return result, nil
}
func (s *memoryStore) Update(_ context.Context, value dao.Skill) error {
	s.skills[skillKey(value.SkillCode, value.WorkspaceID)] = value
	return nil
}
func (s *memoryStore) UpdateVersion(_ context.Context, value dao.SkillVersion) error {
	s.versions[versionKey(value.SkillCode, value.Version)] = value
	return nil
}
func (s *memoryStore) Delete(_ context.Context, code, workspace, _ string, _ time.Time) error {
	skill := s.skills[skillKey(code, workspace)]
	skill.Status = statusDeleted
	s.skills[skillKey(code, workspace)] = skill
	for key, version := range s.versions {
		if version.SkillCode == code {
			version.Status = statusDeleted
			s.versions[key] = version
		}
	}
	return nil
}
