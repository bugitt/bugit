package db

type CIError struct {
	Type CIErrType
	err  error
}

func (ciErr *CIError) Error() string {
	return ciErr.err.Error()
}

type CIErrType int

const (
	InternalErrType CIErrType = 500
	UnknownErrType  CIErrType = 501
	TimeoutErrType  CIErrType = 502

	ValidateErrType CIErrType = 100

	BuildErrType CIErrType = 200

	TestErrType CIErrType = 300

	PushErrType CIErrType = 400

	DeployErrType CIErrType = 600
)
