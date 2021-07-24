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

type ExperimentStudent struct {
	ID        int64
	ExpId     int64
	StudentID int64
	CourseID  int64
}

func GetExpsByCourseID(courseID int64) ([]*Experiment, error) {
	exps := make([]*Experiment, 0)
	err := cloudX.Where("course_id = ?", courseID).Find(&exps)
	return exps, err
}

func GetExpByID(id int64) (*Experiment, error) {
	exp := new(Experiment)
	_, err := cloudX.ID(id).Get(exp)
	return exp, err
}

func ExistExpByID(eid, cid int64) (bool, error) {
	return cloudX.Table("experiment").Where("id = ? and course_id = ?", eid, cid).Exist()
}
