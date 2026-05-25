package models

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/snowflake"
	crudusers "github.com/netlify/gotrue/crud/users"
	"github.com/netlify/gotrue/storage"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

// SystemUserID is the sentinel int64 ID for the system user.
const SystemUserID int64 = 0

// zeroTime is the sentinel "zero" timestamp stored as '2000-01-01 00:00:00' in the DB.
var zeroTime = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

var snowflakeNode *snowflake.Node

func init() {
	var err error
	snowflakeNode, err = snowflake.NewNode(1)
	if err != nil {
		panic(fmt.Sprintf("failed to create snowflake node: %v", err))
	}
}

// User represents a registered user with email/password authentication.
type User struct {
	ID         int64 `json:"id"`
	InstanceID int64 `json:"-"`

	Aud               string    `json:"aud"`
	Role              string    `json:"role"`
	Email             string    `json:"email"`
	EncryptedPassword string    `json:"-"`
	ConfirmedAt       time.Time `json:"confirmed_at,omitempty"`
	InvitedAt         time.Time `json:"invited_at,omitempty"`

	ConfirmationToken  string    `json:"-"`
	ConfirmationSentAt time.Time `json:"confirmation_sent_at,omitempty"`

	RecoveryToken  string    `json:"-"`
	RecoverySentAt time.Time `json:"recovery_sent_at,omitempty"`

	EmailChangeToken  string    `json:"-"`
	EmailChange       string    `json:"new_email,omitempty"`
	EmailChangeSentAt time.Time `json:"email_change_sent_at,omitempty"`

	LastSignInAt time.Time `json:"last_sign_in_at,omitempty"`

	AppMetaData  JSONMap `json:"app_metadata"`
	UserMetaData JSONMap `json:"user_metadata"`

	IsSuperAdmin bool `json:"-"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserForExport is like User but exposes EncryptedPassword for export.
type UserForExport struct {
	ID         int64 `json:"id"`
	InstanceID int64 `json:"-"`

	Aud               string    `json:"aud"`
	Role              string    `json:"role"`
	Email             string    `json:"email"`
	EncryptedPassword string    `json:"encrypted_password"`
	ConfirmedAt       time.Time `json:"confirmed_at,omitempty"`
	InvitedAt         time.Time `json:"invited_at,omitempty"`

	ConfirmationToken  string    `json:"-"`
	ConfirmationSentAt time.Time `json:"confirmation_sent_at,omitempty"`

	RecoveryToken  string    `json:"-"`
	RecoverySentAt time.Time `json:"recovery_sent_at,omitempty"`

	EmailChangeToken  string    `json:"-"`
	EmailChange       string    `json:"new_email,omitempty"`
	EmailChangeSentAt time.Time `json:"email_change_sent_at,omitempty"`

	LastSignInAt time.Time `json:"last_sign_in_at,omitempty"`

	AppMetaData  JSONMap `json:"app_metadata"`
	UserMetaData JSONMap `json:"user_metadata"`

	IsSuperAdmin bool `json:"-"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewUser initializes a new user from email, password, audience and user data.
func NewUser(instanceID int64, email, password, aud string, userData map[string]interface{}) (*User, error) {
	pw, err := hashPassword(password)
	if err != nil {
		return nil, err
	}

	id := snowflakeNode.Generate().Int64()
	user := &User{
		InstanceID:         instanceID,
		ID:                 id,
		Aud:                aud,
		Email:              email,
		UserMetaData:       userData,
		EncryptedPassword:  pw,
		ConfirmedAt:        zeroTime,
		ConfirmationSentAt: zeroTime,
		RecoverySentAt:     zeroTime,
		EmailChangeSentAt:  zeroTime,
		LastSignInAt:       zeroTime,
		InvitedAt:          zeroTime,
	}

	return user, nil
}

// NewSystemUser returns a synthetic system user (not persisted).
func NewSystemUser(instanceID int64, aud string) *User {
	return &User{
		InstanceID:   instanceID,
		ID:           SystemUserID,
		Aud:          aud,
		IsSuperAdmin: true,
	}
}

// validate ensures the user can be persisted.
func (u *User) validate() error {
	if u.ID == SystemUserID {
		return errors.New("Cannot persist system user")
	}
	return nil
}

// IsConfirmed checks if a user has already been registered and confirmed.
func (u *User) IsConfirmed() bool {
	return u.ConfirmedAt.Year() > 2000
}

// HasRole returns true when the user's role matches roleName.
func (u *User) HasRole(roleName string) bool {
	return u.Role == roleName
}

// hashPassword generates a bcrypt hash of a plaintext password.
func hashPassword(password string) (string, error) {
	pw, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(pw), nil
}

// Authenticate verifies a plaintext password against the stored hash.
func (u *User) Authenticate(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.EncryptedPassword), []byte(password))
	return err == nil
}

