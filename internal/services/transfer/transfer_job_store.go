// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"

	"github.com/aptlogica/go-postgres-rest/pkg"
	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/internal/utils/helpers"
)

// TransferJobStore keeps the export jobs. The database implements it; tests use a fake.
type TransferJobStore interface {
	Create(ctx context.Context, schemaName string, job tenant.TransferJob) error
	// Get returns app_errors.TransferJobNotFound for a job that does not exist
	Get(ctx context.Context, schemaName, id string) (tenant.TransferJob, error)
	Update(ctx context.Context, schemaName, id string, fields map[string]interface{}) error
	List(ctx context.Context, schemaName string) ([]tenant.TransferJob, error)
	Delete(ctx context.Context, schemaName, id string) error
}

type dbTransferJobStore struct {
	repo *pkg.DatabaseService
}

// NewDBTransferJobStore keeps the export jobs in the transfer_jobs table
func NewDBTransferJobStore(repo *pkg.DatabaseService) TransferJobStore {
	return &dbTransferJobStore{repo: repo}
}

func (d *dbTransferJobStore) Create(_ context.Context, schemaName string, job tenant.TransferJob) error {
	row := map[string]interface{}{
		"id":                 job.ID,
		"type":               jobType(job),
		"base_title":         job.BaseTitle,
		"user_id":            job.UserID,
		"status":             job.Status,
		"created_time":       job.CreatedAt,
		"last_modified_time": job.UpdatedAt,
	}
	// an import has no base yet and an export has no workspace or uploaded file
	if job.BaseID != "" {
		row["base_id"] = job.BaseID
	}
	if job.WorkspaceID != "" {
		row["workspace_id"] = job.WorkspaceID
	}
	if job.FileName != "" {
		row["file_name"] = job.FileName
	}
	if job.SizeBytes > 0 {
		row["size_bytes"] = job.SizeBytes
	}
	_, err := d.repo.TableService.CreateRecord(tenant.TransferJob{}.TableName(schemaName), row)
	return err
}

func (d *dbTransferJobStore) Get(_ context.Context, schemaName, id string) (tenant.TransferJob, error) {
	limit := 1
	rows, err := d.repo.TableService.GetTableData(tenant.TransferJob{}.TableName(schemaName), dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{{Column: "id", Operator: "eq", Value: id}},
		Limit:   &limit,
	})
	if err != nil {
		return tenant.TransferJob{}, err
	}
	if len(rows) == 0 {
		return tenant.TransferJob{}, app_errors.TransferJobNotFound
	}
	return transferJobFromRow(rows[0])
}

func (d *dbTransferJobStore) Update(_ context.Context, schemaName, id string, fields map[string]interface{}) error {
	_, err := d.repo.TableService.UpdateRecord(tenant.TransferJob{}.TableName(schemaName), id, fields)
	return err
}

func (d *dbTransferJobStore) List(_ context.Context, schemaName string) ([]tenant.TransferJob, error) {
	rows, err := d.repo.TableService.GetTableData(tenant.TransferJob{}.TableName(schemaName), dbModels.QueryParams{})
	if err != nil {
		return nil, err
	}
	jobs := make([]tenant.TransferJob, 0, len(rows))
	for _, row := range rows {
		job, err := transferJobFromRow(row)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func (d *dbTransferJobStore) Delete(_ context.Context, schemaName, id string) error {
	return d.repo.TableService.DeleteRecord(tenant.TransferJob{}.TableName(schemaName), id)
}

func transferJobFromRow(row map[string]interface{}) (tenant.TransferJob, error) {
	var job tenant.TransferJob
	if err := helpers.MapToStruct(row, &job); err != nil {
		return tenant.TransferJob{}, app_errors.ErrMapToStruct
	}
	return job, nil
}
