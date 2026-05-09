package pb_hooks

import (
	"errors"
	"fmt"
)

var (
	ErrRowRequiredFieldMissing = errors.New("row: required field missing")
	ErrRowValidation           = errors.New("row: validation failed")
	ErrDuplicateRecord         = errors.New("row: duplicate record exists")
	ErrRecordNotFound          = errors.New("row: referenced record not found")
	ErrInvalidData             = errors.New("row: invalid data format")
	ErrSystemError             = errors.New("row: system error")
	ErrInvalidMapping          = errors.New("mapping: invalid configuration")
	ErrInvalidJobState         = errors.New("job: invalid state transition")
)

// ImportError represents a row-level error with context
type ImportError struct {
	RowIndex    int    `json:"row_index"`
	RowNum      int    `json:"row_num"` // Excel row number (1-indexed, includes header)
	Field       string `json:"field,omitempty"`
	Reason      string `json:"reason"`
	RawValue    string `json:"raw_value,omitempty"`
	ErrorCode   string `json:"error_code"`
	IsRecoverable bool  `json:"is_recoverable"`
}

func (ie *ImportError) Error() string {
	return fmt.Sprintf("row %d: %s", ie.RowNum, ie.Reason)
}

// NewImportError creates a new ImportError
func NewImportError(rowIndex, rowNum int, field, reason, rawValue string, isRecoverable bool) *ImportError {
	return &ImportError{
		RowIndex:     rowIndex,
		RowNum:       rowNum,
		Field:        field,
		Reason:       reason,
		RawValue:     rawValue,
		ErrorCode:    extractErrorCode(reason),
		IsRecoverable: isRecoverable,
	}
}

func extractErrorCode(reason string) string {
	switch {
	case reason == "required field missing":
		return "REQUIRED_MISSING"
	case reason == "invalid data format":
		return "INVALID_FORMAT"
	case reason == "duplicate record exists":
		return "DUPLICATE"
	case reason == "referenced record not found":
		return "REF_NOT_FOUND"
	default:
		return "SYSTEM_ERROR"
	}
}
