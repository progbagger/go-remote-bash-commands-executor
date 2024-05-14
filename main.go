package main

import (
	"api"
	"db"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
)

func main() {
	port := flag.Uint("port", 8888, "Port where server will be launched")
	flag.Parse()

	log.SetFlags(log.Lshortfile)

	conn, err := db.Open(getCredentials())
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("connected to database")
	defer conn.Close()

	cancelHandler, err := api.NewCancelHandler(conn)
	if err != nil {
		log.Fatalln(err)
	}
	executeHandler, err := api.NewExecuteHandler(conn, cancelHandler)
	if err != nil {
		log.Fatalln(err)
	}
	getCommandsHandler, err := api.NewGetCommandsHandler(conn)
	if err != nil {
		log.Fatalln(err)
	}
	getFullCommandHandler, err := api.NewGetFullCommandHandler(conn)
	if err != nil {
		log.Fatalln(err)
	}

	http.Handle("GET /api/commands", getCommandsHandler)
	http.Handle("GET /api/get_command", getFullCommandHandler)
	http.Handle("POST /api/launch", executeHandler)
	http.Handle("POST /api/cancel", cancelHandler)

	http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
}

func getCredentials() db.Credentials {
	credentials := db.Credentials{
		Username: os.Getenv("POSTGRES_USER"),
		Password: os.Getenv("POSTGRES_PASSWORD"),
		Database: os.Getenv("POSTGRES_DATABASE"),
		Host:     os.Getenv("POSTGRES_HOST"),
	}

	postgresPort := os.Getenv("POSTGRES_PORT")

	if postgresPort != "" {
		intPostgresPort, err := strconv.ParseInt(postgresPort, 10, 32)
		if err != nil {
			log.Println(err)
			log.Println("using default db port 5432")
		} else {
			credentials.Port = int(intPostgresPort)
		}
	}

	return credentials
}
