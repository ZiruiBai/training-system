package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/example/training-platform/internal/model"
	"gorm.io/gorm"
)

var (
	// ErrValidation indicates invalid business input.
	ErrValidation = errors.New("validation failed")
	// ErrNotFound indicates the resource was not found.
	ErrNotFound = errors.New("not found")
	// ErrConflict indicates a uniqueness/state conflict.
	ErrConflict = errors.New("conflict")
	// ErrForbidden indicates insufficient permission.
	ErrForbidden = errors.New("forbidden")
)

// ListOptions is shared pagination/filter input.
type ListOptions struct {
	Page     int
	PageSize int
	Search   string
	// EmployeeIDs, when non-empty, restricts results to these employees
	// (used to scope manager views to their own department).
	EmployeeIDs []uint
}

// OrgService handles departments, positions, competencies and employees.
type OrgService struct {
	db *gorm.DB
}

// NewOrgService creates an OrgService.
func NewOrgService(db *gorm.DB) *OrgService { return &OrgService{db: db} }

// --- Departments ---

// ListDepartments returns departments with an optional filename search.
func (s *OrgService) ListDepartments(ctx context.Context, opt ListOptions) ([]model.Department, int64, error) {
	var items []model.Department
	query := s.db.WithContext(ctx).Model(&model.Department{})
	if opt.Search != "" {
		like := "%" + opt.Search + "%"
		query = query.Where("name LIKE ? OR description LIKE ?", like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size := normalizePage(opt.Page, opt.PageSize)
	if err := query.Preload("Manager").Order("created_at desc").
		Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetDepartment returns one department.
func (s *OrgService) GetDepartment(ctx context.Context, id uint) (*model.Department, error) {
	var d model.Department
	if err := s.db.WithContext(ctx).Preload("Manager").First(&d, id).Error; err != nil {
		return nil, mapGormError(err)
	}
	return &d, nil
}

// CreateDepartment creates a department.
func (s *OrgService) CreateDepartment(ctx context.Context, name, description string, managerID *uint) (*model.Department, error) {
	if name == "" {
		return nil, ErrValidation
	}
	d := model.Department{Name: name, Description: description, ManagerID: managerID}
	if err := s.db.WithContext(ctx).Create(&d).Error; err != nil {
		return nil, mapCreateError(err)
	}
	return &d, nil
}

// UpdateDepartment updates a department.
func (s *OrgService) UpdateDepartment(ctx context.Context, id uint, name, description string, managerID *uint) (*model.Department, error) {
	var d model.Department
	if err := s.db.WithContext(ctx).First(&d, id).Error; err != nil {
		return nil, mapGormError(err)
	}
	if name == "" {
		return nil, ErrValidation
	}
	d.Name = name
	d.Description = description
	d.ManagerID = managerID
	if err := s.db.WithContext(ctx).Save(&d).Error; err != nil {
		return nil, mapCreateError(err)
	}
	return &d, nil
}

// DeleteDepartment removes a department if it has no employees.
func (s *OrgService) DeleteDepartment(ctx context.Context, id uint) error {
	var count int64
	if err := s.db.WithContext(ctx).Model(&model.Employee{}).Where("department_id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrConflict
	}
	return mapGormError(s.db.WithContext(ctx).Delete(&model.Department{}, id).Error)
}

// --- Positions ---

// ListPositions returns positions with optional department filter.
func (s *OrgService) ListPositions(ctx context.Context, opt ListOptions, departmentID uint) ([]model.Position, int64, error) {
	var items []model.Position
	query := s.db.WithContext(ctx).Model(&model.Position{})
	if departmentID > 0 {
		query = query.Where("department_id = ?", departmentID)
	}
	if opt.Search != "" {
		query = query.Where("name LIKE ?", "%"+opt.Search+"%")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size := normalizePage(opt.Page, opt.PageSize)
	if err := query.Preload("Department").Order("created_at desc").
		Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetPosition returns a position with its competencies.
func (s *OrgService) GetPosition(ctx context.Context, id uint) (*model.Position, error) {
	var p model.Position
	if err := s.db.WithContext(ctx).Preload("Department").First(&p, id).Error; err != nil {
		return nil, mapGormError(err)
	}
	return &p, nil
}

// CreatePosition creates a position.
func (s *OrgService) CreatePosition(ctx context.Context, p *model.Position) (*model.Position, error) {
	if p.Name == "" || p.DepartmentID == 0 {
		return nil, ErrValidation
	}
	if p.CycleDays <= 0 {
		p.CycleDays = 90
	}
	if err := s.db.WithContext(ctx).Create(p).Error; err != nil {
		return nil, mapCreateError(err)
	}
	return p, nil
}

// UpdatePosition updates a position.
func (s *OrgService) UpdatePosition(ctx context.Context, id uint, p *model.Position) (*model.Position, error) {
	var existing model.Position
	if err := s.db.WithContext(ctx).First(&existing, id).Error; err != nil {
		return nil, mapGormError(err)
	}
	if p.Name == "" || p.DepartmentID == 0 {
		return nil, ErrValidation
	}
	p.ID = id
	if p.CycleDays <= 0 {
		p.CycleDays = 90
	}
	p.CreatedAt = existing.CreatedAt
	if err := s.db.WithContext(ctx).Save(p).Error; err != nil {
		return nil, mapCreateError(err)
	}
	return p, nil
}

// DeletePosition removes a position if it has no employees.
func (s *OrgService) DeletePosition(ctx context.Context, id uint) error {
	var count int64
	if err := s.db.WithContext(ctx).Model(&model.Employee{}).Where("position_id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrConflict
	}
	// Remove related competencies.
	if err := s.db.WithContext(ctx).Where("position_id = ?", id).Delete(&model.PositionCompetency{}).Error; err != nil {
		return err
	}
	return mapGormError(s.db.WithContext(ctx).Delete(&model.Position{}, id).Error)
}

// --- Position competencies ---

// ListCompetencies returns competencies for a position.
func (s *OrgService) ListCompetencies(ctx context.Context, positionID uint) ([]model.PositionCompetency, error) {
	var items []model.PositionCompetency
	if err := s.db.WithContext(ctx).Where("position_id = ?", positionID).Order("weight desc").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// UpsertCompetencies replaces the competency set of a position.
func (s *OrgService) UpsertCompetencies(ctx context.Context, positionID uint, items []model.PositionCompetency) error {
	if _, err := s.GetPosition(ctx, positionID); err != nil {
		return err
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("position_id = ?", positionID).Delete(&model.PositionCompetency{}).Error; err != nil {
			return err
		}
		for i := range items {
			items[i].ID = 0
			items[i].PositionID = positionID
			if items[i].Name == "" {
				return ErrValidation
			}
			if err := tx.Create(&items[i]).Error; err != nil {
				return mapCreateError(err)
			}
		}
		return nil
	})
}

// --- Employees ---

// ListEmployees returns employees with optional filters.
func (s *OrgService) ListEmployees(ctx context.Context, opt ListOptions, departmentID, positionID uint, status string) ([]model.Employee, int64, error) {
	var items []model.Employee
	query := s.db.WithContext(ctx).Model(&model.Employee{})
	if departmentID > 0 {
		query = query.Where("department_id = ?", departmentID)
	}
	if positionID > 0 {
		query = query.Where("position_id = ?", positionID)
	}
	if status != "" {
		query = query.Where("training_status = ?", status)
	}
	if opt.Search != "" {
		like := "%" + opt.Search + "%"
		query = query.Where("name LIKE ? OR employee_code LIKE ? OR email LIKE ?", like, like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size := normalizePage(opt.Page, opt.PageSize)
	if err := query.Preload("Department").Preload("Position").Preload("Manager").
		Order("created_at desc").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// ManagerCanAccessEmployee reports whether the manager may access the employee
// (direct report or member of a department they manage).
func (s *OrgService) ManagerCanAccessEmployee(ctx context.Context, managerID uint, e *model.Employee) bool {
	return ManagerCanAccessEmployee(ctx, s.db, managerID, e)
}

// ListEmployeesByManager returns employees a manager may access: direct
// reports plus everyone in departments they manage.
func (s *OrgService) ListEmployeesByManager(ctx context.Context, managerID uint, opt ListOptions) ([]model.Employee, int64, error) {
	var items []model.Employee
	deptIDs, err := ManagedDepartmentIDs(ctx, s.db, managerID)
	if err != nil {
		return nil, 0, err
	}
	query := ManagerEmployeeScope(s.db.WithContext(ctx).Model(&model.Employee{}), managerID, deptIDs)
	if opt.Search != "" {
		like := "%" + opt.Search + "%"
		query = query.Where("name LIKE ? OR employee_code LIKE ?", like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size := normalizePage(opt.Page, opt.PageSize)
	if err := query.Preload("Department").Preload("Position").
		Order("created_at desc").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetEmployee returns one employee.
func (s *OrgService) GetEmployee(ctx context.Context, id uint) (*model.Employee, error) {
	var e model.Employee
	if err := s.db.WithContext(ctx).Preload("Department").Preload("Position").Preload("Manager").First(&e, id).Error; err != nil {
		return nil, mapGormError(err)
	}
	return &e, nil
}

// CreateEmployee creates an employee record.
func (s *OrgService) CreateEmployee(ctx context.Context, e *model.Employee) (*model.Employee, error) {
	if e.Name == "" || e.EmployeeCode == "" || e.Email == "" {
		return nil, ErrValidation
	}
	if e.SkillLevel == "" {
		e.SkillLevel = model.LevelJunior
	}
	if e.TrainingStatus == "" {
		e.TrainingStatus = model.StatusNotStarted
	}
	if e.HireDate.IsZero() {
		e.HireDate = time.Now().UTC()
	}
	if err := s.db.WithContext(ctx).Create(e).Error; err != nil {
		return nil, mapCreateError(err)
	}
	return e, nil
}

// UpdateEmployee updates an employee.
func (s *OrgService) UpdateEmployee(ctx context.Context, id uint, e *model.Employee) (*model.Employee, error) {
	var existing model.Employee
	if err := s.db.WithContext(ctx).First(&existing, id).Error; err != nil {
		return nil, mapGormError(err)
	}
	if e.Name == "" || e.Email == "" {
		return nil, ErrValidation
	}
	e.ID = id
	e.EmployeeCode = existing.EmployeeCode
	e.CreatedAt = existing.CreatedAt
	if err := s.db.WithContext(ctx).Save(e).Error; err != nil {
		return nil, mapCreateError(err)
	}
	return e, nil
}

// DeleteEmployee removes an employee.
func (s *OrgService) DeleteEmployee(ctx context.Context, id uint) error {
	return mapGormError(s.db.WithContext(ctx).Delete(&model.Employee{}, id).Error)
}

func normalizePage(page, size int) (int, int) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 200 {
		size = 200
	}
	return page, size
}

func mapGormError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}

func mapCreateError(err error) error {
	if err == nil {
		return nil
	}
	if isUniqueViolation(err) {
		return ErrConflict
	}
	return err
}

// isUniqueViolation detects SQLite/Postgres unique constraint errors.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") || strings.Contains(msg, "duplicate key value")
}
