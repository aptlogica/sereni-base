# API Response Codes

Source: `internal/utils/response/constants`

## Success Codes

| Custom Code | HTTP Status | Message | Description |
|---|---:|---|---|
| `AST_SUCCESS_5000` | 201 | Asset uploaded successfully | The asset has been uploaded successfully |
| `AST_SUCCESS_5001` | 201 | Asset created successfully | The asset has been created successfully |
| `AST_SUCCESS_5002` | 200 | Asset updated successfully | The asset has been updated successfully |
| `AST_SUCCESS_5003` | 200 | Asset deleted successfully | The asset has been deleted successfully |
| `AST_SUCCESS_5004` | 200 | Assets fetched successfully | The assets have been fetched successfully |
| `AUTH_SUCCESS_1001` | 201 | User registered successfully | The user has been registered successfully |
| `AUTH_SUCCESS_1002` | 200 | Login successful | The user has logged in successfully |
| `AUTH_SUCCESS_1003` | 200 | Email verified successfully | The user's email has been verified successfully |
| `AUTH_SUCCESS_1004` | 200 | OTP resent successfully | A new OTP has been sent successfully |
| `AUTH_SUCCESS_1005` | 200 | Token refreshed successfully | The access token has been refreshed successfully |
| `AUTH_SUCCESS_1006` | 200 | Forgot password request successful | Password recovery instructions have been sent successfully |
| `AUTH_SUCCESS_1007` | 200 | Password reset successful | The user's password has been reset successfully |
| `AUTH_SUCCESS_1008` | 200 | Logout successful | The user has been logged out successfully |
| `AUTH_SUCCESS_1009` | 200 | Token valid | The provided token is valid |
| `AUTH_SUCCESS_1010` | 200 | Token verified | The provided token has been verified |
| `BAS_SUCCESS_6001` | 201 | Base created successfully | The Base has been created successfully |
| `BAS_SUCCESS_6002` | 200 | Base updated successfully | The Base has been updated successfully |
| `BAS_SUCCESS_6003` | 200 | Base deleted successfully | The Base has been deleted successfully |
| `ROL_SUCCESS_4001` | 201 | Role created successfully | The role has been created successfully |
| `ROL_SUCCESS_4002` | 200 | Role updated successfully | The role has been updated successfully |
| `ROL_SUCCESS_4003` | 200 | Role deleted successfully | The role has been deleted successfully |
| `SUB_SUCCESS_5001` | 201 | Subscription plan created successfully | The subscription plan has been created successfully |
| `SUB_SUCCESS_5002` | 200 | Subscription plan updated successfully | The subscription plan has been updated successfully |
| `SUB_SUCCESS_5003` | 200 | Subscription plan deleted successfully | The subscription plan has been deleted successfully |
| `TBL_SUCCESS_5001` | 201 | Table created successfully | The table has been created successfully |
| `TBL_SUCCESS_5002` | 200 | Table updated successfully | The table has been updated successfully |
| `TBL_SUCCESS_5003` | 200 | Table deleted successfully | The table has been deleted successfully |
| `TBL_SUCCESS_5004` | 200 | Table fetched successfully | The table has been fetched successfully |
| `TBL_SUCCESS_5005` | 200 | Column added successfully | The column has been added to the table successfully |
| `TBL_SUCCESS_5006` | 200 | Column fetched successfully | The column has been fetched successfully |
| `TBL_SUCCESS_5007` | 201 | View created successfully | The view has been created successfully |
| `TBL_SUCCESS_5008` | 200 | View fetched successfully | The view has been fetched successfully |
| `TBL_SUCCESS_5009` | 200 | View updated successfully | The view has been updated successfully |
| `TBL_SUCCESS_5010` | 200 | View deleted successfully | The view has been deleted successfully |
| `TBL_SUCCESS_5011` | 200 | Column updated successfully | The column has been updated successfully |
| `TBL_SUCCESS_5012` | 200 | Column deleted successfully | The column has been deleted successfully |
| `TBL_SUCCESS_5013` | 201 | Record created successfully | The record has been created successfully |
| `TBL_SUCCESS_5014` | 200 | Records fetched successfully | The records have been fetched successfully |
| `TBL_SUCCESS_5015` | 201 | Row data inserted successfully | The row data has been inserted successfully |
| `TBL_SUCCESS_5016` | 200 | Row deleted successfully | The row has been deleted successfully |
| `TBL_SUCCESS_5017` | 200 | Column reordered successfully | The columns have been reordered successfully |
| `TNT_SUCCESS_3001` | 201 | Tenant created successfully | The tenant has been created successfully |
| `TNT_SUCCESS_3002` | 200 | Tenant updated successfully | The tenant has been updated successfully |
| `TNT_SUCCESS_3003` | 200 | Tenant deleted successfully | The tenant has been deleted successfully |
| `TNT_SUCCESS_3004` | 201 | Tenant subscription created successfully | The tenant subscription has been created successfully |
| `TNT_SUCCESS_3005` | 201 | Tenant membership created successfully | The tenant membership has been created successfully |
| `TNT_SUCCESS_3006` | 200 | Tenant fetched successfully | The tenant has been fetched successfully |
| `USR_SUCCESS_2001` | 201 | User created successfully | The user has been created successfully |
| `USR_SUCCESS_2002` | 200 | User updated successfully | The user has been updated successfully |
| `USR_SUCCESS_2003` | 200 | User deleted successfully | The user has been deleted successfully |
| `USR_SUCCESS_2004` | 200 | User fetched successfully | The user has been fetched successfully |
| `USR_SUCCESS_2005` | 200 | Password updated successfully | The user's password has been updated successfully |
| `USR_SUCCESS_2006` | 200 | Avatar added successfully | The user's avatar has been added successfully |
| `USR_SUCCESS_2007` | 200 | Avatar removed successfully | The user's avatar has been removed successfully |
| `USR_SUCCESS_2008` | 201 | User added successfully | The user has been added to the tenant successfully |
| `USR_SUCCESS_2009` | 200 | User removed successfully | The user has been removed from the tenant successfully |
| `USR_SUCCESS_2010` | 200 | Users fetched successfully | The users have been fetched successfully |
| `USR_SUCCESS_2011` | 201 | User assigned to workspace successfully | The user has been successfully assigned to the workspace |
| `USR_SUCCESS_2012` | 200 | Workspaces fetched successfully | The user's workspaces have been fetched successfully |
| `USR_SUCCESS_2013` | 200 | User removed from workspace successfully | The user has been removed from the workspace successfully |
| `USR_SUCCESS_2014` | 200 | User access details fetched successfully | The user's workspace and base access details have been fetched successfully |
| `WSP_SUCCESS_3001` | 201 | Workspace created successfully | The workspace has been created successfully |
| `WSP_SUCCESS_3002` | 200 | Workspace updated successfully | The workspace has been updated successfully |
| `WSP_SUCCESS_3003` | 200 | Workspace deleted successfully | The workspace has been deleted successfully |