// -------- CRUD helpers --------

// toUsersCrud converts a models.User to a crud/users.Users struct.
func toUsersCrud(u *User) *crudusers.Users {
	appMeta, _ := json.Marshal(u.AppMetaData)
	userMeta, _ := json.Marshal(u.UserMetaData)

	isSuperAdmin := int64(0)
	if u.IsSuperAdmin {
		isSuperAdmin = 1
	}

	return &crudusers.Users{
		Id:                 u.ID,
		InstanceId:         u.InstanceID,
		Aud:                u.Aud,
		Role:               u.Role,
		Email:              u.Email,
		EncryptedPassword:  u.EncryptedPassword,
		ConfirmedAt:        u.ConfirmedAt,
		InvitedAt:          u.InvitedAt,
		ConfirmationToken:  u.ConfirmationToken,
		ConfirmationSentAt: u.ConfirmationSentAt,
		RecoveryToken:      u.RecoveryToken,
		RecoverySentAt:     u.RecoverySentAt,
		EmailChangeToken:   u.EmailChangeToken,
		EmailChange:        u.EmailChange,
		EmailChangeSentAt:  u.EmailChangeSentAt,
		LastSignInAt:       u.LastSignInAt,
		RawAppMetaData:     string(appMeta),
		RawUserMetaData:    string(userMeta),
		IsSuperAdmin:       isSuperAdmin,
		CreatedAt:          u.CreatedAt,
		UpdatedAt:          u.UpdatedAt,
	}
}

// fromUsersCrud converts a crud/users.Users to a models.User.
func fromUsersCrud(cu *crudusers.Users) *User {
	u := &User{
		ID:                 cu.Id,
		InstanceID:         cu.InstanceId,
		Aud:                cu.Aud,
		Role:               cu.Role,
		Email:              cu.Email,
		EncryptedPassword:  cu.EncryptedPassword,
		ConfirmedAt:        cu.ConfirmedAt,
		InvitedAt:          cu.InvitedAt,
		ConfirmationToken:  cu.ConfirmationToken,
		ConfirmationSentAt: cu.ConfirmationSentAt,
		RecoveryToken:      cu.RecoveryToken,
		RecoverySentAt:     cu.RecoverySentAt,
		EmailChangeToken:   cu.EmailChangeToken,
		EmailChange:        cu.EmailChange,
		EmailChangeSentAt:  cu.EmailChangeSentAt,
		LastSignInAt:       cu.LastSignInAt,
		IsSuperAdmin:       cu.IsSuperAdmin != 0,
		CreatedAt:          cu.CreatedAt,
		UpdatedAt:          cu.UpdatedAt,
	}
	// Unmarshal JSON fields
	if cu.RawAppMetaData != "" {
		_ = json.Unmarshal([]byte(cu.RawAppMetaData), &u.AppMetaData)
	}
	if cu.RawUserMetaData != "" {
		_ = json.Unmarshal([]byte(cu.RawUserMetaData), &u.UserMetaData)
	}
	return u
}

