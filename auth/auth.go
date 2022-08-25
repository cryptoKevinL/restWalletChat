package auth

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/mux"
)

var (
	ErrUserNotExists  = errors.New("AuthUser does not exist")
	ErrUserExists     = errors.New("AuthUser already exists")
	ErrInvalidAddress = errors.New("invalid address")
	ErrInvalidNonce   = errors.New("invalid nonce")
	ErrMissingSig     = errors.New("signature is missing")
	ErrAuthError      = errors.New("authentication error")
)

type JwtHmacProvider struct {
	hmacSecret []byte
	issuer     string
	duration   time.Duration
}

func NewJwtHmacProvider(hmacSecret string, issuer string, duration time.Duration) *JwtHmacProvider {
	ans := JwtHmacProvider{
		hmacSecret: []byte(hmacSecret),
		issuer:     issuer,
		duration:   duration,
	}
	return &ans
}

func (j *JwtHmacProvider) CreateStandard(subject string) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Issuer:    j.issuer,
		Subject:   subject,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(j.duration)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.hmacSecret)
}

func (j *JwtHmacProvider) Verify(tokenString string) (*jwt.RegisteredClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return j.hmacSecret, nil
	})
	if err != nil {
		return nil, ErrAuthError
	}
	if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, ErrAuthError
}

type AuthUser struct {
	Address string
	Nonce   string
}

type MemStorage struct {
	lock  sync.RWMutex
	users map[string]AuthUser
}

func (m *MemStorage) CreateIfNotExists(u AuthUser) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	if _, exists := m.users[u.Address]; exists {
		return ErrUserExists
	}
	m.users[u.Address] = u
	return nil
}

func (m *MemStorage) Get(address string) (AuthUser, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	u, exists := m.users[address]
	if !exists {
		return u, ErrUserNotExists
	}
	return u, nil
}

func (m *MemStorage) Update(AuthUser AuthUser) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.users[AuthUser.Address] = AuthUser
	return nil
}

func NewMemStorage() *MemStorage {
	ans := MemStorage{
		users: make(map[string]AuthUser),
	}
	return &ans
}

// ============================================================================

var (
	hexRegex   *regexp.Regexp = regexp.MustCompile(`^0x[a-fA-F0-9]{40}$`)
	nonceRegex *regexp.Regexp = regexp.MustCompile(`^[0-9]+$`)
)

type RegisterPayload struct {
	Address string `json:"address"`
}

func (p RegisterPayload) Validate() error {
	if !hexRegex.MatchString(p.Address) {
		return ErrInvalidAddress
	}
	return nil
}

func RegisterHandler(storage *MemStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestBody, _ := ioutil.ReadAll(r.Body)
		var p RegisterPayload
		if err := json.Unmarshal(requestBody, &p); err != nil { // Parse []byte to the go struct pointer
			fmt.Println("Can not unmarshal JSON in RegisterHandler")
		}
		if err := p.Validate(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		nonce, err := GetNonce()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		u := AuthUser{
			Address: strings.ToLower(p.Address), // let's only store lower case
			Nonce:   nonce,
		}
		if err := storage.CreateIfNotExists(u); err != nil {
			switch errors.Is(err, ErrUserExists) {
			case true:
				w.WriteHeader(http.StatusConflict)
			default:
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}
		w.WriteHeader(http.StatusCreated)
	}
}

func UserNonceHandler(storage *MemStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		address := vars["address"]
		//fmt.Println("getting nonce for user: ", address)
		if !hexRegex.MatchString(address) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		AuthUser, err := storage.Get(strings.ToLower(address))
		if err != nil {
			switch errors.Is(err, ErrUserNotExists) {
			case true:
				w.WriteHeader(http.StatusNotFound)
			default:
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}
		resp := struct {
			Nonce string
		}{
			Nonce: AuthUser.Nonce,
		}
		renderJson(r, w, http.StatusOK, resp)
	}
}

type SigninPayload struct {
	Address string `json:"address"`
	Nonce   string `json:"nonce"`
	Sig     string `json:"sig"`
}

func (s SigninPayload) Validate() error {
	if !hexRegex.MatchString(s.Address) {
		return ErrInvalidAddress
	}
	if !nonceRegex.MatchString(s.Nonce) {
		return ErrInvalidNonce
	}
	if len(s.Sig) == 0 {
		return ErrMissingSig
	}
	return nil
}

func SigninHandler(storage *MemStorage, jwtProvider *JwtHmacProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var p SigninPayload
		requestBody, _ := ioutil.ReadAll(r.Body)
		if err := json.Unmarshal(requestBody, &p); err != nil { // Parse []byte to the go struct pointer
			fmt.Println("Can not unmarshal JSON in SigninHandler")
		}
		if err := p.Validate(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		address := strings.ToLower(p.Address)
		AuthUser, err := Authenticate(storage, address, p.Nonce, p.Sig)
		switch err {
		case nil:
		case ErrAuthError:
			w.WriteHeader(http.StatusUnauthorized)
			return
		default:
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		signedToken, err := jwtProvider.CreateStandard(AuthUser.Address)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		resp := struct {
			AccessToken string `json:"access"`
		}{
			AccessToken: signedToken,
		}
		renderJson(r, w, http.StatusOK, resp)
	}
}

func WelcomeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		AuthUser := getUserFromReqContext(r)
		fmt.Println("getting AuthUser: ", AuthUser)
		resp := struct {
			Msg string `json:"msg"`
		}{
			Msg: "Congrats " + AuthUser.Address + " you made it",
		}
		renderJson(r, w, http.StatusOK, resp)
	}
}

