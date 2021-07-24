package db

type Course struct {
	ID        int64
	TeacherID int64
	TermID    int64
	Name      string
}

type CourseStudentMapping struct {
	CourseID  int64  `xorm:"NOT NULL DEFAULT 0"`
	StudentID string `xorm:"NOT NULL DEFAULT '0'"`
}

func GetCoursesByStudentID(sid string) ([]*Course, error) {
	var courses []*Course
	err := cloudX.Table("course").
		Join("INNER", "course_student_mapping", "course_student_mapping.course_id = course.id").
		Where("course_student_mapping.student_id = ?", sid).
		Find(&courses)
	return courses, err
}
