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
	"rest-go-demo/database"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/mux"

	_ "rest-go-demo/docs"

	"github.com/0xsequence/go-sequence/api"
)

var (
	ErrUserNotExists  = errors.New("Authuser does not exist")
	ErrUserExists     = errors.New("Authuser already exists")
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
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
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

type Authuser struct {
	Address string
	Nonce   string
}

func CreateIfNotExists(u Authuser) error {
	var checkUser Authuser
	dbQuery := database.Connector.Where("address = ?", u.Address).Find(&checkUser)

	if dbQuery.RowsAffected > 0 {
		return ErrUserExists
	}

	//create the item in the database
	database.Connector.Create(&u)
	return nil
}

func Get(address string) (Authuser, error) {
	var checkUser Authuser
	dbQuery := database.Connector.Where("address = ?", address).Find(&checkUser)

	if dbQuery.RowsAffected == 0 {
		return checkUser, ErrUserNotExists
	}

	return checkUser, nil
}

func Update(user Authuser) error {

	database.Connector.Model(&Authuser{}).
		Where("address = ?", user.Address).
		Update("nonce", user.Nonce)

	return nil
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

//Legacy - not needed anymore
func RegisterHandler() http.HandlerFunc {
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
		u := Authuser{
			Address: strings.ToLower(p.Address), // let's only store lower case
			Nonce:   nonce,
		}
		if err := CreateIfNotExists(u); err != nil {
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

// UserNonceHandler godoc
// @Summary If the current wallet doesn't have a valid local JWT, need to request a new nonce to sign
// @Description As part of the login process, we need a user to sign a nonce genrated from the API, to prove the user in fact
// @Description the owner of the wallet they are siging in from.  JWT currently set to 24 hour validity (could change this upon request)
// @Tags Auth
// @Accept  json
// @Produce json
// @Param address path string true "wallet address to get nonce to sign"
// @Success 200 {} Authuser
// @Router /users/{address}/nonce [get]
func UserNonceHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		address := vars["address"]
		//fmt.Println("getting nonce for user: ", address)
		if !hexRegex.MatchString(address) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		//combining /register and /users (no need to call both and check each time)
		nonce, err := GetNonce()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		user := Authuser{
			Address: strings.ToLower(address), // let's only store lower case
			Nonce:   nonce,
		}
		CreateIfNotExists(user)
		//end of copied /register functionality

		Authuser, err := Get(strings.ToLower(address))
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
			Nonce: Authuser.Nonce,
		}
		renderJson(r, w, http.StatusOK, resp)
	}
}

type SigninPayload struct {
	Address string `json:"address"`
	Nonce   string `json:"nonce"`
	Sig     string `json:"sig"`
	Msg     string `json:"msg"`
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

// SigninHandler godoc
// @Summary Sign In with signed nonce value, currently JWT token returned should be valid for 24 hours
// @Description Every call the to API after this signin should present the JWT Bearer token for authenticated access.
// @Description Upon request we can change the timeout to greater than 24 hours, or setup an addtional dedicated API for
// @Description an agreed upon development and maintenance cost
// @Tags Auth
// @Accept  json
// @Produce json
// @Param message body SigninPayload true "json input containing signed message and append nonce for easy processing"
// @Success 200 {integer} int
// @Router /signin [post]
func SigninHandler(jwtProvider *JwtHmacProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var p SigninPayload
		requestBody, _ := ioutil.ReadAll(r.Body)
		if err := json.Unmarshal(requestBody, &p); err != nil { // Parse []byte to the go struct pointer
			fmt.Println("Can not unmarshal JSON in SigninHandler")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := p.Validate(); err != nil {
			fmt.Println("Some fields were invalid")
			fmt.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		address := strings.ToLower(p.Address)
		Authuser, err := Authenticate(address, p.Nonce, p.Msg, p.Sig)
		switch err {
		case nil:
		case ErrAuthError:
			w.WriteHeader(http.StatusUnauthorized)
			return
		default:
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		signedToken, err := jwtProvider.CreateStandard(Authuser.Address)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:  "Authorization",
			Value: signedToken,
			// Path:     "/",
			// SameSite: 4,
			// Secure:   true,
			// MaxAge:   86400,
			// true means no scripts, http requests only. This has
			// nothing to do with https vs http
			HttpOnly: true,
		})
		resp := struct {
			AccessToken string `json:"access"`
		}{
			AccessToken: signedToken,
		}
		renderJson(r, w, http.StatusOK, resp)
		// renderJsonWithCookie(r, w, http.StatusOK, http.Cookie{
		// 	Name:  "jwt",
		// 	Value: signedToken,
		// 	// true means no scripts, http requests only. This has
		// 	// nothing to do with https vs http
		// 	HttpOnly: true,
		// }, resp)
	}
}

func WelcomeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		Authuser := GetUserFromReqContext(r)
		fmt.Println("getting Authuser: ", Authuser)
		resp := struct {
			Msg string `json:"msg"`
		}{
			Msg: "Congrats " + Authuser.Address + " you made it",
		}
		renderJson(r, w, http.StatusOK, resp)
	}
}