// ============================================================================

func getUserFromReqContext(r *http.Request) AuthUser {
	ctx := r.Context()
	key := ctx.Value("AuthUser").(AuthUser)
	return key
}

func AuthMiddleware(storage *MemStorage, jwtProvider *JwtHmacProvider) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			headerValue := r.Header.Get("Authorization")
			const prefix = "Bearer "
			if len(headerValue) < len(prefix) {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			tokenString := headerValue[len(prefix):]
			if len(tokenString) == 0 {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			claims, err := jwtProvider.Verify(tokenString)
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			AuthUser, err := storage.Get(claims.Subject)
			if err != nil {
				if errors.Is(err, ErrUserNotExists) {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			ctx := context.WithValue(r.Context(), "AuthUser", AuthUser)
			next.ServeHTTP(w, r.WithContext(ctx))

		})
	}
}

func Authenticate(storage *MemStorage, address string, nonce string, sigHex string) (AuthUser, error) {
	AuthUser, err := storage.Get(address)
	if err != nil {
		return AuthUser, err
	}
	if AuthUser.Nonce != nonce {
		return AuthUser, ErrAuthError
	}

	sig := hexutil.MustDecode(sigHex)
	// https://github.com/ethereum/go-ethereum/blob/master/internal/ethapi/api.go#L516
	// check here why I am subtracting 27 from the last byte
	sig[crypto.RecoveryIDOffset] -= 27
	msg := accounts.TextHash([]byte(nonce))
	recovered, err := crypto.SigToPub(msg, sig)
	if err != nil {
		return AuthUser, err
	}
	recoveredAddr := crypto.PubkeyToAddress(*recovered)

	if AuthUser.Address != strings.ToLower(recoveredAddr.Hex()) {
		return AuthUser, ErrAuthError
	}

	// update the nonce here so that the signature cannot be resused
	nonce, err = GetNonce()
	if err != nil {
		return AuthUser, err
	}
	AuthUser.Nonce = nonce
	storage.Update(AuthUser)

	return AuthUser, nil
}

var (
	max  *big.Int
	once sync.Once
)

func GetNonce() (string, error) {
	once.Do(func() {
		max = new(big.Int)
		max.Exp(big.NewInt(2), big.NewInt(130), nil).Sub(max, big.NewInt(1))
	})
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return n.Text(10), nil
}

func bindReqBody(r *http.Request, obj any) error {
	return json.NewDecoder(r.Body).Decode(obj)
}

func renderJson(r *http.Request, w http.ResponseWriter, statusCode int, res interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8 ")
	var body []byte
	if res != nil {
		var err error
		body, err = json.Marshal(res)
		if err != nil { // TODO handle me better
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
	w.WriteHeader(statusCode)
	if len(body) > 0 {
		w.Write(body)
	}
}
