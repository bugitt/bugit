package db

func IsStudentIDValid(studendID string) (bool, error) {
	// TODO: 学号合法性进一步检验
	return true, nil
}

func IsStudentIDExist(studentID string) (bool, error) {
	if len(studentID) <= 0 {
		return false, nil
	}
	return x.Get(&User{StudentID: studentID})
}
