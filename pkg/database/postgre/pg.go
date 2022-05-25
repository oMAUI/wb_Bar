package postgre

import (
	"context"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/jmoiron/sqlx"
	models2 "wb_Bar/pkg/models"
)

type DataBaseConn struct {
	conn *sqlx.DB
}

func Connection(url string) (*DataBaseConn, error) {
	db, errConn := sqlx.Connect("pgx", url)
	if errConn != nil {
		return &DataBaseConn{}, errConn
	}

	return &DataBaseConn{
		conn: db,
	}, nil
}

func (db *DataBaseConn) CreateUser(ctx context.Context, authData models2.UserAuthData) (models2.UserWithClaims, error) {
	var user models2.UserWithClaims
	if errGet := db.conn.GetContext(ctx, &user,
		`INSERT INTO bar_user(login, password, money, ppm)
				VALUES($1, $2, $3, $4)
				RETURNING id, login;
				`, authData.Login, authData.Password, 1000, 0); errGet != nil {
		return models2.UserWithClaims{}, errGet
	}

	return user, nil
}

func (db *DataBaseConn) Login(ctx context.Context, authData models2.UserAuthData) (models2.UserWithClaims, error) {
	var user models2.UserWithClaims
	if errGet := db.conn.GetContext(ctx, &user,
		`SELECT id, login
				FROM bar_user
				WHERE $1 = login and $2 = password`, authData.Login, authData.Password); errGet != nil {
		return models2.UserWithClaims{}, errGet
	}

	return user, nil
}

func (db *DataBaseConn) GetVisitor(ctx context.Context, user models2.UserAuthData) (models2.Visitor, error) {
	var visitor models2.Visitor
	if errGet := db.conn.GetContext(ctx, &visitor,
		`SELECT login, money, ppm, is_alive, last_drink
				FROM bar_user
				WHERE $1 = login`, user.Login); errGet != nil {
		return models2.Visitor{}, errGet
	}

	visitor.UpdatePpm()

	return visitor, nil
}

func (db *DataBaseConn) UpdateVisitor(ctx context.Context, visitor models2.Visitor) (models2.Visitor, error) {
	var updatedVisitor models2.Visitor
	if errGet := db.conn.GetContext(ctx, &updatedVisitor,
		`UPDATE bar_user
				SET money = $1,
				    ppm = $2,
					is_alive = $3,
					last_drink = $4
				WHERE login = $5
				RETURNING login, money, ppm, is_alive`,
		visitor.Money, visitor.Ppm, visitor.IsAlive, visitor.LastDrink, visitor.Login); errGet != nil {
		return models2.Visitor{}, errGet
	}

	return updatedVisitor, nil
}

func (db *DataBaseConn) CreateDrink(ctx context.Context, drink models2.Drink) error {
	if _, errExec := db.conn.ExecContext(ctx,
		`INSERT INTO drink(name, price, ppm)
				VALUES($1, $2, $3)`, drink.Name, drink.Price, drink.Ppm); errExec != nil {
		return errExec
	}

	return nil
}

func (db *DataBaseConn) GetDrinkList(ctx context.Context) (models2.DrinkList, error) {
	var list []models2.Drink
	if errGet := db.conn.SelectContext(ctx, &list,
		`SELECT name, price, ppm
				FROM drink`); errGet != nil {
		return models2.DrinkList{}, errGet
	}

	drinkList := models2.DrinkList{}
	drinkList.Init()
	for _, v := range list {
		drinkList.NewDrink(v)
	}

	return drinkList, nil
}

func createUser(visitor models2.Visitor) models2.UserWithClaims {
	//if visitor.ID == 1 {
	//	return models.User{
	//		User: models.Barman{
	//			ID: visitor.ID,
	//		},
	//	}
	//} else {
	//	return models.User{
	//		User: visitor,
	//	}
	//}

	return models2.UserWithClaims{}
}
