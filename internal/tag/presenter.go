package tag

func BuildTagResponse(t *Tag) TagResponse {
	return TagResponse{
		ID:   t.ID,
		Name: t.Name,
	}
}

func BuildTagListResponse(tags []Tag) []TagResponse {
	responses := make([]TagResponse, len(tags))
	for i := range tags {
		responses[i] = BuildTagResponse(&tags[i])
	}
	return responses
}
