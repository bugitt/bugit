package db

type CIError interface {
	Type() CIErrType
	SpecMsg() string
	Source() string
}

type CIErrType int

const (
	InternalErrType CIErrType = 500
	UnknownErrType  CIErrType = 501

	ValidateErrType CIErrType = 100

	BuildErrType CIErrType = 200

	TestErrType CIErrType = 300

	PushErrType CIErrType = 400

	DeployErrType CIErrType = 600
)
