package auth_test

import (
	"context"
	"mime/multipart"
	"serenibase/internal/dto"
	"serenibase/internal/models/tenant"
	authProviderInterface "serenibase/internal/providers/auth"
	emailProvider "serenibase/internal/providers/email"
	"serenibase/internal/services/interfaces"
	"time"

	"github.com/google/uuid"
)

type userManagementServiceMock struct {
	GetUserProfileByIDFn   func(ctx context.Context, schema string, userID string) (dto.UserResponse, error)
	UpdateUserProfileFn    func(ctx context.Context, schema string, userID string, updateData dto.UpdateUserProfileRequest) (dto.UserResponse, error)
	UpdatePasswordFn       func(ctx context.Context, schema string, userID string, updateData dto.UpdateUserPasswordRequest) (tenant.User, error)
	AddAvatarFn            func(ctx context.Context, schema string, userID string, fileHeader *multipart.FileHeader) (dto.UserResponse, error)
	RemoveAvatarFn         func(ctx context.Context, schema string, userID string) (dto.UserResponse, error)
	GetUserByEmailFn       func(ctx context.Context, schema string, email string) (tenant.User, error)
	CreateUserFn           func(ctx context.Context, schema string, req dto.RegisterRequest) (tenant.User, error)
	UpdateUserFn           func(ctx context.Context, schema string, id string, updateData map[string]interface{}) (tenant.User, error)
	GetUserByIDFn          func(ctx context.Context, schema string, id string) (tenant.User, error)
	GetAllUsersFn          func(ctx context.Context, schema string) ([]tenant.User, error)
	GetWorkspacesFn        func(ctx context.Context, schema string, userID string, roles string) ([]dto.UserWorkspaceResponse, error)
	GetBulkUsersFn         func(ctx context.Context, schema string, ids []string) ([]tenant.User, error)
	GetUsersWithRoleFn     func(ctx context.Context, schema string) ([]dto.UserWithRole, error)
	GetActiveUsersForAssignFn func(ctx context.Context, schema string) ([]dto.UserWithRole, error)
	DeleteUserCompletelyFn func(ctx context.Context, schema string, userID string) error
	GetUserAccessDetailsFn func(ctx context.Context, schema string, userID string, roles string, workspaceID string) (dto.UserAccessDetailsResponse, error)
	GetUserRolesAndAccessFn func(ctx context.Context, schema string, userID string, scopeID *string) ([]dto.UserRolesAccessResponse, error)
}

func (m *userManagementServiceMock) GetUserProfileByID(ctx context.Context, schema string, userID string) (dto.UserResponse, error) {
	if m.GetUserProfileByIDFn != nil {
		return m.GetUserProfileByIDFn(ctx, schema, userID)
	}
	return dto.UserResponse{}, nil
}

func (m *userManagementServiceMock) UpdateUserProfile(ctx context.Context, schema string, userID string, updateData dto.UpdateUserProfileRequest) (dto.UserResponse, error) {
	if m.UpdateUserProfileFn != nil {
		return m.UpdateUserProfileFn(ctx, schema, userID, updateData)
	}
	return dto.UserResponse{}, nil
}

func (m *userManagementServiceMock) UpdatePassword(ctx context.Context, schema string, userID string, updateData dto.UpdateUserPasswordRequest) (tenant.User, error) {
	if m.UpdatePasswordFn != nil {
		return m.UpdatePasswordFn(ctx, schema, userID, updateData)
	}
	return tenant.User{}, nil
}

func (m *userManagementServiceMock) AddAvatar(ctx context.Context, schema string, userID string, fileHeader *multipart.FileHeader) (dto.UserResponse, error) {
	if m.AddAvatarFn != nil {
		return m.AddAvatarFn(ctx, schema, userID, fileHeader)
	}
	return dto.UserResponse{}, nil
}

func (m *userManagementServiceMock) RemoveAvatar(ctx context.Context, schema string, userID string) (dto.UserResponse, error) {
	if m.RemoveAvatarFn != nil {
		return m.RemoveAvatarFn(ctx, schema, userID)
	}
	return dto.UserResponse{}, nil
}

