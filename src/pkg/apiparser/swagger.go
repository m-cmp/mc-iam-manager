package apiparser

// GetOperations returns all operations from the spec with their paths and methods
func (s *SwaggerSpec) GetOperations() []OperationInfo {
	var operations []OperationInfo

	for path, pathItem := range s.Paths {
		if pathItem.Get != nil && pathItem.Get.OperationID != "" {
			operations = append(operations, OperationInfo{
				Path:      path,
				Method:    "get",
				Operation: pathItem.Get,
			})
		}
		if pathItem.Post != nil && pathItem.Post.OperationID != "" {
			operations = append(operations, OperationInfo{
				Path:      path,
				Method:    "post",
				Operation: pathItem.Post,
			})
		}
		if pathItem.Put != nil && pathItem.Put.OperationID != "" {
			operations = append(operations, OperationInfo{
				Path:      path,
				Method:    "put",
				Operation: pathItem.Put,
			})
		}
		if pathItem.Delete != nil && pathItem.Delete.OperationID != "" {
			operations = append(operations, OperationInfo{
				Path:      path,
				Method:    "delete",
				Operation: pathItem.Delete,
			})
		}
		if pathItem.Patch != nil && pathItem.Patch.OperationID != "" {
			operations = append(operations, OperationInfo{
				Path:      path,
				Method:    "patch",
				Operation: pathItem.Patch,
			})
		}
		if pathItem.Options != nil && pathItem.Options.OperationID != "" {
			operations = append(operations, OperationInfo{
				Path:      path,
				Method:    "options",
				Operation: pathItem.Options,
			})
		}
		if pathItem.Head != nil && pathItem.Head.OperationID != "" {
			operations = append(operations, OperationInfo{
				Path:      path,
				Method:    "head",
				Operation: pathItem.Head,
			})
		}
	}

	return operations
}
