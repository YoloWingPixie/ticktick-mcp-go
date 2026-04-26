package ticktick

import "fmt"

type Task struct {
	ID            string          `json:"id,omitempty"`
	ProjectID     string          `json:"projectId,omitempty"`
	Title         string          `json:"title,omitempty"`
	Content       string          `json:"content,omitempty"`
	Desc          string          `json:"desc,omitempty"`
	IsAllDay      bool            `json:"isAllDay,omitempty"`
	StartDate     string          `json:"startDate,omitempty"`
	DueDate       string          `json:"dueDate,omitempty"`
	TimeZone      string          `json:"timeZone,omitempty"`
	Reminders     []string        `json:"reminders,omitempty"`
	RepeatFlag    string          `json:"repeatFlag,omitempty"`
	Priority      int             `json:"priority,omitempty"`
	Status        int             `json:"status,omitempty"`
	CompletedTime string          `json:"completedTime,omitempty"`
	SortOrder     int64           `json:"sortOrder,omitempty"`
	Items         []ChecklistItem `json:"items,omitempty"`
}

type ChecklistItem struct {
	ID            string `json:"id,omitempty"`
	Title         string `json:"title,omitempty"`
	Status        int    `json:"status,omitempty"`
	CompletedTime string `json:"completedTime,omitempty"`
	IsAllDay      bool   `json:"isAllDay,omitempty"`
	SortOrder     int64  `json:"sortOrder,omitempty"`
	StartDate     string `json:"startDate,omitempty"`
	TimeZone      string `json:"timeZone,omitempty"`
}

type Project struct {
	ID         string `json:"id,omitempty"`
	Name       string `json:"name,omitempty"`
	Color      string `json:"color,omitempty"`
	SortOrder  int64  `json:"sortOrder,omitempty"`
	Closed     bool   `json:"closed,omitempty"`
	GroupID    string `json:"groupId,omitempty"`
	ViewMode   string `json:"viewMode,omitempty"`
	Permission string `json:"permission,omitempty"`
	Kind       string `json:"kind,omitempty"`
}

type Column struct {
	ID        string `json:"id,omitempty"`
	ProjectID string `json:"projectId,omitempty"`
	Name      string `json:"name,omitempty"`
	SortOrder int64  `json:"sortOrder,omitempty"`
}

type ProjectData struct {
	Project Project  `json:"project"`
	Tasks   []Task   `json:"tasks"`
	Columns []Column `json:"columns"`
}

type APIError struct {
	StatusCode int    `json:"-"`
	RetryAfter string `json:"-"`
	Timestamp  string `json:"timestamp"`
	Status     int    `json:"status"`
	Err        string `json:"error"`
	Path       string `json:"path"`
	Message    string `json:"message,omitempty"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("ticktick: %d %s (%s)", e.StatusCode, e.Err, e.Path)
}

type OperationSafety int

const (
	SafeRead OperationSafety = iota
	IdempotentWrite
	NonIdempotentWrite
)

const (
	TaskStatusNormal    = 0
	TaskStatusCompleted = 2

	ChecklistStatusNormal    = 0
	ChecklistStatusCompleted = 1

	PriorityNone   = 0
	PriorityLow    = 1
	PriorityMedium = 3
	PriorityHigh   = 5
)
