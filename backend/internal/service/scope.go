package service

import (
	"context"

	"github.com/example/training-platform/internal/model"
	"gorm.io/gorm"
)

// ManagedDepartmentIDs returns the IDs of departments managed by the user.
func ManagedDepartmentIDs(ctx context.Context, db *gorm.DB, managerUserID uint) ([]uint, error) {
	var ids []uint
	err := db.WithContext(ctx).Model(&model.Department{}).
		Where("manager_id = ?", managerUserID).
		Pluck("id", &ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// ManagerEmployeeScope filters employees a manager may access: direct reports
// (employee.manager_id = user) plus everyone in departments they manage.
// deptIDs must be fetched beforehand (e.g. via ManagedDepartmentIDs); this
// function only composes the WHERE clause and never executes a query.
func ManagerEmployeeScope(db *gorm.DB, managerUserID uint, deptIDs []uint) *gorm.DB {
	if len(deptIDs) == 0 {
		return db.Where("manager_id = ?", managerUserID)
	}
	return db.Where("manager_id = ? OR department_id IN ?", managerUserID, deptIDs)
}

// ManagedEmployeeIDs returns the IDs of employees a manager may access.
func ManagedEmployeeIDs(ctx context.Context, db *gorm.DB, managerUserID uint) ([]uint, error) {
	deptIDs, err := ManagedDepartmentIDs(ctx, db, managerUserID)
	if err != nil {
		return nil, err
	}
	var ids []uint
	err = ManagerEmployeeScope(db.WithContext(ctx).Model(&model.Employee{}), managerUserID, deptIDs).
		Pluck("id", &ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// ManagerCanAccessEmployee reports whether a manager may access the employee:
// the employee is a direct report or belongs to a department they manage.
func ManagerCanAccessEmployee(ctx context.Context, db *gorm.DB, managerUserID uint, e *model.Employee) bool {
	if e.ManagerID != nil && *e.ManagerID == managerUserID {
		return true
	}
	var count int64
	db.WithContext(ctx).Model(&model.Department{}).
		Where("id = ? AND manager_id = ?", e.DepartmentID, managerUserID).
		Count(&count)
	return count > 0
}