// fromUsersCrudExport converts a crudusers.Users to a UserForExport.
func fromUsersCrudExport(cu *crudusers.Users) *UserForExport {
	u := &UserForExport{
		ID:                 cu.Id,
		InstanceID:         cu.InstanceId,
		Aud:                cu.Aud,
		Role:               cu.Role,
		Email:              cu.Email,
		EncryptedPassword:  cu.EncryptedPassword,
		ConfirmedAt:        cu.ConfirmedAt,
		InvitedAt:          cu.InvitedAt,
		ConfirmationToken:  cu.ConfirmationToken,
		ConfirmationSentAt: cu.ConfirmationSentAt,
		RecoveryToken:      cu.RecoveryToken,
		RecoverySentAt:     cu.RecoverySentAt,
		EmailChangeToken:   cu.EmailChangeToken,
		EmailChange:        cu.EmailChange,
		EmailChangeSentAt:  cu.EmailChangeSentAt,
		LastSignInAt:       cu.LastSignInAt,
		IsSuperAdmin:       cu.IsSuperAdmin != 0,
		CreatedAt:          cu.CreatedAt,
		UpdatedAt:          cu.UpdatedAt,
	}
	if cu.RawAppMetaData != "" {
		_ = json.Unmarshal([]byte(cu.RawAppMetaData), &u.AppMetaData)
	}
	if cu.RawUserMetaData != "" {
		_ = json.Unmarshal([]byte(cu.RawUserMetaData), &u.UserMetaData)
	}
	return u
}

// findUser is the internal helper to query a single user.
func findUser(tx *storage.Connection, wheres ...interface{}) (*User, error) {
	if len(wheres) == 0 {
		return nil, errors.New("findUser requires at least a where clause")
	}
	// wheres[0] is the where func, the rest are additional filters.
	// We accept a variadic of xsql WhereFunc via an adapter approach.
	// For simplicity, we take a query-string + args approach using raw SQL
	// and map the results ourselves. But since the crud package generates WhereFunc
	// constants, let's use a raw SQL approach with QueryContext.
	panic("use named findUser wrappers")
}

func findUserByWheres(tx *storage.Connection, conditions []interface{}) (*User, error) {
	// This won't be called directly - each finder builds the selector inline.
	_ = conditions
	return nil, nil
}

// FindUserByConfirmationToken finds a user with the matching confirmation token.
func FindUserByConfirmationToken(tx *storage.Connection, token string) (*User, error) {
	if strings.TrimSpace(token) == "" {
		return nil, UserNotFoundError{}
	}
	ctx := context.Background()
	cu, err := crudusers.Find(tx.DB()).Where(crudusers.ConfirmationTokenOp.EQ(token)).One(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, UserNotFoundError{}
		}
		return nil, errors.Wrap(err, "error finding user")
	}
	return fromUsersCrud(cu), nil
}

// FindUserByEmailAndAudience finds a user with the matching email and audience.
func FindUserByEmailAndAudience(tx *storage.Connection, instanceID int64, email, aud string) (*User, error) {
	ctx := context.Background()
	cu, err := crudusers.Find(tx.DB()).
		Where(
			crudusers.InstanceIdOp.EQ(instanceID),
			crudusers.EmailOp.EQ(email),
			crudusers.AudOp.EQ(aud),
		).One(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, UserNotFoundError{}
		}
		return nil, errors.Wrap(err, "error finding user")
	}
	return fromUsersCrud(cu), nil
}

// FindUserByID finds a user matching the provided ID.
func FindUserByID(tx *storage.Connection, id int64) (*User, error) {
	ctx := context.Background()
	cu, err := crudusers.Find(tx.DB()).Where(crudusers.IdOp.EQ(id)).One(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, UserNotFoundError{}
		}
		return nil, errors.Wrap(err, "error finding user")
	}
	return fromUsersCrud(cu), nil
}

// FindUserByInstanceIDAndID finds a user matching the instance ID and user ID.
func FindUserByInstanceIDAndID(tx *storage.Connection, instanceID, id int64) (*User, error) {
	ctx := context.Background()
	cu, err := crudusers.Find(tx.DB()).
		Where(
			crudusers.InstanceIdOp.EQ(instanceID),
			crudusers.IdOp.EQ(id),
		).One(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, UserNotFoundError{}
		}
		return nil, errors.Wrap(err, "error finding user")
	}
	return fromUsersCrud(cu), nil
}

// FindUserByRecoveryToken finds a user with the matching recovery token.
func FindUserByRecoveryToken(tx *storage.Connection, token string) (*User, error) {
	if strings.TrimSpace(token) == "" {
		return nil, UserNotFoundError{}
	}
	ctx := context.Background()
	cu, err := crudusers.Find(tx.DB()).Where(crudusers.RecoveryTokenOp.EQ(token)).One(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, UserNotFoundError{}
		}
		return nil, errors.Wrap(err, "error finding user")
	}
	return fromUsersCrud(cu), nil
}

