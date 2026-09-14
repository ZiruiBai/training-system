package service

import (
	"context"

	"github.com/example/training-platform/internal/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// ResourceService handles courses, materials, SOPs and users.
type ResourceService struct {
	db *gorm.DB
}

// NewResourceService creates a ResourceService.
func NewResourceService(db *gorm.DB) *ResourceService { return &ResourceService{db: db} }

// --- Courses ---

func (s *ResourceService) ListCourses(ctx context.Context, opt ListOptions, category, status string) ([]model.Course, int64, error) {
	var items []model.Course
	query := s.db.WithContext(ctx).Model(&model.Course{})
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if opt.Search != "" {
		query = query.Where("title LIKE ?", "%"+opt.Search+"%")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size := normalizePage(opt.Page, opt.PageSize)
	if err := query.Preload("Position").Order("created_at desc").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *ResourceService) GetCourse(ctx context.Context, id uint) (*model.Course, error) {
	var c model.Course
	if err := s.db.WithContext(ctx).Preload("Position").First(&c, id).Error; err != nil {
		return nil, mapGormError(err)
	}
	return &c, nil
}

func (s *ResourceService) CreateCourse(ctx context.Context, c *model.Course) (*model.Course, error) {
	if c.Title == "" || c.Category == "" {
		return nil, ErrValidation
	}
	if c.Status == "" {
		c.Status = model.CourseDraft
	}
	if err := s.db.WithContext(ctx).Create(c).Error; err != nil {
		return nil, mapCreateError(err)
	}
	return c, nil
}

func (s *ResourceService) UpdateCourse(ctx context.Context, id uint, c *model.Course) (*model.Course, error) {
	var existing model.Course
	if err := s.db.WithContext(ctx).First(&existing, id).Error; err != nil {
		return nil, mapGormError(err)
	}
	if c.Title == "" || c.Category == "" {
		return nil, ErrValidation
	}
	c.ID = id
	c.CreatedAt = existing.CreatedAt
	if err := s.db.WithContext(ctx).Save(c).Error; err != nil {
		return nil, mapCreateError(err)
	}
	return c, nil
}

func (s *ResourceService) DeleteCourse(ctx context.Context, id uint) error {
	return mapGormError(s.db.WithContext(ctx).Delete(&model.Course{}, id).Error)
}

// --- Materials ---

func (s *ResourceService) ListMaterials(ctx context.Context, opt ListOptions, category string) ([]model.TrainingMaterial, int64, error) {
	var items []model.TrainingMaterial
	query := s.db.WithContext(ctx).Model(&model.TrainingMaterial{})
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if opt.Search != "" {
		query = query.Where("title LIKE ?", "%"+opt.Search+"%")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size := normalizePage(opt.Page, opt.PageSize)
	if err := query.Preload("Position").Preload("Course").Order("created_at desc").
		Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *ResourceService) CreateMaterial(ctx context.Context, m *model.TrainingMaterial) (*model.TrainingMaterial, error) {
	if m.Title == "" {
		return nil, ErrValidation
	}
	if m.Status == "" {
		m.Status = "active"
	}
	if err := s.db.WithContext(ctx).Create(m).Error; err != nil {
		return nil, mapCreateError(err)
	}
	return m, nil
}

func (s *ResourceService) DeleteMaterial(ctx context.Context, id uint) error {
	return mapGormError(s.db.WithContext(ctx).Delete(&model.TrainingMaterial{}, id).Error)
}

// --- SOP documents ---

func (s *ResourceService) ListSops(ctx context.Context, opt ListOptions, sopType string) ([]model.SopDocument, int64, error) {
	var items []model.SopDocument
	query := s.db.WithContext(ctx).Model(&model.SopDocument{})
	if sopType != "" {
		query = query.Where("sop_type = ?", sopType)
	}
	if opt.Search != "" {
		query = query.Where("title LIKE ?", "%"+opt.Search+"%")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size := normalizePage(opt.Page, opt.PageSize)
	if err := query.Order("updated_at desc").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *ResourceService) CreateSop(ctx context.Context, d *model.SopDocument) (*model.SopDocument, error) {
	if d.Title == "" || d.SopType == "" {
		return nil, ErrValidation
	}
	if d.Version == "" {
		d.Version = "1.0"
	}
	if d.Status == "" {
		d.Status = "active"
	}
	if err := s.db.WithContext(ctx).Create(d).Error; err != nil {
		return nil, mapCreateError(err)
	}
	return d, nil
}

func (s *ResourceService) UpdateSop(ctx context.Context, id uint, d *model.SopDocument) (*model.SopDocument, error) {
	var existing model.SopDocument
	if err := s.db.WithContext(ctx).First(&existing, id).Error; err != nil {
		return nil, mapGormError(err)
	}
	if d.Title == "" || d.SopType == "" {
		return nil, ErrValidation
	}
	d.ID = id
	d.CreatedAt = existing.CreatedAt
	if err := s.db.WithContext(ctx).Save(d).Error; err != nil {
		return nil, mapCreateError(err)
	}
	return d, nil
}

func (s *ResourceService) DeleteSop(ctx context.Context, id uint) error {
	return mapGormError(s.db.WithContext(ctx).Delete(&model.SopDocument{}, id).Error)
}

// --- Users (admin) ---

func (s *ResourceService) ListUsers(ctx context.Context, opt ListOptions) ([]model.User, int64, error) {
	var items []model.User
	query := s.db.WithContext(ctx).Model(&model.User{})
	if opt.Search != "" {
		like := "%" + opt.Search + "%"
		query = query.Where("username LIKE ? OR display_name LIKE ? OR email LIKE ?", like, like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size := normalizePage(opt.Page, opt.PageSize)
	if err := query.Order("created_at desc").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *ResourceService) CreateUser(ctx context.Context, u *model.User, password string) (*model.User, error) {
	if u.Username == "" || u.DisplayName == "" || len(password) < 8 {
		return nil, ErrValidation
	}
	if u.Role == "" {
		u.Role = model.RoleEmployee
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	u.Password = string(hash)
	if err := s.db.WithContext(ctx).Create(u).Error; err != nil {
		return nil, mapCreateError(err)
	}
	return u, nil
}

func (s *ResourceService) UpdateUser(ctx context.Context, id uint, u *model.User, password string) (*model.User, error) {
	var existing model.User
	if err := s.db.WithContext(ctx).First(&existing, id).Error; err != nil {
		return nil, mapGormError(err)
	}
	if u.DisplayName == "" {
		return nil, ErrValidation
	}
	updates := map[string]any{
		"display_name": u.DisplayName, "email": u.Email, "role": u.Role, "active": u.Active, "employee_id": u.EmployeeID,
	}
	if u.Role == "" {
		updates["role"] = existing.Role
	}
	if password != "" {
		if len(password) < 8 {
			return nil, ErrValidation
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		updates["password"] = string(hash)
	}
	if err := s.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}
	return s.GetUser(ctx, id)
}

func (s *ResourceService) GetUser(ctx context.Context, id uint) (*model.User, error) {
	var u model.User
	if err := s.db.WithContext(ctx).First(&u, id).Error; err != nil {
		return nil, mapGormError(err)
	}
	return &u, nil
}

func (s *ResourceService) DeleteUser(ctx context.Context, id uint) error {
	return mapGormError(s.db.WithContext(ctx).Delete(&model.User{}, id).Error)
}

// UserOptions lists all users for dropdowns.
func (s *ResourceService) UserOptions(ctx context.Context) ([]model.UserSummary, error) {
	var users []model.User
	if err := s.db.WithContext(ctx).Select("id", "display_name", "role").Order("display_name asc").Find(&users).Error; err != nil {
		return nil, err
	}
	out := make([]model.UserSummary, 0, len(users))
	for _, u := range users {
		out = append(out, model.UserSummary{ID: u.ID, DisplayName: u.DisplayName, Role: u.Role})
	}
	return out, nil
}
