# SereniBase Role-Based Access Control (RBAC) Guide

This document provides a complete mapping of all API endpoints, their required permissions, and which roles have access.

---

## Table of Contents

1. [Roles Overview](#roles-overview)
2. [Permissions (Actions)](#permissions-actions)
3. [Resources](#resources)
4. [Role Permission Matrix](#role-permission-matrix)
5. [API Access by Role](#api-access-by-role)
6. [Complete API Reference](#complete-api-reference)

---

## Roles Overview

| Role Name              | Code                 | Scope Level | Priority | Description                                                    |
|------------------------|----------------------|-------------|----------|----------------------------------------------------------------|
| Owner                  | `owner`              | System      | 100      | Full access over the entire application (system-level superuser) |
| CoOwner                | `co-owner`           | System      | 90       | Full access similar to Owner, but not primary system owner      |
| WorkspaceMaintainer    | `maintainer`         | Workspace   | 80       | Owner of a workspace, manages only that workspace               |
| WorkspaceMaintainerRO  | `workspace-read`     | Workspace   | 70       | Read-only maintainer of a workspace                             |
| BaseMember             | `base-member`        | Base        | 60       | Standard member with read/write permissions in a base           |
| BaseMemberReadOnly     | `base-read`          | Base        | 50       | Member with read-only permissions in a base                     |
| NoAccess               | `user`               | System      | 10       | No workspace access; can only view/edit own profile             |

---

## Permissions (Actions)

| Action Code | Description                              |
|-------------|------------------------------------------|
| `read`      | View/read resources                      |
| `create`    | Create new resources                     |
| `update`    | Update/edit existing resources           |
| `delete`    | Delete resources                         |
| `share`     | Share resources with others              |
| `invite`    | Invite users/members to scope            |
| `export`    | Export data from the system              |
| `import`    | Import data into the system              |
| `execute`   | Execute actions/scripts/automations      |
| `manage`    | Full management control over settings    |

---

## Resources

| Resource Code   | Description                          |
|-----------------|--------------------------------------|
| `workspace`     | Workspace management                 |
| `base`          | Base/database management             |
| `table`         | Table schema management              |
| `records`       | Row/record data management           |
| `members`       | User/member management               |
| `views`         | View management                      |
| `settings`      | System/organization settings         |
| `api_tokens`    | API token management                 |
| `webhooks`      | Webhook management                   |
| `automations`   | Automation/workflow management       |

---

## Role Permission Matrix

### Owner & CoOwner (System Level)

| Resource      | read | create | update | delete | share | invite | export | import | execute | manage |
|---------------|:----:|:------:|:------:|:------:|:-----:|:------:|:------:|:------:|:-------:|:------:|
| workspace     | ✅   | ✅     | ✅     | ✅     | ✅    | ✅     | ✅     | ✅     | ✅      | ✅     |
| base          | ✅   | ✅     | ✅     | ✅     | ✅    | ✅     | ✅     | ✅     | ✅      | ✅     |
| table         | ✅   | ✅     | ✅     | ✅     | ✅    | ✅     | ✅     | ✅     | ✅      | ✅     |
| records       | ✅   | ✅     | ✅     | ✅     | ✅    | ✅     | ✅     | ✅     | ✅      | ✅     |
| members       | ✅   | ✅     | ✅     | ✅     | ✅    | ✅     | ✅     | ✅     | ✅      | ✅     |
| views         | ✅   | ✅     | ✅     | ✅     | ✅    | ✅     | ✅     | ✅     | ✅      | ✅     |
| settings      | ✅   | ✅     | ✅     | ✅     | ✅    | ✅     | ✅     | ✅     | ✅      | ✅     |
| api_tokens    | ✅   | ✅     | ✅     | ✅     | ✅    | ✅     | ✅     | ✅     | ✅      | ✅     |
| webhooks      | ✅   | ✅     | ✅     | ✅     | ✅    | ✅     | ✅     | ✅     | ✅      | ✅     |
| automations   | ✅   | ✅     | ✅     | ✅     | ✅    | ✅     | ✅     | ✅     | ✅      | ✅     |

### WorkspaceMaintainer (Workspace Level)

| Resource      | read | create | update | delete | share | invite | export | import | execute | manage |
|---------------|:----:|:------:|:------:|:------:|:-----:|:------:|:------:|:------:|:-------:|:------:|
| workspace     | ✅   | ❌     | ✅     | ✅     | ✅    | ✅     | ✅     | ✅     | ✅      | ✅     |
| base          | ✅   | ✅     | ✅     | ✅     | ✅    | ✅     | ✅     | ✅     | ✅      | ✅     |
| table         | ✅   | ✅     | ✅     | ✅     | ✅    | ✅     | ✅     | ✅     | ✅      | ✅     |
| records       | ✅   | ✅     | ✅     | ✅     | ✅    | ✅     | ✅     | ✅     | ✅      | ✅     |
| members       | ✅   | ✅     | ✅     | ✅     | ✅    | ✅     | ✅     | ✅     | ✅      | ✅     |
| views         | ✅   | ✅     | ✅     | ✅     | ✅    | ✅     | ✅     | ✅     | ✅      | ✅     |
| settings      | ✅   | ❌     | ✅     | ❌     | ❌    | ❌     | ✅     | ❌     | ❌      | ❌     |

### WorkspaceMaintainerRO (Workspace Level - Read Only)

| Resource      | read | create | update | delete | share | invite | export | import | execute | manage |
|---------------|:----:|:------:|:------:|:------:|:-----:|:------:|:------:|:------:|:-------:|:------:|
| workspace     | ✅   | ❌     | ❌     | ❌     | ❌    | ❌     | ✅     | ❌     | ❌      | ❌     |
| base          | ✅   | ❌     | ❌     | ❌     | ❌    | ❌     | ✅     | ❌     | ❌      | ❌     |
| table         | ✅   | ❌     | ❌     | ❌     | ❌    | ❌     | ✅     | ❌     | ❌      | ❌     |
| records       | ✅   | ❌     | ❌     | ❌     | ❌    | ❌     | ✅     | ❌     | ❌      | ❌     |
| members       | ✅   | ❌     | ❌     | ❌     | ❌    | ❌     | ❌     | ❌     | ❌      | ❌     |
| views         | ✅   | ❌     | ❌     | ❌     | ❌    | ❌     | ✅     | ❌     | ❌      | ❌     |

### BaseMember (Base Level)

| Resource      | read | create | update | delete | share | invite | export | import | execute | manage |
|---------------|:----:|:------:|:------:|:------:|:-----:|:------:|:------:|:------:|:-------:|:------:|
| base          | ✅   | ❌     | ❌     | ❌     | ❌    | ❌     | ✅     | ✅     | ❌      | ❌     |
| table         | ✅   | ✅     | ✅     | ✅     | ❌    | ❌     | ✅     | ✅     | ❌      | ❌     |
| records       | ✅   | ✅     | ✅     | ✅     | ❌    | ❌     | ✅     | ✅     | ❌      | ❌     |
| views         | ✅   | ✅     | ✅     | ✅     | ❌    | ❌     | ✅     | ❌     | ❌      | ❌     |

### BaseMemberReadOnly (Base Level - Read Only)

| Resource      | read | create | update | delete | share | invite | export | import | execute | manage |
|---------------|:----:|:------:|:------:|:------:|:-----:|:------:|:------:|:------:|:-------:|:------:|
| base          | ✅   | ❌     | ❌     | ❌     | ❌    | ❌     | ✅     | ❌     | ❌      | ❌     |
| table         | ✅   | ❌     | ❌     | ❌     | ❌    | ❌     | ✅     | ❌     | ❌      | ❌     |
| records       | ✅   | ❌     | ❌     | ❌     | ❌    | ❌     | ✅     | ❌     | ❌      | ❌     |
| views         | ✅   | ❌     | ❌     | ❌     | ❌    | ❌     | ✅     | ❌     | ❌      | ❌     |

### NoAccess (System Level - Minimal)

| Resource      | read | create | update | delete | share | invite | export | import | execute | manage |
|---------------|:----:|:------:|:------:|:------:|:-----:|:------:|:------:|:------:|:-------:|:------:|
| profile (own) | ✅   | ❌     | ✅     | ❌     | ❌    | ❌     | ❌     | ❌     | ❌      | ❌     |

---

## API Access by Role

### Legend
- 🔓 **Public** - No authentication required
- 👤 **Self** - User can access their own data only
- ✅ **Allowed** - Role has access
- ❌ **Denied** - Role does not have access

---

## Complete API Reference

### 1. Authentication APIs (Public - No Auth Required) 🔓

| Method | Endpoint                    | Action   | Description                          |
|--------|----------------------------|----------|--------------------------------------|
| POST   | `/api/v1/auth/login`       | -        | User login with credentials          |
| POST   | `/api/v1/auth/forgot-password` | -    | Request password reset               |
| POST   | `/api/v1/auth/reset-password`  | -    | Reset password with token            |
| POST   | `/api/v1/auth/validate-token`  | -    | Validate JWT token                   |
| POST   | `/api/v1/auth/verify-token`    | -    | Verify token validity                |
| POST   | `/api/v1/auth/refresh`     | -        | Refresh access token                 |
| POST   | `/api/v1/auth/logout`      | -        | User logout                          |
| POST   | `/api/v1/auth/otp/verify`  | -        | Verify email with OTP                |
| POST   | `/api/v1/auth/otp/resend`  | -        | Resend OTP to email                  |

---

### 2. User Profile APIs (Self-Access)

| Method | Endpoint                        | Scope  | Resource | Action  | Owner | CoOwner | Maintainer | MaintainerRO | BaseMember | BaseMemberRO | NoAccess |
|--------|--------------------------------|--------|----------|---------|:-----:|:-------:|:----------:|:------------:|:----------:|:------------:|:--------:|
| GET    | `/api/v1/user/profile/:id`     | Self   | -        | read    | 👤    | 👤      | 👤         | 👤           | 👤         | 👤           | 👤       |
| PATCH  | `/api/v1/user/profile/:id`     | Self   | -        | update  | 👤    | 👤      | 👤         | 👤           | 👤         | 👤           | 👤       |
| POST   | `/api/v1/user/change-password/:id` | Self | -      | update  | 👤    | 👤      | 👤         | 👤           | 👤         | 👤           | 👤       |
| POST   | `/api/v1/user/profile/:id/avatar`  | Self | -      | update  | 👤    | 👤      | 👤         | 👤           | 👤         | 👤           | 👤       |
| DELETE | `/api/v1/user/profile/:id/avatar`  | Self | -      | delete  | 👤    | 👤      | 👤         | 👤           | 👤         | 👤           | 👤       |
| GET    | `/api/v1/user/workspaces`      | Self   | -        | read    | 👤    | 👤      | 👤         | 👤           | 👤         | 👤           | 👤       |
| GET    | `/api/v1/user/access-details`  | Self   | -        | read    | 👤    | 👤      | 👤         | 👤           | 👤         | 👤           | 👤       |
| GET    | `/api/v1/user/roles-and-access/:id` | Self | -     | read    | 👤    | 👤      | 👤         | 👤           | 👤         | 👤           | 👤       |

---

### 3. User Assignment APIs (Workspace Members)

| Method | Endpoint                        | Scope     | Resource | Action  | Owner | CoOwner | Maintainer | MaintainerRO | BaseMember | BaseMemberRO | NoAccess |
|--------|--------------------------------|-----------|----------|---------|:-----:|:-------:|:----------:|:------------:|:----------:|:------------:|:--------:|
| POST   | `/api/v1/user/assign`          | Workspace | members  | invite  | ✅    | ✅      | ✅         | ❌           | ❌         | ❌           | ❌       |
| PUT    | `/api/v1/user/access/update`   | Workspace | members  | update  | ✅    | ✅      | ✅         | ❌           | ❌         | ❌           | ❌       |

---

### 4. User Admin APIs (System Level)

| Method | Endpoint                        | Scope  | Resource | Action  | Owner | CoOwner | Maintainer | MaintainerRO | BaseMember | BaseMemberRO | NoAccess |
|--------|--------------------------------|--------|----------|---------|:-----:|:-------:|:----------:|:------------:|:----------:|:------------:|:--------:|
| POST   | `/api/v1/user/create`          | System | members  | create  | ✅    | ✅      | ❌         | ❌           | ❌         | ❌           | ❌       |
| POST   | `/api/v1/user/edit`            | System | members  | update  | ✅    | ✅      | ❌         | ❌           | ❌         | ❌           | ❌       |
| POST   | `/api/v1/user/remove`          | System | members  | delete  | ✅    | ✅      | ❌         | ❌           | ❌         | ❌           | ❌       |
| POST   | `/api/v1/user/activate`        | System | members  | update  | ✅    | ✅      | ❌         | ❌           | ❌         | ❌           | ❌       |
| POST   | `/api/v1/user/deactivate`      | System | members  | update  | ✅    | ✅      | ❌         | ❌           | ❌         | ❌           | ❌       |
| GET    | `/api/v1/user/list`            | System | members  | read    | ✅    | ✅      | ❌         | ❌           | ❌         | ❌           | ❌       |
| GET    | `/api/v1/user/list-for-assign` | System | members  | read    | ✅    | ✅      | ❌         | ❌           | ❌         | ❌           | ❌       |

---

### 5. Organization APIs (System Level)

| Method | Endpoint                        | Scope  | Resource | Action  | Owner | CoOwner | Maintainer | MaintainerRO | BaseMember | BaseMemberRO | NoAccess |
|--------|--------------------------------|--------|----------|---------|:-----:|:-------:|:----------:|:------------:|:----------:|:------------:|:--------:|
| GET    | `/api/v1/organization`         | System | settings | read    | ✅    | ✅      | ❌         | ❌           | ❌         | ❌           | ❌       |
| PUT    | `/api/v1/organization/:id`     | System | settings | update  | ✅    | ✅      | ❌         | ❌           | ❌         | ❌           | ❌       |

---

### 6. Workspace APIs

#### System-Level Workspace Operations

| Method | Endpoint                        | Scope  | Resource  | Action  | Owner | CoOwner | Maintainer | MaintainerRO | BaseMember | BaseMemberRO | NoAccess |
|--------|--------------------------------|--------|-----------|---------|:-----:|:-------:|:----------:|:------------:|:----------:|:------------:|:--------:|
| POST   | `/api/v1/workspace/create`     | System | workspace | create  | ✅    | ✅      | ❌         | ❌           | ❌         | ❌           | ❌       |
| GET    | `/api/v1/workspace/`           | System | workspace | read    | ✅    | ✅      | ❌         | ❌           | ❌         | ❌           | ❌       |

#### Workspace-Level Operations

| Method | Endpoint                                   | Scope     | Resource  | Action  | Owner | CoOwner | Maintainer | MaintainerRO | BaseMember | BaseMemberRO | NoAccess |
|--------|-------------------------------------------|-----------|-----------|---------|:-----:|:-------:|:----------:|:------------:|:----------:|:------------:|:--------:|
| GET    | `/api/v1/workspace/:id`                   | Workspace | workspace | read    | ✅    | ✅      | ✅         | ✅           | ❌         | ❌           | ❌       |
| PUT    | `/api/v1/workspace/:id`                   | Workspace | workspace | update  | ✅    | ✅      | ✅         | ❌           | ❌         | ❌           | ❌       |
| DELETE | `/api/v1/workspace/:id`                   | Workspace | workspace | delete  | ✅    | ✅      | ✅         | ❌           | ❌         | ❌           | ❌       |
| GET    | `/api/v1/workspace/:id/tables`            | Workspace | table     | read    | ✅    | ✅      | ✅         | ✅           | ❌         | ❌           | ❌       |
| GET    | `/api/v1/workspace/:id/bases`             | Workspace | base      | read    | ✅    | ✅      | ✅         | ✅           | ❌         | ❌           | ❌       |
| POST   | `/api/v1/workspace/:id/remove`            | Workspace | members   | delete  | ✅    | ✅      | ✅         | ❌           | ❌         | ❌           | ❌       |
| GET    | `/api/v1/workspace/:id/members`           | Workspace | members   | read    | ✅    | ✅      | ✅         | ✅           | ❌         | ❌           | ❌       |
| GET    | `/api/v1/workspace/:id/members-with-roles`| Workspace | members   | read    | ✅    | ✅      | ✅         | ✅           | ❌         | ❌           | ❌       |
| POST   | `/api/v1/workspace/:id/bulk-add-members`  | Workspace | members   | invite  | ✅    | ✅      | ✅         | ❌           | ❌         | ❌           | ❌       |
| DELETE | `/api/v1/workspace/access/:id`            | Workspace | members   | delete  | ✅    | ✅      | ✅         | ❌           | ❌         | ❌           | ❌       |

---

### 7. Base APIs

#### Workspace-Level Base Operations

| Method | Endpoint                        | Scope     | Resource | Action  | Owner | CoOwner | Maintainer | MaintainerRO | BaseMember | BaseMemberRO | NoAccess |
|--------|--------------------------------|-----------|----------|---------|:-----:|:-------:|:----------:|:------------:|:----------:|:------------:|:--------:|
| POST   | `/api/v1/base/create`          | Workspace | base     | create  | ✅    | ✅      | ✅         | ❌           | ❌         | ❌           | ❌       |

#### Base-Level Operations

| Method | Endpoint                              | Scope | Resource | Action  | Owner | CoOwner | Maintainer | MaintainerRO | BaseMember | BaseMemberRO | NoAccess |
|--------|--------------------------------------|-------|----------|---------|:-----:|:-------:|:----------:|:------------:|:----------:|:------------:|:--------:|
| GET    | `/api/v1/base/:id`                   | Base  | base     | read    | ✅    | ✅      | ✅         | ✅           | ✅         | ✅           | ❌       |
| PUT    | `/api/v1/base/:id`                   | Base  | base     | update  | ✅    | ✅      | ✅         | ❌           | ❌         | ❌           | ❌       |
| DELETE | `/api/v1/base/:id`                   | Base  | base     | delete  | ✅    | ✅      | ✅         | ❌           | ❌         | ❌           | ❌       |
| GET    | `/api/v1/base/:id/tables`            | Base  | table    | read    | ✅    | ✅      | ✅         | ✅           | ✅         | ✅           | ❌       |
| POST   | `/api/v1/base/:id/image`             | Base  | base     | update  | ✅    | ✅      | ✅         | ❌           | ❌         | ❌           | ❌       |
| DELETE | `/api/v1/base/:id/image`             | Base  | base     | delete  | ✅    | ✅      | ✅         | ❌           | ❌         | ❌           | ❌       |
| POST   | `/api/v1/base/:id/remove`            | Base  | members  | delete  | ✅    | ✅      | ✅         | ❌           | ❌         | ❌           | ❌       |
| GET    | `/api/v1/base/:id/members`           | Base  | members  | read    | ✅    | ✅      | ✅         | ✅           | ✅         | ✅           | ❌       |
| GET    | `/api/v1/base/:id/members-with-roles`| Base  | members  | read    | ✅    | ✅      | ✅         | ✅           | ✅         | ✅           | ❌       |
| POST   | `/api/v1/base/:id/bulk-add-members`  | Base  | members  | invite  | ✅    | ✅      | ✅         | ❌           | ❌         | ❌           | ❌       |
| DELETE | `/api/v1/base/access/:id`            | Base  | members  | delete  | ✅    | ✅      | ✅         | ❌           | ❌         | ❌           | ❌       |

---

### 8. Table APIs (Base Level)

| Method | Endpoint                        | Scope | Resource | Action  | Owner | CoOwner | Maintainer | MaintainerRO | BaseMember | BaseMemberRO | NoAccess |
|--------|--------------------------------|-------|----------|---------|:-----:|:-------:|:----------:|:------------:|:----------:|:------------:|:--------:|
| POST   | `/api/v1/table/create`         | Base  | table    | create  | ✅    | ✅      | ✅         | ❌           | ✅         | ❌           | ❌       |
| POST   | `/api/v1/table/import`         | Base  | table    | import  | ✅    | ✅      | ✅         | ❌           | ✅         | ❌           | ❌       |
| GET    | `/api/v1/table/`               | Base  | table    | read    | ✅    | ✅      | ✅         | ✅           | ✅         | ✅           | ❌       |
| GET    | `/api/v1/table/:id`            | Base  | table    | read    | ✅    | ✅      | ✅         | ✅           | ✅         | ✅           | ❌       |
| PATCH  | `/api/v1/table/:id`            | Base  | table    | update  | ✅    | ✅      | ✅         | ❌           | ✅         | ❌           | ❌       |
| DELETE | `/api/v1/table/:id`            | Base  | table    | delete  | ✅    | ✅      | ✅         | ❌           | ✅         | ❌           | ❌       |
| GET    | `/api/v1/table/:id/columns`    | Base  | table    | read    | ✅    | ✅      | ✅         | ✅           | ✅         | ✅           | ❌       |
| GET    | `/api/v1/table/:id/views`      | Base  | views    | read    | ✅    | ✅      | ✅         | ✅           | ✅         | ✅           | ❌       |
| GET    | `/api/v1/table/:id/records`    | Base  | records  | read    | ✅    | ✅      | ✅         | ✅           | ✅         | ✅           | ❌       |

---

### 9. Column APIs (Base Level)

| Method | Endpoint                        | Scope | Resource | Action  | Owner | CoOwner | Maintainer | MaintainerRO | BaseMember | BaseMemberRO | NoAccess |
|--------|--------------------------------|-------|----------|---------|:-----:|:-------:|:----------:|:------------:|:----------:|:------------:|:--------:|
| POST   | `/api/v1/column/create`        | Base  | table    | create  | ✅    | ✅      | ✅         | ❌           | ✅         | ❌           | ❌       |
| GET    | `/api/v1/column/`              | Base  | table    | read    | ✅    | ✅      | ✅         | ✅           | ✅         | ✅           | ❌       |
| GET    | `/api/v1/column/:id`           | Base  | table    | read    | ✅    | ✅      | ✅         | ✅           | ✅         | ✅           | ❌       |
| PATCH  | `/api/v1/column/:id`           | Base  | table    | update  | ✅    | ✅      | ✅         | ❌           | ✅         | ❌           | ❌       |
| DELETE | `/api/v1/column/:id`           | Base  | table    | delete  | ✅    | ✅      | ✅         | ❌           | ✅         | ❌           | ❌       |
| POST   | `/api/v1/column/reorder`       | Base  | table    | update  | ✅    | ✅      | ✅         | ❌           | ✅         | ❌           | ❌       |

---

### 10. Row/Record APIs (Base Level)

| Method | Endpoint                        | Scope | Resource | Action  | Owner | CoOwner | Maintainer | MaintainerRO | BaseMember | BaseMemberRO | NoAccess |
|--------|--------------------------------|-------|----------|---------|:-----:|:-------:|:----------:|:------------:|:----------:|:------------:|:--------:|
| POST   | `/api/v1/row/create`           | Base  | records  | create  | ✅    | ✅      | ✅         | ❌           | ✅         | ❌           | ❌       |
| POST   | `/api/v1/row/remove`           | Base  | records  | delete  | ✅    | ✅      | ✅         | ❌           | ✅         | ❌           | ❌       |
| POST   | `/api/v1/row/bulk-remove`      | Base  | records  | delete  | ✅    | ✅      | ✅         | ❌           | ✅         | ❌           | ❌       |
| POST   | `/api/v1/row/data/insert`      | Base  | records  | update  | ✅    | ✅      | ✅         | ❌           | ✅         | ❌           | ❌       |
| POST   | `/api/v1/row/data/relation`    | Base  | records  | update  | ✅    | ✅      | ✅         | ❌           | ✅         | ❌           | ❌       |
| POST   | `/api/v1/row/attachment/add`   | Base  | records  | create  | ✅    | ✅      | ✅         | ❌           | ✅         | ❌           | ❌       |
| POST   | `/api/v1/row/attachment/update`| Base  | records  | update  | ✅    | ✅      | ✅         | ❌           | ✅         | ❌           | ❌       |
| POST   | `/api/v1/row/attachment/remove`| Base  | records  | delete  | ✅    | ✅      | ✅         | ❌           | ✅         | ❌           | ❌       |

---

### 11. View APIs (Base Level)

| Method | Endpoint                        | Scope | Resource | Action  | Owner | CoOwner | Maintainer | MaintainerRO | BaseMember | BaseMemberRO | NoAccess |
|--------|--------------------------------|-------|----------|---------|:-----:|:-------:|:----------:|:------------:|:----------:|:------------:|:--------:|
| POST   | `/api/v1/view/create`          | Base  | views    | create  | ✅    | ✅      | ✅         | ❌           | ✅         | ❌           | ❌       |
| GET    | `/api/v1/view/`                | Base  | views    | read    | ✅    | ✅      | ✅         | ✅           | ✅         | ✅           | ❌       |
| GET    | `/api/v1/view/:id`             | Base  | views    | read    | ✅    | ✅      | ✅         | ✅           | ✅         | ✅           | ❌       |
| PATCH  | `/api/v1/view/:id`             | Base  | views    | update  | ✅    | ✅      | ✅         | ❌           | ✅         | ❌           | ❌       |
| DELETE | `/api/v1/view/:id`             | Base  | views    | delete  | ✅    | ✅      | ✅         | ❌           | ✅         | ❌           | ❌       |

---

### 12. Asset APIs (Base Level)

| Method | Endpoint                        | Scope | Resource | Action  | Owner | CoOwner | Maintainer | MaintainerRO | BaseMember | BaseMemberRO | NoAccess |
|--------|--------------------------------|-------|----------|---------|:-----:|:-------:|:----------:|:------------:|:----------:|:------------:|:--------:|
| POST   | `/api/v1/asset/upload`         | Base  | records  | create  | ✅    | ✅      | ✅         | ❌           | ✅         | ❌           | ❌       |
| POST   | `/api/v1/asset/upload-image`   | Base  | records  | create  | ✅    | ✅      | ✅         | ❌           | ✅         | ❌           | ❌       |
| POST   | `/api/v1/asset/bulk`           | Base  | records  | read    | ✅    | ✅      | ✅         | ✅           | ✅         | ✅           | ❌       |
| PATCH  | `/api/v1/asset/:id`            | Base  | records  | update  | ✅    | ✅      | ✅         | ❌           | ✅         | ❌           | ❌       |
| DELETE | `/api/v1/asset/:id`            | Base  | records  | delete  | ✅    | ✅      | ✅         | ❌           | ✅         | ❌           | ❌       |

---

## Quick Access Summary by Role

### Owner / CoOwner
- **Full system access** to all APIs
- Can create/manage workspaces, users, organization settings
- Can perform all operations at workspace and base levels

### WorkspaceMaintainer
- **Full workspace access** (limited to assigned workspace)
- Can create bases, manage members, and perform all base operations
- Cannot create new workspaces or manage system-level settings/users

### WorkspaceMaintainerRO
- **Read-only workspace access**
- Can view workspace, bases, tables, records, members
- Cannot create, update, or delete any resources

### BaseMember
- **Full base access** (limited to assigned base)
- Can create/update/delete tables, columns, records, views
- Cannot manage base settings, members, or workspace-level resources

### BaseMemberReadOnly
- **Read-only base access**
- Can view tables, columns, records, views
- Cannot create, update, or delete any resources

### NoAccess
- **Profile access only**
- Can view and edit their own profile
- No access to any workspace, base, or organizational resources

---

## HTTP Method to Action Mapping

| HTTP Method | Typical Action |
|-------------|---------------|
| GET         | `read`        |
| POST        | `create` / `execute` / `import` |
| PUT         | `update`      |
| PATCH       | `update`      |
| DELETE      | `delete`      |

---

## Code Usage Examples

### Using Middleware in Routes

```go
import (
    "github.com/gin-gonic/gin"
    "sereni-base/internal/middleware"
)

func SetupRoutes(r *gin.Engine, accessService interfaces.AccessMemberService) {
    // System Admin Only Routes
    adminRoutes := r.Group("/admin")
    adminRoutes.Use(middleware.RequireSystemAdmin())
    {
        adminRoutes.GET("/users", listAllUsers)
        adminRoutes.POST("/system-settings", updateSystemSettings)
    }

    // Workspace Routes
    workspaceRoutes := r.Group("/workspace/:workspaceId")
    workspaceRoutes.Use(middleware.RequireWorkspaceAccess(accessService))
    {
        // Read access (WorkspaceMaintainerRO and above)
        workspaceRoutes.GET("", getWorkspace)
        
        // Write access (WorkspaceMaintainer and above)
        workspaceRoutes.PUT("", middleware.RequireWorkspaceOwner(), updateWorkspace)
        workspaceRoutes.DELETE("", middleware.RequireWorkspaceOwner(), deleteWorkspace)
    }

    // Base Routes with Permission Checks
    baseRoutes := r.Group("/base/:baseId")
    baseRoutes.Use(middleware.RequireBaseAccess(accessService))
    {
        baseRoutes.GET("/records", getRecords)
        baseRoutes.POST("/records", middleware.RoleGuardMiddleware(middleware.RoleGuardConfig{
            MinRole:        constant.BaseMember,
            Resource:       "records",
            RequiredAction: constant.ActionCreate,
        }), createRecord)
    }
}
```

### Permission Checking in Handlers

```go
import "sereni-base/internal/constant"

func createRecordHandler(c *gin.Context) {
    // Get user role from context (set by auth middleware)
    role := c.GetString("user_role")
    
    // Check if role can create records
    if !constant.HasPermission(role, "records", constant.ActionCreate) {
        c.JSON(403, gin.H{"error": "insufficient permissions"})
        return
    }
    
    // Proceed with record creation...
}

func getWorkspaceHandler(c *gin.Context) {
    role := c.GetString("user_role")
    
    // Check if user can read workspace
    if !middleware.CanRead(c) {
        c.JSON(403, gin.H{"error": "read access denied"})
        return
    }
    
    // Return workspace data...
}
```

### Role Priority Comparison

```go
import "sereni-base/internal/constant"

// Check if user1 can manage user2's role
func canManageUser(managerRole, targetRole string) bool {
    return constant.IsHigherOrEqualPriority(managerRole, targetRole)
}

// Example usage
canPromote := canManageUser(constant.WorkspaceMaintainer, constant.BaseMember) // true
canDemote := canManageUser(constant.BaseMemberReadOnly, constant.WorkspaceMaintainer) // false
```

### Getting All Permissions for a Role

```go
import "sereni-base/internal/constant"

func getUserPermissions(role string) {
    permissions := constant.GetRolePermissionsFlat(role)
    
    // Returns slice like:
    // ["workspace:read", "workspace:create", "base:read", "records:read", "records:create", ...]
    
    for _, perm := range permissions {
        fmt.Println(perm)
    }
}
```

---

## Notes

1. **Scope Inheritance**: Higher-level roles automatically have access to lower-level scopes:
   - System roles (Owner, CoOwner) → Access to all workspaces and bases
   - Workspace roles → Access to all bases within that workspace
   - Base roles → Access to that specific base only

2. **Priority Resolution**: When a user has multiple roles, the highest priority role determines access.

3. **Self-Access**: All authenticated users can access their own profile regardless of role.

4. **Authorization Flow**:
   ```
   Request → Auth Middleware → Scope Validation → Action Authorization → Handler
   ```

---

*Last Updated: March 2026*
