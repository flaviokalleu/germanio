package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Auth handles authentication for Germanio apps.
type Auth struct {
	DB            *sql.DB
	Table         string // user table name
	LoginField    string // email, username, etc
	PassField     string // password field
	Secret        []byte
	Roles         []string
	loginAttempts map[string]int
	loginLockout  map[string]time.Time
	rateMu        sync.Mutex
}

// Novo creates a new Auth handler.
func Novo(db *sql.DB, table, loginField, passField, secret string) *Auth {
	return &Auth{
		DB:            db,
		Table:         table,
		LoginField:    loginField,
		PassField:     passField,
		Secret:        []byte(secret),
		loginAttempts: make(map[string]int),
		loginLockout:  make(map[string]time.Time),
	}
}

// SetupTable ensures the user table has the required auth fields.
func (a *Auth) SetupTable() error {
	// Add role column if not exists (SQLite-safe)
	_, _ = a.DB.Exec(fmt.Sprintf(`ALTER TABLE "%s" ADD COLUMN "role" TEXT DEFAULT 'usuario'`, a.Table))
	return nil
}

// Registrar creates a new user with hashed password.
func (a *Auth) Registrar(w http.ResponseWriter, r *http.Request) {
	var input map[string]any
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonErr(w, "Dados inválidos", 400)
		return
	}

	login, _ := input[a.LoginField].(string)
	pass, _ := input[a.PassField].(string)

	if login == "" || pass == "" {
		jsonErr(w, fmt.Sprintf("Campo '%s' e '%s' são obrigatórios", a.LoginField, a.PassField), 400)
		return
	}

	if len(pass) < 6 {
		jsonErr(w, "Senha deve ter no mínimo 6 caracteres", 400)
		return
	}

	// Check if user exists
	var count int
	err := a.DB.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM "%s" WHERE "%s" = ?`, a.Table, a.LoginField), login).Scan(&count)
	if err == nil && count > 0 {
		jsonErr(w, fmt.Sprintf("%s já cadastrado", a.LoginField), 409)
		return
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		jsonErr(w, "Erro interno", 500)
		return
	}
	input[a.PassField] = string(hash)

	// Sign-up never chooses its own role or system fields, and only real
	// columns of the table reach the INSERT (keys are never trusted as SQL).
	columns, err := a.columns()
	if err != nil {
		jsonErr(w, "Erro interno", 500)
		return
	}
	var cols, phs []string
	var vals []any
	for k, v := range input {
		if !columns[k] || protectedField(k) {
			continue
		}
		cols = append(cols, fmt.Sprintf(`"%s"`, k))
		phs = append(phs, "?")
		vals = append(vals, v)
	}
	input["role"] = "usuario"
	if columns["role"] {
		cols = append(cols, `"role"`)
		phs = append(phs, "?")
		vals = append(vals, "usuario")
	}

	query := fmt.Sprintf(`INSERT INTO "%s" (%s) VALUES (%s)`, a.Table, strings.Join(cols, ","), strings.Join(phs, ","))
	result, err := a.DB.Exec(query, vals...)
	if err != nil {
		jsonErr(w, "Erro ao criar conta: "+err.Error(), 500)
		return
	}

	id, _ := result.LastInsertId()

	// Generate token
	token := a.gerarToken(id, login, fmt.Sprintf("%v", input["role"]))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(map[string]any{
		"token":   token,
		"id":      id,
		"message": "Conta criada com sucesso",
	})
}

var (
	dummyOnce sync.Once
	dummy     []byte
)

// dummyHash is a bcrypt hash at the cost real passwords use.
func dummyHash() []byte {
	dummyOnce.Do(func() { dummy, _ = bcrypt.GenerateFromPassword([]byte("germanio-tempo-igual"), bcrypt.DefaultCost) })
	return dummy
}

// protectedField: fields a person never sets for themselves.
func protectedField(name string) bool {
	switch strings.ToLower(name) {
	case "id", "role", "papel", "admin", "criado_em", "atualizado_em", "deletado_em", "created_at", "updated_at":
		return true
	}
	return false
}

// columns lists the real columns of the user table.
func (a *Auth) columns() (map[string]bool, error) {
	rows, err := a.DB.Query(fmt.Sprintf(`SELECT * FROM "%s" LIMIT 0`, strings.ReplaceAll(a.Table, `"`, `""`)))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	names, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(names))
	for _, n := range names {
		out[n] = true
	}
	return out, nil
}

