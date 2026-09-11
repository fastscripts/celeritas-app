package handlers

import "net/http"

func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	err := h.App.Render.Page(w, r, "login", nil, nil)
	if err != nil {
		h.App.ErrorLog.Println("error login:", err)
	}
}

func (h *Handlers) LoginPost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		h.App.ErrorLog.Println("error parsing login form:", err)
		return
	}

	email := r.Form.Get("email")
	password := r.Form.Get("password")

	user, err := h.Models.Users.GetByEmail(email)
	if err != nil {
		h.App.ErrorLog.Println("error getting user:", err)
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	matches, err := user.PasswordMatches(password)
	if err != nil {
		h.App.ErrorLog.Println("error checking password:", err)
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}
	if !matches {
		h.App.ErrorLog.Println("invalid password")
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	h.App.Session.Put(r.Context(), "userID", user.ID)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	h.App.Session.RenewToken(r.Context())
	h.App.Session.Remove(r.Context(), "userID")

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
