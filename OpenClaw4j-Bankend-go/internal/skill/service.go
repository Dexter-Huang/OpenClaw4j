package skill

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/seaskyland/openclaw4j-backend-go/internal/dao"
)

const (
	statusDeleted          int16 = 0
	statusDraft            int16 = 1
	statusPublished        int16 = 2
	statusPublishedEditing int16 = 3
)

var (
	ErrSkillCodeRequired = errors.New("skill_code is required")
	ErrNameRequired      = errors.New("name is required")
	ErrNotFound          = errors.New("skill not found")
)

type Input struct {
	SkillCode        string  `json:"skill_code"`
	Name             string  `json:"name"`
	Description      *string `json:"description"`
	Source           *string `json:"source"`
	Status           *int16  `json:"status"`
	Version          string  `json:"version"`
	Tags             *string `json:"tags"`
	MainFilePath     *string `json:"main_file_path"`
	Manifest         *string `json:"manifest"`
	ContentHash      *string `json:"content_hash"`
	StorageType      *string `json:"storage_type"`
	StorageBucket    *string `json:"storage_bucket"`
	StoragePrefix    *string `json:"storage_prefix"`
	PackageObjectKey *string `json:"package_object_key"`
	FileCount        *int32  `json:"file_count"`
	TotalSizeBytes   *int64  `json:"total_size_bytes"`
	NeedFiles        bool    `json:"need_files"`
}

type Detail struct {
	SkillCode        string    `json:"skill_code"`
	Name             string    `json:"name"`
	Description      *string   `json:"description,omitempty"`
	Source           string    `json:"source"`
	Status           int16     `json:"status"`
	CurrentVersion   string    `json:"current_version"`
	Version          string    `json:"version"`
	Tags             *string   `json:"tags,omitempty"`
	MainFilePath     string    `json:"main_file_path"`
	Manifest         *string   `json:"manifest,omitempty"`
	ContentHash      *string   `json:"content_hash,omitempty"`
	StorageType      string    `json:"storage_type"`
	StorageBucket    *string   `json:"storage_bucket,omitempty"`
	StoragePrefix    string    `json:"storage_prefix"`
	PackageObjectKey *string   `json:"package_object_key,omitempty"`
	FileCount        *int32    `json:"file_count,omitempty"`
	TotalSizeBytes   *int64    `json:"total_size_bytes,omitempty"`
	NeedFiles        bool      `json:"need_files"`
	GmtModified      time.Time `json:"gmt_modified"`
}

type Page struct {
	Current int64    `json:"current"`
	Size    int64    `json:"size"`
	Total   int64    `json:"total"`
	Records []Detail `json:"records"`
}

type Service struct {
	dao        dao.SkillDAO
	clock      func() time.Time
	storageDir string
}

func NewService(data dao.SkillDAO, clock func() time.Time) *Service {
	return NewServiceWithStorage(data, clock, "data/files")
}

// NewServiceWithStorage 让 Skill 运行时和上传接口共享同一个受控文件根目录。
func NewServiceWithStorage(data dao.SkillDAO, clock func() time.Time, storageDir string) *Service {
	if clock == nil {
		clock = time.Now
	}
	if strings.TrimSpace(storageDir) == "" {
		storageDir = "data/files"
	}
	return &Service{dao: data, clock: clock, storageDir: storageDir}
}

// ReadFile 只允许读取当前工作区已发布 Skill 包中的相对路径文件，防止模型工具
// 借由路径穿越访问包外的配置或密钥文件。
func (s *Service) ReadFile(ctx context.Context, workspaceID, skillCode, path string) (string, error) {
	if strings.TrimSpace(skillCode) == "" {
		return "", ErrSkillCodeRequired
	}
	version, err := s.findVersion(ctx, skillCode, workspaceID, "")
	if errors.Is(err, dao.ErrNotFound) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if version.Status != statusPublished {
		return "", ErrNotFound
	}
	relative, err := cleanPackagePath(path)
	if err != nil {
		return "", err
	}
	root, err := filepath.Abs(s.storageDir)
	if err != nil {
		return "", err
	}
	packageRoot := filepath.Join(root, filepath.FromSlash(version.StoragePrefix))
	target := filepath.Join(packageRoot, filepath.FromSlash(relative))
	cleanRoot, cleanTarget := filepath.Clean(packageRoot), filepath.Clean(target)
	if !strings.HasPrefix(cleanTarget, cleanRoot+string(os.PathSeparator)) {
		return "", errors.New("skill file path is invalid")
	}
	file, err := os.Open(cleanTarget)
	if err != nil {
		return "", err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 1<<20))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func cleanPackagePath(value string) (string, error) {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" {
		value = "SKILL.md"
	}
	if strings.HasPrefix(value, "/") || strings.Contains(value, ":") {
		return "", errors.New("skill file path is invalid")
	}
	clean := path.Clean(value)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", errors.New("skill file path is invalid")
	}
	return clean, nil
}

