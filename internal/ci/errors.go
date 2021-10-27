package ci

type ErrCreatePipeline string

const (
	ErrCreatePipelineParam         ErrCreatePipeline = "params missing"
	ErrCreatePipelineCIFileInvalid ErrCreatePipeline = "ci file invalid"
)

func (err ErrCreatePipeline) Error() string {
	return string(err)
}
