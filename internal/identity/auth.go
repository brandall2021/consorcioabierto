package identity

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/brandall2021/consorcioabierto/internal/config"
	"github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/brandall2021/consorcioabierto/internal/tenancy"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvalidCredentials  = errors.New("credenciales inválidas")
	ErrUserDisabled        = errors.New("usuario deshabilitado")
	ErrTooManyAttempts     = errors.New("demasiados intentos fallidos")
	ErrInvalidRefreshToken = errors.New("refresh token inválido")
	ErrRefreshTokenExpired = errors.New("refresh token expirado")
	ErrRefreshTokenReused  = errors.New("refresh token reutilizado")
	ErrMembershipNotFound  = errors.New("membresía no encontrada")
	ErrMembershipInactive  = errors.New("membresía inactiva")
)

// User es el DTO público de usuario (sin password_hash).
type User struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	MfaEnabled bool   `json:"mfa_enabled"`
}

// Membership es el DTO público de membresía.
type Membership struct {
	ID           string   `json:"id"`
	TenantID     string   `json:"tenant_id"`
	TenantName   string   `json:"tenant_name"`
	TenantStatus string   `json:"tenant_status"`
	Status       string   `json:"status"`
	Roles        []string `json:"roles"`
	Scopes       []string `json:"scopes"`
}

// Me agrupa usuario, membresía activa y permisos efectivos.
type Me struct {
	User        User       `json:"user"`
	Membership  Membership `json:"membership"`
	Permissions []string   `json:"permissions"`
}

// LoginResult es lo que devuelve Login. AccessToken se emite solo cuando hay
// una única membresía activa; con varias, el cliente llama select-tenant.
// Con MFA habilitado se devuelve MfaRequired + MfaToken para completar con
// VerifyMfa (no se abre sesión aún).
type LoginResult struct {
	User         User
	Memberships  []Membership
	RefreshToken string
	AccessToken  string
	MfaRequired  bool
	MfaToken     string
}

// AuthManager implementa el flujo de identidad: login con Argon2id, sesiones,
// refresh tokens con rotación por familia y access token JWT RS256.
type AuthManager struct {
	cfg      *config.Config
	key      *rsa.PrivateKey
	pool     *pgxpool.Pool
	q        *db.Queries
	attempts *attemptRecorder
	now      func() time.Time
}

// NewAuthManager crea el gestor de identidad.
func NewAuthManager(cfg *config.Config, key *rsa.PrivateKey, pool *pgxpool.Pool) *AuthManager {
	return &AuthManager{
		cfg:      cfg,
		key:      key,
		pool:     pool,
		q:        db.New(pool),
		attempts: &attemptRecorder{pool: pool, now: time.Now},
		now:      time.Now,
	}
}

// PublicKey expone la clave pública para verificar access tokens (middleware).
func (am *AuthManager) PublicKey() *rsa.PublicKey {
	if am.key == nil {
		return nil
	}
	return &am.key.PublicKey
}

// Issuer devuelve el issuer esperado en los access tokens.
func (am *AuthManager) Issuer() string { return am.cfg.BaseURL }

// AccessTokenTTL devuelve la duración del access token.
func (am *AuthManager) AccessTokenTTL() time.Duration { return am.cfg.AccessTokenTTL }

func pgUUID(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		return pgtype.UUID{}, err
	}
	return u, nil
}

func newPgUUID() pgtype.UUID {
	var u pgtype.UUID
	_ = u.Scan(uuid.NewString())
	return u
}

func refreshHash(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

func newRefreshToken() (plain, hash string, err error) {
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return "", "", err
	}
	plain = hex.EncodeToString(buf)
	return plain, refreshHash(plain), nil
}

