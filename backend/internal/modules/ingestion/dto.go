package ingestion

import (
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/db"
)

type Upload struct {
	FileName string
	Size     int64
	Content  io.Reader
	UserId   *uuid.UUID
}

type ImportResult struct {
	BatchId              uuid.UUID  `json:"batchId"`
	Rows                 int        `json:"rows"`
	Inserted             int64      `json:"inserted"`
	Skipped              int64      `json:"skipped"`
	MetersCreated        int        `json:"metersCreated"`
	MetersRestored       int        `json:"metersRestored"`
	PreviouslyImportedAt *time.Time `json:"previouslyImportedAt"`
}

type BatchResponse struct {
	Id             uuid.UUID       `json:"id"`
	NumId          int32           `json:"numId"`
	Kind           db.ImportKind   `json:"kind"`
	Status         db.ImportStatus `json:"status"`
	FileName       string          `json:"fileName"`
	FileSize       int64           `json:"fileSize"`
	Checksum       string          `json:"checksum"`
	Rows           int32           `json:"rows"`
	Inserted       int32           `json:"inserted"`
	Skipped        int32           `json:"skipped"`
	MetersCreated  int32           `json:"metersCreated"`
	MetersRestored int32           `json:"metersRestored"`
	ErrorCode      *string         `json:"errorCode"`
	Error          *string         `json:"error"`
	UserName       *string         `json:"userName"`
	CreatedAt      time.Time       `json:"createdAt"`
}

type ListResponse struct {
	Data []BatchResponse `json:"data"`
}

func toBatchResponses(rows []db.ListImportBatchesRow) []BatchResponse {
	responses := make([]BatchResponse, 0, len(rows))

	for _, row := range rows {
		responses = append(responses, BatchResponse{
			Id:             row.UidImportBatch,
			NumId:          row.NumIDImportBatch,
			Kind:           row.StrKindImportBatch,
			Status:         row.StrStatusImportBatch,
			FileName:       row.StrFileNameImportBatch,
			FileSize:       row.NumFileSizeImportBatch,
			Checksum:       row.StrChecksumImportBatch,
			Rows:           row.NumRowsImportBatch,
			Inserted:       row.NumInsertedImportBatch,
			Skipped:        row.NumSkippedImportBatch,
			MetersCreated:  row.NumMetersCreatedImportBatch,
			MetersRestored: row.NumMetersRestoredImportBatch,
			ErrorCode:      row.StrErrorCodeImportBatch,
			Error:          row.StrErrorImportBatch,
			UserName:       row.StrNameUser,
			CreatedAt:      row.DtmCreatedAt.UTC(),
		})
	}

	return responses
}
