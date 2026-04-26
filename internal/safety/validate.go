package safety

import (
	"fmt"
	"regexp"
	"time"
)

const (
	MaxTaskTitleLength   = 256
	MaxTaskContentLength = 16384
	MaxProjectNameLength = 64
	MaxTagLength         = 64
	MaxBatchSize         = 25
	MaxProfileNameLength = 32
)

var (
	hexIDPattern    = regexp.MustCompile(`^[a-f0-9]{24}$`)
	profilePattern  = regexp.MustCompile(`^[a-z0-9_-]{1,32}$`)
	validPriorities = map[int]bool{0: true, 1: true, 3: true, 5: true}
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation: %s: %s", e.Field, e.Message)
}

func ValidateTaskTitle(title string) *ValidationError {
	if title == "" {
		return &ValidationError{Field: "title", Message: "must not be empty"}
	}
	if len(title) > MaxTaskTitleLength {
		return &ValidationError{Field: "title", Message: fmt.Sprintf("must not exceed %d characters", MaxTaskTitleLength)}
	}
	return nil
}

func ValidateTaskContent(content string) *ValidationError {
	if len(content) > MaxTaskContentLength {
		return &ValidationError{Field: "content", Message: fmt.Sprintf("must not exceed %d characters", MaxTaskContentLength)}
	}
	return nil
}

func ValidateProjectName(name string) *ValidationError {
	if name == "" {
		return &ValidationError{Field: "project_name", Message: "must not be empty"}
	}
	if len(name) > MaxProjectNameLength {
		return &ValidationError{Field: "project_name", Message: fmt.Sprintf("must not exceed %d characters", MaxProjectNameLength)}
	}
	return nil
}

func ValidateProjectID(id string) *ValidationError {
	if !hexIDPattern.MatchString(id) {
		return &ValidationError{Field: "project_id", Message: "must be a 24-character hex string"}
	}
	return nil
}

func ValidateTaskID(id string) *ValidationError {
	if !hexIDPattern.MatchString(id) {
		return &ValidationError{Field: "task_id", Message: "must be a 24-character hex string"}
	}
	return nil
}

func ValidateProfileName(name string) *ValidationError {
	if !profilePattern.MatchString(name) {
		return &ValidationError{Field: "profile_name", Message: "must be 1-32 lowercase alphanumeric, underscore, or hyphen characters"}
	}
	return nil
}

func ValidateDate(date string) *ValidationError {
	if date == "" {
		return nil
	}
	if _, err := time.Parse(time.RFC3339, date); err != nil {
		return &ValidationError{Field: "date", Message: "must be a valid RFC3339 date"}
	}
	return nil
}

func ValidatePriority(priority int) *ValidationError {
	if !validPriorities[priority] {
		return &ValidationError{Field: "priority", Message: "must be 0, 1, 3, or 5"}
	}
	return nil
}

func ValidateBatchSize(size int) *ValidationError {
	if size <= 0 || size > MaxBatchSize {
		return &ValidationError{Field: "batch_size", Message: fmt.Sprintf("must be between 1 and %d", MaxBatchSize)}
	}
	return nil
}

func validateTaskFields(content, desc, startDate, dueDate string, priority int) *ValidationError {
	if content != "" {
		if err := ValidateTaskContent(content); err != nil {
			return err
		}
	}
	if len(desc) > MaxTaskContentLength {
		return &ValidationError{Field: "desc", Message: fmt.Sprintf("must not exceed %d characters", MaxTaskContentLength)}
	}
	if err := ValidateDate(startDate); err != nil {
		err.Field = "start_date"
		return err
	}
	if err := ValidateDate(dueDate); err != nil {
		err.Field = "due_date"
		return err
	}
	return ValidatePriority(priority)
}

func ValidateCreateTask(title, projectID, content, desc, startDate, dueDate string, priority int) *ValidationError {
	if err := ValidateTaskTitle(title); err != nil {
		return err
	}
	if projectID != "" {
		if err := ValidateProjectID(projectID); err != nil {
			return err
		}
	}
	return validateTaskFields(content, desc, startDate, dueDate, priority)
}

func ValidateUpdateTask(taskID, projectID, title, content, desc, startDate, dueDate string, priority int) *ValidationError {
	if err := ValidateTaskID(taskID); err != nil {
		return err
	}
	if err := ValidateProjectID(projectID); err != nil {
		return err
	}
	if title != "" && len(title) > MaxTaskTitleLength {
		return &ValidationError{Field: "title", Message: fmt.Sprintf("must not exceed %d characters", MaxTaskTitleLength)}
	}
	return validateTaskFields(content, desc, startDate, dueDate, priority)
}

func ValidateCreateProject(name string) *ValidationError {
	return ValidateProjectName(name)
}
