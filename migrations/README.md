
# buffalo model generate

require!!
Installing the buffalo-plugins Plugin
https://gobuffalo.io/documentation/guides/plugins/


```
buffalo db generate model workspace name:text description:nulls.text

buffalo db generate model project nsId:text name:text description:nulls.text

buffalo db generate model role name:text description:nulls.text

buffalo db generate model workspace_project_mapping workspace_id:uuid project_id:uuid

buffalo db generate model workspace_user_role_mapping workspace_id:uuid user_id:text role_id:uuid
```
**
add_index("workspaces", "name", {"unique": true})
**
add_index("projects", "name", {"unique": true})
**
add_index("roles", "name", {"unique": true})
**
add_foreign_key("mapping_workspace_projects", "workspace_id", {"workspaces": ["id"]}, {
	"on_delete": "CASCADE",
})
add_foreign_key("mapping_workspace_projects", "project_id", {"projects": ["id"]}, {
	"on_delete": "CASCADE",
})
sql("ALTER TABLE mapping_workspace_projects ADD CONSTRAINT unique_workspace_id_project_id UNIQUE (workspace_id, project_id);")
**
add_foreign_key("mapping_workspace_user_roles", "workspace_id", {"workspaces": ["id"]}, {})
add_foreign_key("mapping_workspace_user_roles", "role_id", {"roles": ["id"]}, {})