## Failure Codes

| Custom Code | HTTP Status | Message | Description |
|---|---:|---|---|
| `AST_5000` | 500 | Asset upload failed | The asset could not be uploaded due to an internal error |
| `AST_5001` | 404 | Asset not found | The specified asset could not be found |
| `AST_5002` | 409 | Asset already exists | An asset with the given information already exists |
| `AST_5003` | 500 | Asset not created | The asset could not be created due to an internal error |
| `AST_5004` | 500 | Asset not updated | The asset could not be updated due to an internal error |
| `AST_5005` | 500 | Asset not deleted | The asset could not be deleted due to an internal error |
| `AST_5006` | 400 | Invalid payload | The multipart form data was not found in the request |
| `AST_5007` | 400 | Invalid payload | No files were found in the multipart form data of the request |
| `AST_5008` | 400 | IDs required | One or more IDs are required in the request |
| `AST_5009` | 400 | Invalid IDs | The provided IDs are invalid or malformed |
| `AST_5010` | 400 | Invalid request | The request is invalid or malformed |
| `AST_5011` | 400 | Title required | The title field is required |
| `AST_5012` | 400 | Invalid title | The provided title is invalid or malformed |
| `AST_5013` | 500 | Failed to open file for storage | The file could not be opened for storage due to an internal error |
| `AST_5014` | 500 | Failed to upload file to storage | The file could not be uploaded to storage due to an internal error |
| `AST_5015` | 400 | File too large | The uploaded file exceeds the maximum allowed size |
| `AST_5016` | 400 | One or more files too large | One or more uploaded files exceed the maximum allowed size |
| `AST_5017` | 400 | Too many files | The number of uploaded files exceeds the allowed limit |
| `AST_5018` | 400 | Virus detected | The uploaded file contains a virus and was rejected |
| `AST_5019` | 400 | Invalid file format | Only image files are allowed |
| `AUTH_ERR_3001` | 401 | Authentication provider login failed | Unable to login using authentication provider |
| `AUTH_ERR_3002` | 401 | Authentication provider refresh token failed | Refresh token failed for authentication provider |
| `AUTH_ERR_3003` | 401 | Authentication provider token invalid | Authentication provider token is invalid |
| `AUTH_ERR_3004` | 503 | Authentication provider ping failed | Unable to ping authentication provider |
| `AUTH_ERR_3005` | 401 | Authentication provider auth header required | Authentication header required by provider |
| `AUTH_ERR_3006` | 401 | Authentication provider token decode failed | Failed to decode authentication provider token |
| `AUTH_ERR_3007` | 401 | Authentication provider claims not found | Claims not found in authentication provider token |
| `AUTH_ERR_3008` | 401 | Authentication provider user ID not found | User ID not found in authentication provider claims |
| `AUTH_ERR_3009` | 401 | User ID not found in token | User ID not available in token |
| `AUTH_ERR_3010` | 500 | Access token signing failed | Signing access token failed |
| `AUTH_ERR_3011` | 500 | Refresh token signing failed | Signing refresh token failed |
| `AUTH_ERR_3012` | 401 | Refresh token invalid | Supplied refresh token is invalid |
| `AUTH_ERR_3013` | 401 | Refresh token claims invalid | Claims in refresh token are invalid |
| `AUTH_ERR_3014` | 401 | Token claims invalid | Supplied token contains invalid claims |
| `AUTH_ERR_3015` | 401 | Token authorization header required | Authorization header is required |
| `AUTH_ERR_3016` | 401 | Token claims not found | Token claims were not found |
| `AUTH_ERR_3017` | 401 | Authentication provider admin login failed | Admin login in authentication provider failed |
| `AUTH_ERR_3018` | 500 | Authentication provider user create failed | User creation in authentication provider failed |
| `AUTH_ERR_3019` | 500 | Authentication provider set password failed | Setting password in authentication provider failed |
| `AUTH_ERR_3020` | 401 | Token expired | The authentication token has expired |
| `AUTH_ERR_3021` | 401 | Authentication provider token expired | Authentication provider's token has expired |
| `AUTH_VAL_1001` | 400 | Invalid request payload | First name is required |
| `AUTH_VAL_1002` | 400 | Invalid request payload | Last name is required |
| `AUTH_VAL_1003` | 400 | Invalid request payload | First name is invalid |
| `AUTH_VAL_1004` | 400 | Invalid request payload | Last name is invalid |
| `AUTH_VAL_1005` | 400 | Invalid request payload | Email is required |
| `AUTH_VAL_1006` | 400 | Invalid request payload | Email format is invalid |
| `AUTH_VAL_1007` | 400 | Invalid request payload | Email is invalid |
| `AUTH_VAL_1008` | 400 | Invalid request payload | Password is required |
| `AUTH_VAL_1009` | 400 | Invalid request payload | Password is too short |
| `AUTH_VAL_1010` | 400 | Invalid request payload | Password is invalid |
| `AUTH_VAL_1011` | 400 | Invalid request payload | Organization name is required |
| `AUTH_VAL_1012` | 400 | Invalid request payload | Organization name is invalid |
| `AUTH_VAL_1013` | 400 | Invalid OTP | Invalid OTP |
| `AUTH_VAL_1014` | 400 | Invalid request payload | User ID is required |
| `AUTH_VAL_1015` | 400 | Invalid request payload | User ID format is invalid |
| `AUTH_VAL_1016` | 400 | Invalid request payload | User ID is invalid |
| `AUTH_VAL_1017` | 400 | Invalid request payload | OTP is required |
| `AUTH_VAL_1018` | 400 | Invalid request payload | OTP is invalid |
| `AUTH_VAL_1019` | 400 | Invalid request payload | Subscription ID is required |
| `AUTH_VAL_1020` | 400 | Invalid request payload | Subscription ID is invalid |
| `AUTH_VAL_1021` | 400 | Invalid request payload | Role ID is required |
| `AUTH_VAL_1022` | 400 | Invalid request payload | Role ID is invalid |
| `AUTH_VAL_1023` | 400 | Invalid request payload | Header 'schema' is required |
| `AUTH_VAL_1044` | 400 | Invalid schema | Provided schema is invalid |
| `AUTH_VAL_1047` | 400 | Invalid request payload | Refresh token is required |
| `AUTH_VAL_1048` | 400 | Invalid request payload | Refresh token is invalid |
| `AUTH_VAL_1049` | 401 | Token required | Authentication token is required |
| `AUTH_VAL_1050` | 401 | Token invalid format | Submitted token format is invalid |
| `AUTH_VAL_1051` | 401 | Token invalid | The provided token is invalid |
| `AUTH_VAL_1052` | 400 | New password required | New password is required |
| `AUTH_VAL_1053` | 400 | New password invalid | Provided new password is invalid |
| `AUTH_VAL_1054` | 400 | Invalid request payload | Date of birth is required |
| `AUTH_VAL_1055` | 400 | Invalid request payload | Date of birth is invalid |
| `AUTH_VAL_1056` | 400 | Invalid request payload | Country is required |
| `AUTH_VAL_1057` | 400 | Invalid request payload | Country is invalid |
| `AUTH_VAL_1058` | 400 | Invalid request payload | Timezone is required |
| `AUTH_VAL_1059` | 400 | Invalid request payload | Timezone is invalid |
| `AUTH_VAL_1060` | 400 | Invalid request payload | Header 'workspace' is required |
| `AUTH_VAL_1061` | 400 | Invalid request payload | Header 'base' is required |
| `AUTH_VAL_2001` | 403 | Account locked | Your account has been locked |
| `AUTH_VAL_2002` | 429 | Too many requests | Too many authentication attempts, try again later |
| `AUTH_VAL_2003` | 401 | Unauthorized | You are not authorized to perform this action |
| `AUTH_VAL_2004` | 403 | Forbidden | You do not have permission to access this resource |
| `AUTH_VAL_2005` | 500 | Internal server error | An error occurred on the server |
| `AUTH_VAL_2006` | 404 | Not found | The requested resource was not found |
| `AUTH_VAL_2007` | 401 | Unauthorized | Token is unauthorized |
| `BAS_6001` | 404 | Base not found | The specified Base could not be found |
| `BAS_6002` | 409 | Base already exists | A Base with the given information already exists |
| `BAS_6003` | 500 | Base not created | The Base could not be created due to an internal error |
| `BAS_6004` | 500 | Base not updated | The Base could not be updated due to an internal error |
| `BAS_6005` | 500 | Base not deleted | The Base could not be deleted due to an internal error |
| `BAS_6006` | 400 | Base name is required | The Base name field is required and was not provided |
| `BAS_6007` | 400 | Base name is invalid | The Base name provided is invalid |
| `BAS_6008` | 400 | Base description is invalid | The Base description provided is invalid |
| `BAS_6009` | 400 | Workspace ID is required | The workspace ID field is required and was not provided |
| `BAS_6010` | 400 | Workspace ID is invalid | The workspace ID provided is invalid |
| `BAS_6011` | 404 | Base not found | The specified Base could not be found |
| `ERR_0001` | 400 | Invalid ID | The provided ID is invalid |
| `ERR_0002` | 401 | Unauthorized access | Unauthorized access |
| `ERR_0003` | 403 | Forbidden | Forbidden |
| `ERR_0004` | 401 | Unauthorized access | Session expired |
| `ERR_0005` | 400 | Bad request | Invalid request payload |
| `ERR_0006` | 422 | Validation failed | Validation failed |
| `ERR_0007` | 500 | Database error | Database error |
| `ERR_0008` | 404 | Record not found | Record not found |
| `ERR_0009` | 500 | Internal server error | Internal server error |
| `ERR_0010` | 503 | Service unavailable | Service unavailable |
| `ERR_0011` | 504 | Gateway timeout | Gateway timeout |
| `ERR_0012` | 429 | Too many requests | Too many requests |
| `ERR_0013` | 409 | User already exists | User already exists |
| `ERR_0014` | 409 | Record already exists | Record already exists |
| `ERR_0015` | 500 | Record not inserted | Record not inserted |
| `ERR_0016` | 400 | Bad request | Bad request |
| `ERR_0017` | 409 | Conflict | Conflict |
| `ERR_0018` | 501 | Not implemented | Not implemented |
| `ERR_0019` | 408 | Timeout | Timeout |
| `ERR_0020` | 424 | Dependency failed | Dependency failed |
| `ERR_0021` | 500 | Map to struct error | Map to struct error |
| `ERR_0022` | 500 | Struct to struct error | Struct to struct error |
| `ERR_0023` | 500 | Hashing error | Hashing error |
| `ERR_0024` | 401 | Invalid credentials | Invalid credentials |
| `ERR_0025` | 400 | Invalid driver | Invalid driver |
| `ERR_0026` | 404 | Not found | Not found |
| `ERR_0027` | 500 | JSON marshal error | JSON marshal error |
| `ERR_0028` | 500 | HTTP request creation error | HTTP request creation error |
| `ERR_0029` | 500 | HTTP do request error | HTTP do request error |
| `ERR_0030` | 403 | User not active | User not active |
| `ERR_1001` | 404 | File not found | File not found |
| `ERR_1002` | 409 | File already exists | File already exists |
| `ERR_1003` | 500 | File read failed | File read failed |
| `ERR_1004` | 500 | File write failed | File write failed |
| `ERR_1005` | 500 | File delete failed | File delete failed |
| `ERR_1006` | 403 | File permission denied | File permission denied |
| `ERR_1007` | 400 | File invalid path | File invalid path |
| `ERR_1008` | 404 | Folder not found | Folder not found |
| `ERR_1009` | 409 | Folder already exists | Folder already exists |
| `ERR_1010` | 500 | Folder create failed | Folder create failed |
| `ERR_1011` | 500 | Folder delete failed | Folder delete failed |
| `ERR_1012` | 403 | Folder permission denied | Folder permission denied |
| `ERR_1013` | 400 | Folder invalid path | Folder invalid path |
| `ERR_1014` | 400 | Invalid date of birth | Invalid date of birth |
| `ERR_1015` | 500 | Role creation failed | Role creation failed |
| `ERR_1016` | 404 | Subscription plan not found | Subscription plan not found |
| `ERR_1017` | 404 | Role not found | Role not found |
| `ERR_1018` | 500 | User disable failed | User disable failed |
| `ERR_1019` | 500 | Invalid workspace member data | Invalid workspace member data |
| `ERR_1020` | 401 | User context not found | User context not found |
| `RBAC_ERR_7001` | 404 | role not found | The requested role does not exist |
| `RBAC_ERR_7002` | 409 | role already exists | A role with this name already exists in the system |
| `RBAC_ERR_7003` | 500 | failed to delete role | An error occurred while deleting the role |
| `RBAC_ERR_7004` | 500 | failed to update role | An error occurred while updating the role |
| `RBAC_ERR_7005` | 400 | invalid role priority value | Role priority must be a valid number |
| `RBAC_ERR_7006` | 400 | failed to assign role to user | An error occurred while assigning the role to the user |
| `RBAC_ERR_7007` | 400 | failed to remove role from user | An error occurred while removing the role from the user |
| `RBAC_ERR_7008` | 404 | resource not found | The requested resource does not exist |
| `RBAC_ERR_7009` | 409 | resource already exists | A resource with this code already exists |
| `RBAC_ERR_7010` | 500 | failed to create resource | An error occurred while creating the resource |
| `RBAC_ERR_7011` | 500 | failed to delete resource | An error occurred while deleting the resource |
| `RBAC_ERR_7012` | 400 | invalid resource code | Resource code must be alphanumeric and unique |
| `RBAC_ERR_7013` | 404 | action not found | The requested action does not exist |
| `RBAC_ERR_7014` | 409 | action already exists | An action with this code already exists |
| `RBAC_ERR_7015` | 500 | failed to create action | An error occurred while creating the action |
| `RBAC_ERR_7016` | 500 | failed to delete action | An error occurred while deleting the action |
| `RBAC_ERR_7017` | 400 | invalid action code | Action code must be alphanumeric and unique |
| `RBAC_ERR_7018` | 404 | permission not found | The requested permission does not exist |
| `RBAC_ERR_7019` | 409 | permission already exists | This resource-action combination already exists |
| `RBAC_ERR_7020` | 500 | failed to create permission | An error occurred while creating the permission |
| `RBAC_ERR_7021` | 500 | failed to delete permission | An error occurred while deleting the permission |
| `RBAC_ERR_7022` | 400 | invalid resource-action combination | The provided resource and action combination is invalid |
| `RBAC_ERR_7023` | 404 | role permission mapping not found | The role permission mapping does not exist |
| `RBAC_ERR_7024` | 409 | role permission mapping already exists | This role already has this permission |
| `RBAC_ERR_7025` | 500 | failed to create role permission | An error occurred while assigning permission to role |
| `RBAC_ERR_7026` | 500 | failed to delete role permission | An error occurred while removing permission from role |
| `RBAC_ERR_7027` | 404 | access member record not found | The user does not have a role in the specified scope |
| `RBAC_ERR_7028` | 409 | user already has this role in the scope | The user already has this role assignment |
| `RBAC_ERR_7029` | 400 | failed to assign role to user | An error occurred while assigning the role to the user |
| `RBAC_ERR_7030` | 400 | failed to remove role from user | An error occurred while removing the role from the user |
| `RBAC_ERR_7031` | 400 | invalid access scope | The access scope type is invalid |
| `RBAC_ERR_7032` | 400 | scope ID is required for workspace or base scope | Scope ID must be provided for workspace and base level scopes |
| `RBAC_ERR_7033` | 403 | user does not have access to this scope | The user does not have access to the specified scope |
| `RBAC_ERR_7034` | 403 | user does not have permission to perform this action | The user lacks the required permission for this operation |
| `RBAC_ERR_7035` | 403 | access denied | You do not have access to this resource |
| `RBAC_ERR_7036` | 403 | insufficient privileges for this operation | Your current role does not have sufficient privileges |
| `RBAC_ERR_7037` | 400 | failed to assign roles to one or more users | One or more role assignments failed during the bulk operation |
| `RBAC_ERR_7038` | 400 | failed to remove roles from one or more users | One or more role removals failed during the bulk operation |
| `RBAC_ERR_7039` | 400 | user list cannot be empty for bulk operations | At least one user must be specified for bulk operations |
| `RBAC_ERR_7040` | 400 | invalid scope type. Must be 'system', 'workspace', or 'base' | Valid scope types are: system, workspace, base |
| `RBAC_ERR_7041` | 404 | scope not found | The specified scope does not exist |
| `ROL_4001` | 404 | Role not found | The specified role could not be found |
| `ROL_4002` | 409 | Role already exists | A role with the given information already exists |
| `ROL_4003` | 500 | Role not created | The role could not be created due to an internal error |
| `ROL_4004` | 500 | Role not updated | The role could not be updated due to an internal error |
| `ROL_4005` | 500 | Role not deleted | The role could not be deleted due to an internal error |
| `ROL_4006` | 400 | Role is required | A role must be provided in the request |
| `ROL_4007` | 400 | Invalid role | The specified role is not valid |
| `SUB_5001` | 404 | Subscription plan not found | The specified subscription plan could not be found |
| `SUB_5002` | 409 | Subscription plan already exists | A subscription plan with the given information already exists |
| `SUB_5003` | 500 | Subscription plan not created | The subscription plan could not be created due to an internal error |
| `SUB_5004` | 500 | Subscription plan not updated | The subscription plan could not be updated due to an internal error |
| `SUB_5005` | 500 | Subscription plan not deleted | The subscription plan could not be deleted due to an internal error |
| `TBL_1001` | 400 | Base ID is required | The base_id field is required |
| `TBL_1002` | 400 | Invalid Base ID | The provided base_id is invalid or malformed |
| `TBL_1003` | 400 | Workspace ID is required | The workspace_id field is required |
| `TBL_1004` | 400 | Invalid Workspace ID | The provided workspace_id is invalid or malformed |
| `TBL_1005` | 400 | Title is required | The title field is required |
| `TBL_1006` | 400 | Invalid title | The provided title is invalid or malformed |
| `TBL_1007` | 400 | Description is required | The description field is required |
| `TBL_1008` | 400 | Invalid description | The provided description is invalid or malformed |
| `TBL_1009` | 400 | Order index is required | The order_index field is required |
| `TBL_1010` | 400 | Invalid order index | The provided order_index is invalid or malformed |
| `TBL_1011` | 400 | Validation failed | One or more fields failed validation |
| `TBL_1012` | 404 | Table not found | The specified table could not be found |
| `TBL_1013` | 409 | Table already exists | A table with the given information already exists |
| `TBL_1014` | 500 | Table not created | The table could not be created due to an internal error |
| `TBL_1015` | 500 | Table not updated | The table could not be updated due to an internal error |
| `TBL_1016` | 500 | Table not deleted | The table could not be deleted due to an internal error |
| `TBL_1017` | 400 | Model ID is required | The model_id field is required |
| `TBL_1018` | 400 | Invalid model ID | The provided model_id is invalid or malformed |
| `TBL_1019` | 400 | Column name is required | The column_name field is required |
| `TBL_1020` | 400 | Invalid column name | The provided column_name is invalid or malformed |
| `TBL_1021` | 400 | UIDT is required | The uidt field is required |
| `TBL_1022` | 400 | Invalid UIDT | The provided uidt is invalid or malformed |
| `TBL_1023` | 400 | DT is required | The dt field is required |
| `TBL_1024` | 400 | Invalid DT | The provided dt is invalid or malformed |
| `TBL_1025` | 400 | Virtual is required | The virtual field is required |
| `TBL_1026` | 400 | Invalid virtual | The provided virtual value is invalid or malformed |
| `TBL_1027` | 400 | System is required | The system field is required |
| `TBL_1028` | 400 | Invalid system | The provided system value is invalid or malformed |
| `TBL_1029` | 400 | Type is required | The type field is required |
| `TBL_1030` | 400 | Invalid type | The provided type is invalid or malformed |
| `TBL_1031` | 404 | View not found | The requested view could not be found |
| `TBL_1032` | 500 | View upload failed | Failed to upload the view due to an internal error |
| `TBL_1033` | 403 | Update not allowed | Updates are not permitted for this table |
| `TBL_1034` | 403 | Delete not allowed | Deletes are not permitted for this table |
| `TBL_1035` | 404 | Column not found | The requested column could not be found |
| `TBL_1036` | 500 | Column update failed | Failed to update the column due to an internal error |
| `TBL_1037` | 400 | Column ID is required | The column ID field is required |
| `TBL_1038` | 400 | Invalid column ID | The provided column ID is invalid or malformed |
| `TBL_1039` | 400 | Value is required | The value field is required |
| `TBL_1040` | 400 | Invalid value | The provided value is invalid or malformed |
| `TBL_1041` | 400 | Row ID is required | The row ID field is required |
| `TBL_1042` | 400 | Invalid row ID | The provided row ID is invalid or malformed |
| `TBL_1043` | 400 | Meta is required | The meta field is required |
| `TBL_1044` | 400 | Invalid meta | The provided meta value is invalid or malformed |
| `TBL_1045` | 404 | Row not found | The requested row could not be found |
| `TBL_1046` | 400 | Action is required | The action field is required |
| `TBL_1047` | 400 | Invalid action | The provided action is invalid or not supported |
| `TBL_1048` | 400 | Invalid payload | The meta field for a link type column is missing required relation information |
| `TBL_1049` | 400 | Attachment is required | The attachment field is required |
| `TBL_1050` | 400 | Invalid attachment | The provided attachment is invalid or malformed |
| `TBL_1051` | 400 | Invalid payload | The meta field for a lookup type column is missing required information |
| `TBL_1052` | 400 | Limit is required | The page size (limit) field is required |
| `TBL_1053` | 400 | Invalid limit | The provided page size (limit) is invalid or malformed |
| `TBL_1054` | 400 | Page number is required | The page number field is required |
| `TBL_1055` | 400 | Invalid page number | The provided page number is invalid or malformed |
| `TNT_3001` | 404 | Tenant not found | The specified tenant could not be found |
| `TNT_3002` | 409 | Tenant already exists | A tenant with the given information already exists |
| `TNT_3003` | 500 | Tenant not created | The tenant could not be created due to an internal error |
| `TNT_3004` | 500 | Tenant not updated | The tenant could not be updated due to an internal error |
| `TNT_3005` | 500 | Tenant not deleted | The tenant could not be deleted due to an internal error |
| `TNT_3006` | 500 | Tenant subscription not created | The tenant subscription could not be created due to an internal error |
| `TNT_3007` | 500 | Tenant membership not created | The tenant membership could not be created due to an internal error |
| `TNT_3008` | 404 | Tenant subscription not found | The specified tenant subscription could not be found |
| `TNT_3009` | 404 | Tenant membership not found | The specified tenant membership could not be found |
| `USR_2005` | 409 | User already exists | A user with the given information already exists |
| `USR_2006` | 404 | User not found | The specified user could not be found |
| `USR_2007` | 500 | User not created | The user could not be created due to an internal error |
| `USR_2008` | 500 | User not updated | The user could not be updated due to an internal error |
| `USR_2009` | 500 | User not deleted | The user could not be deleted due to an internal error |
| `USR_2010` | 409 | Email already verified | The user's email address has already been verified |
| `USR_2011` | 401 | Invalid old password | The provided old password is incorrect |
| `USR_2012` | 400 | Old password is required | You must provide your old password to proceed |
| `USR_2013` | 400 | Old password is invalid | The old password provided does not meet requirements or is invalid |
| `USR_2014` | 400 | New password is required | You must provide a new password to update your credentials |
| `USR_2015` | 400 | New password is invalid | The new password provided does not meet the required criteria |
| `USR_2016` | 400 | Email is required | An email address must be provided |
| `USR_2017` | 400 | Invalid email address | The provided email address does not match the required format |
| `USR_2018` | 400 | First name is required | A first name must be provided |
| `USR_2019` | 400 | Invalid first name | The provided first name is invalid |
| `USR_2020` | 400 | Last name is required | A last name must be provided |
| `USR_2021` | 400 | Invalid last name | The provided last name is invalid |
| `USR_2022` | 400 | Role ID is required | A role identifier must be provided |
| `USR_2023` | 400 | Invalid role ID | The provided role identifier is not valid or not a proper UUID |
| `USR_2024` | 400 | User ID is required | A user identifier must be provided |
| `USR_2025` | 400 | Invalid user ID | The provided user identifier is not valid or not a proper UUID |
| `WSP_3005` | 409 | Workspace already exists | A workspace with the given information already exists |
| `WSP_3006` | 404 | Workspace not found | The specified workspace could not be found |
| `WSP_3007` | 500 | Workspace not created | The workspace could not be created due to an internal error |
| `WSP_3008` | 500 | Workspace not updated | The workspace could not be updated due to an internal error |
| `WSP_3009` | 500 | Workspace not deleted | The workspace could not be deleted due to an internal error |
| `WSP_3010` | 400 | Workspace name is required | The workspace name field is required and was not provided |
| `WSP_3011` | 400 | Workspace name is invalid | The workspace name provided is invalid |
| `WSP_3012` | 400 | Workspace description is invalid | The workspace description provided is invalid |
| `WSP_3013` | 400 | Workspace ID is required | The workspace ID field is required and was not provided |
| `WSP_3014` | 400 | Workspace ID is invalid | The workspace ID provided is invalid |
| `WSP_3015` | 404 | Workspace member not found | The specified workspace member does not exist |
| `WSP_3016` | 400 | User already in workspace | The user is already a member of the specified workspace |
| `WSP_3017` | 500 | Workspace insertion failed | Failed to insert workspace due to an internal error |

## Totals

- Success codes: **64**
- Failure codes: **299**