// Login valida credenciales (con límite de intentos por email+IP), registra la
// sesión y emite refresh token. Si el usuario tiene MFA habilitado, devuelve un
// mfa_token de corta vida y NO abre sesión: el segundo factor se valida en
// VerifyMfa. La IP se usa solo para el contador de intentos, nunca se registra
// el email en logs.
func (am *AuthManager) Login(ctx context.Context, email, password, ip string) (*LoginResult, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))

	failures, err := am.attempts.countFailures(ctx, normalized, ip, am.cfg.LoginAttemptWindow)
	if err != nil {
		return nil, err
	}
	if failures >= int64(am.cfg.LoginMaxAttempts) {
		return nil, ErrTooManyAttempts
	}

	user, err := am.q.GetUserByEmail(ctx, normalized)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Mismo código de error para no filtrar existencia de cuentas.
			_ = am.attempts.record(ctx, normalized, ip, "password", false)
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if user.Status != "active" {
		return nil, ErrUserDisabled
	}

	ok, err := ComparePasswordAndHash(password, user.PasswordHash)
	if err != nil {
		return nil, err
	}
	if !ok {
		_ = am.attempts.record(ctx, normalized, ip, "password", false)
		return nil, ErrInvalidCredentials
	}
	if err := am.attempts.record(ctx, normalized, ip, "password", true); err != nil {
		return nil, err
	}

	if user.MfaEnabled {
		token, err := am.signMfaToken(user.ID.String())
		if err != nil {
			return nil, err
		}
		return &LoginResult{User: userDTO(user), MfaRequired: true, MfaToken: token}, nil
	}

	return am.openSession(ctx, user)
}

