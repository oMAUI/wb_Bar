package router

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
	"wb_Bar/pkg/httpError"
	jwtAuth "wb_Bar/pkg/middleware/authorization"
	"wb_Bar/pkg/models"
)

type IDataBase interface {
	CreateUser(context.Context, models.UserAuthData) (models.UserWithClaims, error)
	Login(context.Context, models.UserAuthData) (models.UserWithClaims, error)
	GetVisitor(context.Context, models.UserAuthData) (models.Visitor, error)
	UpdateVisitor(context.Context, models.Visitor) (models.Visitor, error)
	CreateDrink(context.Context, models.Drink) error
	GetDrinkList(context.Context) (models.DrinkList, error)
}

var (
	drinkList       models.DrinkList
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
		authRouter.Use(jwtAuth.Jwt())

		authRouter.Get("/me", func(w http.ResponseWriter, r *http.Request) {
			userCtx := models.UserFromCtx(r.Context())
			if userCtx.Role != models.VisitorRole {
				fmt.Println(userCtx.Login, userCtx.Role)
				httpError.Json(w, models.ErrUnauthorized, "",
					"unauthorized", http.StatusUnauthorized)
				return
			}

			visitor, errGet := db.GetVisitor(r.Context(), userCtx)
			if errGet != nil {
				httpError.Json(w, errGet, "", "server error", http.StatusInternalServerError)
				return
			} else if !visitor.IsAlive {
				httpError.Json(w, models.ErrDead, "", "you died", http.StatusUnauthorized)
				return
			}

			visitor.UpdatePpm()
			updatedVisitor, errUpdate := db.UpdateVisitor(r.Context(), visitor)
			if errUpdate != nil {
				httpError.Json(w, errUpdate, "update visitor",
					"server error", http.StatusBadRequest)
				return
			}

			visitorJ, _ := json.Marshal(updatedVisitor)

			w.Header().Set("Content-Type", "application/json")
			w.Write(visitorJ)
		})

		authRouter.Patch("/buy", func(w http.ResponseWriter, r *http.Request) {
			userCtx := models.UserFromCtx(r.Context())
			if userCtx.Role != models.VisitorRole {
				httpError.Json(w, models.ErrUnauthorized, "unauthorized",
					"unauthorized", http.StatusUnauthorized)
				return
			}
			visitor, errGet := db.GetVisitor(r.Context(), userCtx)
			if errGet != nil {
				httpError.Json(w, errGet, "get visitor", "server error", http.StatusInternalServerError)
				return
			}

			drinkName := r.URL.Query().Get("name")
			if ok := drinkList.DrinkContain(drinkName); !ok {
				httpError.Json(w, models.ErrUnauthorized, "",
					"drink not found", http.StatusBadRequest)
				return
			}

			drink := drinkList.Drink(drinkName)
			errBuy := visitor.BuyDrink(drink)
			if errBuy != nil {
				if errors.Is(errBuy, models.ErrNoMoney) {
					httpError.Json(w, errBuy, "",
						"no money", http.StatusBadRequest)
					return
				}
				if errors.Is(errBuy, models.ErrDead) {
					if _, errUpdate := db.UpdateVisitor(r.Context(), visitor); errUpdate != nil {
						httpError.Json(w, errUpdate, "update visitor",
							"server error", http.StatusInternalServerError)
						return
					}
					httpError.Json(w, errBuy, "",
						"you dead", http.StatusUnauthorized)
					return
				}
			}

			updatedVisitor, errUpdate := db.UpdateVisitor(r.Context(), visitor)
			if errUpdate != nil {
				httpError.Json(w, errUpdate, "update visitor",
					"server error", http.StatusInternalServerError)
				return
			}

			visitorJ, _ := json.Marshal(updatedVisitor)
			w.Header().Set("Content-Type", "application/json")
			w.Write(visitorJ)
		})

		authRouter.Post("/create", func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()

			userCtx := models.UserFromCtx(r.Context())
			if userCtx.Role != models.BarmanRole {
				httpError.Json(w, models.ErrUnauthorized, "",
					"unauthorized", http.StatusUnauthorized)
				return
			}

			var drink models.Drink
			if errUnmarshalBody := UnmarshalBody(r.Body, &drink); errUnmarshalBody != nil {
				httpError.Json(w, errUnmarshalBody, "",
					"bad request", http.StatusBadRequest)
				return
			}

			barman := models.Barman{
				DrinkList: &drinkList,
			}
			if ok := barman.DrinkList.DrinkContain(drink.Name); ok {
				httpError.Json(w, models.ErrDrinkAlreadyExist, "",
					"drink already exist", http.StatusBadRequest)
				return
			}

			if errCreate := db.CreateDrink(r.Context(), drink); errCreate != nil {
				httpError.Json(w, errCreate, "",
					"server error", http.StatusInternalServerError)
				return
			}
			barman.CreateDrink(drink)

			drinkListJ, _ := json.Marshal(drinkList.DrinkList())

			w.Header().Set("Content-Type", "application/json")
			w.Write(drinkListJ)
		})

		authRouter.Get("/list", func(w http.ResponseWriter, r *http.Request) {
			userCtx := models.UserFromCtx(r.Context())
			if userCtx.Role != models.BarmanRole && userCtx.Role != models.VisitorRole {
				httpError.Json(w, models.ErrUnauthorized, "",
					"unauthorized", http.StatusUnauthorized)
				return
			}

			var respJ []byte
			switch userCtx.Role {
			case models.BarmanRole:
				barman := models.Barman{}
				list := barman.DrinkLIst(drinkList)

				respJ, _ = json.Marshal(list)
				break
			case models.VisitorRole:
				visitor, errGet := db.GetVisitor(r.Context(), userCtx)
				if errGet != nil {
					httpError.Json(w, errGet, "get visitor",
						"server error", http.StatusInternalServerError)
					return
				} else if !visitor.IsAlive {
					httpError.Json(w, models.ErrDead, "",
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
		var userAuthData models.UserAuthData

		if errUnmarshalBody := UnmarshalBody(r.Body, &userAuthData); errUnmarshalBody != nil {
			httpError.Json(w, errUnmarshalBody, "",
				"bad request", http.StatusBadRequest)
			return
		}

		user, errCreate := db.CreateUser(context.Background(), userAuthData)
		if errCreate != nil {
			httpError.Json(w, errCreate, "",
				"server error", http.StatusInternalServerError)
			return
		}

		user.SetRole()
		token, errGetToken := user.Token()
		if errGetToken != nil {
			httpError.Json(w, errGetToken, "",
				"server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(token)
	})
	router.Get("/login", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var userAuthData models.UserAuthData

		if errUnmarshalBody := UnmarshalBody(r.Body, &userAuthData); errUnmarshalBody != nil {
			httpError.Json(w, errUnmarshalBody, "",
				"bad request", http.StatusBadRequest)
			return
		}

		user, errGetUser := db.Login(context.Background(), userAuthData)
		if errGetUser != nil {
			if errors.Is(errGetUser, sql.ErrNoRows) {
				httpError.Json(w, errGetUser, "",
					"bad request", http.StatusBadRequest)
				return
			} else {
				httpError.Json(w, errGetUser, "",
					"server error", http.StatusInternalServerError)
				return
			}
		}

		user.SetRole()
		token, errGetToken := user.Token()
		if errGetToken != nil {
			httpError.Json(w, errGetUser, "",
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
		//ErrorPorcessing.Json(w, errResp, "failed to get body", "Bad Request", httpError.StatusBadRequest)
		return fmt.Errorf("server error: %w", errResp)
	}

	if errUnmarshalJson := json.Unmarshal(resp, v); errUnmarshalJson != nil {
		return fmt.Errorf("server error: %w", errUnmarshalJson)
	}

	return nil
}
