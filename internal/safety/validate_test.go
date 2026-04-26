package safety

import (
	"strings"
	"testing"
)

func TestValidateTaskTitle(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		wantErr bool
	}{
		{"empty", "", true},
		{"valid", "Buy groceries", false},
		{"at max length", strings.Repeat("a", MaxTaskTitleLength), false},
		{"over max length", strings.Repeat("a", MaxTaskTitleLength+1), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTaskTitle(tt.title)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTaskTitle(%q) error = %v, wantErr %v", tt.title, err, tt.wantErr)
			}
		})
	}
}

func TestValidateTaskContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{"empty", "", false},
		{"at max", strings.Repeat("x", MaxTaskContentLength), false},
		{"over max", strings.Repeat("x", MaxTaskContentLength+1), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTaskContent(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTaskContent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateProjectName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty", "", true},
		{"valid", "My Project", false},
		{"over max", strings.Repeat("a", MaxProjectNameLength+1), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProjectName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateProjectName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateProjectID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{"valid 24-char hex", "aabbccddeeff00112233aabb", false},
		{"too short", "aabbcc", true},
		{"contains uppercase", "AABBCCDDEEFF00112233AABB", true},
		{"contains non-hex", "zzzzccddeeff00112233aabb", true},
		{"empty", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProjectID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateProjectID(%q) error = %v, wantErr %v", tt.id, err, tt.wantErr)
			}
		})
	}
}

func TestValidateTaskID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{"valid 24-char hex", "aabbccddeeff00112233aabb", false},
		{"too short", "aabbcc", true},
		{"contains uppercase", "AABBCCDDEEFF00112233AABB", true},
		{"contains non-hex", "zzzzccddeeff00112233aabb", true},
		{"empty", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTaskID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTaskID(%q) error = %v, wantErr %v", tt.id, err, tt.wantErr)
			}
		})
	}
}

func TestValidateProfileName(t *testing.T) {
	tests := []struct {
		name    string
		profile string
		wantErr bool
	}{
		{"default", "default", false},
		{"alphanumeric with hyphen and underscore", "my-profile_01", false},
		{"uppercase", "UPPERCASE", true},
		{"empty", "", true},
		{"contains spaces", "a b", true},
		{"over max length", strings.Repeat("a", MaxProfileNameLength+1), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProfileName(tt.profile)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateProfileName(%q) error = %v, wantErr %v", tt.profile, err, tt.wantErr)
			}
		})
	}
}

func TestValidateDate(t *testing.T) {
	tests := []struct {
		name    string
		date    string
		wantErr bool
	}{
		{"empty", "", false},
		{"valid RFC3339", "2024-01-15T10:30:00Z", false},
		{"invalid", "not-a-date", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDate(tt.date)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDate(%q) error = %v, wantErr %v", tt.date, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePriority(t *testing.T) {
	tests := []struct {
		name     string
		priority int
		wantErr  bool
	}{
		{"none (0)", 0, false},
		{"low (1)", 1, false},
		{"medium (3)", 3, false},
		{"high (5)", 5, false},
		{"invalid 2", 2, true},
		{"invalid 4", 4, true},
		{"negative", -1, true},
		{"too high", 6, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePriority(tt.priority)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePriority(%d) error = %v, wantErr %v", tt.priority, err, tt.wantErr)
			}
		})
	}
}

func TestValidateBatchSize(t *testing.T) {
	tests := []struct {
		name    string
		size    int
		wantErr bool
	}{
		{"zero", 0, true},
		{"one", 1, false},
		{"max", MaxBatchSize, false},
		{"over max", MaxBatchSize + 1, true},
		{"negative", -1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBatchSize(tt.size)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBatchSize(%d) error = %v, wantErr %v", tt.size, err, tt.wantErr)
			}
		})
	}
}

func TestValidateCreateTask(t *testing.T) {
	validProjectID := "aabbccddeeff00112233aabb"

	tests := []struct {
		name      string
		title     string
		projectID string
		content   string
		desc      string
		start     string
		due       string
		priority  int
		wantErr   bool
	}{
		{"valid inputs", "Task", "", "", "", "", "", 0, false},
		{"valid with project", "Task", validProjectID, "", "", "", "", 0, false},
		{"empty title", "", "", "", "", "", "", 0, true},
		{"invalid priority", "Task", "", "", "", "", "", 2, true},
		{"invalid project ID", "Task", "not-a-hex-id", "", "", "", "", 0, true},
		{"valid with all fields", "Task", validProjectID, "content", "desc", "2024-01-15T10:30:00Z", "2024-01-16T10:30:00Z", 5, false},
		{"invalid start date", "Task", "", "", "", "bad-date", "", 0, true},
		{"invalid due date", "Task", "", "", "", "", "bad-date", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCreateTask(tt.title, tt.projectID, tt.content, tt.desc, tt.start, tt.due, tt.priority)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCreateTask() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateUpdateTask(t *testing.T) {
	validTaskID := "aabbccddeeff00112233aabb"
	validProjectID := "112233aabbccddeeff001122"

	tests := []struct {
		name      string
		taskID    string
		projectID string
		title     string
		content   string
		desc      string
		start     string
		due       string
		priority  int
		wantErr   bool
	}{
		{"valid with empty title", validTaskID, validProjectID, "", "", "", "", "", 0, false},
		{"valid with title", validTaskID, validProjectID, "New Title", "", "", "", "", 0, false},
		{"invalid taskID", "bad", validProjectID, "", "", "", "", "", 0, true},
		{"invalid projectID", validTaskID, "bad", "", "", "", "", "", 0, true},
		{"invalid priority", validTaskID, validProjectID, "", "", "", "", "", 2, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUpdateTask(tt.taskID, tt.projectID, tt.title, tt.content, tt.desc, tt.start, tt.due, tt.priority)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdateTask() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidationError_Error(t *testing.T) {
	err := &ValidationError{Field: "title", Message: "must not be empty"}
	want := "validation: title: must not be empty"
	if got := err.Error(); got != want {
		t.Errorf("ValidationError.Error() = %q, want %q", got, want)
	}
}
