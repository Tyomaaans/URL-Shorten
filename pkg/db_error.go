package pkg

import (
    "errors"
    "fmt"
    "net"

    "github.com/lib/pq"
    "github.com/jackc/pgx/v5/pgconn"
    "gorm.io/gorm"
)

// PostgreSQL error codes
// https://www.postgresql.org/docs/current/errcodes-appendix.html
const (
    // Class 08 — Connection
    PgErrConnectionException    = "08000"
    PgErrConnectionDoesNotExist = "08003"
    PgErrConnectionFailure      = "08006"

    // Class 22 — Data Exception
    PgErrDataException              = "22000"
    PgErrNumericValueOutOfRange     = "22003"
    PgErrStringDataRightTruncation  = "22001"
    PgErrDivisionByZero             = "22012"
    PgErrInvalidTextRepresentation  = "22P02"

    // Class 23 — Integrity Constraint Violation
    PgErrIntegrityConstraintViolation = "23000"
    PgErrRestrictViolation            = "23001"
    PgErrNotNullViolation             = "23502"
    PgErrForeignKeyViolation          = "23503"
    PgErrUniqueViolation              = "23505"
    PgErrCheckViolation               = "23514"
    PgErrExclusionViolation           = "23P01"

    // Class 40 — Transaction Rollback
    PgErrTransactionRollback  = "40000"
    PgErrSerializationFailure = "40001"
    PgErrDeadlockDetected     = "40P01"

    // Class 42 — Syntax Error or Access Rule Violation
    PgErrUndefinedTable        = "42P01"
    PgErrUndefinedColumn       = "42703"
    PgErrInsufficientPrivilege = "42501"

    // Class 53 — Insufficient Resources
    PgErrTooManyConnections = "53300"
    PgErrDiskFull           = "53100"
    PgErrOutOfMemory        = "53200"

    // Class 55 — Object Not In Prerequisite State
    PgErrObjectNotInPrerequisiteState = "55000"
    PgErrLockNotAvailable             = "55P03"

    // Class 57 — Operator Intervention
    PgErrQueryCanceled  = "57014"
    PgErrAdminShutdown  = "57P01"
    PgErrCrashShutdown  = "57P02"
)

// DB infrastructure errors
var (
    ErrAlreadyExists         = errors.New("record already exists")
    ErrNotFound              = errors.New("record not found")
    ErrForeignKeyViolation   = errors.New("related record not found or still referenced")
    ErrNotNullViolation      = errors.New("required field is missing")
    ErrCheckViolation        = errors.New("value does not satisfy constraint")
    ErrDataTooLong           = errors.New("data exceeds maximum length")
    ErrInvalidDataFormat     = errors.New("invalid data format")
    ErrDeadlock              = errors.New("deadlock detected, please retry")
    ErrSerializationFailure  = errors.New("transaction conflict, please retry")
    ErrConnectionFailed      = errors.New("database connection failed")
    ErrTooManyConnections    = errors.New("database is overloaded, too many connections")
    ErrQueryCanceled         = errors.New("query was canceled due to timeout")
    ErrInsufficientPrivilege = errors.New("insufficient database privilege")
    ErrUndefinedTable        = errors.New("table does not exist")
    ErrUndefinedColumn       = errors.New("column does not exist")
    ErrInternal              = errors.New("internal database error")
)

// HandleDBError maps driver and GORM errors to domain errors to prevent leakage.
func HandleDBError(err error) error {
    if err == nil {
        return nil
    }

    if errors.Is(err, gorm.ErrRecordNotFound) {
        return ErrNotFound
    }
    if errors.Is(err, gorm.ErrDuplicatedKey) {
        return ErrAlreadyExists
    }
    if errors.Is(err, gorm.ErrForeignKeyViolated) {
        return ErrForeignKeyViolation
    }
    if errors.Is(err, gorm.ErrCheckConstraintViolated) {
        return ErrCheckViolation
    }

    var pgErr *pgconn.PgError
    if errors.As(err, &pgErr) {
        return handlePGError(pgErr)
    }

    var pqErr *pq.Error
    if errors.As(err, &pqErr) {
        return handlePQError(pqErr)
    }

    var netErr *net.OpError
    if errors.As(err, &netErr) {
        return fmt.Errorf("%w: %s", ErrConnectionFailed, netErr.Op)
    }

    return fmt.Errorf("%w: %s", ErrInternal, err.Error())
}

