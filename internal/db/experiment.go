package db

type Experiment struct {
	ID         int64
	CourseID   int64
	Name       string
	CreateTime string
	StartTime  string
	EndTime    string
	Deadline   string
}
