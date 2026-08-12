package identity

import (
	"context"
	"errors"

	"github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/brandall2021/consorcioabierto/internal/tenancy"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pquerna/otp/totp"
)

const mfaTokenPurpose = "mfa"

// MfaSetup es el resultado de iniciar MFA: el secret se muestra una sola vez
// (para escanear el QR) y se persiste en users.mfa_secret.
type MfaSetup struct {
	Secret     string `json:"secret"`
	OtpauthURL string `json:"otpauth_url"`
}

// Errores del flujo MFA.
var (
	ErrInvalidMfaCode  = errors.New("código MFA inválido")
	ErrInvalidMFAToken = errors.New("token MFA inválido")
	ErrMFANotRequired  = errors.New("MFA no está habilitado")
)

// SetupMfa genera un secret TOTP nuevo, lo persiste (sin habilitar) y devuelve
// el secret y la URL otpauth para el QR del autenticador.
func (am *AuthManager) SetupMfa(ctx context.Context, userID string) (*MfaSetup, error) {
	uid, err := pgUUID(userID)
	if err != nil {
		return nil, err
	}
	user, err := am.q.GetUserByID(ctx, uid)
	if err != nil {
		return nil, err
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      am.cfg.BaseURL,
		AccountName: user.EmailNormalized,
		Period:      30,
		SecretSize:  20,
	})
	if err != nil {
		return nil, err
	}

	tx, err := am.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := am.q.WithTx(tx)
	if err := tenancy.SetUser(ctx, tx, userID); err != nil {
		return nil, err
	}
	if err := q.UpdateMfaSecret(ctx, db.UpdateMfaSecretParams{
		ID:        uid,
		MfaSecret: pgtype.Text{String: key.Secret(), Valid: true},
	}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &MfaSetup{Secret: key.Secret(), OtpauthURL: key.URL()}, nil
}

// ConfirmMfa habilita MFA validando el código TOTP contra el secret guardado.
// El secret no se regenere aquí; si el código no coincide no se habilita.
func (am *AuthManager) ConfirmMfa(ctx context.Context, userID, code string) error {
	uid, err := pgUUID(userID)
	if err != nil {
		return err
	}
	user, err := am.q.GetUserByID(ctx, uid)
	if err != nil {
		return err
	}
	if !user.MfaSecret.Valid || user.MfaSecret.String == "" {
		return ErrInvalidMfaCode
	}
	ok := totp.Validate(code, user.MfaSecret.String)
	if !ok {
		return ErrInvalidMfaCode
	}

	tx, err := am.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := am.q.WithTx(tx)
	if err := tenancy.SetUser(ctx, tx, userID); err != nil {
		return err
	}
	if err := q.EnableMfa(ctx, uid); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// DisableMfa desactiva MFA y borra el secret.
func (am *AuthManager) DisableMfa(ctx context.Context, userID string) error {
	uid, err := pgUUID(userID)
	if err != nil {
		return err
	}
	tx, err := am.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := am.q.WithTx(tx)
	if err := tenancy.SetUser(ctx, tx, userID); err != nil {
		return err
	}
	if err := q.DisableMfa(ctx, uid); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// signMfaToken firma un token de corta vida que autoriza completar el login
// con el segundo factor. El token es de un solo propósito y expira en minutos.
func (am *AuthManager) signMfaToken(userID string) (string, error) {
	if am.key == nil {
		return "", errors.New("clave JWT no configurada")
	}
	now := am.now()
	return signJWT(am.key, Claims{
		Issuer:    am.cfg.BaseURL,
		Subject:   userID,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(am.cfg.MFATokenTTL).Unix(),
		Purpose:   mfaTokenPurpose,
	})
}

// VerifyMfa completa el login verificando el mfa_token y el código TOTP.
func (am *AuthManager) VerifyMfa(ctx context.Context, mfaToken, code, ip string) (*LoginResult, error) {
	if am.key == nil {
		return nil, errors.New("clave JWT no configurada")
	}
	claims, err := parseAndVerifyJWT(am.PublicKey(), mfaToken, am.cfg.BaseURL)
	if err != nil || claims.Purpose != mfaTokenPurpose {
		return nil, ErrInvalidMFAToken
	}
	uid, err := pgUUID(claims.Subject)
	if err != nil {
		return nil, ErrInvalidMFAToken
	}

	user, err := am.q.GetUserByID(ctx, uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidMFAToken
		}
		return nil, err
	}
	if user.Status != "active" {
		return nil, ErrUserDisabled
	}
	if !user.MfaEnabled || !user.MfaSecret.Valid || user.MfaSecret.String == "" {
		return nil, ErrMFANotRequired
	}

	failures, err := am.attempts.countFailures(ctx, user.EmailNormalized, ip, am.cfg.LoginAttemptWindow)
	if err != nil {
		return nil, err
	}
	if failures >= int64(am.cfg.LoginMaxAttempts) {
		return nil, ErrTooManyAttempts
	}

	ok := totp.Validate(code, user.MfaSecret.String)
	if !ok {
		_ = am.attempts.record(ctx, user.EmailNormalized, ip, "totp", false)
		return nil, ErrInvalidMfaCode
	}
	if err := am.attempts.record(ctx, user.EmailNormalized, ip, "totp", true); err != nil {
		return nil, err
	}
	return am.openSession(ctx, user)
}