// FindUserWithRefreshToken finds a user from the provided refresh token.
func FindUserWithRefreshToken(tx *storage.Connection, token string) (*User, *RefreshToken, error) {
	rt, err := findRefreshTokenByToken(tx, token)
	if err != nil {
		return nil, nil, err
	}
	user, err := FindUserByID(tx, rt.UserID)
	if err != nil {
		return nil, nil, err
	}
	return user, rt, nil
}

// FindUsersInAudience finds users with the matching audience, with optional pagination/sorting/filter.
func FindUsersInAudience(tx *storage.Connection, instanceID int64, aud string, pageParams *Pagination, sortParams *SortParams, filter string) ([]*User, error) {
	ctx := context.Background()
	sel := crudusers.Find(tx.DB()).
		Where(crudusers.InstanceIdOp.EQ(instanceID), crudusers.AudOp.EQ(aud))

	if filter != "" {
		lf := "%" + filter + "%"
		// Use raw SQL fragment via ExecQuerier for the complex filter.
		// Instead, build with multiple conditions via OR using a raw query approach.
		// We implement this via a subquery pattern using the connection directly.
		_ = lf
		// Apply filter via raw count/select queries below.
		return findUsersInAudienceFiltered(tx, instanceID, aud, pageParams, sortParams, filter)
	}

	if sortParams != nil && len(sortParams.Fields) > 0 {
		for _, field := range sortParams.Fields {
			if field.Dir == Ascending {
				sel = sel.OrderAsc(field.Name)
			} else {
				sel = sel.OrderDesc(field.Name)
			}
		}
	}

	if pageParams != nil {
		// Count
		countSel := crudusers.Find(tx.DB()).
			Where(crudusers.InstanceIdOp.EQ(instanceID), crudusers.AudOp.EQ(aud)).
			Count()
		count, err := countSel.Int64(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "error counting users")
		}
		pageParams.Count = uint64(count)

		sel = sel.Limit(int32(pageParams.PerPage)).Offset(int32(pageParams.Offset()))
	}

	cus, err := sel.All(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error finding users")
	}
	users := make([]*User, len(cus))
	for i, cu := range cus {
		users[i] = fromUsersCrud(cu)
	}
	return users, nil
}

func findUsersInAudienceFiltered(tx *storage.Connection, instanceID int64, aud string, pageParams *Pagination, sortParams *SortParams, filter string) ([]*User, error) {
	ctx := context.Background()
	lf := "%" + filter + "%"

	orderClause := "created_at DESC"
	if sortParams != nil && len(sortParams.Fields) > 0 {
		parts := make([]string, 0, len(sortParams.Fields))
		for _, field := range sortParams.Fields {
			parts = append(parts, field.Name+" "+string(field.Dir))
		}
		orderClause = strings.Join(parts, ", ")
	}

	var limitClause string
	var offset int64
	if pageParams != nil {
		offset = int64(pageParams.Offset())
		// Count
		countQuery := `SELECT COUNT(*) FROM users WHERE instance_id = ? AND aud = ? AND (email LIKE ? OR raw_user_meta_data->>'$.full_name' COLLATE utf8mb4_unicode_ci LIKE ?)`
		rows, err := tx.DB().QueryContext(ctx, countQuery, instanceID, aud, lf, lf)
		if err != nil {
			return nil, errors.Wrap(err, "error counting users")
		}
		var count int64
		if rows.Next() {
			_ = rows.Scan(&count)
		}
		rows.Close()
		pageParams.Count = uint64(count)
		limitClause = fmt.Sprintf(" LIMIT %d OFFSET %d", pageParams.PerPage, offset)
	}

	query := `SELECT id, instance_id, aud, role, email, encrypted_password, confirmed_at, invited_at, confirmation_token, confirmation_sent_at, recovery_token, recovery_sent_at, email_change_token, email_change, email_change_sent_at, last_sign_in_at, raw_app_meta_data, raw_user_meta_data, is_super_admin, created_at, updated_at FROM users WHERE instance_id = ? AND aud = ? AND (email LIKE ? OR raw_user_meta_data->>'$.full_name' COLLATE utf8mb4_unicode_ci LIKE ?) ORDER BY ` + orderClause + limitClause
	rows, err := tx.DB().QueryContext(ctx, query, instanceID, aud, lf, lf)
	if err != nil {
		return nil, errors.Wrap(err, "error finding users")
	}
	defer rows.Close()

	return scanUsersFromRows(rows)
}