func (m *userManagementServiceMock) GetUserByEmail(ctx context.Context, schema string, email string) (tenant.User, error) {
	if m.GetUserByEmailFn != nil {
		return m.GetUserByEmailFn(ctx, schema, email)
	}
	return tenant.User{}, nil
}

func (m *userManagementServiceMock) CreateUser(ctx context.Context, schema string, req dto.RegisterRequest) (tenant.User, error) {
	if m.CreateUserFn != nil {
		return m.CreateUserFn(ctx, schema, req)
	}
	return tenant.User{}, nil
}

func (m *userManagementServiceMock) UpdateUser(ctx context.Context, schema string, id string, updateData map[string]interface{}) (tenant.User, error) {
	if m.UpdateUserFn != nil {
		return m.UpdateUserFn(ctx, schema, id, updateData)
	}
	return tenant.User{}, nil
}

func (m *userManagementServiceMock) GetUserByID(ctx context.Context, schema string, id string) (tenant.User, error) {
	if m.GetUserByIDFn != nil {
		return m.GetUserByIDFn(ctx, schema, id)
	}
	return tenant.User{}, nil
}

func (m *userManagementServiceMock) GetAllUsers(ctx context.Context, schema string) ([]tenant.User, error) {
	if m.GetAllUsersFn != nil {
		return m.GetAllUsersFn(ctx, schema)
	}
	return nil, nil
}

func (m *userManagementServiceMock) GetWorkspaces(ctx context.Context, schema string, userID string, roles string) ([]dto.UserWorkspaceResponse, error) {
	if m.GetWorkspacesFn != nil {
		return m.GetWorkspacesFn(ctx, schema, userID, roles)
	}
	return nil, nil
}

func (m *userManagementServiceMock) GetBulkUsers(ctx context.Context, schema string, ids []string) ([]tenant.User, error) {
	if m.GetBulkUsersFn != nil {
		return m.GetBulkUsersFn(ctx, schema, ids)
	}
	return nil, nil
}

func (m *userManagementServiceMock) GetUsersWithRole(ctx context.Context, schema string) ([]dto.UserWithRole, error) {
	if m.GetUsersWithRoleFn != nil {
		return m.GetUsersWithRoleFn(ctx, schema)
	}
	return nil, nil
}

func (m *userManagementServiceMock) GetActiveUsersForAssign(ctx context.Context, schema string) ([]dto.UserWithRole, error) {
	if m.GetActiveUsersForAssignFn != nil {
		return m.GetActiveUsersForAssignFn(ctx, schema)
	}
	return nil, nil
}

func (m *userManagementServiceMock) DeleteUserCompletely(ctx context.Context, schema string, userID string) error {
	if m.DeleteUserCompletelyFn != nil {
		return m.DeleteUserCompletelyFn(ctx, schema, userID)
	}
	return nil
}

func (m *userManagementServiceMock) GetUserAccessDetails(ctx context.Context, schema string, userID string, roles string, workspaceID string) (dto.UserAccessDetailsResponse, error) {
	if m.GetUserAccessDetailsFn != nil {
		return m.GetUserAccessDetailsFn(ctx, schema, userID, roles, workspaceID)
	}
	return dto.UserAccessDetailsResponse{}, nil
}

func (m *userManagementServiceMock) GetUserRolesAndAccess(ctx context.Context, schema string, userID string, scopeID *string) ([]dto.UserRolesAccessResponse, error) {
	if m.GetUserRolesAndAccessFn != nil {
		return m.GetUserRolesAndAccessFn(ctx, schema, userID, scopeID)
	}
	return nil, nil
}

