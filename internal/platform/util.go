package platform

import "strings"

func getInt64Ptr(x int64) *int64 {
	return &x
}

// PrettyName get harborName
// FIXME 这里需要细化一下转成转换 name 的逻辑
func PrettyName(name string) string {
	return strings.ToLower(name)
}