func (s *Service) Create(ctx context.Context, workspaceID, accountID string, input Input) (string, error) {
	if strings.TrimSpace(input.Name) == "" {
		return "", ErrNameRequired
	}
	code, err := newSkillCode()
	if err != nil {
		return "", err
	}
	now := s.clock()
	status := defaultStatus(input.Status, statusDraft)
	value := dao.Skill{SkillCode: code, WorkspaceID: workspaceID, AccountID: stringPointer(accountID), Name: strings.TrimSpace(input.Name), Description: trimmedPointer(input.Description), Source: defaultString(input.Source, "CUSTOMER"), Status: status, Tags: trimmedPointer(input.Tags), GmtCreate: now, GmtModified: now, Creator: stringPointer(accountID), Modifier: stringPointer(accountID)}
	if err := s.dao.Create(ctx, value); err != nil {
		return "", err
	}
	version := skillVersionFromInput(code, workspaceID, "1", accountID, now, input, status)
	if err := s.dao.CreateVersion(ctx, version); err != nil {
		return "", err
	}
	return code, nil
}

func (s *Service) Update(ctx context.Context, workspaceID, accountID string, input Input) error {
	if strings.TrimSpace(input.SkillCode) == "" {
		return ErrSkillCodeRequired
	}
	if strings.TrimSpace(input.Name) == "" {
		return ErrNameRequired
	}
	entity, err := s.dao.Find(ctx, input.SkillCode, workspaceID)
	if errors.Is(err, dao.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	latest, err := s.dao.FindLatestVersion(ctx, input.SkillCode, workspaceID)
	if errors.Is(err, dao.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	now := s.clock()
	entity.Name = strings.TrimSpace(input.Name)
	entity.Description = trimmedPointer(input.Description)
	entity.Source = defaultString(input.Source, entity.Source)
	entity.Tags = trimmedPointer(input.Tags)
	if entity.Status == statusPublished {
		entity.Status = statusPublishedEditing
	}
	entity.Modifier = stringPointer(accountID)
	entity.GmtModified = now
	if err := s.dao.Update(ctx, *entity); err != nil {
		return err
	}
	newVersion := nextVersion(latest.Version)
	copy := skillVersionFromInput(input.SkillCode, workspaceID, newVersion, accountID, now, input, statusDraft)
	copyFallbacks(&copy, latest)
	return s.dao.CreateVersion(ctx, copy)
}

func (s *Service) Publish(ctx context.Context, workspaceID, accountID, skillCode string) error {
	if strings.TrimSpace(skillCode) == "" {
		return ErrSkillCodeRequired
	}
	entity, err := s.dao.Find(ctx, skillCode, workspaceID)
	if errors.Is(err, dao.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	version, err := s.dao.FindLatestVersion(ctx, skillCode, workspaceID)
	if errors.Is(err, dao.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	now := s.clock()
	version.Status, version.Modifier, version.GmtModified = statusPublished, stringPointer(accountID), now
	if err := s.dao.UpdateVersion(ctx, *version); err != nil {
		return err
	}
	entity.Status, entity.Modifier, entity.GmtModified = statusPublished, stringPointer(accountID), now
	return s.dao.Update(ctx, *entity)
}

func (s *Service) Delete(ctx context.Context, workspaceID, accountID, skillCode string) error {
	if strings.TrimSpace(skillCode) == "" {
		return ErrSkillCodeRequired
	}
	if _, err := s.dao.Find(ctx, skillCode, workspaceID); errors.Is(err, dao.ErrNotFound) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	return s.dao.Delete(ctx, skillCode, workspaceID, accountID, s.clock())
}

func (s *Service) Get(ctx context.Context, workspaceID, skillCode, requestedVersion string, needFiles bool) (*Detail, error) {
	if strings.TrimSpace(skillCode) == "" {
		return nil, ErrSkillCodeRequired
	}
	entity, err := s.dao.Find(ctx, skillCode, workspaceID)
	if errors.Is(err, dao.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	version, err := s.findVersion(ctx, skillCode, workspaceID, requestedVersion)
	if errors.Is(err, dao.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return mapDetail(*entity, *version, needFiles), nil
}

func (s *Service) List(ctx context.Context, workspaceID, name string, status int16, current, size int64) (*Page, error) {
	if current < 1 {
		current = 1
	}
	if size < 1 {
		size = 10
	}
	if size > 100 {
		size = 100
	}
	values, err := s.dao.List(ctx, workspaceID, strings.TrimSpace(name), status)
	if err != nil {
		return nil, err
	}
	all := make([]Detail, 0, len(values))
	for _, value := range values {
		version, findErr := s.dao.FindLatestVersion(ctx, value.SkillCode, workspaceID)
		if errors.Is(findErr, dao.ErrNotFound) {
			continue
		}
		if findErr != nil {
			return nil, findErr
		}
		all = append(all, *mapDetail(value, *version, false))
	}
	start := (current - 1) * size
	if start >= int64(len(all)) {
		return &Page{Current: current, Size: size, Total: int64(len(all)), Records: []Detail{}}, nil
	}
	end := start + size
	if end > int64(len(all)) {
		end = int64(len(all))
	}
	return &Page{Current: current, Size: size, Total: int64(len(all)), Records: all[start:end]}, nil
}

func (s *Service) ListByCodes(ctx context.Context, workspaceID string, codes []string, needFiles bool) ([]Detail, error) {
	all, err := s.dao.List(ctx, workspaceID, "", -1)
	if err != nil {
		return nil, err
	}
	selected := make(map[string]struct{}, len(codes))
	for _, code := range codes {
		selected[strings.TrimSpace(code)] = struct{}{}
	}
	result := make([]Detail, 0, len(codes))
	for _, value := range all {
		if _, ok := selected[value.SkillCode]; !ok {
			continue
		}
		version, findErr := s.dao.FindLatestVersion(ctx, value.SkillCode, workspaceID)
		if errors.Is(findErr, dao.ErrNotFound) {
			continue
		}
		if findErr != nil {
			return nil, findErr
		}
		result = append(result, *mapDetail(value, *version, needFiles))
	}
	return result, nil
}

func (s *Service) findVersion(ctx context.Context, skillCode, workspaceID, requestedVersion string) (*dao.SkillVersion, error) {
	if strings.TrimSpace(requestedVersion) != "" {
		return s.dao.FindVersion(ctx, skillCode, workspaceID, requestedVersion)
	}
	return s.dao.FindLatestVersion(ctx, skillCode, workspaceID)
}

func skillVersionFromInput(skillCode, workspaceID, version, accountID string, now time.Time, input Input, status int16) dao.SkillVersion {
	prefix := defaultString(input.StoragePrefix, fmt.Sprintf("skills/%s/%s/%s/", workspaceID, skillCode, version))
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return dao.SkillVersion{SkillCode: skillCode, WorkspaceID: workspaceID, Version: version, Description: trimmedPointer(input.Description), MainFilePath: defaultString(input.MainFilePath, "SKILL.md"), Manifest: trimmedPointer(input.Manifest), ContentHash: trimmedPointer(input.ContentHash), StorageType: defaultString(input.StorageType, "file"), StorageBucket: trimmedPointer(input.StorageBucket), StoragePrefix: prefix, PackageObjectKey: trimmedPointer(input.PackageObjectKey), FileCount: input.FileCount, TotalSizeBytes: input.TotalSizeBytes, Status: status, GmtCreate: now, GmtModified: now, Creator: stringPointer(accountID), Modifier: stringPointer(accountID)}
}

func copyFallbacks(target *dao.SkillVersion, source *dao.SkillVersion) {
	if target.Description == nil {
		target.Description = source.Description
	}
	if target.Manifest == nil {
		target.Manifest = source.Manifest
	}
	if target.ContentHash == nil {
		target.ContentHash = source.ContentHash
	}
	if target.StorageBucket == nil {
		target.StorageBucket = source.StorageBucket
	}
	if target.PackageObjectKey == nil {
		target.PackageObjectKey = source.PackageObjectKey
	}
	if target.FileCount == nil {
		target.FileCount = source.FileCount
	}
	if target.TotalSizeBytes == nil {
		target.TotalSizeBytes = source.TotalSizeBytes
	}
}

func mapDetail(skill dao.Skill, version dao.SkillVersion, needFiles bool) *Detail {
	return &Detail{SkillCode: skill.SkillCode, Name: skill.Name, Description: skill.Description, Source: skill.Source, Status: skill.Status, CurrentVersion: version.Version, Version: version.Version, Tags: skill.Tags, MainFilePath: version.MainFilePath, Manifest: version.Manifest, ContentHash: version.ContentHash, StorageType: version.StorageType, StorageBucket: version.StorageBucket, StoragePrefix: version.StoragePrefix, PackageObjectKey: version.PackageObjectKey, FileCount: version.FileCount, TotalSizeBytes: version.TotalSizeBytes, NeedFiles: needFiles, GmtModified: skill.GmtModified}
}

func defaultString(value *string, fallback string) string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return fallback
	}
	return strings.TrimSpace(*value)
}
func defaultStatus(value *int16, fallback int16) int16 {
	if value == nil {
		return fallback
	}
	return *value
}
func stringPointer(value string) *string { return &value }
func trimmedPointer(value *string) *string {
	if value == nil {
		return nil
	}
	result := strings.TrimSpace(*value)
	if result == "" {
		return nil
	}
	return &result
}
func nextVersion(value string) string {
	var number int
	if _, err := fmt.Sscanf(value, "%d", &number); err != nil || number < 1 {
		return "1"
	}
	return fmt.Sprintf("%d", number+1)
}
func newSkillCode() (string, error) {
	var data [12]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", err
	}
	return "skill_" + hex.EncodeToString(data[:]), nil
}

var _ = statusDeleted