type workspaceManagementServiceMock struct {
	CreateFn                    func(ctx context.Context, req dto.CreateWorkspaceRequest, schemaName string, userId string) (dto.WorkspaceResponse, error)
	GetByIDFn                   func(ctx context.Context, schemaName string, id string) (tenant.Workspace, error)
	GetAllFn                    func(ctx context.Context, schemaName string) ([]tenant.Workspace, error)
	UpdateFn                    func(ctx context.Context, schemaName string, id string, req dto.WorkspaceUpdate, userId string) (tenant.Workspace, error)
	DeleteFn                    func(ctx context.Context, schemaName string, id string) error
	GetTablesByWorkspaceIdFn    func(ctx context.Context, schemaName string, workspaceID string) ([]dto.TableResponse, error)
	GetBasesByWorkspaceIdFn     func(ctx context.Context, schemaName string, workspaceMemberData *tenant.WorkspaceMember) ([]tenant.Base, error)
	GetAllBasesByWorkspaceIdFn  func(ctx context.Context, schemaName string, workspaceID string, role string, userID string) ([]dto.BaseResponse, error)
	GetWorkspaceMemberByUserFn  func(ctx context.Context, schemaName string, userID string) ([]tenant.WorkspaceMember, error)
	GetWorkspaceMembersFn       func(ctx context.Context, schemaName string, workspaceID string) ([]tenant.WorkspaceMember, error)
	GetBulkWorkspacesFn         func(ctx context.Context, schemaName string, workspaceIDs []string) ([]tenant.Workspace, error)
	GetWorkspaceBaseMembersFn   func(ctx context.Context, schemaName string, baseID string) ([]tenant.WorkspaceMember, error)
	DeleteUserMappingsFn        func(ctx context.Context, schemaName string, userID string) error
	UpdateWorkspaceMemberBasesFn func(ctx context.Context, schemaName string, workspaceID string, userID string, accessLevel string, basesIds string) error
	RemoveUserFromWorkspaceFn   func(ctx context.Context, schemaName string, workspaceID string, userID string) error
}

func (m *workspaceManagementServiceMock) Create(ctx context.Context, req dto.CreateWorkspaceRequest, schemaName string, userId string) (dto.WorkspaceResponse, error) {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, req, schemaName, userId)
	}
	return dto.WorkspaceResponse{}, nil
}

func (m *workspaceManagementServiceMock) GetByID(ctx context.Context, schemaName string, id string) (tenant.Workspace, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, schemaName, id)
	}
	return tenant.Workspace{}, nil
}

func (m *workspaceManagementServiceMock) GetAll(ctx context.Context, schemaName string) ([]tenant.Workspace, error) {
	if m.GetAllFn != nil {
		return m.GetAllFn(ctx, schemaName)
	}
	return nil, nil
}

func (m *workspaceManagementServiceMock) Update(ctx context.Context, schemaName string, id string, req dto.WorkspaceUpdate, userId string) (tenant.Workspace, error) {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, schemaName, id, req, userId)
	}
	return tenant.Workspace{}, nil
}

func (m *workspaceManagementServiceMock) Delete(ctx context.Context, schemaName string, id string) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, schemaName, id)
	}
	return nil
}

func (m *workspaceManagementServiceMock) GetTablesByWorkspaceId(ctx context.Context, schemaName string, workspaceID string) ([]dto.TableResponse, error) {
	if m.GetTablesByWorkspaceIdFn != nil {
		return m.GetTablesByWorkspaceIdFn(ctx, schemaName, workspaceID)
	}
	return nil, nil
}

func (m *workspaceManagementServiceMock) GetBasesByWorkspaceId(ctx context.Context, schemaName string, workspaceMemberData *tenant.WorkspaceMember) ([]tenant.Base, error) {
	if m.GetBasesByWorkspaceIdFn != nil {
		return m.GetBasesByWorkspaceIdFn(ctx, schemaName, workspaceMemberData)
	}
	return nil, nil
}

func (m *workspaceManagementServiceMock) GetAllBasesByWorkspaceId(ctx context.Context, schemaName string, workspaceID string, role string, userID string) ([]dto.BaseResponse, error) {
	if m.GetAllBasesByWorkspaceIdFn != nil {
		return m.GetAllBasesByWorkspaceIdFn(ctx, schemaName, workspaceID, role, userID)
	}
	return nil, nil
}

