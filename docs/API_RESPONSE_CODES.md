# API Response Codes Documentation

This document provides a comprehensive list of all success and error response codes used in the Sereni Base API.

## Table of Contents
- [Authentication Codes](#authentication-codes)
- [User Management Codes](#user-management-codes)
- [Workspace Management Codes](#workspace-management-codes)
- [Base Management Codes](#base-management-codes)
- [Table Management Codes](#table-management-codes)
- [Asset Management Codes](#asset-management-codes)
- [Tenant Management Codes](#tenant-management-codes)
- [Role Management Codes](#role-management-codes)
- [RBAC Management Codes](#rbac-management-codes)
- [Subscription Plan Codes](#subscription-plan-codes)
- [Core Error Codes](#core-error-codes)

---

## Authentication Codes

### Success Codes

| HTTP Code | Code | UI Message | Dev Message (Description) |
|-----------|------|------------|---------------------------|
| 201 | AUTH_SUCCESS_1001 | User registered successfully | The user has been registered successfully |
| 200 | AUTH_SUCCESS_1002 | Login successful | The user has logged in successfully |
| 200 | AUTH_SUCCESS_1003 | Email verified successfully | The user's email has been verified successfully |
| 200 | AUTH_SUCCESS_1004 | OTP resent successfully | A new OTP has been sent successfully |
| 200 | AUTH_SUCCESS_1005 | Token refreshed successfully | The access token has been refreshed successfully |
| 200 | AUTH_SUCCESS_1006 | Forgot password request successful | Password recovery instructions have been sent successfully |
| 200 | AUTH_SUCCESS_1007 | Password reset successful | The user's password has been reset successfully |
| 200 | AUTH_SUCCESS_1008 | Logout successful | The user has been logged out successfully |
| 200 | AUTH_SUCCESS_1009 | Token valid | The provided token is valid |
| 200 | AUTH_SUCCESS_1010 | Token verified | The provided token has been verified |

### Error Codes

| HTTP Code | Code | UI Message | Dev Message (Description) |
|-----------|------|------------|---------------------------|
| 400 | AUTH_VAL_1001 | Invalid request payload | First name is required |
| 400 | AUTH_VAL_1002 | Invalid request payload | Last name is required |
| 400 | AUTH_VAL_1003 | Invalid request payload | First name is invalid |
| 400 | AUTH_VAL_1004 | Invalid request payload | Last name is invalid |
| 400 | AUTH_VAL_1005 | Invalid request payload | Email is required |
| 400 | AUTH_VAL_1006 | Invalid request payload | Email format is invalid |
| 400 | AUTH_VAL_1007 | Invalid request payload | Email is invalid |
| 400 | AUTH_VAL_1008 | Invalid request payload | Password is required |
| 400 | AUTH_VAL_1009 | Invalid request payload | Password is too short |
| 400 | AUTH_VAL_1010 | Invalid request payload | Password is invalid |
| 400 | AUTH_VAL_1011 | Invalid request payload | Organization name is required |
| 400 | AUTH_VAL_1012 | Invalid request payload | Organization name is invalid |
| 400 | AUTH_VAL_1013 | Invalid OTP | Invalid OTP |
| 400 | AUTH_VAL_1014 | Invalid request payload | User ID is required |
| 400 | AUTH_VAL_1015 | Invalid request payload | User ID format is invalid |
| 400 | AUTH_VAL_1016 | Invalid request payload | User ID is invalid |
| 400 | AUTH_VAL_1017 | Invalid request payload | OTP is required |
| 400 | AUTH_VAL_1018 | Invalid request payload | OTP is invalid |
| 400 | AUTH_VAL_1019 | Invalid request payload | Subscription ID is required |
| 400 | AUTH_VAL_1020 | Invalid request payload | Subscription ID is invalid |
| 400 | AUTH_VAL_1021 | Invalid request payload | Role ID is required |
| 400 | AUTH_VAL_1022 | Invalid request payload | Role ID is invalid |
| 400 | AUTH_VAL_1023 | Invalid request payload | Header 'schema' is required |
| 400 | AUTH_VAL_1044 | Invalid schema | Provided schema is invalid |
| 400 | AUTH_VAL_1047 | Invalid request payload | Refresh token is required |
| 400 | AUTH_VAL_1048 | Invalid request payload | Refresh token is invalid |
| 401 | AUTH_VAL_1049 | Token required | Authentication token is required |
| 401 | AUTH_VAL_1050 | Token invalid format | Submitted token format is invalid |
| 401 | AUTH_VAL_1051 | Token invalid | The provided token is invalid |
| 400 | AUTH_VAL_1052 | New password required | New password is required |
| 400 | AUTH_VAL_1053 | New password invalid | Provided new password is invalid |
| 400 | AUTH_VAL_1054 | Invalid request payload | Date of birth is required |
| 400 | AUTH_VAL_1055 | Invalid request payload | Date of birth is invalid |
| 400 | AUTH_VAL_1056 | Invalid request payload | Country is required |
| 400 | AUTH_VAL_1057 | Invalid request payload | Country is invalid |
| 400 | AUTH_VAL_1058 | Invalid request payload | Timezone is required |
| 400 | AUTH_VAL_1059 | Invalid request payload | Timezone is invalid |
| 400 | AUTH_VAL_1060 | Invalid request payload | Header 'workspace' is required |
| 400 | AUTH_VAL_1061 | Invalid request payload | Header 'base' is required |
| 403 | AUTH_VAL_2001 | Account locked | Your account has been locked |
| 429 | AUTH_VAL_2002 | Too many requests | Too many authentication attempts, try again later |
| 401 | AUTH_VAL_2003 | Unauthorized | You are not authorized to perform this action |
| 403 | AUTH_VAL_2004 | Forbidden | You do not have permission to access this resource |
| 500 | AUTH_VAL_2005 | Internal server error | An error occurred on the server |
| 404 | AUTH_VAL_2006 | Not found | The requested resource was not found |
| 401 | AUTH_VAL_2007 | Unauthorized | Token is unauthorized |
| 401 | AUTH_ERR_3001 | Authentication provider login failed | Unable to login using authentication provider |
| 401 | AUTH_ERR_3002 | Authentication provider refresh token failed | Refresh token failed for authentication provider |
| 401 | AUTH_ERR_3003 | Authentication provider token invalid | Authentication provider token is invalid |
| 503 | AUTH_ERR_3004 | Authentication provider ping failed | Unable to ping authentication provider |
| 401 | AUTH_ERR_3005 | Authentication provider auth header required | Authentication header required by provider |
| 401 | AUTH_ERR_3006 | Authentication provider token decode failed | Failed to decode authentication provider token |
| 401 | AUTH_ERR_3007 | Authentication provider claims not found | Claims not found in authentication provider token |
| 401 | AUTH_ERR_3008 | Authentication provider user ID not found | User ID not found in authentication provider claims |
| 401 | AUTH_ERR_3009 | User ID not found in token | User ID not available in token |
| 500 | AUTH_ERR_3010 | Access token signing failed | Signing access token failed |
| 500 | AUTH_ERR_3011 | Refresh token signing failed | Signing refresh token failed |
| 401 | AUTH_ERR_3012 | Refresh token invalid | Supplied refresh token is invalid |
| 401 | AUTH_ERR_3013 | Refresh token claims invalid | Claims in refresh token are invalid |
| 401 | AUTH_ERR_3014 | Token claims invalid | Supplied token contains invalid claims |
| 401 | AUTH_ERR_3015 | Token authorization header required | Authorization header is required |
| 401 | AUTH_ERR_3016 | Token claims not found | Token claims were not found |
| 401 | AUTH_ERR_3017 | Authentication provider admin login failed | Admin login in authentication provider failed |
| 500 | AUTH_ERR_3018 | Authentication provider user create failed | User creation in authentication provider failed |
| 500 | AUTH_ERR_3019 | Authentication provider set password failed | Setting password in authentication provider failed |
| 401 | AUTH_ERR_3020 | Token expired | The authentication token has expired |
| 401 | AUTH_ERR_3021 | Authentication provider token expired | Authentication provider's token has expired |

---

## User Management Codes

### Success Codes

| HTTP Code | Code | UI Message | Dev Message (Description) |
|-----------|------|------------|---------------------------|
| 201 | USR_SUCCESS_2001 | User created successfully | The user has been created successfully |
| 200 | USR_SUCCESS_2002 | User updated successfully | The user has been updated successfully |
| 200 | USR_SUCCESS_2003 | User deleted successfully | The user has been deleted successfully |
| 200 | USR_SUCCESS_2004 | User fetched successfully | The user has been fetched successfully |
| 200 | USR_SUCCESS_2005 | Password updated successfully | The user's password has been updated successfully |
| 200 | USR_SUCCESS_2006 | Avatar added successfully | The user's avatar has been added successfully |
| 200 | USR_SUCCESS_2007 | Avatar removed successfully | The user's avatar has been removed successfully |
| 201 | USR_SUCCESS_2008 | User added successfully | The user has been added to the tenant successfully |
| 200 | USR_SUCCESS_2009 | User removed successfully | The user has been removed from the tenant successfully |
| 200 | USR_SUCCESS_2010 | Users fetched successfully | The users have been fetched successfully |
| 201 | USR_SUCCESS_2011 | User assigned to workspace successfully | The user has been successfully assigned to the workspace |
| 200 | USR_SUCCESS_2012 | Workspaces fetched successfully | The user's workspaces have been fetched successfully |
| 200 | USR_SUCCESS_2013 | User removed from workspace successfully | The user has been removed from the workspace successfully |
| 200 | USR_SUCCESS_2014 | User access details fetched successfully | The user's workspace and base access details have been fetched successfully |

### Error Codes

| HTTP Code | Code | UI Message | Dev Message (Description) |
|-----------|------|------------|---------------------------|
| 404 | USR_2006 | User not found | The specified user could not be found |
| 409 | USR_2005 | User already exists | A user with the given information already exists |
| 500 | USR_2007 | User not created | The user could not be created due to an internal error |
| 500 | USR_2008 | User not updated | The user could not be updated due to an internal error |
| 500 | USR_2009 | User not deleted | The user could not be deleted due to an internal error |
| 409 | USR_2010 | Email already verified | The user's email address has already been verified |
| 401 | USR_2011 | Invalid old password | The provided old password is incorrect |
| 400 | USR_2012 | Old password is required | You must provide your old password to proceed |
| 400 | USR_2013 | Old password is invalid | The old password provided does not meet requirements or is invalid |
| 400 | USR_2014 | New password is required | You must provide a new password to update your credentials |
| 400 | USR_2015 | New password is invalid | The new password provided does not meet the required criteria |
| 400 | USR_2016 | Email is required | An email address must be provided |
| 400 | USR_2017 | Invalid email address | The provided email address does not match the required format |
| 400 | USR_2018 | First name is required | A first name must be provided |
| 400 | USR_2019 | Invalid first name | The provided first name is invalid |
| 400 | USR_2020 | Last name is required | A last name must be provided |
| 400 | USR_2021 | Invalid last name | The provided last name is invalid |
| 400 | USR_2022 | Role ID is required | A role identifier must be provided |
| 400 | USR_2023 | Invalid role ID | The provided role identifier is not valid or not a proper UUID |
| 400 | USR_2024 | User ID is required | A user identifier must be provided |
| 400 | USR_2025 | Invalid user ID | The provided user identifier is not valid or not a proper UUID |

---

## Workspace Management Codes

### Success Codes

| HTTP Code | Code | UI Message | Dev Message (Description) |
|-----------|------|------------|---------------------------|
| 201 | WSP_SUCCESS_3001 | Workspace created successfully | The workspace has been created successfully |
| 200 | WSP_SUCCESS_3002 | Workspace updated successfully | The workspace has been updated successfully |
| 200 | WSP_SUCCESS_3003 | Workspace deleted successfully | The workspace has been deleted successfully |

### Error Codes

| HTTP Code | Code | UI Message | Dev Message (Description) |
|-----------|------|------------|---------------------------|
| 404 | WSP_3006 | Workspace not found | The specified workspace could not be found |
| 409 | WSP_3005 | Workspace already exists | A workspace with the given information already exists |
| 500 | WSP_3007 | Workspace not created | The workspace could not be created due to an internal error |
| 500 | WSP_3008 | Workspace not updated | The workspace could not be updated due to an internal error |
| 500 | WSP_3009 | Workspace not deleted | The workspace could not be deleted due to an internal error |
| 400 | WSP_3010 | Workspace name is required | The workspace name field is required and was not provided |
| 400 | WSP_3011 | Workspace name is invalid | The workspace name provided is invalid |
| 400 | WSP_3012 | Workspace description is invalid | The workspace description provided is invalid |
| 400 | WSP_3013 | Workspace ID is required | The workspace ID field is required and was not provided |
| 400 | WSP_3014 | Workspace ID is invalid | The workspace ID provided is invalid |
| 500 | WSP_3017 | Workspace insertion failed | Failed to insert workspace due to an internal error |
| 404 | WSP_3015 | Workspace member not found | The specified workspace member does not exist |
| 400 | WSP_3016 | User already in workspace | The user is already a member of the specified workspace |

---

## Base Management Codes

### Success Codes

| HTTP Code | Code | UI Message | Dev Message (Description) |
|-----------|------|------------|---------------------------|
| 201 | BAS_SUCCESS_6001 | Base created successfully | The Base has been created successfully |
| 200 | BAS_SUCCESS_6002 | Base updated successfully | The Base has been updated successfully |
| 200 | BAS_SUCCESS_6003 | Base deleted successfully | The Base has been deleted successfully |

### Error Codes

| HTTP Code | Code | UI Message | Dev Message (Description) |
|-----------|------|------------|---------------------------|
| 404 | BAS_6001 | Base not found | The specified Base could not be found |
| 409 | BAS_6002 | Base already exists | A Base with the given information already exists |
| 500 | BAS_6003 | Base not created | The Base could not be created due to an internal error |
| 500 | BAS_6004 | Base not updated | The Base could not be updated due to an internal error |
| 500 | BAS_6005 | Base not deleted | The Base could not be deleted due to an internal error |
| 400 | BAS_6006 | Base name is required | The Base name field is required and was not provided |
| 400 | BAS_6007 | Base name is invalid | The Base name provided is invalid |
| 400 | BAS_6008 | Base description is invalid | The Base description provided is invalid |
| 404 | BAS_6011 | Base not found | The specified Base could not be found |

---

## Table Management Codes

### Success Codes

| HTTP Code | Code | UI Message | Dev Message (Description) |
|-----------|------|------------|---------------------------|
| 201 | TBL_SUCCESS_5001 | Table created successfully | The table has been created successfully |
| 200 | TBL_SUCCESS_5002 | Table updated successfully | The table has been updated successfully |
| 200 | TBL_SUCCESS_5003 | Table deleted successfully | The table has been deleted successfully |
| 200 | TBL_SUCCESS_5004 | Table fetched successfully | The table has been fetched successfully |
| 200 | TBL_SUCCESS_5005 | Column added successfully | The column has been added to the table successfully |
| 200 | TBL_SUCCESS_5006 | Column fetched successfully | The column has been fetched successfully |
| 200 | TBL_SUCCESS_5011 | Column updated successfully | The column has been updated successfully |
| 201 | TBL_SUCCESS_5007 | View created successfully | The view has been created successfully |
| 200 | TBL_SUCCESS_5008 | View fetched successfully | The view has been fetched successfully |
| 200 | TBL_SUCCESS_5009 | View updated successfully | The view has been updated successfully |
| 200 | TBL_SUCCESS_5010 | View deleted successfully | The view has been deleted successfully |
| 200 | TBL_SUCCESS_5012 | Column deleted successfully | The column has been deleted successfully |
| 201 | TBL_SUCCESS_5013 | Record created successfully | The record has been created successfully |
| 200 | TBL_SUCCESS_5014 | Records fetched successfully | The records have been fetched successfully |
| 201 | TBL_SUCCESS_5015 | Row data inserted successfully | The row data has been inserted successfully |
| 200 | TBL_SUCCESS_5016 | Row deleted successfully | The row has been deleted successfully |
| 200 | TBL_SUCCESS_5017 | Column reordered successfully | The columns have been reordered successfully |

### Error Codes

| HTTP Code | Code | UI Message | Dev Message (Description) |
|-----------|------|------------|---------------------------|
| 400 | TBL_1001 | Base ID is required | The base_id field is required |
| 400 | TBL_1002 | Invalid Base ID | The provided base_id is invalid or malformed |
| 400 | TBL_1003 | Workspace ID is required | The workspace_id field is required |
| 400 | TBL_1004 | Invalid Workspace ID | The provided workspace_id is invalid or malformed |
| 400 | TBL_1005 | Title is required | The title field is required |
| 400 | TBL_1006 | Invalid title | The provided title is invalid or malformed |
| 400 | TBL_1007 | Description is required | The description field is required |
| 400 | TBL_1008 | Invalid description | The provided description is invalid or malformed |
| 400 | TBL_1009 | Order index is required | The order_index field is required |
| 400 | TBL_1010 | Invalid order index | The provided order_index is invalid or malformed |
| 400 | TBL_1011 | Validation failed | One or more fields failed validation |
| 404 | TBL_1012 | Table not found | The specified table could not be found |
| 409 | TBL_1013 | Table already exists | A table with the given information already exists |
| 500 | TBL_1014 | Table not created | The table could not be created due to an internal error |
| 500 | TBL_1015 | Table not updated | The table could not be updated due to an internal error |
| 500 | TBL_1016 | Table not deleted | The table could not be deleted due to an internal error |
| 400 | TBL_1017 | Model ID is required | The model_id field is required |
| 400 | TBL_1018 | Invalid model ID | The provided model_id is invalid or malformed |
| 400 | TBL_1019 | Column name is required | The column_name field is required |
| 400 | TBL_1020 | Invalid column name | The provided column_name is invalid or malformed |
| 400 | TBL_1021 | UIDT is required | The uidt field is required |
| 400 | TBL_1022 | Invalid UIDT | The provided uidt is invalid or malformed |
| 400 | TBL_1023 | DT is required | The dt field is required |
| 400 | TBL_1024 | Invalid DT | The provided dt is invalid or malformed |
| 400 | TBL_1025 | Virtual is required | The virtual field is required |
| 400 | TBL_1026 | Invalid virtual | The provided virtual value is invalid or malformed |
| 400 | TBL_1027 | System is required | The system field is required |
| 400 | TBL_1028 | Invalid system | The provided system value is invalid or malformed |
| 400 | TBL_1029 | Type is required | The type field is required |
| 400 | TBL_1030 | Invalid type | The provided type is invalid or malformed |
| 404 | TBL_1031 | View not found | The requested view could not be found |
| 500 | TBL_1032 | View upload failed | Failed to upload the view due to an internal error |
| 403 | TBL_1033 | Update not allowed | Updates are not permitted for this table |
| 403 | TBL_1034 | Delete not allowed | Deletes are not permitted for this table |
| 404 | TBL_1035 | Column not found | The requested column could not be found |
| 500 | TBL_1036 | Column update failed | Failed to update the column due to an internal error |
| 400 | TBL_1037 | Column ID is required | The column ID field is required |
| 400 | TBL_1038 | Invalid column ID | The provided column ID is invalid or malformed |
| 400 | TBL_1039 | Value is required | The value field is required |
| 400 | TBL_1040 | Invalid value | The provided value is invalid or malformed |
| 400 | TBL_1041 | Row ID is required | The row ID field is required |
| 400 | TBL_1042 | Invalid row ID | The provided row ID is invalid or malformed |
| 400 | TBL_1043 | Meta is required | The meta field is required |
| 400 | TBL_1044 | Invalid meta | The provided meta value is invalid or malformed |
| 404 | TBL_1045 | Row not found | The requested row could not be found |
| 400 | TBL_1046 | Action is required | The action field is required |
| 400 | TBL_1047 | Invalid action | The provided action is invalid or not supported |
| 400 | TBL_1048 | Invalid payload | The meta field for a link type column is missing required relation information |
| 400 | TBL_1049 | Attachment is required | The attachment field is required |
| 400 | TBL_1050 | Invalid attachment | The provided attachment is invalid or malformed |
| 400 | TBL_1051 | Invalid payload | The meta field for a lookup type column is missing required information |
| 400 | TBL_1052 | Limit is required | The page size (limit) field is required |
| 400 | TBL_1053 | Invalid limit | The provided page size (limit) is invalid or malformed |
| 400 | TBL_1054 | Page number is required | The page number field is required |
| 400 | TBL_1055 | Invalid page number | The provided page number is invalid or malformed |

---

## Asset Management Codes

### Success Codes

| HTTP Code | Code | UI Message | Dev Message (Description) |
|-----------|------|------------|---------------------------|
| 201 | AST_SUCCESS_5000 | Asset uploaded successfully | The asset has been uploaded successfully |
| 201 | AST_SUCCESS_5001 | Asset created successfully | The asset has been created successfully |
| 200 | AST_SUCCESS_5002 | Asset updated successfully | The asset has been updated successfully |
| 200 | AST_SUCCESS_5003 | Asset deleted successfully | The asset has been deleted successfully |
| 200 | AST_SUCCESS_5004 | Assets fetched successfully | The assets have been fetched successfully |

### Error Codes

| HTTP Code | Code | UI Message | Dev Message (Description) |
|-----------|------|------------|---------------------------|
| 500 | AST_5000 | Asset upload failed | The asset could not be uploaded due to an internal error |
| 404 | AST_5001 | Asset not found | The specified asset could not be found |
| 409 | AST_5002 | Asset already exists | An asset with the given information already exists |
| 500 | AST_5003 | Asset not created | The asset could not be created due to an internal error |
| 500 | AST_5004 | Asset not updated | The asset could not be updated due to an internal error |
| 500 | AST_5005 | Asset not deleted | The asset could not be deleted due to an internal error |
| 400 | AST_5006 | Invalid payload | The multipart form data was not found in the request |
| 400 | AST_5007 | Invalid payload | No files were found in the multipart form data of the request |
| 400 | AST_5008 | IDs required | One or more IDs are required in the request |
| 400 | AST_5009 | Invalid IDs | The provided IDs are invalid or malformed |
| 400 | AST_5010 | Invalid request | The request is invalid or malformed |
| 400 | AST_5011 | Title required | The title field is required |
| 400 | AST_5012 | Invalid title | The provided title is invalid or malformed |
| 500 | AST_5013 | Failed to open file for storage | The file could not be opened for storage due to an internal error |
| 500 | AST_5014 | Failed to upload file to storage | The file could not be uploaded to storage due to an internal error |
| 400 | AST_5015 | File too large | The uploaded file exceeds the maximum allowed size |
| 400 | AST_5016 | One or more files too large | One or more uploaded files exceed the maximum allowed size |
| 400 | AST_5017 | Too many files | The number of uploaded files exceeds the allowed limit |
| 400 | AST_5018 | Virus detected | The uploaded file contains a virus and was rejected |
| 400 | AST_5019 | Invalid file format | Only image files are allowed |

---

## Tenant Management Codes

### Success Codes

| HTTP Code | Code | UI Message | Dev Message (Description) |
|-----------|------|------------|---------------------------|
| 201 | TNT_SUCCESS_3001 | Tenant created successfully | The tenant has been created successfully |
| 200 | TNT_SUCCESS_3002 | Tenant updated successfully | The tenant has been updated successfully |
| 200 | TNT_SUCCESS_3003 | Tenant deleted successfully | The tenant has been deleted successfully |
| 201 | TNT_SUCCESS_3004 | Tenant subscription created successfully | The tenant subscription has been created successfully |
| 201 | TNT_SUCCESS_3005 | Tenant membership created successfully | The tenant membership has been created successfully |
| 200 | TNT_SUCCESS_3006 | Tenant fetched successfully | The tenant has been fetched successfully |

### Error Codes

| HTTP Code | Code | UI Message | Dev Message (Description) |
|-----------|------|------------|---------------------------|
| 404 | TNT_3001 | Tenant not found | The specified tenant could not be found |
| 409 | TNT_3002 | Tenant already exists | A tenant with the given information already exists |
| 500 | TNT_3003 | Tenant not created | The tenant could not be created due to an internal error |
| 500 | TNT_3004 | Tenant not updated | The tenant could not be updated due to an internal error |
| 500 | TNT_3005 | Tenant not deleted | The tenant could not be deleted due to an internal error |
| 500 | TNT_3006 | Tenant subscription not created | The tenant subscription could not be created due to an internal error |
| 500 | TNT_3007 | Tenant membership not created | The tenant membership could not be created due to an internal error |
| 404 | TNT_3008 | Tenant subscription not found | The specified tenant subscription could not be found |
| 404 | TNT_3009 | Tenant membership not found | The specified tenant membership could not be found |

---

## Role Management Codes

### Success Codes

| HTTP Code | Code | UI Message | Dev Message (Description) |
|-----------|------|------------|---------------------------|
| 201 | ROL_SUCCESS_4001 | Role created successfully | The role has been created successfully |
| 200 | ROL_SUCCESS_4002 | Role updated successfully | The role has been updated successfully |
| 200 | ROL_SUCCESS_4003 | Role deleted successfully | The role has been deleted successfully |

### Error Codes

| HTTP Code | Code | UI Message | Dev Message (Description) |
|-----------|------|------------|---------------------------|
| 404 | ROL_4001 | Role not found | The specified role could not be found |
| 409 | ROL_4002 | Role already exists | A role with the given information already exists |
| 500 | ROL_4003 | Role not created | The role could not be created due to an internal error |
| 500 | ROL_4004 | Role not updated | The role could not be updated due to an internal error |
| 500 | ROL_4005 | Role not deleted | The role could not be deleted due to an internal error |
| 400 | ROL_4006 | Role is required | A role must be provided in the request |
| 400 | ROL_4007 | Invalid role | The specified role is not valid |

---

## RBAC Management Codes

### Error Codes

| HTTP Code | Code | UI Message | Dev Message (Description) |
|-----------|------|------------|---------------------------|
| 404 | RBAC_ERR_7001 | role not found | The requested role does not exist |
| 409 | RBAC_ERR_7002 | role already exists | A role with this name already exists in the system |
| 500 | RBAC_ERR_7003 | failed to delete role | An error occurred while deleting the role |
| 500 | RBAC_ERR_7004 | failed to update role | An error occurred while updating the role |
| 400 | RBAC_ERR_7005 | invalid role priority value | Role priority must be a valid number |
| 400 | RBAC_ERR_7006 | failed to assign role to user | An error occurred while assigning the role to the user |
| 400 | RBAC_ERR_7007 | failed to remove role from user | An error occurred while removing the role from the user |
| 404 | RBAC_ERR_7008 | resource not found | The requested resource does not exist |
| 409 | RBAC_ERR_7009 | resource already exists | A resource with this code already exists |
| 500 | RBAC_ERR_7010 | failed to create resource | An error occurred while creating the resource |
| 500 | RBAC_ERR_7011 | failed to delete resource | An error occurred while deleting the resource |
| 400 | RBAC_ERR_7012 | invalid resource code | Resource code must be alphanumeric and unique |
| 404 | RBAC_ERR_7013 | action not found | The requested action does not exist |
| 409 | RBAC_ERR_7014 | action already exists | An action with this code already exists |
| 500 | RBAC_ERR_7015 | failed to create action | An error occurred while creating the action |
| 500 | RBAC_ERR_7016 | failed to delete action | An error occurred while deleting the action |
| 400 | RBAC_ERR_7017 | invalid action code | Action code must be alphanumeric and unique |
| 404 | RBAC_ERR_7018 | permission not found | The requested permission does not exist |
| 409 | RBAC_ERR_7019 | permission already exists | This resource-action combination already exists |
| 500 | RBAC_ERR_7020 | failed to create permission | An error occurred while creating the permission |
| 500 | RBAC_ERR_7021 | failed to delete permission | An error occurred while deleting the permission |
| 400 | RBAC_ERR_7022 | invalid resource-action combination | The provided resource and action combination is invalid |
| 404 | RBAC_ERR_7023 | role permission mapping not found | The role permission mapping does not exist |
| 409 | RBAC_ERR_7024 | role permission mapping already exists | This role already has this permission |
| 500 | RBAC_ERR_7025 | failed to create role permission | An error occurred while assigning permission to role |
| 500 | RBAC_ERR_7026 | failed to delete role permission | An error occurred while removing permission from role |
| 404 | RBAC_ERR_7027 | access member record not found | The user does not have a role in the specified scope |
| 409 | RBAC_ERR_7028 | user already has this role in the scope | The user already has this role assignment |
| 400 | RBAC_ERR_7029 | failed to assign role to user | An error occurred while assigning the role to the user |
| 400 | RBAC_ERR_7030 | failed to remove role from user | An error occurred while removing the role from the user |
| 400 | RBAC_ERR_7031 | invalid access scope | The access scope type is invalid |
| 400 | RBAC_ERR_7032 | scope ID is required for workspace or base scope | Scope ID must be provided for workspace and base level scopes |
| 403 | RBAC_ERR_7033 | user does not have access to this scope | The user does not have access to the specified scope |
| 403 | RBAC_ERR_7034 | user does not have permission to perform this action | The user lacks the required permission for this operation |
| 403 | RBAC_ERR_7035 | access denied | You do not have access to this resource |
| 403 | RBAC_ERR_7036 | insufficient privileges for this operation | Your current role does not have sufficient privileges |
| 400 | RBAC_ERR_7037 | failed to assign roles to one or more users | One or more role assignments failed during the bulk operation |
| 400 | RBAC_ERR_7038 | failed to remove roles from one or more users | One or more role removals failed during the bulk operation |
| 400 | RBAC_ERR_7039 | user list cannot be empty for bulk operations | At least one user must be specified for bulk operations |
| 400 | RBAC_ERR_7040 | invalid scope type. Must be 'system', 'workspace', or 'base' | Valid scope types are: system, workspace, base |
| 404 | RBAC_ERR_7041 | scope not found | The specified scope does not exist |

---

## Subscription Plan Codes

### Success Codes

| HTTP Code | Code | UI Message | Dev Message (Description) |
|-----------|------|------------|---------------------------|
| 201 | SUB_SUCCESS_5001 | Subscription plan created successfully | The subscription plan has been created successfully |
| 200 | SUB_SUCCESS_5002 | Subscription plan updated successfully | The subscription plan has been updated successfully |
| 200 | SUB_SUCCESS_5003 | Subscription plan deleted successfully | The subscription plan has been deleted successfully |

### Error Codes

| HTTP Code | Code | UI Message | Dev Message (Description) |
|-----------|------|------------|---------------------------|
| 404 | SUB_5001 | Subscription plan not found | The specified subscription plan could not be found |
| 409 | SUB_5002 | Subscription plan already exists | A subscription plan with the given information already exists |
| 500 | SUB_5003 | Subscription plan not created | The subscription plan could not be created due to an internal error |
| 500 | SUB_5004 | Subscription plan not updated | The subscription plan could not be updated due to an internal error |
| 500 | SUB_5005 | Subscription plan not deleted | The subscription plan could not be deleted due to an internal error |

---

## Core Error Codes

These are generic error codes that can be used across all modules.

| HTTP Code | Code | UI Message | Dev Message (Description) |
|-----------|------|------------|---------------------------|
| 400 | ERR_0001 | Invalid ID | The provided ID is invalid |
| 401 | ERR_0002 | Unauthorized access | Unauthorized access |
| 403 | ERR_0003 | Forbidden | Forbidden |
| 401 | ERR_0004 | Unauthorized access | Session expired |
| 400 | ERR_0005 | Bad request | Invalid request payload |
| 422 | ERR_0006 | Validation failed | Validation failed |
| 500 | ERR_0007 | Database error | Database error |
| 404 | ERR_0008 | Record not found | Record not found |
| 500 | ERR_0009 | Internal server error | Internal server error |
| 503 | ERR_0010 | Service unavailable | Service unavailable |
| 504 | ERR_0011 | Gateway timeout | Gateway timeout |
| 429 | ERR_0012 | Too many requests | Too many requests |
| 409 | ERR_0013 | User already exists | User already exists |
| 409 | ERR_0014 | Record already exists | Record already exists |
| 500 | ERR_0015 | Record not inserted | Record not inserted |
| 400 | ERR_0016 | Bad request | Bad request |
| 409 | ERR_0017 | Conflict | Conflict |
| 501 | ERR_0018 | Not implemented | Not implemented |
| 408 | ERR_0019 | Timeout | Timeout |
| 424 | ERR_0020 | Dependency failed | Dependency failed |
| 500 | ERR_0021 | Map to struct error | Map to struct error |
| 500 | ERR_0022 | Struct to struct error | Struct to struct error |
| 500 | ERR_0023 | Hashing error | Hashing error |
| 401 | ERR_0024 | Invalid credentials | Invalid credentials |
| 400 | ERR_0025 | Invalid driver | Invalid driver |
| 404 | ERR_0026 | Not found | Not found |
| 500 | ERR_0027 | JSON marshal error | JSON marshal error |
| 500 | ERR_0028 | HTTP request creation error | HTTP request creation error |
| 500 | ERR_0029 | HTTP do request error | HTTP do request error |
| 403 | ERR_0030 | User not active | User not active |
| 404 | ERR_1001 | File not found | File not found |
| 409 | ERR_1002 | File already exists | File already exists |
| 500 | ERR_1003 | File read failed | File read failed |
| 500 | ERR_1004 | File write failed | File write failed |
| 500 | ERR_1005 | File delete failed | File delete failed |
| 403 | ERR_1006 | File permission denied | File permission denied |
| 400 | ERR_1007 | File invalid path | File invalid path |
| 404 | ERR_1008 | Folder not found | Folder not found |
| 409 | ERR_1009 | Folder already exists | Folder already exists |
| 500 | ERR_1010 | Folder create failed | Folder create failed |
| 500 | ERR_1011 | Folder delete failed | Folder delete failed |
| 403 | ERR_1012 | Folder permission denied | Folder permission denied |
| 400 | ERR_1013 | Folder invalid path | Folder invalid path |
| 400 | ERR_1014 | Invalid date of birth | Invalid date of birth |
| 500 | ERR_1015 | Role creation failed | Role creation failed |
| 404 | ERR_1016 | Subscription plan not found | Subscription plan not found |
| 404 | ERR_1017 | Role not found | Role not found |
| 500 | ERR_1018 | User disable failed | User disable failed |
| 500 | ERR_1019 | Invalid workspace member data | Invalid workspace member data |
| 401 | ERR_1020 | User context not found | User context not found |

---

## Response Structure

All API responses follow a standardized format:

### Success Response
```json
{
  "success": true,
  "message": "User created successfully",
  "data": { ... },
  "meta": {
    "code": "USR_SUCCESS_2001",
    "http_status": 201
  }
}
```

### Error Response
```json
{
  "success": false,
  "error": {
    "code": "USR_2016",
    "message": "Email is required"
  },
  "meta": {
    "code": "USR_2016",
    "http_status": 400
  }
}
```

---

## Notes

1. **HTTP Code**: The standard HTTP status code returned by the API
2. **Code**: A unique identifier for the specific success/error condition
3. **UI Message**: User-friendly message suitable for display in the user interface
4. **Dev Message (Description)**: More detailed technical description for developers and logs

## Code Prefixes

- **AUTH_**: Authentication related codes
- **USR_**: User management codes
- **WSP_**: Workspace management codes
- **TNT_**: Tenant management codes
- **ROL_**: Role management codes
- **AST_**: Asset management codes
- **TBL_**: Table management codes
- **SUB_**: Subscription plan codes
- **RBAC_**: Role-Based Access Control codes
- **ERR_**: Core/Generic error codes

## HTTP Status Code Reference

- **200**: OK - Request succeeded
- **201**: Created - Resource created successfully
- **400**: Bad Request - Invalid request parameters
- **401**: Unauthorized - Authentication required or failed
- **403**: Forbidden - Insufficient permissions
- **404**: Not Found - Resource not found
- **408**: Request Timeout - Request took too long
- **409**: Conflict - Resource already exists
- **422**: Unprocessable Entity - Validation failed
- **424**: Failed Dependency - Dependent operation failed
- **429**: Too Many Requests - Rate limit exceeded
- **500**: Internal Server Error - Server error
- **501**: Not Implemented - Feature not implemented
- **503**: Service Unavailable - Service temporarily unavailable
- **504**: Gateway Timeout - Upstream service timeout



