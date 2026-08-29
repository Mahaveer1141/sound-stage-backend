package category

func BuildCategoryResponse(c *Category) CategoryResponse {
	return CategoryResponse{
		ID:          c.ID,
		Name:        c.Name,
		Description: c.Description,
	}
}

func BuildCategoryListResponse(categories []Category) []CategoryResponse {
	responses := make([]CategoryResponse, len(categories))
	for i := range categories {
		responses[i] = BuildCategoryResponse(&categories[i])
	}
	return responses
}
