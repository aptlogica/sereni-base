package interfaces

import (
	"context"
	"mime/multipart"
	"serenibase/internal/dto"
)

type ImportService interface {
	Import(ctx context.Context, schemaName string, req dto.CreateTableRequest, file *multipart.FileHeader) (dto.ImportTableResponse, error)
}
