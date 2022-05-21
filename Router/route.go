package Router

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	jwtAuth "wb_Bar/Middleware"
	"wb_Bar/Models"
)

type IDataBase interface {
	CreateUser(context.Context, Models.UserAuthData) (Models.UserWithClaims, error)
	Login(context.Context, Models.UserAuthData) (Models.UserWithClaims, error)
	GetVisitor(context.Context, Models.UserAuthData) (Models.Visitor, error)
	UpdateVisitor(context.Context, Models.Visitor) (Models.Visitor, error)
	CreateDrink(context.Context, Models.Drink) error
	GetDrinkList(context.Context) (Models.DrinkList, error)
}

var (
	drinkList       Models.DrinkList
	errGetDrinkList error
)

func Route(db IDataBase) *chi.Mux {
	router := chi.NewRouter()
	router.Use(middleware.Logger)

	drinkList, errGetDrinkList = db.GetDrinkList(context.Background())
	if errGetDrinkList != nil && !errors.Is(errGetDrinkList, sql.ErrNoRows) {
		log.Default().Panic(errGetDrinkList)
		return nil
	}

	router.Group(func(authRouter chi.Router) {
		authRouter.Use(jwtAuth.JwtAuthorization())

		authRouter.Get("/me", func(w http.ResponseWriter, r *http.Request) {
			userCtx := Models.UserFromCtx(r.Context())
			if userCtx.Role != Models.VisitorRole {
				fmt.Println(userCtx.Login, userCtx.Role)
				HttpError(w, Models.ErrUnauthorized, "",
					"unauthorized", http.StatusUnauthorized)
				return
			}

			visitor, errGet := db.GetVisitor(r.Context(), userCtx)
			if errGet != nil {
				HttpError(w, errGet, "", "server error", http.StatusInternalServerError)
				return
			} else if !visitor.IsAlive {
				HttpError(w, Models.ErrDead, "", "you died", http.StatusUnauthorized)
				return
			}

			visitor.UpdatePpm()
			updatedVisitor, errUpdate := db.UpdateVisitor(r.Context(), visitor)
			if errUpdate != nil {
				HttpError(w, errUpdate, "update visitor",
					"server error", http.StatusBadRequest)
				return
			}

			visitorJ, _ := json.Marshal(updatedVisitor)

			w.Header().Set("Content-Type", "application/json")
			w.Write(visitorJ)
		})

		authRouter.Patch("/buy", func(w http.ResponseWriter, r *http.Request) {
			userCtx := Models.UserFromCtx(r.Context())
			if userCtx.Role != Models.VisitorRole {
				HttpError(w, Models.ErrUnauthorized, "unauthorized",
					"unauthorized", http.StatusUnauthorized)
				return
			}
			visitor, errGet := db.GetVisitor(r.Context(), userCtx)
			if errGet != nil {
				HttpError(w, errGet, "get visitor", "server error", http.StatusInternalServerError)
				return
			}

			drinkName := r.URL.Query().Get("name")
			if ok := drinkList.DrinkContain(drinkName); !ok {
				HttpError(w, Models.ErrUnauthorized, "",
					"drink not found", http.StatusBadRequest)
				return
			}

			drink := drinkList.GetDrink(drinkName)
			errBuy := visitor.BuyDrink(drink)
			if errBuy != nil {
				if errors.Is(errBuy, Models.ErrNoMoney) {
					HttpError(w, errBuy, "",
						"no money", http.StatusBadRequest)
					return
				}
				if errors.Is(errBuy, Models.ErrDead) {
					if _, errUpdate := db.UpdateVisitor(r.Context(), visitor); errUpdate != nil {
						HttpError(w, errUpdate, "update visitor",
							"server error", http.StatusInternalServerError)
						return
					}
					HttpError(w, errBuy, "",
						"you dead", http.StatusUnauthorized)
					return
				}
			}

			updatedVisitor, errUpdate := db.UpdateVisitor(r.Context(), visitor)
			if errUpdate != nil {
				HttpError(w, errUpdate, "update visitor",
					"server error", http.StatusInternalServerError)
				return
			}

			visitorJ, _ := json.Marshal(updatedVisitor)
			w.Header().Set("Content-Type", "application/json")
			w.Write(visitorJ)
		})

		authRouter.Post("/create", func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()

			userCtx := Models.UserFromCtx(r.Context())
			if userCtx.Role != Models.BarmanRole {
				HttpError(w, Models.ErrUnauthorized, "",
					"unauthorized", http.StatusUnauthorized)
				return
			}

			var drink Models.Drink
			if errUnmarshalBody := UnmarshalBody(r.Body, &drink); errUnmarshalBody != nil {
				HttpError(w, errUnmarshalBody, "",
					"bad request", http.StatusBadRequest)
				return
			}

			barman := Models.Barman{
				DrinkList: &drinkList,
			}
			if ok := barman.DrinkList.DrinkContain(drink.Name); ok {
				HttpError(w, Models.ErrDrinkAlreadyExist, "",
					"drink already exist", http.StatusBadRequest)
				return
			}

			if errCreate := db.CreateDrink(r.Context(), drink); errCreate != nil {
				HttpError(w, errCreate, "",
					"server error", http.StatusInternalServerError)
				return
			}
			barman.CreateDrink(drink)

			drinkListJ, _ := json.Marshal(drinkList.GetDrinkList())

			w.Header().Set("Content-Type", "application/json")
			w.Write(drinkListJ)
		})

		authRouter.Get("/list", func(w http.ResponseWriter, r *http.Request) {
			userCtx := Models.UserFromCtx(r.Context())
			if userCtx.Role != Models.BarmanRole && userCtx.Role != Models.VisitorRole {
				HttpError(w, Models.ErrUnauthorized, "",
					"unauthorized", http.StatusUnauthorized)
				return
			}

			var respJ []byte
			switch userCtx.Role {
			case Models.BarmanRole:
				barman := Models.Barman{}
				list := barman.GetDrinkLIst(drinkList)

				respJ, _ = json.Marshal(list)
				break
			case Models.VisitorRole:
				visitor, errGet := db.GetVisitor(r.Context(), userCtx)
				if errGet != nil {
					HttpError(w, errGet, "get visitor",
						"server error", http.StatusInternalServerError)
					return
				} else if !visitor.IsAlive {
					HttpError(w, Models.ErrDead, "",
						"you died", http.StatusUnauthorized)
					return
				}

				list := visitor.AvailableDrinkList(drinkList)
				respJ, _ = json.Marshal(list)
				break
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write(respJ)
		})
	})

	router.Post("/register", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var userAuthData Models.UserAuthData

		if errUnmarshalBody := UnmarshalBody(r.Body, &userAuthData); errUnmarshalBody != nil {
			HttpError(w, errUnmarshalBody, "",
				"bad request", http.StatusBadRequest)
			return
		}

		user, errCreate := db.CreateUser(context.Background(), userAuthData)
		if errCreate != nil {
			HttpError(w, errCreate, "",
				"server error", http.StatusInternalServerError)
			return
		}

		user.SetRole()
		token, errGetToken := user.GetToken()
		if errGetToken != nil {
			HttpError(w, errGetToken, "",
				"server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(token)
	})
	router.Get("/login", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var userAuthData Models.UserAuthData

		if errUnmarshalBody := UnmarshalBody(r.Body, &userAuthData); errUnmarshalBody != nil {
			HttpError(w, errUnmarshalBody, "",
				"bad request", http.StatusBadRequest)
			return
		}

		user, errGetUser := db.Login(context.Background(), userAuthData)
		if errGetUser != nil {
			if errors.Is(errGetUser, sql.ErrNoRows) {
				HttpError(w, errGetUser, "",
					"bad request", http.StatusBadRequest)
				return
			} else {
				HttpError(w, errGetUser, "",
					"server error", http.StatusInternalServerError)
				return
			}
		}

		user.SetRole()
		token, errGetToken := user.GetToken()
		if errGetToken != nil {
			HttpError(w, errGetUser, "",
				"server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(token)
	})

	return router
}

func UnmarshalBody(r io.Reader, v interface{}) error {
	resp, errResp := ioutil.ReadAll(r)
	if errResp != nil {
		//ErrorPorcessing.HttpError(w, errResp, "failed to get body", "Bad Request", http.StatusBadRequest)
		return fmt.Errorf("server error: %w", errResp)
	}

	if errUnmarshalJson := json.Unmarshal(resp, v); errUnmarshalJson != nil {
		return fmt.Errorf("server error: %w", errUnmarshalJson)
	}

	return nil
}
