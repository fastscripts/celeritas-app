package main

import (
	"fmt"
	"myapp/data"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (a *application) routes() *chi.Mux {
	// middleware must come before any routes

	// add routes here
	a.App.Routes.Get("/", a.Handlers.Home)
	a.App.Routes.Get("/go-page", a.Handlers.GoPage)
	a.App.Routes.Get("/jet-page", a.Handlers.JetPage)
	a.App.Routes.Get("/sessions", a.Handlers.SessionTest)

	a.App.Routes.Get("/users/login", a.Handlers.Login)
	a.App.Routes.Post("/users/login", a.Handlers.LoginPost)
	a.App.Routes.Get("/users/logout", a.Handlers.Logout)

	/*
		a.App.Routes.Get("/test-database", func(w http.ResponseWriter, r *http.Request) {

			query := "select id, first_name from users where id = 1"
			row := a.App.DB.Pool.QueryRowContext(r.Context(), query)

			var id int
			var firstName string

			err := row.Scan(&id, &firstName)
			if err != nil {
				a.App.ErrorLog.Println("Error scanning row:", err)
				return
			}

			fmt.Fprintf(w, "%d %s", id, firstName)
		})
	*/

	a.App.Routes.Get("/create-user", func(w http.ResponseWriter, r *http.Request) {
		user := &data.User{
			FirstName: "John",
			LastName:  "Doe",
			Email:     "john.doe@example.com",
			Active:    1,
			Password:  "password",
		}
		_, err := a.Models.Users.Insert(*user)
		if err != nil {
			a.App.ErrorLog.Println("Error creating user:", err)
			return
		}

		fmt.Fprintf(w, "User created: %d %s %s", user.ID, user.FirstName, user.LastName)
	})

	a.App.Routes.Get("/get-all-users", func(w http.ResponseWriter, r *http.Request) {
		users, err := a.Models.Users.GetAll()
		if err != nil {
			a.App.ErrorLog.Println("Error getting all users:", err)
			return
		}

		for _, user := range users {
			fmt.Fprintf(w, "%d %s %s\n", user.ID, user.FirstName, user.LastName)
		}
	})

	a.App.Routes.Get("/get-user/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			a.App.ErrorLog.Println("Invalid user ID:", err)
			return
		}
		user, err := a.Models.Users.Get(id)
		if err != nil {
			a.App.ErrorLog.Println("Error getting user by ID:", err)
			return
		}

		fmt.Fprintf(w, "%d %s %s", user.ID, user.FirstName, user.LastName)
	})

	a.App.Routes.Get("/update-user/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			a.App.ErrorLog.Println("Invalid user ID:", err)
			return
		}
		user, err := a.Models.Users.Get(id)
		if err != nil {
			a.App.ErrorLog.Println("Error getting user by ID:", err)
			return
		}
		user.LastName = a.App.RandomString(10)
		err = a.Models.Users.Update(*user)
		if err != nil {
			a.App.ErrorLog.Println("Error updating user:", err)
			return
		}

		fmt.Fprintf(w, "User updated: %d %s %s", user.ID, user.FirstName, user.LastName)
	})

	// static routes
	fileServer := http.FileServer(http.Dir("./public"))
	a.App.Routes.Handle("/public/*", http.StripPrefix("/public", fileServer))

	return a.App.Routes
}
