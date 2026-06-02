// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package interfaces

import (
	"context"
	"mime/multipart"

	"github.com/aptlogica/sereni-base/internal/dto"
)

type ImportService interface {
	ImportWithConfig(ctx context.Context, schemaName string, req dto.ImportWithConfigRequest, file *multipart.FileHeader, tableTitle string) (dto.ImportTableResponse, error)
	FetchAiSchema(ctx context.Context, prompt string) (dto.AiTableResponse, error)
	ApplyAiSchema(ctx context.Context, schemaName string, req dto.CreateTableRequest, aiResponse dto.AiTableResponse, sample bool, rows int) (dto.ImportTableResponse, error)
}