// Login authenticates a user and returns a JWT.
func (a *Auth) Login(w http.ResponseWriter, r *http.Request) {
	var input map[string]any
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonErr(w, "Dados inválidos", 400)
		return
	}

	login, _ := input[a.LoginField].(string)
	pass, _ := input[a.PassField].(string)

	if login == "" || pass == "" {
		jsonErr(w, fmt.Sprintf("Campo '%s' e '%s' são obrigatórios", a.LoginField, a.PassField), 400)
		return
	}

	// Rate limiting
	a.rateMu.Lock()
	if lockUntil, locked := a.loginLockout[login]; locked && time.Now().Before(lockUntil) {
		a.rateMu.Unlock()
		jsonErr(w, "Conta bloqueada. Tente novamente em 5 minutos.", 429)
		return
	}
	a.rateMu.Unlock()

	// Find user
	var id int64
	var storedHash, role string
	err := a.DB.QueryRow(
		fmt.Sprintf(`SELECT "id", "%s", COALESCE("role",'usuario') FROM "%s" WHERE "%s" = ?`,
			a.PassField, a.Table, a.LoginField), login,
	).Scan(&id, &storedHash, &role)

	if err == sql.ErrNoRows {
		// Spend the same time as a wrong password: the answer's timing must
		// not reveal which logins exist.
		bcrypt.CompareHashAndPassword(dummyHash(), []byte(pass))
		jsonErr(w, "Credenciais inválidas", 401)
		return
	}
	if err != nil {
		jsonErr(w, "Erro interno", 500)
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(pass)); err != nil {
		jsonErr(w, "Credenciais inválidas", 401)
		// Track failed attempts
		a.rateMu.Lock()
		a.loginAttempts[login]++
		if a.loginAttempts[login] >= 5 {
			a.loginLockout[login] = time.Now().Add(5 * time.Minute)
			a.loginAttempts[login] = 0
		}
		a.rateMu.Unlock()
		return
	}

	// Clear failed attempts on success
	a.rateMu.Lock()
	delete(a.loginAttempts, login)
	delete(a.loginLockout, login)
	a.rateMu.Unlock()

	token := a.gerarToken(id, login, role)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"token": token,
		"id":    id,
		"role":  role,
	})
}

// Me returns the current user info from token.
func (a *Auth) Me(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r)
	if claims == nil {
		jsonErr(w, "Não autenticado", 401)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(claims)
}

// Middleware checks JWT token and adds claims to context.
func (a *Auth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Identity headers are set here from a verified token, never taken
		// from the client.
		r.Header.Del("X-User-ID")
		r.Header.Del("X-User-Login")
		r.Header.Del("X-User-Role")

		// Skip auth for login/register endpoints
		if r.URL.Path == "/api/login" || r.URL.Path == "/api/registro" ||
			r.URL.Path == "/api/register" ||
			r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		// Skip auth for GET on root (frontend)
		if r.URL.Path == "/" {
			next.ServeHTTP(w, r)
			return
		}

		token := extractToken(r)
		if token == "" {
			// Allow GET on root (frontend), static files, and health
			if r.Method == http.MethodGet && (r.URL.Path == "/" ||
				strings.HasPrefix(r.URL.Path, "/uploads/") ||
				r.URL.Path == "/health") {
				next.ServeHTTP(w, r)
				return
			}
			// With authentication configured, data is not readable anonymously;
			// only the API description stays public.
			if r.Method == http.MethodGet && (r.URL.Path == "/api/docs" || r.URL.Path == "/api/openapi.json") {
				next.ServeHTTP(w, r)
				return
			}
			jsonErr(w, "Token não fornecido", 401)
			return
		}

		claims, err := a.validarToken(token)
		if err != nil {
			jsonErr(w, "Token inválido", 401)
			return
		}

		// Store claims in header for downstream use
		r.Header.Set("X-User-ID", fmt.Sprintf("%v", claims["id"]))
		r.Header.Set("X-User-Login", fmt.Sprintf("%v", claims["login"]))
		r.Header.Set("X-User-Role", fmt.Sprintf("%v", claims["role"]))

		next.ServeHTTP(w, r)
	})
}

// CheckRole verifies if the user has the required role.
func (a *Auth) CheckRole(r *http.Request, required string) bool {
	if required == "" || required == "publico" || required == "public" {
		return true
	}
	role := r.Header.Get("X-User-Role")
	if role == "" {
		return false
	}
	if role == "admin" || role == "superadmin" {
		return true // admin/superadmin has access to everything
	}
	return role == required
}

// ==================== JWT (simple HMAC-SHA256) ====================

func (a *Auth) gerarToken(id int64, login, role string) string {
	header := base64url([]byte(`{"alg":"HS256","typ":"JWT"}`))

	payload := map[string]any{
		"id":    id,
		"login": login,
		"role":  role,
		"exp":   time.Now().Add(72 * time.Hour).Unix(),
	}
	payloadJSON, _ := json.Marshal(payload)
	payloadB64 := base64url(payloadJSON)

	data := header + "." + payloadB64
	sig := a.sign([]byte(data))
	sigB64 := base64url(sig)

	return data + "." + sigB64
}

func (a *Auth) validarToken(token string) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("token inválido")
	}

	// Verify signature
	data := parts[0] + "." + parts[1]
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("token inválido")
	}

	expected := a.sign([]byte(data))
	if !hmac.Equal(sig, expected) {
		return nil, fmt.Errorf("assinatura inválida")
	}

	// Decode payload
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("payload inválido")
	}

	var claims map[string]any
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return nil, fmt.Errorf("claims inválidos")
	}

	// Check expiration
	if exp, ok := claims["exp"].(float64); ok {
		if time.Now().Unix() > int64(exp) {
			return nil, fmt.Errorf("token expirado")
		}
	}

	return claims, nil
}

func (a *Auth) sign(data []byte) []byte {
	mac := hmac.New(sha256.New, a.Secret)
	mac.Write(data)
	return mac.Sum(nil)
}

func base64url(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func extractToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return r.URL.Query().Get("token")
}

// GetClaims extracts user claims from request headers (set by middleware).
func GetClaims(r *http.Request) map[string]any {
	id := r.Header.Get("X-User-ID")
	if id == "" {
		return nil
	}
	return map[string]any{
		"id":    id,
		"login": r.Header.Get("X-User-Login"),
		"role":  r.Header.Get("X-User-Role"),
	}
}

func jsonErr(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"erro": msg})
}
