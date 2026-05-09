package models

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	crudale "github.com/netlify/gotrue/crud/auditlogentries"
	"github.com/netlify/gotrue/storage"
	"github.com/pkg/errors"
)

type AuditAction string
type auditLogType string

const (
	LoginAction                 AuditAction = "login"
	LogoutAction                AuditAction = "logout"
	InviteAcceptedAction        AuditAction = "invite_accepted"
	UserSignedUpAction          AuditAction = "user_signedup"
	UserInvitedAction           AuditAction = "user_invited"
	UserDeletedAction           AuditAction = "user_deleted"
	UserModifiedAction          AuditAction = "user_modified"
	UserRecoveryRequestedAction AuditAction = "user_recovery_requested"
	TokenRevokedAction          AuditAction = "token_revoked"
	TokenRefreshedAction        AuditAction = "token_refreshed"

	account auditLogType = "account"
	team    auditLogType = "team"
	token   auditLogType = "token"
	user    auditLogType = "user"
)

var actionLogTypeMap = map[AuditAction]auditLogType{
	LoginAction:                 account,
	LogoutAction:                account,
	InviteAcceptedAction:        account,
	UserSignedUpAction:          team,
	UserInvitedAction:           team,
	UserDeletedAction:           team,
	TokenRevokedAction:          token,
	TokenRefreshedAction:        token,
	UserModifiedAction:          user,
	UserRecoveryRequestedAction: user,
}

// AuditLogEntry is the database model for audit log entries.
type AuditLogEntry struct {
	ID         int64     `json:"id"`
	InstanceID int64     `json:"-"`
	Payload    JSONMap   `json:"payload"`
	CreatedAt  time.Time `json:"created_at"`
}

// NewAuditLogEntry creates and persists a new audit log entry.
func NewAuditLogEntry(tx *storage.Connection, instanceID int64, actor *User, action AuditAction, traits map[string]interface{}) error {
	payload := JSONMap{
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"actor_id":    actor.ID,
		"actor_email": actor.Email,
		"action":      action,
		"log_type":    actionLogTypeMap[action],
	}

	if name, ok := actor.UserMetaData["full_name"]; ok {
		payload["actor_name"] = name
	}

	if traits != nil {
		payload["traits"] = traits
	}

	payloadData, err := json.Marshal(payload)
	if err != nil {
		return errors.Wrap(err, "marshalling audit log payload")
	}

	now := time.Now()
	entry := &crudale.AuditLogEntries{
		InstanceId: instanceID,
		Payload:    string(payloadData),
		CreatedAt:  now,
	}

	ctx := context.Background()
	_, err = crudale.Create(tx.DB()).SetAuditLogEntries(entry).Save(ctx)
	return errors.Wrap(err, "database error creating audit log entry")
}

// FindAuditLogEntries returns audit log entries for the given instance with optional filter and pagination.
func FindAuditLogEntries(tx *storage.Connection, instanceID int64, filterColumns []string, filterValue string, pageParams *Pagination) ([]*AuditLogEntry, error) {
	ctx := context.Background()

	baseWhere := "instance_id = ?"
	baseArgs := []interface{}{instanceID}

	if len(filterColumns) > 0 && filterValue != "" {
		lf := "%" + filterValue + "%"
		builder := bytes.NewBufferString("(")
		values := make([]interface{}, len(filterColumns))
		for idx, col := range filterColumns {
			builder.WriteString(fmt.Sprintf("payload->>'$.%s' COLLATE utf8mb4_unicode_ci LIKE ?", col))
			values[idx] = lf
			if idx+1 < len(filterColumns) {
				builder.WriteString(" OR ")
			}
		}
		builder.WriteString(")")
		baseWhere += " AND " + builder.String()
		baseArgs = append(baseArgs, values...)
	}

	var limitClause string
	if pageParams != nil {
		countQuery := "SELECT COUNT(*) FROM audit_log_entries WHERE " + baseWhere
		rows, err := tx.DB().QueryContext(ctx, countQuery, baseArgs...)
		if err != nil {
			return nil, errors.Wrap(err, "error counting audit log entries")
		}
		var count int64
		if rows.Next() {
			_ = rows.Scan(&count)
		}
		rows.Close()
		pageParams.Count = uint64(count)
		limitClause = fmt.Sprintf(" LIMIT %d OFFSET %d", pageParams.PerPage, pageParams.Offset())
	}

	query := "SELECT id, instance_id, payload, created_at FROM audit_log_entries WHERE " + baseWhere + " ORDER BY created_at DESC" + limitClause
	rows, err := tx.DB().QueryContext(ctx, query, baseArgs...)
	if err != nil {
		return nil, errors.Wrap(err, "error finding audit log entries")
	}
	defer rows.Close()

	var logs []*AuditLogEntry
	for rows.Next() {
		var e AuditLogEntry
		var payloadStr string
		if err := rows.Scan(&e.ID, &e.InstanceID, &payloadStr, &e.CreatedAt); err != nil {
			return nil, errors.Wrap(err, "error scanning audit log entry")
		}
		if payloadStr != "" {
			_ = json.Unmarshal([]byte(payloadStr), &e.Payload)
		}
		logs = append(logs, &e)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "error iterating audit log entries")
	}

	// Fix for empty filter columns: build a proper query from the selector
	_ = strings.Join // used indirectly above
	return logs, nil
}
