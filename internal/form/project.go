package form

type CreateProject struct {
	ProjectName string `binding:"Required" locale:"project.project_name_holder"`
}