// openSession crea sesión + refresh token y arma la respuesta de login.
func (am *AuthManager) openSession(ctx context.Context, user db.User) (*LoginResult, error) {
	tx, err := am.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := am.q.WithTx(tx)
	if err := tenancy.SetUser(ctx, tx, user.ID.String()); err != nil {
		return nil, err
	}

	rows, err := q.ListMembershipsForUser(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	memberships := make([]Membership, 0, len(rows))
	for _, r := range rows {
		m, err := am.membershipDTO(ctx, q, r)
		if err != nil {
			return nil, err
		}
		memberships = append(memberships, *m)
	}

	expiresAt := am.now().Add(am.cfg.RefreshTokenTTL)
	firstMembership := pgtype.UUID{}
	if len(memberships) > 0 {
		firstMembership, err = pgUUID(memberships[0].ID)
		if err != nil {
			return nil, err
		}
	}

	sessionID, err := q.InsertSession(ctx, db.InsertSessionParams{
		UserID:       user.ID,
		MembershipID: firstMembership,
		SessionKey:   newPgUUID(),
		ExpiresAt:    pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	plain, hash, err := newRefreshToken()
	if err != nil {
		return nil, err
	}
	if err := q.InsertRefreshToken(ctx, db.InsertRefreshTokenParams{
		SessionID: sessionID,
		UserID:    user.ID,
		TokenHash: hash,
		FamilyID:  newPgUUID(),
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	res := &LoginResult{
		User:         userDTO(user),
		Memberships:  memberships,
		RefreshToken: plain,
	}
	if len(memberships) == 1 {
		at, err := am.issueAccessFor(ctx, user.ID.String(), memberships[0].ID)
		if err == nil {
			res.AccessToken = at
		}
	}
	return res, nil
}

func (am *AuthManager) membershipDTO(ctx context.Context, q *db.Queries, r db.ListMembershipsForUserRow) (*Membership, error) {
	roles, err := q.ListRolesForMembership(ctx, r.MembershipID)
	if err != nil {
		return nil, err
	}
	scopes, err := q.ListScopesForMembership(ctx, r.MembershipID)
	if err != nil {
		return nil, err
	}

	m := &Membership{
		ID:           r.MembershipID.String(),
		TenantID:     r.TenantID.String(),
		TenantName:   r.TenantName,
		TenantStatus: r.TenantStatus,
		Status:       r.MembershipStatus,
		Roles:        make([]string, 0, len(roles)),
		Scopes:       make([]string, 0, len(scopes)),
	}
	for _, rl := range roles {
		m.Roles = append(m.Roles, rl.Code)
	}
	for _, sc := range scopes {
		s := sc.ScopeType
		if sc.ScopeID.Valid {
			s += ":" + sc.ScopeID.String()
		}
		m.Scopes = append(m.Scopes, s)
	}
	return m, nil
}

// SelectTenant valida la membresía del usuario y emite el access token JWT.
func (am *AuthManager) SelectTenant(ctx context.Context, userID, membershipID string) (*Me, string, error) {
	uid, err := pgUUID(userID)
	if err != nil {
		return nil, "", err
	}
	mid, err := pgUUID(membershipID)
	if err != nil {
		return nil, "", err
	}

	user, err := am.q.GetUserByID(ctx, uid)
	if err != nil {
		return nil, "", err
	}

	tx, err := am.pool.Begin(ctx)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := am.q.WithTx(tx)
	if err := tenancy.SetUser(ctx, tx, userID); err != nil {
		return nil, "", err
	}

	rows, err := q.ListMembershipsForUser(ctx, uid)
	if err != nil {
		return nil, "", err
	}
	var target *db.ListMembershipsForUserRow
	for i := range rows {
		if rows[i].MembershipID.String() == membershipID {
			target = &rows[i]
			break
		}
	}
	if target == nil {
		return nil, "", ErrMembershipNotFound
	}
	if target.MembershipStatus != "active" {
		return nil, "", ErrMembershipInactive
	}

	m, err := am.membershipDTO(ctx, q, *target)
	if err != nil {
		return nil, "", err
	}
	me := &Me{
		User:        userDTO(user),
		Membership:  *m,
		Permissions: PermissionsForRoles(m.Roles),
	}

	access, err := am.signAccessToken(uid, mid, m)
	if err != nil {
		return nil, "", err
	}
	return me, access, nil
}

// revokeFamily revoca todos los tokens de la familia (detección de reuso).
// Corre en transacción con el contexto del usuario: la policy refresh_tokens_update
// exige user_id = app.current_user_id().
func (am *AuthManager) revokeFamily(ctx context.Context, userID string, familyID pgtype.UUID) error {
	tx, err := am.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := am.q.WithTx(tx)
	if err := tenancy.SetUser(ctx, tx, userID); err != nil {
		return err
	}
	now := pgtype.Timestamptz{Time: am.now(), Valid: true}
	if err := q.RevokeTokensByFamily(ctx, db.RevokeTokensByFamilyParams{FamilyID: familyID, RevokedAt: now}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Refresh rota el refresh token: valida, revoca el anterior y emite uno nuevo
// junto con un access token. El reuso de un token ya revocado revoca la familia.
func (am *AuthManager) Refresh(ctx context.Context, plain string) (string, string, error) {
	tok, err := am.q.LookupRefreshToken(ctx, refreshHash(plain))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", ErrInvalidRefreshToken
		}
		return "", "", err
	}

	if tok.RevokedAt.Valid {
		if err := am.revokeFamily(ctx, tok.UserID.String(), tok.FamilyID); err != nil {
			return "", "", err
		}
		return "", "", ErrRefreshTokenReused
	}
	if !tok.ExpiresAt.Valid || tok.ExpiresAt.Time.Before(am.now()) {
		return "", "", ErrRefreshTokenExpired
	}

	uid := tok.UserID.String()

	tx, err := am.pool.Begin(ctx)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := am.q.WithTx(tx)
	if err := tenancy.SetUser(ctx, tx, uid); err != nil {
		return "", "", err
	}

	now := am.now()
	if err := q.RevokeRefreshToken(ctx, db.RevokeRefreshTokenParams{
		ID:        tok.ID,
		RevokedAt: pgtype.Timestamptz{Time: now, Valid: true},
	}); err != nil {
		return "", "", err
	}

	newPlain, newHash, err := newRefreshToken()
	if err != nil {
		return "", "", err
	}
	uidPg, err := pgUUID(uid)
	if err != nil {
		return "", "", err
	}
	if err := q.InsertRefreshToken(ctx, db.InsertRefreshTokenParams{
		SessionID: tok.SessionID,
		UserID:    uidPg,
		TokenHash: newHash,
		FamilyID:  tok.FamilyID,
		ExpiresAt: pgtype.Timestamptz{Time: now.Add(am.cfg.RefreshTokenTTL), Valid: true},
	}); err != nil {
		return "", "", err
	}

	// Sesión → membresía activa → claims del access token.
	session, err := q.GetSessionByID(ctx, tok.SessionID)
	if err != nil {
		return "", "", err
	}
	m, err := am.membershipByID(ctx, q, uidPg, session.MembershipID)
	if err != nil {
		return "", "", err
	}

	access, err := am.signAccessToken(uidPg, session.MembershipID, m)
	if err != nil {
		return "", "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", "", err
	}
	return access, newPlain, nil
}

// Logout revoca el refresh token y su sesión.
func (am *AuthManager) Logout(ctx context.Context, plain string) error {
	tok, err := am.q.LookupRefreshToken(ctx, refreshHash(plain))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	if tok.RevokedAt.Valid {
		return nil
	}

	tx, err := am.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := am.q.WithTx(tx)
	if err := tenancy.SetUser(ctx, tx, tok.UserID.String()); err != nil {
		return err
	}
	now := pgtype.Timestamptz{Time: am.now(), Valid: true}
	if err := q.RevokeRefreshToken(ctx, db.RevokeRefreshTokenParams{ID: tok.ID, RevokedAt: now}); err != nil {
		return err
	}
	if err := q.RevokeSession(ctx, db.RevokeSessionParams{ID: tok.SessionID, RevokedAt: now}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Me resuelve el perfil desde un access token verificado.
func (am *AuthManager) Me(ctx context.Context, claims *Claims) (*Me, error) {
	uid, err := pgUUID(claims.Subject)
	if err != nil {
		return nil, err
	}
	user, err := am.q.GetUserByID(ctx, uid)
	if err != nil {
		return nil, err
	}

	tx, err := am.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := am.q.WithTx(tx)
	if err := tenancy.SetUser(ctx, tx, claims.Subject); err != nil {
		return nil, err
	}

	mid, err := pgUUID(claims.Membership)
	if err != nil {
		return nil, ErrMembershipNotFound
	}
	m, err := am.membershipByID(ctx, q, uid, mid)
	if err != nil {
		return nil, err
	}
	return &Me{
		User:        userDTO(user),
		Membership:  *m,
		Permissions: PermissionsForRoles(m.Roles),
	}, nil
}

func userDTO(u db.User) User {
	return User{
		ID:         u.ID.String(),
		Email:      u.EmailNormalized,
		Name:       u.Name,
		Status:     u.Status,
		MfaEnabled: u.MfaEnabled,
	}
}

func (am *AuthManager) membershipByID(ctx context.Context, q *db.Queries, uid, mid pgtype.UUID) (*Membership, error) {
	rows, err := q.ListMembershipsForUser(ctx, uid)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		if rows[i].MembershipID.String() == mid.String() {
			if rows[i].MembershipStatus != "active" {
				return nil, ErrMembershipInactive
			}
			return am.membershipDTO(ctx, q, rows[i])
		}
	}
	return nil, ErrMembershipNotFound
}

// ListMemberships devuelve las membresías del usuario (todas, sin filtrar por
// la activa del token).
func (am *AuthManager) ListMemberships(ctx context.Context, claims *Claims) ([]Membership, error) {
	uid, err := pgUUID(claims.Subject)
	if err != nil {
		return nil, err
	}
	tx, err := am.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := am.q.WithTx(tx)
	if err := tenancy.SetUser(ctx, tx, claims.Subject); err != nil {
		return nil, err
	}
	rows, err := q.ListMembershipsForUser(ctx, uid)
	if err != nil {
		return nil, err
	}
	memberships := make([]Membership, 0, len(rows))
	for _, r := range rows {
		m, err := am.membershipDTO(ctx, q, r)
		if err != nil {
			return nil, err
		}
		memberships = append(memberships, *m)
	}
	return memberships, nil
}

// issueAccessFor emite access token para una membresía (login de una sola).
func (am *AuthManager) issueAccessFor(ctx context.Context, userID, membershipID string) (string, error) {
	uid, err := pgUUID(userID)
	if err != nil {
		return "", err
	}
	mid, err := pgUUID(membershipID)
	if err != nil {
		return "", err
	}
	tx, err := am.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := am.q.WithTx(tx)
	if err := tenancy.SetUser(ctx, tx, userID); err != nil {
		return "", err
	}
	m, err := am.membershipByID(ctx, q, uid, mid)
	if err != nil {
		return "", err
	}
	return am.signAccessToken(uid, mid, m)
}

func (am *AuthManager) signAccessToken(uid, mid pgtype.UUID, m *Membership) (string, error) {
	if am.key == nil {
		return "", errors.New("clave JWT no configurada")
	}
	now := am.now()
	claims := Claims{
		Issuer:     am.cfg.BaseURL,
		Subject:    uid.String(),
		IssuedAt:   now.Unix(),
		ExpiresAt:  now.Add(am.cfg.AccessTokenTTL).Unix(),
		Email:      "",
		Name:       "",
		Membership: mid.String(),
		Tenant:     m.TenantID,
		Roles:      m.Roles,
	}
	if len(m.Scopes) > 0 {
		parts := strings.SplitN(m.Scopes[0], ":", 2)
		claims.ScopeType = parts[0]
		if len(parts) == 2 {
			claims.ScopeID = parts[1]
		}
	}
	return signJWT(am.key, claims)
}

// VerifyAccessToken valida un access token y devuelve sus claims.
func (am *AuthManager) VerifyAccessToken(token string) (*Claims, error) {
	if am.key == nil {
		return nil, errors.New("clave JWT no configurada")
	}
	return parseAndVerifyJWT(am.PublicKey(), token, am.cfg.BaseURL)
}