// FindUsersForExportInAudience finds users for export with the matching audience.
func FindUsersForExportInAudience(tx *storage.Connection, instanceID int64, aud string, pageParams *Pagination, sortParams *SortParams, filter string) ([]*UserForExport, error) {
	ctx := context.Background()

	orderClause := "created_at DESC"
	if sortParams != nil && len(sortParams.Fields) > 0 {
		parts := make([]string, 0, len(sortParams.Fields))
		for _, field := range sortParams.Fields {
			parts = append(parts, field.Name+" "+string(field.Dir))
		}
		orderClause = strings.Join(parts, ", ")
	}

	baseWhere := "instance_id = ? AND aud = ?"
	baseArgs := []interface{}{instanceID, aud}

	if filter != "" {
		lf := "%" + filter + "%"
		baseWhere += " AND (email LIKE ? OR raw_user_meta_data->>'$.full_name' COLLATE utf8mb4_unicode_ci LIKE ?)"
		baseArgs = append(baseArgs, lf, lf)
	}

	var limitClause string
	if pageParams != nil {
		countQuery := `SELECT COUNT(*) FROM users WHERE ` + baseWhere
		rows, err := tx.DB().QueryContext(ctx, countQuery, baseArgs...)
		if err != nil {
			return nil, errors.Wrap(err, "error counting users")
		}
		var count int64
		if rows.Next() {
			_ = rows.Scan(&count)
		}
		rows.Close()
		pageParams.Count = uint64(count)
		limitClause = fmt.Sprintf(" LIMIT %d OFFSET %d", pageParams.PerPage, pageParams.Offset())
	}

	query := `SELECT id, instance_id, aud, role, email, encrypted_password, confirmed_at, invited_at, confirmation_token, confirmation_sent_at, recovery_token, recovery_sent_at, email_change_token, email_change, email_change_sent_at, last_sign_in_at, raw_app_meta_data, raw_user_meta_data, is_super_admin, created_at, updated_at FROM users WHERE ` + baseWhere + ` ORDER BY ` + orderClause + limitClause
	rows, err := tx.DB().QueryContext(ctx, query, baseArgs...)
	if err != nil {
		return nil, errors.Wrap(err, "error finding users")
	}
	defer rows.Close()

	plainUsers, err := scanUsersFromRows(rows)
	if err != nil {
		return nil, err
	}
	result := make([]*UserForExport, len(plainUsers))
	for i, u := range plainUsers {
		result[i] = &UserForExport{
			ID:                 u.ID,
			InstanceID:         u.InstanceID,
			Aud:                u.Aud,
			Role:               u.Role,
			Email:              u.Email,
			EncryptedPassword:  u.EncryptedPassword,
			ConfirmedAt:        u.ConfirmedAt,
			InvitedAt:          u.InvitedAt,
			ConfirmationToken:  u.ConfirmationToken,
			ConfirmationSentAt: u.ConfirmationSentAt,
			RecoveryToken:      u.RecoveryToken,
			RecoverySentAt:     u.RecoverySentAt,
			EmailChangeToken:   u.EmailChangeToken,
			EmailChange:        u.EmailChange,
			EmailChangeSentAt:  u.EmailChangeSentAt,
			LastSignInAt:       u.LastSignInAt,
			IsSuperAdmin:       u.IsSuperAdmin,
			AppMetaData:        u.AppMetaData,
			UserMetaData:       u.UserMetaData,
			CreatedAt:          u.CreatedAt,
			UpdatedAt:          u.UpdatedAt,
		}
	}
	return result, nil
}

