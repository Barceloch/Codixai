package measurement

import (
	_ "embed"
	"encoding/json"
)

//go:embed tasks.json
var tasksJSON []byte

// Task is one representative measurement task.
type Task struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
	Repository  string `json:"repository,omitempty"`
}

// TaskRegistry holds the registered tasks.
type TaskRegistry struct {
	SchemaVersion string `json:"schema_version"`
	Tasks         []Task `json:"tasks"`
}

// LoadTasks returns the embedded task registry.
func LoadTasks() (TaskRegistry, error) {
	var r TaskRegistry
	if err := json.Unmarshal(tasksJSON, &r); err != nil {
		return TaskRegistry{}, err
	}
	if r.SchemaVersion == "" {
		r.SchemaVersion = SchemaVersion
	}
	return r, nil
}
