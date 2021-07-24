package db

import "xorm.io/xorm"

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

func buildCourseQueryWithStudentID(sid string) *xorm.Session {
	return cloudX.Table("course").
		Join("INNER", "course_student_mapping", "course_student_mapping.course_id = course.id").
		Where("course_student_mapping.student_id = ?", sid)
}

func GetCoursesByStudentID(sid string) ([]*Course, error) {
	var courses []*Course
	err := buildCourseQueryWithStudentID(sid).Find(&courses)
	return courses, err
}

func GetCourseByID(id int64) (*Course, error) {
	course := new(Course)
	_, err := cloudX.ID(id).Get(course)
	return course, err
}

func ExistCourseByStudentID(cid int64, sid string) (bool, error) {
	return buildCourseQueryWithStudentID(sid).And("course.id = ?", cid).Get(new(Course))
}
