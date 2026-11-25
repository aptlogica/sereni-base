package services

import (
	"context"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
)

// Remove deletes a finished job from the list: the row, and the zip of an export. A job that is still queued or
// running is refused, since it is still at work.
func (s *transferJobService) Remove(ctx context.Context, schemaName, jobID, userID string) error {
	job, err := s.owned(ctx, schemaName, jobID, userID)
	if err != nil {
		return err
	}
	if isActive(job) {
		return app_errors.TransferJobNotReady
	}
	if jobType(job) == tenant.TransferJobTypeExport {
		if err := s.deleteExportFile(ctx, job); err != nil {
			return err
		}
	}
	return s.store.Delete(ctx, schemaName, jobID)
}
