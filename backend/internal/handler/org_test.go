package handler_test

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/example/training-platform/internal/model"
)

func TestAdminCreatesDepartment(t *testing.T) {
	r, _ := testServer()
	token := login(r, "Admin", "System Admin123")
	if token == "" {
		t.Fatal("login failed")
	}
	w := doJSON(r, "POST", "/api/v1/departments",
		`{"name":"QA","description":"Quality assurance"}`, token)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", w.Code, w.Body.String())
	}
}

func TestDuplicateDepartmentConflicts(t *testing.T) {
	r, _ := testServer()
	token := login(r, "Admin", "System Admin123")
	doJSON(r, "POST", "/api/v1/departments", `{"name":"Engineering","description":"x"}`, token)
	w := doJSON(r, "POST", "/api/v1/departments", `{"name":"Engineering","description":"y"}`, token)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", w.Code)
	}
}

func TestAdminCreatesEmployeeAndBindsPosition(t *testing.T) {
	r, db := testServer()
	token := login(r, "Admin", "System Admin123")
	// Seeded: AI Platform dept id and AI Product Manager position exist.
	var dept model.Department
	db.First(&dept, "name = ?", "产品部")
	var pos model.Position
	db.First(&pos, "name = ?", "AI 产品经理")

	body := `{"employee_code":"EMPX1","name":"Test New Hire","email":"test@x.com",` +
		`"department_id":` + itoa(int(dept.ID)) + `,"position_id":` + itoa(int(pos.ID)) + `,"hire_date":"2026-01-02"}`
	w := doJSON(r, "POST", "/api/v1/employees", body, token)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", w.Code, w.Body.String())
	}
	var out struct {
		Data model.Employee `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Data.DepartmentID != dept.ID || out.Data.PositionID != pos.ID {
		t.Fatalf("department/position binding failed: %+v", out.Data)
	}
}

func TestListEmployees(t *testing.T) {
	r, _ := testServer()
	token := login(r, "Admin", "System Admin123")
	w := doJSON(r, "GET", "/api/v1/employees", "", token)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var out struct {
		Data []model.Employee `json:"data"`
		Meta model.PageMeta   `json:"meta"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Meta.Total < 3 {
		t.Fatalf("expected at least 3 seeded employees, got %d", out.Meta.Total)
	}
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
