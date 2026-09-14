package database

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/example/training-platform/internal/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// hashPassword returns a bcrypt hash for a plaintext password.
func hashPassword(plain string) string {
	h, _ := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	return string(h)
}

// newToken returns a random hex token.
func newToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Seed inserts the initial demo dataset for development and UAT. It must not
// be a security boundary; the default admin password is for demonstration only.
func Seed(db *gorm.DB) error {
	var count int64
	if err := db.Model(&model.User{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	now := time.Now()

	admin := model.User{Username: "Admin", Email: "admin@example.com",
		Password: hashPassword("System Admin123"), DisplayName: "Admin", Role: model.RoleAdmin, Active: true}
	manager := model.User{Username: "Manager", Email: "manager@example.com",
		Password: hashPassword("Alice Manager123"), DisplayName: "Manager", Role: model.RoleManager, Active: true}
	employeeUser := model.User{Username: "Bob New Hire", Email: "employee@example.com",
		Password: hashPassword("Bob New Hire123"), DisplayName: "Bob New Hire", Role: model.RoleEmployee, Active: true}

	if err := db.Create([]*model.User{&admin, &manager, &employeeUser}).Error; err != nil {
		return err
	}

	engDept := model.Department{Name: "Engineering", Description: "Product engineering"}
	salesDept0 := model.Department{Name: "Sales", Description: "Sales and revenue"}
	hrDept := model.Department{Name: "Human Resources", Description: "HR operations"}
	db.Create([]*model.Department{&engDept, &salesDept0, &hrDept})

	// Product department (owns the AI Product Manager role). Manager is the Product Manager of this department.
	productDept := model.Department{Name: "产品部", Description: "Product & AI product management", ManagerID: &manager.ID}
	db.Create(&productDept)

	pmPos := model.Position{Name: "Product Manager", DepartmentID: engDept.ID, Description: "Owns product roadmap",
		Responsibilities: "Gather requirements; write PRDs; drive delivery", Requirement: "3+ years PM experience",
		CycleDays: 90}
	aiPM := model.Position{Name: "AI 产品经理", DepartmentID: productDept.ID, Description: "AI product ownership",
		Responsibilities: "Spec AI features; coordinate ML; evaluate model outcomes",
		Requirement:      "AI fundamentals + PM experience", CoreAbilities: "AI literacy; data analysis", CycleDays: 90}
	db.Create([]*model.Position{&pmPos, &aiPM})

	// Position competencies for AI PM.
	db.Create([]*model.PositionCompetency{
		{PositionID: aiPM.ID, Name: "Product Fundamentals", TargetLevel: 80, Weight: 1.0, Description: "Baseline product skills"},
		{PositionID: aiPM.ID, Name: "Requirements Analysis", TargetLevel: 85, Weight: 1.2, Description: "Mine and structure requirements"},
		{PositionID: aiPM.ID, Name: "PRD Writing", TargetLevel: 85, Weight: 1.2, Description: "Author clear PRDs"},
		{PositionID: aiPM.ID, Name: "Data Analysis", TargetLevel: 75, Weight: 1.0, Description: "Interpret data for decisions"},
		{PositionID: aiPM.ID, Name: "AI Knowledge", TargetLevel: 80, Weight: 1.0, Description: "Model, prompt, and eval literacy"},
		{PositionID: aiPM.ID, Name: "Communication", TargetLevel: 80, Weight: 0.8, Description: "Cross-team communication"},
		{PositionID: aiPM.ID, Name: "Project Management", TargetLevel: 70, Weight: 0.8, Description: "Plan and track delivery"},
	})

	// Position competencies for Product Manager.
	db.Create([]*model.PositionCompetency{
		{PositionID: pmPos.ID, Name: "Product Strategy", TargetLevel: 80, Weight: 1.0, Description: "Shape roadmap and priorities"},
		{PositionID: pmPos.ID, Name: "Requirements Analysis", TargetLevel: 85, Weight: 1.2, Description: "Mine and structure requirements"},
		{PositionID: pmPos.ID, Name: "PRD Writing", TargetLevel: 85, Weight: 1.2, Description: "Author clear PRDs"},
		{PositionID: pmPos.ID, Name: "Stakeholder Communication", TargetLevel: 80, Weight: 0.8, Description: "Align cross-team stakeholders"},
		{PositionID: pmPos.ID, Name: "Data Analysis", TargetLevel: 75, Weight: 1.0, Description: "Interpret data for decisions"},
	})

	bob := model.Employee{EmployeeCode: "EMP001", Name: "Bob New Hire", Email: "employee@example.com",
		Phone: "13800000001", DepartmentID: productDept.ID, PositionID: aiPM.ID, ManagerID: &manager.ID,
		HireDate: now.AddDate(0, 0, -10), WorkExperience: 0, SkillLevel: model.LevelJunior,
		CurrentStage: 1, TrainingStatus: model.StatusInProgress, Progress: 40}
	carol := model.Employee{EmployeeCode: "EMP002", Name: "Carol Chen", Email: "carol@example.com",
		Phone: "13800000002", DepartmentID: engDept.ID, PositionID: pmPos.ID,
		HireDate: now.AddDate(0, 0, -55), WorkExperience: 2, SkillLevel: model.LevelMiddle,
		CurrentStage: 2, TrainingStatus: model.StatusInProgress, Progress: 70}
	dave := model.Employee{EmployeeCode: "EMP003", Name: "Dave Liu", Email: "dave@example.com",
		Phone: "13800000003", DepartmentID: engDept.ID, PositionID: pmPos.ID,
		HireDate: now.AddDate(0, 0, -95), WorkExperience: 5, SkillLevel: model.LevelSenior,
		CurrentStage: 3, TrainingStatus: model.StatusCompleted, Progress: 100}
	erin := model.Employee{EmployeeCode: "EMP004", Name: "Erin Wu", Email: "erin@example.com",
		Phone: "13800000004", DepartmentID: productDept.ID, PositionID: aiPM.ID, ManagerID: &manager.ID,
		HireDate: now.AddDate(0, 0, -3), WorkExperience: 0, SkillLevel: model.LevelJunior,
		CurrentStage: 1, TrainingStatus: model.StatusNotStarted, Progress: 0}
	db.Create([]*model.Employee{&bob, &carol, &dave, &erin})

	// Link the employee user to 张三.
	db.Model(&employeeUser).Update("employee_id", bob.ID)

	// Plans.
	bobPlan := model.TrainingPlan{EmployeeID: bob.ID, StartDate: bob.HireDate,
		EndDate: bob.HireDate.AddDate(0, 0, 89), CycleDays: 90, CurrentStage: 1, Progress: 40,
		Status: model.PlanInProgress, CreatedBy: &admin.ID, CreatedSource: model.SourceManual}
	carolPlan := model.TrainingPlan{EmployeeID: carol.ID, StartDate: carol.HireDate,
		EndDate: carol.HireDate.AddDate(0, 0, 89), CycleDays: 90, CurrentStage: 2, Progress: 70,
		Status: model.PlanInProgress, CreatedBy: &admin.ID, CreatedSource: model.SourceManual}
	davePlan := model.TrainingPlan{EmployeeID: dave.ID, StartDate: dave.HireDate,
		EndDate: dave.HireDate.AddDate(0, 0, 89), CycleDays: 90, CurrentStage: 3, Progress: 100,
		Status: model.PlanCompleted, CreatedBy: &admin.ID, CreatedSource: model.SourceManual}
	db.Create([]*model.TrainingPlan{&bobPlan, &carolPlan, &davePlan})

	stage1 := model.TrainingStage{PlanID: bobPlan.ID, StageNumber: 1, Name: "Onboarding & Fundamentals",
		Objectives: "Familiarize with company, role duties, tools, and policies",
		StartDate:  bob.HireDate, EndDate: bob.HireDate.AddDate(0, 0, 29), Status: model.StageInProgress, Progress: 60}
	stage2 := model.TrainingStage{PlanID: bobPlan.ID, StageNumber: 2, Name: "Capability Building",
		Objectives: "Master core knowledge, SOPs, real tasks, start independent work",
		StartDate:  bob.HireDate.AddDate(0, 0, 30), EndDate: bob.HireDate.AddDate(0, 0, 59), Status: model.StageNotStarted, Progress: 0}
	stage3 := model.TrainingStage{PlanID: bobPlan.ID, StageNumber: 3, Name: "Independent Competence",
		Objectives: "Complete work independently, meet capability standard, final assessment",
		StartDate:  bob.HireDate.AddDate(0, 0, 60), EndDate: bob.HireDate.AddDate(0, 0, 89), Status: model.StageNotStarted, Progress: 0}
	db.Create([]*model.TrainingStage{&stage1, &stage2, &stage3})

	// Tasks across stages (today + upcoming + completed).
	baseTasks := []model.LearningTask{
		{EmployeeID: bob.ID, PlanID: bobPlan.ID, StageID: stage1.ID, Title: "Company culture review",
			Description: "Read the employee handbook", TaskType: model.TaskReading, StartDate: bob.HireDate,
			DueDate: bob.HireDate.AddDate(0, 0, 3), EstimateHours: 2, Priority: model.PriorityHigh,
			Status: model.TaskCompleted, Score: float64p(88), Outcome: "Read handbook and quiz 88%"},
		{EmployeeID: bob.ID, PlanID: bobPlan.ID, StageID: stage1.ID, Title: "Role & department onboarding",
			Description: "Meet the team and review role duties", TaskType: model.TaskPractice, StartDate: bob.HireDate,
			DueDate: bob.HireDate.AddDate(0, 0, 5), EstimateHours: 4, Priority: model.PriorityHigh,
			Status: model.TaskCompleted, Score: float64p(90), Outcome: "Completed onboarding session"},
		{EmployeeID: bob.ID, PlanID: bobPlan.ID, StageID: stage1.ID, Title: "AI fundamentals course",
			Description: "Complete the AI basics video series", TaskType: model.TaskVideo, StartDate: bob.HireDate.AddDate(0, 0, 6),
			DueDate: bob.HireDate.AddDate(0, 0, 12), EstimateHours: 6, Priority: model.PriorityMedium,
			Status: model.TaskCompleted, Score: float64p(85), Outcome: "Finished video series"},
		{EmployeeID: bob.ID, PlanID: bobPlan.ID, StageID: stage1.ID, Title: "Write a sample PRD",
			Description: "Draft a PRD for a simple AI feature", TaskType: model.TaskHomework, StartDate: bob.HireDate.AddDate(0, 0, 10),
			DueDate: bob.HireDate.AddDate(0, 0, 16), EstimateHours: 8, Priority: model.PriorityHigh,
			Status: model.TaskInProgress, Outcome: ""},
		{EmployeeID: bob.ID, PlanID: bobPlan.ID, StageID: stage1.ID, Title: "Data analysis exercise",
			Description: "Analyze sample product metrics", TaskType: model.TaskPractice, StartDate: bob.HireDate.AddDate(0, 0, 17),
			DueDate: bob.HireDate.AddDate(0, 0, 22), EstimateHours: 5, Priority: model.PriorityMedium,
			Status: model.TaskPending, Outcome: ""},
		{EmployeeID: bob.ID, PlanID: bobPlan.ID, StageID: stage1.ID, Title: "Weekly reflection",
			Description: "Write a reflection on week one learnings", TaskType: model.TaskReview, StartDate: bob.HireDate,
			DueDate: bob.HireDate.AddDate(0, 0, 7), EstimateHours: 1, Priority: model.PriorityLow,
			Status: model.TaskCompleted, Outcome: "Submitted reflection"},
		{EmployeeID: bob.ID, PlanID: bobPlan.ID, StageID: stage1.ID, Title: "AI fundamentals quiz",
			Description: "Take a short quiz on AI fundamentals", TaskType: model.TaskTest, StartDate: bob.HireDate,
			DueDate: bob.HireDate.AddDate(0, 0, 7), EstimateHours: 1, Priority: model.PriorityMedium,
			Status: model.TaskPending, Outcome: "",
			Quiz: []model.QuizQuestion{
				{Prompt: "What is an LLM primarily good at?", Options: []string{"Image editing", "Text generation and reasoning", "Database queries", "Hardware acceleration"}, Answer: 1},
				{Prompt: "What does 'grounding' mean in AI product work?", Options: []string{"Connecting to the ground network", "Anchoring model output to trusted data", "Training with no data", "Deploying to production"}, Answer: 1},
				{Prompt: "Which is a core risk of LLM products?", Options: []string{"Faster compile times", "Hallucination", "Lower server costs", "No licensing needed"}, Answer: 1},
				{Prompt: "What should a PM define before shipping an AI feature?", Options: []string{"The exact model weights", "A success metric and evaluation", "Only the UI color", "The marketing slogan"}, Answer: 1},
			},
		},
	}
	db.Create(&baseTasks)

	// Bob's daily quiz (pending) so he also has fresh questions on the
	// "Today's Quiz" page.
	bobDaily := model.LearningTask{EmployeeID: bob.ID, PlanID: bobPlan.ID, StageID: stage1.ID,
		Title: dailyQuizTitle("pm"), TaskType: model.TaskTest,
		StartDate: bob.HireDate, DueDate: bob.HireDate.AddDate(0, 0, 14),
		EstimateHours: 0.5, Priority: model.PriorityLow, Status: model.TaskPending,
		CreatedSource: model.SourceManual, Quiz: dailyQuizQuestions("pm")}
	db.Create(&bobDaily)

	// Learning records for completed tasks.
	record1 := model.LearningRecord{EmployeeID: bob.ID, TaskID: baseTasks[0].ID,
		Content: "Read handbook and completed quiz", StartTime: bob.HireDate.Add(time.Hour),
		EndTime: timeP(bob.HireDate.Add(2 * time.Hour)), DurationMin: 120, Status: model.TaskCompleted,
		TestScore: float64p(88), SelfEval: "Clear onboarding material", Outcome: "Quiz passed"}
	record2 := model.LearningRecord{EmployeeID: bob.ID, TaskID: baseTasks[1].ID,
		Content: "Onboarding session with team", StartTime: bob.HireDate.AddDate(0, 0, 1).Add(2 * time.Hour),
		EndTime: timeP(bob.HireDate.AddDate(0, 0, 1).Add(5 * time.Hour)), DurationMin: 180,
		Status: model.TaskCompleted, SelfEval: "Great intro", Outcome: "Met the team"}
	db.Create([]*model.LearningRecord{&record1, &record2})

	// Assessments for bob (stage 1).
	assess1 := model.Assessment{EmployeeID: bob.ID, PlanID: bobPlan.ID, StageNumber: 1,
		AssessDate: bob.HireDate.AddDate(0, 0, 10), AssessorID: &manager.ID, CompositeScore: 78,
		Strengths: "Quick learner; good communication", Weaknesses: "Data analysis needs practice",
		Improvement: "Focus on data exercises", GoalsMet: true, Status: model.AssessmentDone,
		AssessmentSource: model.AssessmentSourceManual}
	db.Create(&assess1)
	db.Create([]*model.AssessmentScore{
		{AssessmentID: assess1.ID, Capability: "Product Fundamentals", Score: 82, TargetLevel: 80, Weight: 1.0},
		{AssessmentID: assess1.ID, Capability: "Communication", Score: 76, TargetLevel: 80, Weight: 0.8},
		{AssessmentID: assess1.ID, Capability: "Data Analysis", Score: 65, TargetLevel: 75, Weight: 1.0},
		{AssessmentID: assess1.ID, Capability: "AI Knowledge", Score: 88, TargetLevel: 80, Weight: 1.0},
	})

	// Courses, materials, SOPs.
	db.Create([]*model.Course{
		{
			Title: "Employee Handbook", Category: model.CourseCategoryPolicy, Description: "Company policies",
			Difficulty: 1, EstimateHours: 2, Status: model.CoursePublished,
			Objectives: "Understand company culture, code of conduct and key employment policies.",
			Content:    "1. Culture & Values: We are an AI-first product company that values learning, ownership and transparency.\n2. Code of Conduct: Act with integrity, protect user data, and report any ethics or compliance concerns to your manager or HR.\n3. Work Hours: Core collaboration hours are 10:00-17:00; you are expected to keep a consistent schedule and coordinate delays with your team.\n4. Attendance & Leave: Submit leave via the Attendance SOP before taking time off; notify your team in advance.\n5. Security: Never share credentials, keep company data inside approved tools, and follow the IT onboarding checklist.\n6. Performance & Growth: You will have 30/60/90 checkpoints with your manager to review progress and capability growth.",
		},
		{
			Title: "AI Fundamentals", Category: model.CourseCategorySpecialty, PositionID: &aiPM.ID, Description: "AI basics for PMs",
			Difficulty: 2, EstimateHours: 6, Status: model.CoursePublished,
			Objectives: "Grasp core AI terminology (model, prompt, evaluation), understand how to scope and evaluate AI features.",
			Content:    "1. What is an AI model? A model is trained on data to predict an output. As a PM you define the input/output contract and the quality target.\n2. LLMs and prompts: Large language models turn a prompt into a response. Prompt design, grounding and safety are product decisions, not just engineering ones.\n3. Evaluation & metrics: Define clear success metrics (accuracy, latency, cost, user satisfaction) before shipping. A feature without evaluation is not shippable.\n4. Common failure modes: hallucination, data leakage, bias, and cost creep. Your job is to surface these early and set expectations.\n5. Working with ML engineers: Share the business problem, agree on the evaluation set, and iterate on small experiments rather than big-bang delivery.",
		},
		{
			Title: "PRD Writing Essentials", Category: model.CourseCategoryPosition, PositionID: &aiPM.ID, Description: "How to write great PRDs",
			Difficulty: 2, EstimateHours: 4, Status: model.CoursePublished,
			Objectives: "Produce a structured one-page PRD: problem, users, success metrics, scope and open decisions.",
			Content:    "A PRD (Product Requirements Document) turns an idea into a buildable plan. Use this structure:\n1. Problem & Context: state the problem and why it matters now.\n2. Users & Personas: who is impacted and what jobs they are trying to do.\n3. Success Metrics: how to know the feature worked (north-star metric + guardrails).\n4. Scope & Requirements: list must-have vs nice-to-have; defer unknowns.\n5. Design & Flow: describe key user flows and edge cases.\n6. Risks & Open Questions: call out data, cost, safety and dependency risks.\nKeep it one page when possible. The goal is shared understanding, not documentation theatre.",
		},
		{
			Title: "Data Analysis for PMs", Category: model.CourseCategorySpecialty, PositionID: &aiPM.ID, Description: "Metrics and analysis",
			Difficulty: 2, EstimateHours: 5, Status: model.CourseDraft,
			Objectives: "Read core product metrics and turn data into product decisions.",
			Content:    "1. Funnel analysis: track activation, engagement, retention and conversion stages to find where users drop off.\n2. Segmentation: compare new vs returning users, channels and cohorts instead of only looking at aggregates.\n3. Hypothesis before dashboards: start from a question, then pick the smallest data that answers it.\n4. Common pitfalls: sampling bias, survivorship bias and correlation vs causation.\n5. Talking to data: always pair a number with its context and a proposed action.",
		},
	})
	db.Create([]*model.TrainingMaterial{
		{
			Title: "Onboarding Deck", FileType: "PPT", Category: model.MaterialPolicy,
			Description: "New hire orientation slides. Covers company culture, our AI product strategy, team structure, and the first-week checklist. Review these slides before your team kickoff to get the most out of onboarding.\nSections: 1) Welcome & values 2) How we build AI products 3) Your team & stakeholders 4) First 30 days checklist.",
			FilePath:    "",
		},
		{
			Title: "AI Product Handbook", FileType: "PDF", Category: model.MaterialSpecialty,
			Description: "AI product playbook. A practical reference for scoping AI features: how to define the problem, pick success metrics, run small experiments, and hand off to ML engineers with a clear evaluation plan.\nIncludes: prompt design checklist, evaluation template, and common failure modes to watch for.",
			FilePath:    "",
		},
	})
	db.Create([]*model.SopDocument{
		{
			Title: "Employee Handbook", SopType: model.SopTypeHandbook, Version: "1.0", EffectiveAt: now,
			Content: "Welcome to the company! This handbook is your day-one reference.\n1. Our Mission: build AI products that make work simpler.\n2. Working Environment: hybrid-friendly, with core hours 10:00-17:00 and flexible start times.\n3. Your First Week: complete onboarding tasks, meet your team, and set up your accounts before your first 1:1.\n4. Communication: use the official chat for daily updates and email for formal approvals.\n5. Code of Conduct: be honest, protect customer data and never post internal info publicly.\n6. Where to Ask: manager for role questions, HR for policies and benefits, IT for tools and access.",
		},
		{
			Title: "Attendance Policy", SopType: model.SopTypeAttendance, Version: "2.0", EffectiveAt: now,
			Content: "This policy sets expectations for working time and leave.\n1. Core Hours: 10:00 - 17:00 daily; outside these you may flex as long as work is covered.\n2. Reporting Attendance: mark your attendance daily in the HR app by 10 AM.\n3. Leave Requests: submit leave at least 1 day in advance through the HR app; emergencies are allowed with a same-day notice.\n4. Approval: leave is approved by your manager; incomplete requests may be declined.\n5. Overtime: if you work extra hours, coordinate with your manager and record them.\n6. Consequences: unauthorised absence affects your monthly review.",
		},
		{
			Title: "AI PM Weekly Workflow", SopType: model.SopTypeSOP, PositionID: &aiPM.ID, Version: "1.0", EffectiveAt: now,
			Content: "Standard weekly cadence for product managers.\n1. Monday - Plan: update the roadmap, review blocking issues and set weekly goals.\n2. Wednesday - Build Review: share progress, review experiments and resolve cross-team dependencies.\n3. Thursday - Evaluations: review feature metrics and model quality with the ML team.\n4. Friday - Retro: write a short retro and update the project tracker before 17:00.\n5. Continuous: keep the PRD and decision log updated as requirements change.\nDeliverables each week: updated roadmap, meeting notes and a one-line status for leadership.",
		},
	})

	// Reports for bob.
	db.Create(&model.TrainingReport{
		EmployeeID: bob.ID, PlanID: &bobPlan.ID, ReportType: model.ReportMonthly, Period: "Month 1",
		Progress: 40, TaskCompletion: 60, LearningHours: 5, CompositeScore: float64p(78),
		Strengths: "Good domain ramp-up", Weaknesses: "Data analysis gap",
		Suggestions: "Prioritize data exercises", ManagerEval: "On track", AIGenerated: false,
	})

	// Reusable training template for AI PM new hires.
	template := model.TrainingTemplate{
		Title:       "AI Product Manager 90-day Onboarding",
		Description: "Standard 30/60/90 onboarding for AI PM new hires.",
		PositionID:  &aiPM.ID, CycleDays: 90, Status: model.TemplatePublished,
		Stages: []model.TrainingTemplateStage{
			{StageNumber: 1, Name: "Onboarding & Fundamentals",
				Objectives:       "Get familiar with the company, department, role duties, policies and basic tools; complete induction training.",
				SampleTaskTitles: "Read employee handbook;Complete department onboarding;Watch AI fundamentals course;Take induction quiz;Set up dev tools",
				DurationDays:     30},
			{StageNumber: 2, Name: "Capability Building",
				Objectives:       "Master core job knowledge and SOPs, complete real tasks and begin working independently to build core competence.",
				SampleTaskTitles: "Learn position SOPs;Complete hands-on task;Shadow a senior colleague;Write a sample PRD;Weekly reflection",
				DurationDays:     30},
			{StageNumber: 3, Name: "Independent Competence",
				Objectives:       "Complete work independently, solve problems, meet the capability standard, pass the final assessment.",
				SampleTaskTitles: "Deliver an independent task;Solve a real problem;Final capability review;Final assessment",
				DurationDays:     30},
		},
	}
	db.Create(&template)

	// ===== Demo accounts: 3 new hires (R&D / Sales / Support) =====
	// Demo R&D engineer.
	devUser := model.User{Username: "Dev", Email: "dev@example.com",
		Password: hashPassword("Dev123"), DisplayName: "Dev", Role: model.RoleEmployee, Active: true}
	salesUser := model.User{Username: "Sarah", Email: "sales@example.com",
		Password: hashPassword("Sarah123"), DisplayName: "Sarah", Role: model.RoleEmployee, Active: true}
	serviceUser := model.User{Username: "Tom", Email: "service@example.com",
		Password: hashPassword("Tom123"), DisplayName: "Tom", Role: model.RoleEmployee, Active: true}
	db.Create([]*model.User{&devUser, &salesUser, &serviceUser})

	// Product-department new hires also get login accounts so the employee
	// view (today tasks, my plan, supplement tasks from AI coaching) can be
	// experienced end-to-end.
	erinUser := model.User{Username: "Erin", Email: "erin@example.com",
		Password: hashPassword("Erin123"), DisplayName: "Erin Wu", Role: model.RoleEmployee, Active: true}
	frankUser := model.User{Username: "Frank", Email: "frank@example.com",
		Password: hashPassword("Frank123"), DisplayName: "Frank New Hire", Role: model.RoleEmployee, Active: true}
	graceUser := model.User{Username: "Grace", Email: "grace@example.com",
		Password: hashPassword("Grace123"), DisplayName: "Grace New Hire", Role: model.RoleEmployee, Active: true}
	db.Create([]*model.User{&erinUser, &frankUser, &graceUser})

	rdDept := model.Department{Name: "R&D", Description: "Research and development"}
	svcDept := model.Department{Name: "Customer Support", Description: "Customer service"}
	// Reuse the Sales department created earlier (avoids a unique-name conflict).
	salesDept := salesDept0
	db.Create([]*model.Department{&rdDept, &svcDept})

	engPos := model.Position{Name: "Software Engineer", DepartmentID: rdDept.ID, Description: "Build and ship software",
		Responsibilities: "Develop features; write tests; fix bugs", Requirement: "CS degree or equivalent", CycleDays: 90}
	salesPos := model.Position{Name: "Sales Specialist", DepartmentID: salesDept.ID, Description: "Drive new business",
		Responsibilities: "Qualify leads; close deals; manage pipeline", Requirement: "Sales experience", CycleDays: 90}
	svcPos := model.Position{Name: "Support Specialist", DepartmentID: svcDept.ID, Description: "Resolve customer issues",
		Responsibilities: "Triage tickets; respond to customers; escalate", Requirement: "Customer service experience", CycleDays: 90}
	db.Create([]*model.Position{&engPos, &salesPos, &svcPos})

	db.Create([]*model.PositionCompetency{
		{PositionID: engPos.ID, Name: "Programming", TargetLevel: 85, Weight: 1.2, Description: "Write clean code", PassStandard: "Ship a feature end-to-end"},
		{PositionID: engPos.ID, Name: "Debugging", TargetLevel: 75, Weight: 1.0, Description: "Find and fix defects"},
		{PositionID: engPos.ID, Name: "Code Review", TargetLevel: 70, Weight: 0.8, Description: "Review peer PRs"},
		{PositionID: salesPos.ID, Name: "Lead Gen", TargetLevel: 80, Weight: 1.0, Description: "Find prospects", PassStandard: "Book qualified meetings"},
		{PositionID: salesPos.ID, Name: "Negotiation", TargetLevel: 75, Weight: 1.1, Description: "Close deals"},
		{PositionID: salesPos.ID, Name: "CRM", TargetLevel: 65, Weight: 0.7, Description: "Manage pipeline"},
		{PositionID: svcPos.ID, Name: "Product Knowledge", TargetLevel: 80, Weight: 1.0, Description: "Know product deeply", PassStandard: "Resolve tickets independently"},
		{PositionID: svcPos.ID, Name: "Communication", TargetLevel: 85, Weight: 1.2, Description: "Empathetic replies"},
		{PositionID: svcPos.ID, Name: "Troubleshooting", TargetLevel: 75, Weight: 1.0, Description: "Diagnose issues"},
	})

	dev := model.Employee{EmployeeCode: "EMP005", Name: "Dev", Email: "dev@example.com",
		Phone: "13800000005", DepartmentID: rdDept.ID, PositionID: engPos.ID,
		HireDate: now.AddDate(0, 0, -20), WorkExperience: 1, SkillLevel: model.LevelJunior,
		CurrentStage: 1, TrainingStatus: model.StatusInProgress, Progress: 35}
	salesEmp := model.Employee{EmployeeCode: "EMP006", Name: "Sarah", Email: "sales@example.com",
		Phone: "13800000006", DepartmentID: salesDept.ID, PositionID: salesPos.ID,
		HireDate: now.AddDate(0, 0, -45), WorkExperience: 3, SkillLevel: model.LevelMiddle,
		CurrentStage: 2, TrainingStatus: model.StatusInProgress, Progress: 62}
	svcEmp := model.Employee{EmployeeCode: "EMP007", Name: "Tom", Email: "service@example.com",
		Phone: "13800000007", DepartmentID: svcDept.ID, PositionID: svcPos.ID,
		HireDate: now.AddDate(0, 0, -80), WorkExperience: 4, SkillLevel: model.LevelMiddle,
		CurrentStage: 3, TrainingStatus: model.StatusInProgress, Progress: 82}
	db.Create([]*model.Employee{&dev, &salesEmp, &svcEmp})

	db.Model(&devUser).Update("employee_id", dev.ID)
	db.Model(&salesUser).Update("employee_id", salesEmp.ID)
	db.Model(&serviceUser).Update("employee_id", svcEmp.ID)

	// Additional new hires in the Product department (managed by Manager, the
	// Product Manager), plus one R&D engineer who must NOT be visible to Manager.
	frank := model.Employee{EmployeeCode: "EMP008", Name: "Frank New Hire", Email: "frank@example.com",
		Phone: "13800000008", DepartmentID: productDept.ID, PositionID: aiPM.ID, ManagerID: &manager.ID,
		HireDate: now.AddDate(0, 0, -5), WorkExperience: 0, SkillLevel: model.LevelJunior,
		CurrentStage: 1, TrainingStatus: model.StatusNotStarted, Progress: 0}
	grace := model.Employee{EmployeeCode: "EMP009", Name: "Grace New Hire", Email: "grace@example.com",
		Phone: "13800000009", DepartmentID: productDept.ID, PositionID: aiPM.ID, ManagerID: &manager.ID,
		HireDate: now.AddDate(0, 0, -15), WorkExperience: 1, SkillLevel: model.LevelJunior,
		CurrentStage: 1, TrainingStatus: model.StatusInProgress, Progress: 20}
	ivan := model.Employee{EmployeeCode: "EMP010", Name: "Ivan Engineer", Email: "ivan@example.com",
		Phone: "13800000010", DepartmentID: rdDept.ID, PositionID: engPos.ID,
		HireDate: now.AddDate(0, 0, -30), WorkExperience: 2, SkillLevel: model.LevelMiddle,
		CurrentStage: 1, TrainingStatus: model.StatusInProgress, Progress: 50}
	// Senior employee in the Product department: hired long ago, onboarding
	// already completed. Shows up in Employees but NOT in My New Hires.
	kate := model.Employee{EmployeeCode: "EMP011", Name: "Kate Senior", Email: "kate@example.com",
		Phone: "13800000011", DepartmentID: productDept.ID, PositionID: aiPM.ID, ManagerID: &manager.ID,
		HireDate: now.AddDate(0, 0, -200), WorkExperience: 6, SkillLevel: model.LevelSenior,
		CurrentStage: 3, TrainingStatus: model.StatusCompleted, Progress: 100}
	db.Create([]*model.Employee{&frank, &grace, &ivan, &kate})

	db.Model(&erinUser).Update("employee_id", erin.ID)
	db.Model(&frankUser).Update("employee_id", frank.ID)
	db.Model(&graceUser).Update("employee_id", grace.ID)

	// Real plans, stages, tasks and assessments for the Product-department new
	// hires so AI coaching analysis has genuine data for every one of them.
	erinPlan := model.TrainingPlan{EmployeeID: erin.ID, StartDate: erin.HireDate,
		EndDate: erin.HireDate.AddDate(0, 0, 89), CycleDays: 90, CurrentStage: 1, Progress: 5,
		Status: model.PlanInProgress, CreatedBy: &admin.ID, CreatedSource: model.SourceManual}
	frankPlan := model.TrainingPlan{EmployeeID: frank.ID, StartDate: frank.HireDate,
		EndDate: frank.HireDate.AddDate(0, 0, 89), CycleDays: 90, CurrentStage: 1, Progress: 0,
		Status: model.PlanInProgress, CreatedBy: &admin.ID, CreatedSource: model.SourceManual}
	gracePlan := model.TrainingPlan{EmployeeID: grace.ID, StartDate: grace.HireDate,
		EndDate: grace.HireDate.AddDate(0, 0, 89), CycleDays: 90, CurrentStage: 1, Progress: 20,
		Status: model.PlanInProgress, CreatedBy: &admin.ID, CreatedSource: model.SourceManual}
	db.Create([]*model.TrainingPlan{&erinPlan, &frankPlan, &gracePlan})
	seedEmployeeStagesAndTasks(db, erin, erinPlan, admin.ID, manager.ID, "pm", 65)
	seedEmployeeStagesAndTasks(db, frank, frankPlan, admin.ID, manager.ID, "pm", 58)
	seedEmployeeStagesAndTasks(db, grace, gracePlan, admin.ID, manager.ID, "pm", 88)

	// Stage-1 assessments DERIVED from each new hire's real learning data
	// (graded quiz, learning-record scores, task completion) so the manager's
	// Growth Assessments page shows every department member with genuine
	// numbers. These are dated AFTER the learning happened, not hire-day
	// baselines. Kate has no learning data, so she is honestly left out.
	seedDerivedAssessment(db, erin, erinPlan, manager.ID)
	seedDerivedAssessment(db, frank, frankPlan, manager.ID)
	seedDerivedAssessment(db, grace, gracePlan, manager.ID)

	devPlan := model.TrainingPlan{EmployeeID: dev.ID, StartDate: dev.HireDate,
		EndDate: dev.HireDate.AddDate(0, 0, 89), CycleDays: 90, CurrentStage: 1, Progress: 35,
		Status: model.PlanInProgress, CreatedBy: &admin.ID, CreatedSource: model.SourceManual}
	salesPlan := model.TrainingPlan{EmployeeID: salesEmp.ID, StartDate: salesEmp.HireDate,
		EndDate: salesEmp.HireDate.AddDate(0, 0, 89), CycleDays: 90, CurrentStage: 2, Progress: 62,
		Status: model.PlanInProgress, CreatedBy: &admin.ID, CreatedSource: model.SourceManual}
	svcPlan := model.TrainingPlan{EmployeeID: svcEmp.ID, StartDate: svcEmp.HireDate,
		EndDate: svcEmp.HireDate.AddDate(0, 0, 89), CycleDays: 90, CurrentStage: 3, Progress: 82,
		Status: model.PlanInProgress, CreatedBy: &admin.ID, CreatedSource: model.SourceManual}
	db.Create([]*model.TrainingPlan{&devPlan, &salesPlan, &svcPlan})

	// Stages + tasks for each new hire (concise but complete). Quizzes stay
	// pending so employees can experience the exam flow themselves.
	seedEmployeeStagesAndTasks(db, dev, devPlan, admin.ID, manager.ID, "dev", 0)
	seedEmployeeStagesAndTasks(db, salesEmp, salesPlan, admin.ID, manager.ID, "sales", 0)
	seedEmployeeStagesAndTasks(db, svcEmp, svcPlan, admin.ID, manager.ID, "support", 0)

	// Weekly reports + monthly summaries for every demo employee.
	seedReports(db, bob, &bobPlan, "pm")
	seedReports(db, carol, &carolPlan, "pm")
	seedReports(db, dave, &davePlan, "pm")
	seedReports(db, erin, nil, "pm")
	seedReports(db, dev, &devPlan, "dev")
	seedReports(db, salesEmp, &salesPlan, "sales")
	seedReports(db, svcEmp, &svcPlan, "support")

	return nil
}

func float64p(v float64) *float64  { return &v }
func timeP(t time.Time) *time.Time { return &t }

// seedReports creates weekly progress reports and a monthly summary for an
// employee so the manager/admin/employee views have realistic data.
func seedReports(db *gorm.DB, emp model.Employee, plan *model.TrainingPlan, role string) {
	var planID *uint
	if plan != nil {
		planID = &plan.ID
	}

	// Role-specific content tone.
	var focus, tool, obstacle string
	switch role {
	case "dev":
		focus, tool, obstacle = "API integration and code quality", "the shared dev environment", "occasional flaky tests"
	case "sales":
		focus, tool, obstacle = "customer discovery and pitch practice", "the CRM", "a slow approval cycle"
	case "support":
		focus, tool, obstacle = "ticket triage and customer empathy", "the support console", "escalation hand-offs"
	default:
		focus, tool, obstacle = "PRD writing and AI fundamentals", "the product workspace", "data analysis"
	}

	weekly := []model.TrainingReport{
		{
			EmployeeID: emp.ID, PlanID: planID, ReportType: model.ReportWeekly, Period: "Week 1",
			Progress: 12, TaskCompletion: 70, LearningHours: 4, CompositeScore: float64p(72),
			Strengths: "Ramped quickly on " + focus + ".", Weaknesses: "Still getting comfortable with " + tool + ".",
			Risks: "None.", Suggestions: "Keep the daily stand-up cadence.",
			ManagerEval: "Good start.",
			Content:     "Week 1 focus was " + focus + ". Completed the onboarding checklist and shadowed a senior teammate. Main blocker: " + obstacle + ". Next week: deliver first independent task.",
			AIGenerated: false,
		},
		{
			EmployeeID: emp.ID, PlanID: planID, ReportType: model.ReportWeekly, Period: "Week 2",
			Progress: 28, TaskCompletion: 82, LearningHours: 6, CompositeScore: float64p(80),
			Strengths: "Consistent progress and asks good questions.", Weaknesses: "Needs to speed up hand-written docs.",
			Risks: "None.", Suggestions: "Adopt the shared template for write-ups.",
			ManagerEval: "On track.",
			Content:     "Week 2 focused on " + focus + ". Delivered a small task end-to-end and got quick feedback. Practicing with " + tool + " is going well. Blocked briefly by " + obstacle + ", resolved same day.",
			AIGenerated: false,
		},
		{
			EmployeeID: emp.ID, PlanID: planID, ReportType: model.ReportWeekly, Period: "Week 3",
			Progress: 44, TaskCompletion: 90, LearningHours: 5, CompositeScore: float64p(84),
			Strengths: "Becoming more independent.", Weaknesses: "Sometimes over-scopes a task.",
			Risks: "Low.", Suggestions: "Break work into smaller reviewable chunks.",
			ManagerEval: "Expecting strong output.",
			Content:     "Week 3 focus was " + focus + ". Worked more autonomously and contributed to a team discussion. The main improvement is scoping; will tighten before the month review.",
			AIGenerated: false,
		},
	}
	monthly := model.TrainingReport{
		EmployeeID: emp.ID, PlanID: planID, ReportType: model.ReportMonthly, Period: "Month 1",
		Progress: 55, TaskCompletion: 86, LearningHours: 21, CompositeScore: float64p(82),
		Strengths:   "Ramped fast, communicates well, and delivers on time.",
		Weaknesses:  "Still building depth in " + focus + " and writing.",
		Risks:       "Low risk; monitor time management during busy weeks.",
		Suggestions: "Schedule one deep-work block daily and review the PRD checklist.",
		ManagerEval: "Solid first month; on track for the 30/60/90 plan.",
		Content:     "Month 1 summary:" + emp.Name + " completed onboarding and the first stage of the 30/60/90 plan. Overall progress 55%, task completion 86%, ~21 learning hours. Main achievements: " + focus + ". Areas to grow: depth and documentation speed. Recommended next steps: independent task deliveries and weekly reflection.",
		AIGenerated: false,
	}

	db.Create(append(weekly, monthly))
}

// seedEmployeeStagesAndTasks seeds stages, tasks and learning records for a
// demo employee, with role-specific sample tasks. No entry/baseline
// assessment is fabricated: AI coaching judges purely from real learning
// data. When quizScore > 0 the stage-1 quiz is seeded as completed with that
// score (a real exam result the employee supposedly achieved).
func seedEmployeeStagesAndTasks(db *gorm.DB, emp model.Employee, plan model.TrainingPlan, adminID, managerID uint, role string, quizScore float64) {
	// Create 3 standard stages with role-specific objectives/tasks.
	titlesByStage := [][]string{
		titlesFor(role, 1),
		titlesFor(role, 2),
		titlesFor(role, 3),
	}
	var stages []model.TrainingStage
	statuses := []model.StageStatus{model.StageInProgress, model.StageNotStarted, model.StageNotStarted}
	progresses := []float64{emp.Progress * 0.6, 0, 0}
	if emp.CurrentStage == 2 {
		statuses[0] = model.StageCompleted
		statuses[1] = model.StageInProgress
		progresses[0] = 100
		progresses[1] = (emp.Progress - 30) * 1.2
	}
	if emp.CurrentStage == 3 {
		statuses[0] = model.StageCompleted
		statuses[1] = model.StageCompleted
		statuses[2] = model.StageInProgress
		progresses[0] = 100
		progresses[1] = 100
		progresses[2] = (emp.Progress - 60) * 1.8
	}
	for n := 1; n <= 3; n++ {
		st := model.TrainingStage{
			PlanID: plan.ID, StageNumber: n,
			Name:       stageName(n),
			Objectives: stageObjectives(n, role),
			StartDate:  emp.HireDate.AddDate(0, 0, (n-1)*30),
			EndDate:    emp.HireDate.AddDate(0, 0, n*30-1),
			Status:     statuses[n-1], Progress: clamp(progresses[n-1]),
		}
		db.Create(&st)
		stages = append(stages, st)
	}

	// Seed tasks. Mark the first N complete based on stage progress.
	stageDone := make([]int, 3)
	stageTotal := make([]int, 3)
	for n := 0; n < 3; n++ {
		titles := titlesByStage[n]
		for i, t := range titles {
			status := model.TaskPending
			if n == 0 && i < 2 {
				status = model.TaskCompleted
			}
			if n == 1 && emp.CurrentStage >= 2 && i < 1 {
				status = model.TaskCompleted
			}
			if n == 2 && emp.CurrentStage == 3 {
				status = model.TaskCompleted
			}
			task := model.LearningTask{
				EmployeeID: emp.ID, PlanID: plan.ID, StageID: stages[n].ID,
				Title: t, TaskType: model.TaskLearning, StartDate: emp.HireDate.AddDate(0, 0, (n)*30+i),
				DueDate: emp.HireDate.AddDate(0, 0, n*30+5+i), EstimateHours: 2.5,
				Priority: model.PriorityMedium, Status: status, CreatedSource: model.SourceManual,
			}
			db.Create(&task)
			stageTotal[n]++
			if status == model.TaskCompleted {
				stageDone[n]++
				now := time.Now().UTC()
				db.Create(&model.LearningRecord{
					EmployeeID: emp.ID, TaskID: task.ID, Content: "Completed: " + t,
					StartTime: now.Add(-90 * time.Minute), EndTime: timeP(now.Add(-20 * time.Minute)),
					DurationMin: 70, Status: model.TaskCompleted, TestScore: float64p(80 + float64((i*7+n*3)%20)),
					SelfEval: "Done", Outcome: "Task finished",
				})
			}
		}
	}

	// A role-specific quiz (test) task in stage 1 so employees experience the exam flow.
	quizTask := model.LearningTask{
		EmployeeID: emp.ID, PlanID: plan.ID, StageID: stages[0].ID,
		Title: roleQuizTitle(role), TaskType: model.TaskTest,
		StartDate: emp.HireDate, DueDate: emp.HireDate.AddDate(0, 0, 7),
		EstimateHours: 1, Priority: model.PriorityMedium, Status: model.TaskPending,
		CreatedSource: model.SourceManual, Quiz: roleQuizQuestions(role),
	}
	db.Create(&quizTask)
	stageTotal[0]++
	if quizScore > 0 {
		// The employee really took the quiz: record the graded result exactly
		// the way CompleteTask would (task score + learning record).
		stageDone[0]++
		now := time.Now().UTC()
		db.Model(&quizTask).Updates(map[string]interface{}{
			"status": model.TaskCompleted, "score": quizScore, "outcome": "Quiz completed",
		})
		db.Create(&model.LearningRecord{
			EmployeeID: emp.ID, TaskID: quizTask.ID, Content: "Completed: " + quizTask.Title,
			StartTime: now.Add(-45 * time.Minute), EndTime: timeP(now.Add(-10 * time.Minute)),
			DurationMin: 35, Status: model.TaskCompleted, TestScore: float64p(quizScore),
			SelfEval: "Quiz submitted", Outcome: "Quiz completed",
		})
	}

	// A daily quiz (always pending) so every employee has fresh questions to
	// take from the "Today's Quiz" page at any time.
	dailyQuiz := model.LearningTask{
		EmployeeID: emp.ID, PlanID: plan.ID, StageID: stages[0].ID,
		Title: dailyQuizTitle(role), TaskType: model.TaskTest,
		StartDate: emp.HireDate, DueDate: emp.HireDate.AddDate(0, 0, 14),
		EstimateHours: 0.5, Priority: model.PriorityLow, Status: model.TaskPending,
		CreatedSource: model.SourceManual, Quiz: dailyQuizQuestions(role),
	}
	db.Create(&dailyQuiz)
	stageTotal[0]++

	// Sync every progress field with the real completed-task counts so the
	// numbers are always consistent; nothing is fabricated.
	done, total := 0, 0
	for n := 0; n < 3; n++ {
		done += stageDone[n]
		total += stageTotal[n]
		if stageTotal[n] > 0 {
			db.Model(&stages[n]).Update("progress", clamp(float64(stageDone[n])/float64(stageTotal[n])*100))
		}
	}
	pct := 0.0
	if total > 0 {
		pct = float64(done) / float64(total) * 100
	}
	db.Model(&plan).Update("progress", pct)
	empUpdates := map[string]interface{}{"progress": pct}
	if pct > 0 && emp.TrainingStatus == model.StatusNotStarted {
		empUpdates["training_status"] = model.StatusInProgress
	}
	db.Model(&emp).Updates(empUpdates)
}

// dailyQuizTitle returns the title of the always-available daily quiz.
func dailyQuizTitle(role string) string {
	switch role {
	case "dev":
		return "Today's quiz: Code quality"
	case "sales":
		return "Today's quiz: Customer discovery"
	case "support":
		return "Today's quiz: Ticket handling"
	default:
		return "Today's quiz: Product sense"
	}
}

// dailyQuizQuestions returns a fresh set of role-specific questions for the
// daily quiz, distinct from the stage-1 quiz. Answer is 0-based.
func dailyQuizQuestions(role string) []model.QuizQuestion {
	switch role {
	case "dev":
		return []model.QuizQuestion{
			{Prompt: "What is a code review mainly for?", Options: []string{"Assigning blame", "Slowing down releases", "Catching issues and sharing knowledge", "Formatting code"}, Answer: 2},
			{Prompt: "When should you write a unit test?", Options: []string{"Before or alongside the code it covers", "Only after a production bug", "Once a month", "Never for small functions"}, Answer: 0},
			{Prompt: "What does 'refactoring' mean?", Options: []string{"Adding new features", "Improving code structure without changing behavior", "Deleting dead services", "Rewriting in another language"}, Answer: 1},
		}
	case "sales":
		return []model.QuizQuestion{
			{Prompt: "What is the goal of a discovery call?", Options: []string{"Close the deal fast", "Understand the customer's situation and needs", "Demo every feature", "Discuss discounts"}, Answer: 1},
			{Prompt: "What should you do when a prospect objects to price?", Options: []string{"Explore the value and the underlying concern", "Lower the price immediately", "End the call", "Argue with data"}, Answer: 0},
			{Prompt: "What belongs in the CRM after a call?", Options: []string{"Only the deal size", "Personal opinions about the prospect", "Key needs, next steps and timeline", "Nothing until closing"}, Answer: 2},
		}
	case "support":
		return []model.QuizQuestion{
			{Prompt: "First step when a ticket arrives?", Options: []string{"Understand and reproduce the issue", "Escalate right away", "Send a template reply", "Wait for more info"}, Answer: 0},
			{Prompt: "When should you escalate a ticket?", Options: []string{"After a week", "Never", "When it is beyond your scope or at SLA risk", "Only if the customer asks"}, Answer: 2},
			{Prompt: "What makes a good knowledge-base article?", Options: []string{"Long explanations", "Clear steps that solve a recurring issue", "Marketing language", "Screenshots only"}, Answer: 1},
		}
	default:
		return []model.QuizQuestion{
			{Prompt: "What is the main purpose of a PRD?", Options: []string{"Replacing team meetings", "Aligning the team on what to build and why", "Tracking bugs", "Assigning tasks"}, Answer: 1},
			{Prompt: "Which metric best shows an AI feature works?", Options: []string{"Lines of prompts written", "Number of demos given", "The success metric defined before launch", "Model size"}, Answer: 2},
			{Prompt: "What should you do first when a requirement is ambiguous?", Options: []string{"Clarify it with stakeholders", "Guess and build", "Defer it forever", "Escalate to HR"}, Answer: 0},
		}
	}
}

// seedDerivedAssessment creates a stage-1 assessment whose capability scores
// are computed from the employee's real learning evidence: the graded
// stage-1 quiz (AI Knowledge), the average score across learning records
// (Product Fundamentals) and the real task completion rate (Task Execution).
// No numbers are invented; employees without such evidence get nothing.
func seedDerivedAssessment(db *gorm.DB, emp model.Employee, plan model.TrainingPlan, managerID uint) {
	var quiz model.LearningTask
	if err := db.Where("employee_id = ? AND task_type = ? AND status = ? AND score IS NOT NULL",
		emp.ID, model.TaskTest, model.TaskCompleted).Order("id").First(&quiz).Error; err != nil || quiz.Score == nil {
		return // no graded quiz: nothing honest to derive
	}
	var recAvg float64
	db.Model(&model.LearningRecord{}).Where("employee_id = ? AND test_score IS NOT NULL", emp.ID).
		Select("COALESCE(AVG(test_score),0)").Scan(&recAvg)
	var freshPlan model.TrainingPlan
	if err := db.First(&freshPlan, plan.ID).Error; err != nil {
		freshPlan = plan
	}

	scores := []model.AssessmentScore{
		{Capability: "AI Knowledge", Score: *quiz.Score, TargetLevel: 80, Weight: 1.0},
		{Capability: "Product Fundamentals", Score: recAvg, TargetLevel: 80, Weight: 1.0},
		{Capability: "Task Execution", Score: freshPlan.Progress, TargetLevel: 80, Weight: 0.8},
	}
	goalsMet := true
	var sum, wsum float64
	best, worst := scores[0], scores[0]
	for _, s := range scores {
		if s.Score < s.TargetLevel {
			goalsMet = false
		}
		if s.Score > best.Score {
			best = s
		}
		if s.Score < worst.Score {
			worst = s
		}
		sum += s.Score * s.Weight
		wsum += s.Weight
	}
	assess := model.Assessment{
		EmployeeID: emp.ID, PlanID: plan.ID, StageNumber: 1,
		AssessDate: time.Now().UTC(), AssessorID: &managerID,
		Strengths:   "Strongest: " + best.Capability,
		Weaknesses:  "Needs focus: " + worst.Capability,
		Improvement: "Focus on " + worst.Capability + " in the next stage.",
		GoalsMet:    goalsMet, Status: model.AssessmentDone,
		AssessmentSource: model.AssessmentSourceManual,
	}
	db.Create(&assess)
	for i := range scores {
		scores[i].AssessmentID = assess.ID
		db.Create(&scores[i])
	}
	if wsum > 0 {
		db.Model(&assess).Update("composite_score", sum/wsum)
	}
}

// roleQuizTitle returns the title of the stage-1 quiz task for a role.
func roleQuizTitle(role string) string {
	switch role {
	case "dev":
		return "Dev environment quiz"
	case "sales":
		return "Sales fundamentals quiz"
	case "support":
		return "Support fundamentals quiz"
	default:
		return "AI fundamentals quiz"
	}
}

// roleQuizQuestions returns the role-specific multiple-choice quiz for a role.
// Answer is 0-based (index of the correct option).
func roleQuizQuestions(role string) []model.QuizQuestion {
	switch role {
	case "dev":
		return []model.QuizQuestion{
			{Prompt: "Which tool is used to build and run Go server code?", Options: []string{"npm", "go build", "pip install", "cargo run"}, Answer: 1},
			{Prompt: "What does Git help you do?", Options: []string{"Design wireframes", "Version-control source code", "Schedule meetings", "Write tests"}, Answer: 1},
			{Prompt: "What is a unit test used for?", Options: []string{"Deploying to production", "Verifying a single piece of code", "Drawing diagrams", "Sending emails"}, Answer: 1},
		}
	case "sales":
		return []model.QuizQuestion{
			{Prompt: "What is the first step of a discovery call?", Options: []string{"Close the deal", "Listen to the customer's needs", "Send a proposal", "Pitch pricing"}, Answer: 1},
			{Prompt: "What does CRM stand for?", Options: []string{"Customer Relationship Management", "Client Risk Monitor", "Credit Reporting Module", "Content Review Model"}, Answer: 0},
			{Prompt: "What is a good indicator of a qualified lead?", Options: []string{"Has budget and need", "Follows social media", "Requests a demo", "Is a competitor"}, Answer: 0},
		}
	case "support":
		return []model.QuizQuestion{
			{Prompt: "What is the most important part of a support reply?", Options: []string{"Being brief at all costs", "Understanding and addressing the issue", "Ending with a discount", "Copy-pasting a template"}, Answer: 1},
			{Prompt: "What does CSAT measure?", Options: []string{"Customer satisfaction", "Code quality", "Server load", "Revenue growth"}, Answer: 0},
			{Prompt: "What should you do when you cannot resolve an issue?", Options: []string{"Close the ticket", "Escalate to the right team", "Ignore the customer", "Mark it spam"}, Answer: 1},
		}
	default:
		return []model.QuizQuestion{
			{Prompt: "What is an LLM primarily good at?", Options: []string{"Image editing", "Text generation and reasoning", "Database queries", "Hardware acceleration"}, Answer: 1},
			{Prompt: "What does 'grounding' mean in AI product work?", Options: []string{"Connecting to the ground network", "Anchoring model output to trusted data", "Training with no data", "Deploying to production"}, Answer: 1},
			{Prompt: "Which is a core risk of LLM products?", Options: []string{"Faster compile times", "Hallucination", "Lower server costs", "No licensing needed"}, Answer: 1},
		}
	}
}

func titlesFor(role string, stage int) []string {
	m := map[string][][]string{
		"dev": {
			{"Set up dev environment", "Read code style guide", "Complete Git basics", "Run first build"},
			{"Write a unit test", "Implement a small feature", "Do a code review", "Ship to staging"},
			{"Deliver an independent module", "Debug a production issue", "Final code quality review"},
		},
		"sales": {
			{"Product knowledge training", "Learn CRM basics", "Shadow a senior rep", "Craft a pitch"},
			{"Run discovery calls", "Qualify 5 leads", "Prepare a proposal", "Close a demo"},
			{"Close a deal independently", "Pipeline review", "Final sales assessment"},
		},
		"support": {
			{"Product onboarding", "Customer service policy", "Ticket system basics", "Handle 3 tickets"},
			{"Resolve common issues", "Customer escalation drill", "Write a knowledge article", "CSAT review"},
			{"Resolve tickets independently", "Root-cause a complex case", "Final support assessment"},
		},
		"pm": {
			{"Product team onboarding", "Read the PRD template", "Learn the roadmap cadence", "Shadow a senior PM"},
			{"Write a PRD draft", "Run a requirements review", "Analyze feature metrics", "Coordinate an AI feature eval"},
			{"Own a feature end-to-end", "Lead a roadmap planning session", "Final PM capability review"},
		},
	}
	if stage < 1 || stage > 3 {
		return []string{"Complete a stage learning task"}
	}
	return m[role][stage-1]
}

func stageName(n int) string {
	names := []string{"Onboarding & Fundamentals", "Capability Building", "Independent Competence"}
	return names[n-1]
}

func stageObjectives(n int, role string) string {
	m := map[int]string{
		1: "Familiarize with the " + role + " role, company and tooling; complete induction.",
		2: "Master core " + role + " knowledge and SOPs; complete real tasks.",
		3: "Work independently, solve problems and pass the final assessment.",
	}
	return m[n]
}

func capabilitiesFor(role string) []struct {
	name                  string
	score, target, weight float64
} {
	switch role {
	case "dev":
		return []struct {
			name                  string
			score, target, weight float64
		}{
			{"Programming", 82, 85, 1.2}, {"Debugging", 70, 75, 1.0}, {"Code Review", 76, 70, 0.8},
		}
	case "sales":
		return []struct {
			name                  string
			score, target, weight float64
		}{
			{"Lead Gen", 78, 80, 1.0}, {"Negotiation", 72, 75, 1.1}, {"CRM", 88, 65, 0.7},
		}
	case "pm":
		return []struct {
			name                  string
			score, target, weight float64
		}{
			{"Product Fundamentals", 78, 80, 1.0}, {"Data Analysis", 68, 75, 1.0}, {"AI Knowledge", 85, 80, 1.0}, {"Communication", 82, 80, 0.8},
		}
	default:
		return []struct {
			name                  string
			score, target, weight float64
		}{
			{"Product Knowledge", 84, 80, 1.0}, {"Communication", 90, 85, 1.2}, {"Troubleshooting", 74, 75, 1.0},
		}
	}
}

func clamp(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}
