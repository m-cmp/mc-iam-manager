
# buffalo model generate

require!!
Installing the buffalo-plugins Plugin
https://gobuffalo.io/documentation/guides/plugins/


```
buffalo db generate model mc_iam_project project_id:text name:text description:nulls.text 

buffalo db generate model mc_iam_workspace workspace_id:text name:text description:nulls.text 

buffalo db generate model mc_iam_roletype type:text role_id:text role_name:text 

buffalo db generate model mc_iam_mapping_workspace_project workspace_id:text project_id:text

buffalo db generate model mc_iam_mapping_workspace_user_role workspace_id:text role_name:text user_id:text
```