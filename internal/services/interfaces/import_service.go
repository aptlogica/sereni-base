// Copyright (c) 2026 Aptlogica Technologies Private Limited
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
	Import(ctx context.Context, schemaName string, req dto.CreateTableRequest, file *multipart.FileHeader) (dto.ImportTableResponse, error)
}
