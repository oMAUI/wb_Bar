package main

import (
	"fmt"
	"net/http"
	"wb_Bar/PgDataBase"
	"wb_Bar/Router"
)

func main() {
	conn, errConn := PgDataBase.Connection("postgresql://maui:maui@127.0.0.1:5432/postgres")
	if errConn != nil {
		fmt.Println("failed to connect db: ", errConn)
		return
	}

	router := Router.Route(conn)
	http.ListenAndServe(":3030", router)
}
