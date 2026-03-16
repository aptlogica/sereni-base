// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package interfaces

import (
	"context"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
)

type RelationshipService interface {
	Create(ctx context.Context, req dto.RelationInsertion, schemaName string) (tenant.Relation, error)
	GetRelationByID(ctx context.Context, id string, schemaName string) (tenant.Relation, error)
	DeleteRelation(ctx context.Context, relationId string, schemaName string) error
	UpdateRelation(ctx context.Context, relationId string, relationData dto.RelationUpdate, schemaName string) (tenant.Relation, error)
}
