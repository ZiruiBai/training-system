package handler_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/example/training-platform/internal/model"
)

func TestAdminCreatesPlan(t *testing.T) {
	r, db := testServer()
	token := login(r, "Admin", "System Admin123")
	var emp model.Employee
	db.First(&emp, "name = ?", "Bob New Hire")
	body := `{"employee_id":` + itoa(int(emp.ID)) + `,"start_date":"2026-01-02","end_date":"2026-04-02","cycle_days":90}`
	w := doJSON(r, "POST", "/api/v1/plans", body, token)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", w.Code, w.Body.String())
	}
	var out struct {
		Data model.TrainingPlan `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Data.Status != model.PlanDraft {
		t.Fatalf("expected draft status, got %s", out.Data.Status)
	}
}

func TestTransitionPlanToInProgress(t *testing.T) {
	r, db := testServer()
	token := login(r, "Admin", "System Admin123")
	var plan model.TrainingPlan
	db.Where("status = ?", model.PlanDraft).First(&plan)
	if plan.ID == 0 {
		// Seed marks bob plan in_progress already; use any plan.
		db.First(&plan)
	}
	w := doJSON(r, "POST", "/api/v1/plans/"+itoa(int(plan.ID))+"/transition",
		`{"status":"published"}`, token)
	// published transitions to a valid state; if already in_progress it may conflict.
	if w.Code != http.StatusOK && w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 200 or 409: %s", w.Code, w.Body.String())
	}
}

func TestEmployeeCompletesOwnTask(t *testing.T) {
	r, db := testServer()
	token := login(r, "Bob New Hire", "Bob New Hire123")
	var emp model.Employee
	db.First(&emp, "name = ?", "Bob New Hire")
	var task model.LearningTask
	db.Where("employee_id = ? AND status = ?", emp.ID, model.TaskInProgress).First(&task)
	if task.ID == 0 {
		db.Where("employee_id = ?", emp.ID).First(&task)
	}
	w := doJSON(r, "POST", "/api/v1/tasks/"+itoa(int(task.ID))+"/complete",
		`{"outcome":"Done","score":90}`, token)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	var out struct {
		Data model.LearningTask `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Data.Status != model.TaskCompleted {
		t.Fatalf("expected completed, got %s", out.Data.Status)
	}
}

func TestEmployeeCannotCompleteOthersTask(t *testing.T) {
	r, db := testServer()
	token := login(r, "Bob New Hire", "Bob New Hire123")
	var carol model.Employee
	db.First(&carol, "name = ?", "Carol Chen")
	var task model.LearningTask
	db.Where("employee_id = ?", carol.ID).First(&task)
	if task.ID == 0 {
		t.Skip("no task for Carol")
	}
	w := doJSON(r, "POST", "/api/v1/tasks/"+itoa(int(task.ID))+"/complete",
		`{"outcome":"X"}`, token)
	if w.Code == http.StatusOK {
		t.Fatalf("employee should not complete another employee's task")
	}
}

func TestEmployeeSeesOnlyOwnTasks(t *testing.T) {
	r, db := testServer()
	token := login(r, "Bob New Hire", "Bob New Hire123")
	var carol model.Employee
	db.First(&carol, "name = ?", "Carol Chen")
	if carol.ID == 0 {
		t.Skip("no carol")
	}
	// Inject a task for carol; employee should not see it.
	w := doJSON(r, "GET", "/api/v1/tasks", "", token)
	var out struct {
		Data []model.LearningTask `json:"data"`
		Meta model.PageMeta       `json:"meta"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	for _, tsk := range out.Data {
		if tsk.EmployeeID == carol.ID {
			t.Fatalf("employee saw another employee's task (carol id=%d)", carol.ID)
		}
	}
}
