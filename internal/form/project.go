package form

type CreateProject struct {
	ProjectName string `binding:"Required;AlphaDashDot;MaxSize(35)" locale:"org.org_name_holder"`
}
