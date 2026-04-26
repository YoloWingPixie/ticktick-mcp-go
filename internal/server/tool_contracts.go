package server

import "github.com/zachshepherd/ticktick-mcp-go/internal/ticktick"

type GetProjectsInput struct{}

type GetProjectInput struct {
	ProjectID string `json:"project_id" jsonschema:"The project ID (24-char hex)"`
}

type GetProjectWithDataInput struct {
	ProjectID string `json:"project_id" jsonschema:"The project ID (24-char hex)"`
}

type GetTaskInput struct {
	ProjectID string `json:"project_id" jsonschema:"The project ID (24-char hex)"`
	TaskID    string `json:"task_id" jsonschema:"The task ID (24-char hex)"`
}

type GetAllTasksInput struct{}
type GetTasksDueTodayInput struct{}
type GetTasksDueThisWeekInput struct{}
type GetOverdueTasksInput struct{}

type GetTasksByPriorityInput struct {
	Priority int `json:"priority" jsonschema:"Priority level: 0=None 1=Low 3=Medium 5=High"`
}

type SearchTasksInput struct {
	Query string `json:"query" jsonschema:"Search string to match against task titles and content"`
}

type GetEngagedTasksInput struct{}
type GetNextTasksInput struct{}

type CreateTaskInput struct {
	Title      string   `json:"title" jsonschema:"Task title (max 256 chars)"`
	ProjectID  string   `json:"project_id,omitempty" jsonschema:"Project ID to create the task in. If omitted uses the inbox."`
	Content    string   `json:"content,omitempty" jsonschema:"Task notes/content (max 16384 chars)"`
	Desc       string   `json:"desc,omitempty" jsonschema:"Description of checklist"`
	IsAllDay   bool     `json:"is_all_day,omitempty" jsonschema:"Whether this is an all-day task"`
	StartDate  string   `json:"start_date,omitempty" jsonschema:"Start date in RFC3339 format"`
	DueDate    string   `json:"due_date,omitempty" jsonschema:"Due date in RFC3339 format"`
	TimeZone   string   `json:"time_zone,omitempty" jsonschema:"Timezone e.g. America/Los_Angeles"`
	Priority   int      `json:"priority,omitempty" jsonschema:"Priority: 0=None 1=Low 3=Medium 5=High"`
	Reminders  []string `json:"reminders,omitempty" jsonschema:"Reminder triggers e.g. TRIGGER:PT0S"`
	RepeatFlag string   `json:"repeat_flag,omitempty" jsonschema:"iCalendar RRULE e.g. RRULE:FREQ=DAILY;INTERVAL=1"`
}

func (input CreateTaskInput) toTask() *ticktick.Task {
	return &ticktick.Task{
		Title:      input.Title,
		ProjectID:  input.ProjectID,
		Content:    input.Content,
		Desc:       input.Desc,
		IsAllDay:   input.IsAllDay,
		StartDate:  input.StartDate,
		DueDate:    input.DueDate,
		TimeZone:   input.TimeZone,
		Priority:   input.Priority,
		Reminders:  input.Reminders,
		RepeatFlag: input.RepeatFlag,
	}
}

type UpdateTaskInput struct {
	TaskID     string   `json:"task_id" jsonschema:"The task ID to update (24-char hex)"`
	ProjectID  string   `json:"project_id" jsonschema:"The project ID the task belongs to (24-char hex)"`
	Title      string   `json:"title,omitempty" jsonschema:"New task title"`
	Content    string   `json:"content,omitempty" jsonschema:"New task content"`
	Desc       string   `json:"desc,omitempty" jsonschema:"New checklist description"`
	IsAllDay   *bool    `json:"is_all_day,omitempty" jsonschema:"Whether this is an all-day task"`
	StartDate  string   `json:"start_date,omitempty" jsonschema:"New start date in RFC3339 format"`
	DueDate    string   `json:"due_date,omitempty" jsonschema:"New due date in RFC3339 format"`
	TimeZone   string   `json:"time_zone,omitempty" jsonschema:"Timezone"`
	Priority   *int     `json:"priority,omitempty" jsonschema:"Priority: 0=None 1=Low 3=Medium 5=High"`
	Reminders  []string `json:"reminders,omitempty" jsonschema:"Reminder triggers"`
	RepeatFlag string   `json:"repeat_flag,omitempty" jsonschema:"iCalendar RRULE"`
}

type CompleteTaskInput struct {
	ProjectID string `json:"project_id" jsonschema:"The project ID (24-char hex)"`
	TaskID    string `json:"task_id" jsonschema:"The task ID (24-char hex)"`
}

type MoveTaskInput struct {
	TaskID      string `json:"task_id" jsonschema:"The task ID to move (24-char hex)"`
	FromProject string `json:"from_project" jsonschema:"Source project ID (24-char hex)"`
	ToProject   string `json:"to_project" jsonschema:"Destination project ID (24-char hex)"`
}

type CreateProjectInput struct {
	Name     string `json:"name" jsonschema:"Project name (max 64 chars)"`
	Color    string `json:"color,omitempty" jsonschema:"Hex color e.g. #F18181"`
	ViewMode string `json:"view_mode,omitempty" jsonschema:"View mode: list kanban or timeline"`
	Kind     string `json:"kind,omitempty" jsonschema:"Project kind: TASK or NOTE"`
}

type UpdateProjectInput struct {
	ProjectID string `json:"project_id" jsonschema:"The project ID to update (24-char hex)"`
	Name      string `json:"name,omitempty" jsonschema:"New project name"`
	Color     string `json:"color,omitempty" jsonschema:"New hex color"`
	ViewMode  string `json:"view_mode,omitempty" jsonschema:"New view mode"`
}

type BatchCreateTasksInput struct {
	Tasks []CreateTaskInput `json:"tasks" jsonschema:"List of tasks to create (max 25)"`
}

type BatchCompleteTasksInput struct {
	Tasks []CompleteTaskInput `json:"tasks" jsonschema:"List of tasks to complete (max 25)"`
}

type DeleteTaskInput struct {
	ProjectID string `json:"project_id" jsonschema:"The project ID (24-char hex)"`
	TaskID    string `json:"task_id" jsonschema:"The task ID (24-char hex)"`
	Confirmed bool   `json:"confirmed" jsonschema:"Must be true to confirm deletion. This permanently deletes the task."`
}

type DeleteProjectInput struct {
	ProjectID string `json:"project_id" jsonschema:"The project ID (24-char hex)"`
	Confirmed bool   `json:"confirmed" jsonschema:"Must be true to confirm deletion. This permanently deletes the project and all its tasks."`
}

type BatchResult struct {
	Requested int          `json:"requested"`
	Succeeded int          `json:"succeeded"`
	Failed    int          `json:"failed"`
	Results   []ItemResult `json:"results"`
}

type ItemResult struct {
	Index  int    `json:"index"`
	Status string `json:"status"`
	ID     string `json:"id,omitempty"`
	Error  string `json:"error,omitempty"`
}