func scanUsersFromRows(rows interface {
	Next() bool
	Scan(...interface{}) error
	Err() error
}) ([]*User, error) {
	var users []*User
	for rows.Next() {
		var cu crudusers.Users
		var isSuperAdmin int64
		err := rows.Scan(
			&cu.Id, &cu.InstanceId, &cu.Aud, &cu.Role, &cu.Email,
			&cu.EncryptedPassword, &cu.ConfirmedAt, &cu.InvitedAt,
			&cu.ConfirmationToken, &cu.ConfirmationSentAt,
			&cu.RecoveryToken, &cu.RecoverySentAt,
			&cu.EmailChangeToken, &cu.EmailChange, &cu.EmailChangeSentAt,
			&cu.LastSignInAt, &cu.RawAppMetaData, &cu.RawUserMetaData,
			&isSuperAdmin, &cu.CreatedAt, &cu.UpdatedAt,
		)
		if err != nil {
			return nil, errors.Wrap(err, "error scanning user row")
		}
		cu.IsSuperAdmin = isSuperAdmin
		users = append(users, fromUsersCrud(&cu))
	}
	return users, rows.Err()
}

// CountOtherUsers counts how many other users exist besides the one provided.
func CountOtherUsers(tx *storage.Connection, instanceID, id int64) (int, error) {
	ctx := context.Background()
	count, err := crudusers.Find(tx.DB()).
		Where(crudusers.InstanceIdOp.EQ(instanceID), crudusers.IdOp.NEQ(id)).
		Count().Int64(ctx)
	if err != nil {
		return 0, errors.Wrap(err, "error counting users")
	}
	return int(count), nil
}