func (m *workspaceManagementServiceMock) GetWorkspaceMemberByUser(ctx context.Context, schemaName string, userID string) ([]tenant.WorkspaceMember, error) {
	if m.GetWorkspaceMemberByUserFn != nil {
		return m.GetWorkspaceMemberByUserFn(ctx, schemaName, userID)
	}
	return nil, nil
}

func (m *workspaceManagementServiceMock) GetWorkspaceMembers(ctx context.Context, schemaName string, workspaceID string) ([]tenant.WorkspaceMember, error) {
	if m.GetWorkspaceMembersFn != nil {
		return m.GetWorkspaceMembersFn(ctx, schemaName, workspaceID)
	}
	return nil, nil
}

func (m *workspaceManagementServiceMock) GetBulkWorkspaces(ctx context.Context, schemaName string, workspaceIDs []string) ([]tenant.Workspace, error) {
	if m.GetBulkWorkspacesFn != nil {
		return m.GetBulkWorkspacesFn(ctx, schemaName, workspaceIDs)
	}
	return nil, nil
}

func (m *workspaceManagementServiceMock) GetWorkspaceBaseMembers(ctx context.Context, schemaName string, baseID string) ([]tenant.WorkspaceMember, error) {
	if m.GetWorkspaceBaseMembersFn != nil {
		return m.GetWorkspaceBaseMembersFn(ctx, schemaName, baseID)
	}
	return nil, nil
}

func (m *workspaceManagementServiceMock) DeleteUserMappings(ctx context.Context, schemaName string, userID string) error {
	if m.DeleteUserMappingsFn != nil {
		return m.DeleteUserMappingsFn(ctx, schemaName, userID)
	}
	return nil
}

func (m *workspaceManagementServiceMock) UpdateWorkspaceMemberBases(ctx context.Context, schemaName string, workspaceID string, userID string, accessLevel string, basesIds string) error {
	if m.UpdateWorkspaceMemberBasesFn != nil {
		return m.UpdateWorkspaceMemberBasesFn(ctx, schemaName, workspaceID, userID, accessLevel, basesIds)
	}
	return nil
}

func (m *workspaceManagementServiceMock) RemoveUserFromWorkspace(ctx context.Context, schemaName string, workspaceID string, userID string) error {
	if m.RemoveUserFromWorkspaceFn != nil {
		return m.RemoveUserFromWorkspaceFn(ctx, schemaName, workspaceID, userID)
	}
	return nil
}

type userResetTokenServiceMock struct {
	CreateUserResetTokenFn func(ctx context.Context, req dto.UserResetTokenInsertion) (tenant.UserResetToken, error)
	GetUserResetTokenFn    func(ctx context.Context, token string) (tenant.UserResetToken, error)
	DeleteTokensByUserIdFn func(ctx context.Context, userId string) error
}

func (m *userResetTokenServiceMock) CreateUserResetToken(ctx context.Context, req dto.UserResetTokenInsertion) (tenant.UserResetToken, error) {
	if m.CreateUserResetTokenFn != nil {
		return m.CreateUserResetTokenFn(ctx, req)
	}
	return tenant.UserResetToken{}, nil
}

func (m *userResetTokenServiceMock) GetUserResetToken(ctx context.Context, token string) (tenant.UserResetToken, error) {
	if m.GetUserResetTokenFn != nil {
		return m.GetUserResetTokenFn(ctx, token)
	}
	return tenant.UserResetToken{}, nil
}

func (m *userResetTokenServiceMock) DeleteTokensByUserId(ctx context.Context, userId string) error {
	if m.DeleteTokensByUserIdFn != nil {
		return m.DeleteTokensByUserIdFn(ctx, userId)
	}
	return nil
}

type rbacManagementServiceMock struct {
	MockRBACManagementServiceUM
	GetRoleByNameFn        func(ctx context.Context, schemaName string, name string) (tenant.AccessRole, error)
	GetRoleByIDFn          func(ctx context.Context, schemaName string, roleID uuid.UUID) (tenant.AccessRole, error)
	AssignRoleToUserFn     func(ctx context.Context, schemaName string, req dto.AccessMemberDTO) (interface{}, error)
	GetUserAccessMembersFn func(ctx context.Context, schemaName string, userID string) ([]dto.AccessMemberDTO, error)
	ProcessUserMembershipsFn func(ctx context.Context, schema string, userID string, assignedBy string, memberships []dto.MembershipRequest) (interface{}, error)
}

