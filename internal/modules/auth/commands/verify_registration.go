package commands

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/eskrenkovic/vertical-slice-go/internal/modules/auth/domain"
	"github.com/eskrenkovic/vertical-slice-go/internal/modules/core"
	"github.com/google/uuid"

	"github.com/jmoiron/sqlx"
)

type VerifyRegistrationCommand struct {
	Token string
}

func (c VerifyRegistrationCommand) Validate() error {
	if c.Token == "" {
		return fmt.Errorf("invalid Token: '%s'", c.Token)
	}

	return nil
}

type VerifyRegistrationCommandHandler struct {
	db *sqlx.DB
}

func NewVerifyRegistrationCommandHandler(db *sqlx.DB) *VerifyRegistrationCommandHandler {
	return &VerifyRegistrationCommandHandler{db}
}

func (h *VerifyRegistrationCommandHandler) Handle(
	ctx context.Context,
	request VerifyRegistrationCommand,
) (core.Unit, error) {
	const invalidTokenMessage = "invalid confirmation token"

	const getCodeQuery = `
		SELECT
			*
		FROM
			auth.activation_code
		WHERE
			token = $1;`

	var activationCode domain.ActivationCode
	if err := h.db.GetContext(ctx, &activationCode, getCodeQuery, request.Token); err != nil {
		return core.Unit{}, core.NewCommandError(400, fmt.Errorf("invalid activation code"), "")
	}

	const stmt = `
		SELECT
			*
		FROM
			auth.user
		WHERE
			id = $1 AND security_stamp = $2;`

	var user domain.User
	if err := h.db.GetContext(ctx, &user, stmt, activationCode.UserID, activationCode.SecurityStamp); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return core.Unit{}, core.NewCommandError(500, err, invalidTokenMessage)
		}

		return core.Unit{}, core.NewCommandError(500, err, "failed to get user from database")
	}

	if err := domain.ValidateUserActivationCode(activationCode, user); err != nil {
		// TODO: should the security stamp be updated if the confirmation fails?
		return core.Unit{}, core.NewCommandError(500, err, invalidTokenMessage)
	}

	updateParams := map[string]interface{}{
		"old_security_stamp": activationCode.SecurityStamp,
		"new_security_stamp": uuid.New(),
	}

	err := core.Tx(ctx, h.db, func(ctx context.Context, tx *sqlx.Tx) error {
		const updateUserStmt = `
			UPDATE
				auth.user
			SET
				security_stamp = :new_security_stamp,
				email_confirmed = true
			WHERE
				id = :user_id AND security_stamp = :old_security_stamp;`

		if _, err := h.db.ExecContext(ctx, updateUserStmt, updateParams); err != nil {
			return core.NewCommandError(500, err, "failed to store confirmed user")
		}

		const updateActivationCodeStmt = `
			UPDATE
				auth.activation_code
			SET
				used = true
			WHERE
				token = $1;`

		if _, err := h.db.ExecContext(ctx, updateActivationCodeStmt, activationCode.Token); err != nil {
			return core.NewCommandError(500, err, "failed to update activation code")
		}

		return nil
	})

	return core.Unit{}, err
}