// IsDuplicatedEmail returns whether a user exists with the matching email and audience.
func IsDuplicatedEmail(tx *storage.Connection, instanceID int64, email, aud string) (bool, error) {
	_, err := FindUserByEmailAndAudience(tx, instanceID, email, aud)
	if err != nil {
		if IsNotFoundError(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// -------- Mutation methods --------

// Create persists the user to the database.
func (u *User) Create(tx *storage.Connection) error {
	if err := u.validate(); err != nil {
		return err
	}

	now := time.Now()
	if u.CreatedAt.IsZero() {
		u.CreatedAt = now
	}
	if u.UpdatedAt.IsZero() {
		u.UpdatedAt = now
	}

	ctx := context.Background()
	cu := toUsersCrud(u)
	_, err := crudusers.Create(tx.DB()).SetUsers(cu).Save(ctx)
	return errors.Wrap(err, "error creating user")
}

// SetRole sets the user's Role to roleName.
func (u *User) SetRole(tx *storage.Connection, roleName string) error {
	u.Role = strings.TrimSpace(roleName)
	ctx := context.Background()
	_, err := crudusers.Update(tx.DB()).
		SetRole(u.Role).
		Where(crudusers.IdOp.EQ(u.ID)).
		Save(ctx)
	return errors.Wrap(err, "error updating role")
}

// UpdateUserMetaData merges updates into UserMetaData and persists it.
func (u *User) UpdateUserMetaData(tx *storage.Connection, updates map[string]interface{}) error {
	if u.UserMetaData == nil {
		u.UserMetaData = updates
	} else {
		for key, value := range updates {
			if value != nil {
				u.UserMetaData[key] = value
			} else {
				delete(u.UserMetaData, key)
			}
		}
	}
	data, err := json.Marshal(u.UserMetaData)
	if err != nil {
		return err
	}
	ctx := context.Background()
	_, err = crudusers.Update(tx.DB()).
		SetRawUserMetaData(string(data)).
		Where(crudusers.IdOp.EQ(u.ID)).
		Save(ctx)
	return errors.Wrap(err, "error updating user metadata")
}

// UpdateAppMetaData merges updates into AppMetaData and persists it.
func (u *User) UpdateAppMetaData(tx *storage.Connection, updates map[string]interface{}) error {
	if u.AppMetaData == nil {
		u.AppMetaData = updates
	} else {
		for key, value := range updates {
			if value != nil {
				u.AppMetaData[key] = value
			} else {
				delete(u.AppMetaData, key)
			}
		}
	}
	data, err := json.Marshal(u.AppMetaData)
	if err != nil {
		return err
	}
	ctx := context.Background()
	_, err = crudusers.Update(tx.DB()).
		SetRawAppMetaData(string(data)).
		Where(crudusers.IdOp.EQ(u.ID)).
		Save(ctx)
	return errors.Wrap(err, "error updating app metadata")
}

// SetEmail sets the user's email.
func (u *User) SetEmail(tx *storage.Connection, email string) error {
	u.Email = email
	ctx := context.Background()
	_, err := crudusers.Update(tx.DB()).
		SetEmail(u.Email).
		Where(crudusers.IdOp.EQ(u.ID)).
		Save(ctx)
	return errors.Wrap(err, "error updating email")
}

// UpdatePassword updates the user's encrypted password.
func (u *User) UpdatePassword(tx *storage.Connection, password string) error {
	pw, err := hashPassword(password)
	if err != nil {
		return err
	}
	u.EncryptedPassword = pw
	ctx := context.Background()
	_, err = crudusers.Update(tx.DB()).
		SetEncryptedPassword(u.EncryptedPassword).
		Where(crudusers.IdOp.EQ(u.ID)).
		Save(ctx)
	return errors.Wrap(err, "error updating password")
}

// Confirm resets the confirmation token and sets the confirmed_at timestamp.
func (u *User) Confirm(tx *storage.Connection) error {
	u.ConfirmationToken = ""
	now := time.Now()
	u.ConfirmedAt = now
	ctx := context.Background()
	_, err := crudusers.Update(tx.DB()).
		SetConfirmationToken("").
		SetConfirmedAt(now).
		Where(crudusers.IdOp.EQ(u.ID)).
		Save(ctx)
	return errors.Wrap(err, "error confirming user")
}

// ConfirmEmailChange confirms the email change for a user.
func (u *User) ConfirmEmailChange(tx *storage.Connection) error {
	u.Email = u.EmailChange
	u.EmailChange = ""
	u.EmailChangeToken = ""
	ctx := context.Background()
	_, err := crudusers.Update(tx.DB()).
		SetEmail(u.Email).
		SetEmailChange("").
		SetEmailChangeToken("").
		Where(crudusers.IdOp.EQ(u.ID)).
		Save(ctx)
	return errors.Wrap(err, "error confirming email change")
}

// Recover resets the recovery token.
func (u *User) Recover(tx *storage.Connection) error {
	u.RecoveryToken = ""
	ctx := context.Background()
	_, err := crudusers.Update(tx.DB()).
		SetRecoveryToken("").
		Where(crudusers.IdOp.EQ(u.ID)).
		Save(ctx)
	return errors.Wrap(err, "error recovering user")
}

// Update persists all mutable fields of the user to the database.
func (u *User) Update(tx *storage.Connection) error {
	u.UpdatedAt = time.Now()
	ctx := context.Background()
	appData, _ := json.Marshal(u.AppMetaData)
	userData, _ := json.Marshal(u.UserMetaData)
	isSuperAdmin := int64(0)
	if u.IsSuperAdmin {
		isSuperAdmin = 1
	}
	_, err := crudusers.Update(tx.DB()).
		SetRole(u.Role).
		SetEmail(u.Email).
		SetEncryptedPassword(u.EncryptedPassword).
		SetIsSuperAdmin(isSuperAdmin).
		SetConfirmedAt(u.ConfirmedAt).
		SetInvitedAt(u.InvitedAt).
		SetConfirmationToken(u.ConfirmationToken).
		SetConfirmationSentAt(u.ConfirmationSentAt).
		SetRecoveryToken(u.RecoveryToken).
		SetRecoverySentAt(u.RecoverySentAt).
		SetEmailChangeToken(u.EmailChangeToken).
		SetEmailChange(u.EmailChange).
		SetEmailChangeSentAt(u.EmailChangeSentAt).
		SetLastSignInAt(u.LastSignInAt).
		SetRawAppMetaData(string(appData)).
		SetRawUserMetaData(string(userData)).
		SetUpdatedAt(u.UpdatedAt).
		Where(crudusers.IdOp.EQ(u.ID)).
		Save(ctx)
	return errors.Wrap(err, "error updating user")
}

// Destroy deletes the user from the database.
func (u *User) Destroy(tx *storage.Connection) error {
	ctx := context.Background()
	_, err := crudusers.Delete(tx.DB()).
		Where(crudusers.IdOp.EQ(u.ID)).
		Exec(ctx)
	return errors.Wrap(err, "error deleting user")
}