// ============================================================================

func GetUserFromReqContext(r *http.Request) Authuser {
	ctx := r.Context()
	key := ctx.Value("Authuser").(Authuser)
	return key
}

func AuthMiddleware(jwtProvider *JwtHmacProvider) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			headerValue := r.Header.Get("Authorization")
			if len(headerValue) > 0 {
				const prefix = "Bearer "
				if len(headerValue) < len(prefix) {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				//fmt.Println("Found JWT in Authorization header")
				headerValue = headerValue[len(prefix):]
			} else {
				tokenCookie, err := r.Cookie("Authorization")
				if err != nil {
					//log.Fatalf("Error occured while reading cookie")
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				fmt.Println("Found JWT in Cookie")
				headerValue = tokenCookie.Value
			}
			// fmt.Println("Authorization: ", headerValue)
			// fmt.Println("headerValue: ", r.Header)

			tokenString := headerValue //headerValue[len(prefix):]
			if len(tokenString) == 0 {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			claims, err := jwtProvider.Verify(tokenString)
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			Authuser, err := Get(claims.Subject)
			if err != nil {
				if errors.Is(err, ErrUserNotExists) {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			ctx := context.WithValue(r.Context(), "Authuser", Authuser)
			next.ServeHTTP(w, r.WithContext(ctx))

		})
	}
}

func ValidateMessageSignatureSequenceWallet(chainID string, walletAddress string, signature string, message string) bool {
	seqAPI := api.NewAPIClient("https://api.sequence.app", http.DefaultClient)

	isValid, err := seqAPI.IsValidMessageSignature(context.Background(), chainID, walletAddress, message, signature)
	if err != nil {
		fmt.Println("Failed to Verify Sequence Wallet Signature?", err)
		isValid = false
	}
	//fmt.Println("isValid?", isValid)
	return isValid
}

func Authenticate(address string, nonce string, message string, sigHex string) (Authuser, error) {
	fmt.Println("Authenticate: " + address + " nonce: " + message + " sig: " + sigHex)
	Authuser, err := Get(address)
	if err != nil {
		return Authuser, err
	}
	if Authuser.Nonce != message {
		return Authuser, ErrAuthError
	}

	recoveredAddr := " "
	if len(sigHex) > 590 { //594 without the 0x to be exact, but we can clean this up
		fmt.Println("Using Sequence Wallet Signature")
		isValidSequenceWalletSig := ValidateMessageSignatureSequenceWallet("mainnet", address, sigHex, message)

		if isValidSequenceWalletSig {
			recoveredAddr = address
			fmt.Println("Sequence Wallet Signature Valid!")
		}

	} else {
		sig := hexutil.MustDecode(sigHex)
		// https://github.com/ethereum/go-ethereum/blob/master/internal/ethapi/api.go#L516
		// check here why I am subtracting 27 from the last byte
		sig[crypto.RecoveryIDOffset] -= 27
		msg := accounts.TextHash([]byte(message))
		recovered, err := crypto.SigToPub(msg, sig)
		if err != nil {
			return Authuser, err
		}
		recoveredAddr = strings.ToLower(crypto.PubkeyToAddress(*recovered).Hex())
	}

	if Authuser.Address != recoveredAddr {
		return Authuser, ErrAuthError
	}

	// update the nonce here so that the signature cannot be resused
	nonce, err = GetNonce()
	if err != nil {
		return Authuser, err
	}
	Authuser.Nonce = nonce
	Update(Authuser)

	return Authuser, nil
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

// func renderJsonWithCookie(r *http.Request, w http.ResponseWriter, statusCode int, cookie http.Cookie, res interface{}) {
// 	w.Header().Set("Content-Type", "application/json; charset=utf-8 ")
// 	var body []byte
// 	if res != nil {
// 		var err error
// 		body, err = json.Marshal(res)
// 		if err != nil { // TODO handle me better
// 			w.WriteHeader(http.StatusInternalServerError)
// 		}
// 	}
// 	w.WriteHeader(statusCode)
// 	if len(body) > 0 {
// 		w.Write(body)
// 	}
// 		// Finally, we set the client cookie for "token" as the JWT we just generated
// 	// we also set an expiry time which is the same as the token itself
// 	http.SetCookie(w, &cookie)
// }
