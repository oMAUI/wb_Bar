package PgDataBase

import (
	"context"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/jmoiron/sqlx"
	"wb_Bar/Models"
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

func (db *DataBaseConn) CreateUser(ctx context.Context, authData Models.UserAuthData) (Models.UserWithClaims, error) {
	var user Models.UserWithClaims
	if errGet := db.conn.GetContext(ctx, &user,
		`INSERT INTO bar_user(login, password, money, ppm)
				VALUES($1, $2, $3, $4)
				RETURNING id, login;
				`, authData.Login, authData.Password, 1000, 0); errGet != nil {
		return Models.UserWithClaims{}, errGet
	}

	return user, nil
}

func (db *DataBaseConn) Login(ctx context.Context, authData Models.UserAuthData) (Models.UserWithClaims, error) {
	var user Models.UserWithClaims
	if errGet := db.conn.GetContext(ctx, &user,
		`SELECT id, login
				FROM bar_user
				WHERE $1 = login and $2 = password`, authData.Login, authData.Password); errGet != nil {
		return Models.UserWithClaims{}, errGet
	}

	return user, nil
}

func (db *DataBaseConn) GetVisitor(ctx context.Context, user Models.UserAuthData) (Models.Visitor, error) {
	var visitor Models.Visitor
	if errGet := db.conn.GetContext(ctx, &visitor,
		`SELECT login, money, ppm, is_alive, last_drink
				FROM bar_user
				WHERE $1 = login`, user.Login); errGet != nil {
		return Models.Visitor{}, errGet
	}

	visitor.UpdatePpm()

	return visitor, nil
}

func (db *DataBaseConn) UpdateVisitor(ctx context.Context, visitor Models.Visitor) (Models.Visitor, error) {
	var updatedVisitor Models.Visitor
	if errGet := db.conn.GetContext(ctx, &updatedVisitor,
		`UPDATE bar_user
				SET money = $1,
				    ppm = $2,
					is_alive = $3,
					last_drink = $4
				WHERE login = $5
				RETURNING login, money, ppm, is_alive`,
		visitor.Money, visitor.Ppm, visitor.IsAlive, visitor.LastDrink, visitor.Login); errGet != nil {
		return Models.Visitor{}, errGet
	}

	return updatedVisitor, nil
}

func (db *DataBaseConn) CreateDrink(ctx context.Context, drink Models.Drink) error {
	if _, errExec := db.conn.ExecContext(ctx,
		`INSERT INTO drink(name, price, ppm)
				VALUES($1, $2, $3)`, drink.Name, drink.Price, drink.Ppm); errExec != nil {
		return errExec
	}

	return nil
}

func (db *DataBaseConn) GetDrinkList(ctx context.Context) (Models.DrinkList, error) {
	var list []Models.Drink
	if errGet := db.conn.SelectContext(ctx, &list,
		`SELECT name, price, ppm
				FROM drink`); errGet != nil {
		return Models.DrinkList{}, errGet
	}

	drinkList := Models.DrinkList{}
	drinkList.Init()
	for _, v := range list {
		drinkList.NewDrink(v)
	}

	return drinkList, nil
}

func createUser(visitor Models.Visitor) Models.UserWithClaims {
	//if visitor.ID == 1 {
	//	return Models.User{
	//		User: Models.Barman{
	//			ID: visitor.ID,
	//		},
	//	}
	//} else {
	//	return Models.User{
	//		User: visitor,
	//	}
	//}

	return Models.UserWithClaims{}
}
