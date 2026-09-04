package role

func BuildRoleResponse(r *Role) RoleResponse {
	return RoleResponse{
		ID:          r.ID,
		Name:        string(r.Name),
		Description: r.Description,
	}
}