func (m *rbacManagementServiceMock) GetRoleByName(ctx context.Context, schemaName string, name string) (tenant.AccessRole, error) {
	if m.GetRoleByNameFn != nil {
		return m.GetRoleByNameFn(ctx, schemaName, name)
	}
	return m.MockRBACManagementServiceUM.GetRoleByName(ctx, schemaName, name)
}

func (m *rbacManagementServiceMock) GetRoleByID(ctx context.Context, schemaName string, roleID uuid.UUID) (tenant.AccessRole, error) {
	if m.GetRoleByIDFn != nil {
		return m.GetRoleByIDFn(ctx, schemaName, roleID)
	}
	return m.MockRBACManagementServiceUM.GetRoleByID(ctx, schemaName, roleID)
}

func (m *rbacManagementServiceMock) AssignRoleToUser(ctx context.Context, schemaName string, req dto.AccessMemberDTO) (interface{}, error) {
	if m.AssignRoleToUserFn != nil {
		return m.AssignRoleToUserFn(ctx, schemaName, req)
	}
	return m.MockRBACManagementServiceUM.AssignRoleToUser(ctx, schemaName, req)
}

func (m *rbacManagementServiceMock) GetUserAccessMembers(ctx context.Context, schemaName string, userID string) ([]dto.AccessMemberDTO, error) {
	if m.GetUserAccessMembersFn != nil {
		return m.GetUserAccessMembersFn(ctx, schemaName, userID)
	}
	return m.MockRBACManagementServiceUM.GetUserAccessMembers(ctx, schemaName, userID)
}

func (m *rbacManagementServiceMock) ProcessUserMemberships(ctx context.Context, schema string, userID string, assignedBy string, memberships []dto.MembershipRequest) (interface{}, error) {
	if m.ProcessUserMembershipsFn != nil {
		return m.ProcessUserMembershipsFn(ctx, schema, userID, assignedBy, memberships)
	}
	return m.MockRBACManagementServiceUM.ProcessUserMemberships(ctx, schema, userID, assignedBy, memberships)
}

type otpServiceMock struct {
	GenerateFn func(identifier string) string
	VerifyFn   func(identifier, input string) bool
}

func (m *otpServiceMock) StartCleanup(interval time.Duration) {}
func (m *otpServiceMock) StopCleanup()                         {}

func (m *otpServiceMock) Generate(identifier string) string {
	if m.GenerateFn != nil {
		return m.GenerateFn(identifier)
	}
	return ""
}

func (m *otpServiceMock) Verify(identifier, input string) bool {
	if m.VerifyFn != nil {
		return m.VerifyFn(identifier, input)
	}
	return false
}

type emailTemplateServiceMock struct {
	EmailVerificationOTPBodyFn func(otp string) emailProvider.EmailContent
	PasswordResetBodyFn        func(resetLink string) emailProvider.EmailContent
	PlatformInvitationBodyFn   func(firstName, tenantName, resetLink string) emailProvider.EmailContent
	AddedToWorkspaceBodyFn     func(workspaceName, access string) emailProvider.EmailContent
	RemovedFromWorkspaceBodyFn func(workspaceLabel string) emailProvider.EmailContent
	InvitedToWorkspaceBodyFn   func(workspaceName, access string) emailProvider.EmailContent
	WorkspaceAccessUpdatedBodyFn func(workspaceName, access string) emailProvider.EmailContent
}

func (m *emailTemplateServiceMock) EmailVerificationOTPBody(otp string) emailProvider.EmailContent {
	if m.EmailVerificationOTPBodyFn != nil {
		return m.EmailVerificationOTPBodyFn(otp)
	}
	return emailProvider.EmailContent{Subject: "otp", Body: "otp"}
}

