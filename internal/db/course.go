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
