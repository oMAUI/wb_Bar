package Models

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/golang-jwt/jwt"
)

const lethalDose = 6
const SigningKey = "maui"

type (
	Role int

	UserWithClaims struct {
		Role
		jwt.StandardClaims
		ID    int
		Login string
	}

	UserAuthData struct {
		Login    string `json:"login,omitempty"`
		Password string `json:"password,omitempty"`
		Role     Role   `json:"role"`
	}

	tokenResp struct {
		Token string `json:"token"`
	}

	Key struct {
		Key string
	}
)

var (
	BarmanRole      Role  = 0
	VisitorRole     Role  = 1
	ErrUnauthorized error = errors.New("unauthorized")
)

func CtxKey() Key {
	return Key{Key: "id"}
}

func UserFromCtx(ctx context.Context) UserAuthData {
	return ctx.Value(CtxKey()).(UserAuthData)
}

func CheckRole(has Role, need Role) bool {
	return false
}

func (u *UserWithClaims) ToUserAuthData() UserAuthData {
	return UserAuthData{
		Login: u.Login,
		Role:  u.Role,
	}
}

func (u *UserWithClaims) SetRole() {
	if u.ID != 1 {
		u.Role = VisitorRole
	} else {
		u.Role = BarmanRole
	}
}

func (u *UserWithClaims) GetToken() ([]byte, error) {
	tokenWithClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, UserWithClaims{
		Role:           u.Role,
		Login:          u.Login,
		StandardClaims: jwt.StandardClaims{},
	})

	token, errSigningToken := tokenWithClaims.SignedString([]byte(SigningKey))
	if errSigningToken != nil {
		return nil, errSigningToken
	}

	Resp := tokenResp{
		Token: token,
	}

	tokenJ, errMarshal := json.Marshal(Resp)
	if errMarshal != nil {
		return nil, errMarshal
	}

	return tokenJ, nil
}
