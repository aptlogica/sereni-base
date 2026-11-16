package transfer_test

import (
	"context"
	"io"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/aptlogica/sereni-base/internal/services/transfer/adapter"
	"github.com/aptlogica/sereni-base/internal/services/transfer/adapter/sbase"
)

// writeArchive writes an export with data as the SereniBase adapter does: base.sbase.json and the files as a zip.
// The core service only builds the model and reads the stored files (it is the adapter's FileSource).
func writeArchive(svc interfaces.BaseTransferService, file *dto.BaseExportFile, w io.Writer) error {
	return sbase.New().Export(context.Background(), file, adapter.ExportOptions{IncludeData: true, Files: svc}, w)
}