func (m *emailTemplateServiceMock) PasswordResetBody(resetLink string) emailProvider.EmailContent {
	if m.PasswordResetBodyFn != nil {
		return m.PasswordResetBodyFn(resetLink)
	}
	return emailProvider.EmailContent{Subject: "reset", Body: "reset"}
}

func (m *emailTemplateServiceMock) PlatformInvitationBody(firstName, tenantName, resetLink string) emailProvider.EmailContent {
	if m.PlatformInvitationBodyFn != nil {
		return m.PlatformInvitationBodyFn(firstName, tenantName, resetLink)
	}
	return emailProvider.EmailContent{Subject: "invite", Body: "invite"}
}

func (m *emailTemplateServiceMock) AddedToWorkspaceBody(workspaceName, access string) emailProvider.EmailContent {
	if m.AddedToWorkspaceBodyFn != nil {
		return m.AddedToWorkspaceBodyFn(workspaceName, access)
	}
	return emailProvider.EmailContent{Subject: "added", Body: "added"}
}

func (m *emailTemplateServiceMock) RemovedFromWorkspaceBody(workspaceLabel string) emailProvider.EmailContent {
	if m.RemovedFromWorkspaceBodyFn != nil {
		return m.RemovedFromWorkspaceBodyFn(workspaceLabel)
	}
	return emailProvider.EmailContent{Subject: "removed", Body: "removed"}
}

func (m *emailTemplateServiceMock) InvitedToWorkspaceBody(workspaceName, access string) emailProvider.EmailContent {
	if m.InvitedToWorkspaceBodyFn != nil {
		return m.InvitedToWorkspaceBodyFn(workspaceName, access)
	}
	return emailProvider.EmailContent{Subject: "invited", Body: "invited"}
}

func (m *emailTemplateServiceMock) WorkspaceAccessUpdatedBody(workspaceName, access string) emailProvider.EmailContent {
	if m.WorkspaceAccessUpdatedBodyFn != nil {
		return m.WorkspaceAccessUpdatedBodyFn(workspaceName, access)
	}
	return emailProvider.EmailContent{Subject: "updated", Body: "updated"}
}

type emailServiceMock struct {
	EnqueueFn func(job emailProvider.EmailJob)
}

func (m *emailServiceMock) Start(workers int) {}
func (m *emailServiceMock) Stop()             {}

func (m *emailServiceMock) Enqueue(job emailProvider.EmailJob) {
	if m.EnqueueFn != nil {
		m.EnqueueFn(job)
	}
}

type authProviderMock struct {
	GenerateTokenFn func(ctx context.Context, user tenant.User) (authProviderInterface.Tokens, error)
	RefreshTokenFn  func(ctx context.Context, token string) (authProviderInterface.Tokens, error)
	ValidateTokenFn func(ctx context.Context, tokenStr string) (authProviderInterface.Claims, error)
}

func (m *authProviderMock) GenerateToken(ctx context.Context, user tenant.User) (authProviderInterface.Tokens, error) {
	if m.GenerateTokenFn != nil {
		return m.GenerateTokenFn(ctx, user)
	}
	return authProviderInterface.Tokens{}, nil
}

func (m *authProviderMock) RefreshToken(ctx context.Context, token string) (authProviderInterface.Tokens, error) {
	if m.RefreshTokenFn != nil {
		return m.RefreshTokenFn(ctx, token)
	}
	return authProviderInterface.Tokens{}, nil
}

func (m *authProviderMock) ValidateToken(ctx context.Context, tokenStr string) (authProviderInterface.Claims, error) {
	if m.ValidateTokenFn != nil {
		return m.ValidateTokenFn(ctx, tokenStr)
	}
	return authProviderInterface.Claims{}, nil
}

var _ interfaces.UserManagementService = (*userManagementServiceMock)(nil)
var _ interfaces.WorkspaceManagementService = (*workspaceManagementServiceMock)(nil)
var _ interfaces.UserResetTokenService = (*userResetTokenServiceMock)(nil)
var _ interfaces.RBACManagementService = (*rbacManagementServiceMock)(nil)
