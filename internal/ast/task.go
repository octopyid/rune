package ast

import "strings"

// Parameter represents a positional task argument.
type Parameter struct {
	Name         string
	DefaultValue string
	HasDefault   bool
	Description  string
}

// Flag represents an optional boolean flag (e.g., --race?).
type Flag struct {
	Name        string // e.g. "race"
	Description string
}

// FullName returns the CLI flag with dashes, e.g. "--race".
func (f Flag) FullName() string {
	return "--" + f.Name
}

// Passthrough represents explicit passthrough arguments (e.g., *args).
type Passthrough struct {
	Name        string // e.g. "args"
	Description string
}

// Task represents a single task definition.
type Task struct {
	Name         string            // e.g. "db:fresh" or "build"
	Namespace    string            // e.g. "db" or ""
	ShortName    string            // e.g. "fresh" or "build"
	Description  string            // e.g. "Reset the database"
	Confirmation string            // confirmation message, or empty if none
	Dir          string            // working directory relative to Runefile dir, or empty
	Env          map[string]string // task-scoped environment variables
	Parameters   []Parameter       // positional arguments (required and default)
	Flags        []Flag            // optional boolean flags
	Passthrough  *Passthrough      // optional passthrough (*args)
	Dependencies []string          // task dependency names
	Commands     []string          // command lines to execute
	Line         int               // source line number
}

// HasFlag checks whether a flag with the given name exists in this task.
func (t *Task) HasFlag(name string) bool {
	clean := strings.TrimPrefix(name, "--")
	for _, f := range t.Flags {
		if f.Name == clean {
			return true
		}
	}
	return false
}

// File represents a parsed Runefile containing all tasks.
type File struct {
	Path     string
	Tasks    []*Task
	taskMap  map[string]*Task
	nsMap    map[string][]*Task
	nsList   []string
	rootList []*Task
}

// NewFile creates a new File representation from a list of tasks.
func NewFile(path string, tasks []*Task) *File {
	f := &File{
		Path:    path,
		Tasks:   tasks,
		taskMap: make(map[string]*Task, len(tasks)),
		nsMap:   make(map[string][]*Task),
	}

	seenNs := make(map[string]bool)
	for _, task := range tasks {
		f.taskMap[task.Name] = task
		if task.Namespace != "" {
			f.nsMap[task.Namespace] = append(f.nsMap[task.Namespace], task)
			if !seenNs[task.Namespace] {
				seenNs[task.Namespace] = true
				f.nsList = append(f.nsList, task.Namespace)
			}
		} else {
			f.rootList = append(f.rootList, task)
		}
	}

	return f
}

// GetTask retrieves a task by full name (e.g. "build" or "db:fresh").
func (f *File) GetTask(name string) (*Task, bool) {
	t, ok := f.taskMap[name]
	return t, ok
}

// HasTask checks whether a task with the given name exists.
func (f *File) HasTask(name string) bool {
	_, ok := f.taskMap[name]
	return ok
}

// Namespaces returns all unique namespaces in declaration order.
func (f *File) Namespaces() []string {
	return f.nsList
}

// IsNamespace checks whether a given name is a valid namespace.
func (f *File) IsNamespace(name string) bool {
	_, ok := f.nsMap[name]
	return ok
}

// TasksInNamespace returns all tasks belonging to the given namespace.
func (f *File) TasksInNamespace(ns string) []*Task {
	return f.nsMap[ns]
}

// RootTasks returns all tasks that do not belong to any namespace.
func (f *File) RootTasks() []*Task {
	return f.rootList
}

// AllTaskNames returns a list of all task names.
func (f *File) AllTaskNames() []string {
	names := make([]string, len(f.Tasks))
	for i, t := range f.Tasks {
		names[i] = t.Name
	}
	return names
}