func handlePGError(err *pgconn.PgError) error {
    switch err.Code {
    case PgErrUniqueViolation:
        if err.ConstraintName != "" {
            return fmt.Errorf("%w: constraint=%s", ErrAlreadyExists, err.ConstraintName)
        }
        return ErrAlreadyExists

    case PgErrForeignKeyViolation:
        if err.ConstraintName != "" {
            return fmt.Errorf("%w: constraint=%s", ErrForeignKeyViolation, err.ConstraintName)
        }
        return ErrForeignKeyViolation

    case PgErrNotNullViolation:
        if err.ColumnName != "" {
            return fmt.Errorf("%w: column=%s", ErrNotNullViolation, err.ColumnName)
        }
        return ErrNotNullViolation

    case PgErrCheckViolation:
        if err.ConstraintName != "" {
            return fmt.Errorf("%w: constraint=%s", ErrCheckViolation, err.ConstraintName)
        }
        return ErrCheckViolation

    case PgErrDeadlockDetected:
        return ErrDeadlock

    case PgErrSerializationFailure:
        return ErrSerializationFailure

    case PgErrTooManyConnections:
        return ErrTooManyConnections

    case PgErrQueryCanceled:
        return ErrQueryCanceled

    case PgErrInsufficientPrivilege:
        return ErrInsufficientPrivilege

    default:
        return fmt.Errorf("%w: pg_code=%s msg=%s", ErrInternal, err.Code, err.Message)
    }
}

func handlePQError(err *pq.Error) error {
    code := string(err.Code)

    switch code {
    case PgErrUniqueViolation:
        if err.Constraint != "" {
            return fmt.Errorf("%w: constraint=%s", ErrAlreadyExists, err.Constraint)
        }
        return ErrAlreadyExists

    case PgErrForeignKeyViolation:
        if err.Constraint != "" {
            return fmt.Errorf("%w: constraint=%s", ErrForeignKeyViolation, err.Constraint)
        }
        return ErrForeignKeyViolation

    case PgErrNotNullViolation:
        if err.Column != "" {
            return fmt.Errorf("%w: column=%s", ErrNotNullViolation, err.Column)
        }
        return ErrNotNullViolation

    case PgErrCheckViolation:
        if err.Constraint != "" {
            return fmt.Errorf("%w: constraint=%s", ErrCheckViolation, err.Constraint)
        }
        return ErrCheckViolation

    case PgErrExclusionViolation:
        return fmt.Errorf("%w: exclusion constraint=%s", ErrCheckViolation, err.Constraint)

    case PgErrStringDataRightTruncation:
        return fmt.Errorf("%w: column=%s", ErrDataTooLong, err.Column)

    case PgErrInvalidTextRepresentation, PgErrDataException:
        return fmt.Errorf("%w: %s", ErrInvalidDataFormat, err.Detail)

    case PgErrNumericValueOutOfRange:
        return fmt.Errorf("%w: numeric out of range column=%s", ErrInvalidDataFormat, err.Column)

    case PgErrDivisionByZero:
        return fmt.Errorf("%w: division by zero", ErrInvalidDataFormat)

    case PgErrDeadlockDetected:
        return ErrDeadlock

    case PgErrSerializationFailure:
        return ErrSerializationFailure

    case PgErrTransactionRollback:
        return fmt.Errorf("%w: transaction was rolled back", ErrInternal)

    case PgErrConnectionException, PgErrConnectionDoesNotExist, PgErrConnectionFailure:
        return fmt.Errorf("%w: %s", ErrConnectionFailed, err.Message)

    case PgErrTooManyConnections:
        return ErrTooManyConnections

    case PgErrDiskFull:
        return fmt.Errorf("%w: disk full", ErrInternal)

    case PgErrOutOfMemory:
        return fmt.Errorf("%w: out of memory", ErrInternal)

    case PgErrQueryCanceled:
        return ErrQueryCanceled

    case PgErrAdminShutdown, PgErrCrashShutdown:
        return fmt.Errorf("%w: server is shutting down", ErrConnectionFailed)

    case PgErrInsufficientPrivilege:
        return ErrInsufficientPrivilege

    case PgErrUndefinedTable:
        return fmt.Errorf("%w: %s", ErrUndefinedTable, err.Message)

    case PgErrUndefinedColumn:
        return fmt.Errorf("%w: %s", ErrUndefinedColumn, err.Message)

    case PgErrLockNotAvailable:
        return fmt.Errorf("%w: lock not available, try again", ErrDeadlock)

    default:
        return fmt.Errorf("%w: pg_code=%s msg=%s", ErrInternal, code, err.Message)
    }
}