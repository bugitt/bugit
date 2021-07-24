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

func GetExpsByCourseID(courseID int64) ([]*Experiment, error) {
	exps := make([]*Experiment, 0)
	err := cloudX.Where("course_id = ?", courseID).Find(&exps)
	return exps, err
}
