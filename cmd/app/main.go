package main

import (
	"fmt"
	"net/http"
	"wb_Bar/internal/router"
	"wb_Bar/pkg/database/postgre"
)

func main() {
	conn, errConn := postgre.Connection("postgresql://maui:maui@127.0.0.1:5432/postgres")
	if errConn != nil {
		fmt.Println("failed to connect db: ", errConn)
		return
	}

	router := router.Route(conn)
	http.ListenAndServe(":3030", router)
}
