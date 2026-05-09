package models

import (
	"context"
	"database/sql"
	"time"

	crudrt "github.com/netlify/gotrue/crud/refreshtokens"
	"github.com/netlify/gotrue/crypto"
	"github.com/netlify/gotrue/storage"
	"github.com/pkg/errors"
)

// RefreshToken is the database model for refresh tokens.
type RefreshToken struct {
	ID         int64     `json:"id"`
	InstanceID int64     `json:"-"`
	Token      string    `json:"token"`
	UserID     int64     `json:"-"`
	Revoked    bool      `json:"-"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// fromRefreshTokensCrud converts a crud RefreshTokens to a models.RefreshToken.
func fromRefreshTokensCrud(cr *crudrt.RefreshTokens) *RefreshToken {
	return &RefreshToken{
		ID:         cr.Id,
		InstanceID: cr.InstanceId,
		Token:      cr.Token,
		UserID:     cr.UserId,
		Revoked:    cr.Revoked != 0,
		CreatedAt:  cr.CreatedAt,
		UpdatedAt:  cr.UpdatedAt,
	}
}

// findRefreshTokenByToken finds a refresh token by its token string.
func findRefreshTokenByToken(tx *storage.Connection, token string) (*RefreshToken, error) {
	ctx := context.Background()
	cr, err := crudrt.Find(tx.DB()).Where(crudrt.TokenOp.EQ(token)).One(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, RefreshTokenNotFoundError{}
		}
		return nil, errors.Wrap(err, "error finding refresh token")
	}
	return fromRefreshTokensCrud(cr), nil
}

// Expired returns true if the refresh token has expired.
func (r *RefreshToken) Expired(lifetimeSeconds int) bool {
	if lifetimeSeconds <= 0 {
		return false
	}
	lifetime := time.Second * time.Duration(lifetimeSeconds)
	expiresAt := r.CreatedAt.Add(lifetime)
	return time.Now().After(expiresAt)
}

// GrantAuthenticatedUser creates a refresh token for the provided user.
func GrantAuthenticatedUser(tx *storage.Connection, user *User) (*RefreshToken, error) {
	return createRefreshToken(tx, user)
}

// GrantRefreshTokenSwap swaps a refresh token for a new one, revoking the provided token.
func GrantRefreshTokenSwap(tx *storage.Connection, user *User, token *RefreshToken) (*RefreshToken, error) {
	var newToken *RefreshToken
	err := tx.Transaction(func(rtx *storage.Connection) error {
		var terr error
		if terr = NewAuditLogEntry(tx, user.InstanceID, user, TokenRevokedAction, nil); terr != nil {
			return errors.Wrap(terr, "error creating audit log entry")
		}

		ctx := context.Background()
		revokedVal := int64(1)
		if _, terr = crudrt.Update(tx.DB()).
			SetRevoked(revokedVal).
			Where(crudrt.IdOp.EQ(token.ID)).
			Save(ctx); terr != nil {
			return terr
		}
		token.Revoked = true

		newToken, terr = createRefreshToken(rtx, user)
		return terr
	})
	return newToken, err
}

// Logout deletes all refresh tokens for a user.
func Logout(tx *storage.Connection, instanceID int64, userID int64) error {
	return tx.Exec("DELETE FROM refresh_tokens WHERE instance_id = ? AND user_id = ?", instanceID, userID)
}

// createRefreshToken creates a new refresh token for the given user.
func createRefreshToken(tx *storage.Connection, user *User) (*RefreshToken, error) {
	now := time.Now()
	cr := &crudrt.RefreshTokens{
		InstanceId: user.InstanceID,
		UserId:     user.ID,
		Token:      crypto.SecureToken(),
		Revoked:    0,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	ctx := context.Background()
	_, err := crudrt.Create(tx.DB()).SetRefreshTokens(cr).Save(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error creating refresh token")
	}
	return fromRefreshTokensCrud(cr), nil
}
