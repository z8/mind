package main

import (
	"controllers"
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

func main() {
	// Routing
	http.Handle("/style/", http.FileServer(http.Dir("static")))
	http.Handle("/script/", http.FileServer(http.Dir("static")))

	r := mux.NewRouter()

	r.HandleFunc("/insert", controllers.PostInsert).Methods("GET", "POST")
	r.HandleFunc("/{id:[0-9]+}", controllers.PostSelect).Methods("GET")
	r.HandleFunc("/{id:[0-9]+}/update", controllers.PostUpdate).Methods("GET", "POST")
	r.HandleFunc("/login", controllers.Login).Methods("GET", "POST")
	r.HandleFunc("/", controllers.Index).Methods("GET")

	http.Handle("/", r)

	err := http.ListenAndServe(":8002", nil)
	checkErr(err)
}

func checkErr(err error) {
	if err != nil {
		//log.Fatal(err)
		log.Println(err)
	}
}
