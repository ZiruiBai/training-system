package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/example/training-platform/internal/middleware"
	"github.com/example/training-platform/internal/model"
	"github.com/gin-gonic/gin"
)

// queryUint parses an optional unsigned int query parameter.
func queryUint(c *gin.Context, key string) uint {
	v := c.Query(key)
	if v == "" {
		return 0
	}
	n, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return 0
	}
	return uint(n)
}

// fail403 writes a standard forbidden response.
func fail403(c *gin.Context) {
	c.JSON(http.StatusForbidden, model.ErrorEnvelope{
		Error: model.ErrorDetail{Code: "FORBIDDEN", Message: "You do not have permission to perform this action", RequestID: middleware.GetRequestID(c)},
	})
}

// timeFromJSON is an RFC3339 time.Time that unmarshals from a JSON string or
// a numeric Unix-timestamp, tolerating both formats sent by clients.
type timeFromJSON time.Time

// UnmarshalJSON parses RFC3339 or Unix-ms timestamps.
func (t *timeFromJSON) UnmarshalJSON(data []byte) error {
	s := string(data)
	switch s {
	case "null", `""`:
		*t = timeFromJSON(time.Time{})
		return nil
	}
	// Numeric Unix seconds.
	if v, err := strconv.ParseInt(s, 10, 64); err == nil {
		*t = timeFromJSON(time.Unix(v, 0).UTC())
		return nil
	}
	// RFC3339 quoted string.
	if v, err := time.Parse(`"2006-01-02T15:04:05Z"`, s); err == nil {
		*t = timeFromJSON(v.UTC())
		return nil
	}
	if v, err := time.Parse(`"2006-01-02T15:04:05.999999999Z07:00"`, s); err == nil {
		*t = timeFromJSON(v.UTC())
		return nil
	}
	// Date-only.
	if v, err := time.Parse(`"2006-01-02"`, s); err == nil {
		*t = timeFromJSON(v.UTC())
		return nil
	}
	_t := time.Time{}
	_ = _t
	*t = timeFromJSON(time.Time{})
	return nil
}

func (t *timeFromJSON) Time() time.Time { return time.Time(*t) }

// modelFromEmployeeUpsert converts the human-readable upsert into an Employee.
func modelFromEmployeeUpsert(req employeeUpsert) *model.Employee {
	e := &model.Employee{
		EmployeeCode: req.EmployeeCode, Name: req.Name, Email: req.Email, Phone: req.Phone,
		DepartmentID: req.DepartmentID, PositionID: req.PositionID, ManagerID: req.ManagerID,
		WorkExperience: req.WorkExperience, SkillLevel: req.SkillLevel,
		CurrentStage: req.CurrentStage, TrainingStatus: req.TrainingStatus, Progress: req.Progress,
	}
	if req.HireDate != nil {
		e.HireDate = req.HireDate.Time()
	}
	if e.SkillLevel == "" {
		e.SkillLevel = model.LevelJunior
	}
	if e.TrainingStatus == "" {
		e.TrainingStatus = model.StatusNotStarted
	}
	return e
}
