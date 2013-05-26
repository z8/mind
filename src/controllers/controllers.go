package controllers

import (
	//_ "github.com/bmizerany/pq"
	"config"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"html/template"
	"log"
	. "model"
	"net/http"
	"strconv"
	"time"
)

var conf = config.GetConfig()
var sess = sessions.NewCookieStore([]byte("auth"))

func StringTime(unixTime int64) string {
	return time.Unix(unixTime, 0).UTC().Format("2006-01-02")
}

func Index(w http.ResponseWriter, r *http.Request) {
	var posts = List(1)

	t, err := template.New("").Funcs(template.FuncMap{"stringTime": StringTime}).ParseFiles("views/list.html", "views/header.html", "views/footer.html")

	checkErr(err)
	t.ExecuteTemplate(w, "content", &posts)
}

func PostInsert(w http.ResponseWriter, r *http.Request) {

	session, _ := sess.Get(r, "auth")
	auth := session.Values["auth"]

	if auth == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	t, err := template.ParseFiles("views/insert.html", "views/header.html", "views/footer.html")
	checkErr(err)

	switch r.Method {

	case "GET":
		t.ExecuteTemplate(w, "content", nil)

	case "POST":
		var p Post
		var c Category

		p, err = p.Render(r.FormValue("origin"))
		checkErr(err)

		c.Name = p.Category
		c, err = c.Select()
		checkErr(err)

		if c.Id == 0 {
			c.Name = p.Category
			c, err = c.Insert()
			checkErr(err)
		}

		p.CategoryId = c.Id

		p, err = p.Insert()
		checkErr(err)

		http.Redirect(w, r, "/"+strconv.FormatInt(p.Id, 10), http.StatusFound)
	}
}

func PostSelect(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var data struct {
		Id       int64
		Title    string
		Category string
		Time     string
		Content  template.HTML
		Origin   string
	}

	var p Post
	var err error
	p.Id, err = strconv.ParseInt(id, 10, 32)
	checkErr(err)

	p, err = p.Select()

	data.Id = p.Id
	data.Title = p.Title
	data.Category = p.Category
	data.Time = time.Unix(p.Time, 0).UTC().Format("2006-01-02")
	data.Content = template.HTML(p.Content)
	data.Origin = p.Origin

	t, err := template.ParseFiles("views/select.html", "views/header.html", "views/footer.html")
	checkErr(err)

	t.ExecuteTemplate(w, "content", data)
}

func PostUpdate(w http.ResponseWriter, r *http.Request) {

	session, _ := sess.Get(r, "auth")
	auth := session.Values["auth"]

	if auth == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	var p Post
	p.Id, _ = strconv.ParseInt(id, 10, 32)

	p, err := p.Select()
	checkErr(err)

	switch r.Method {
	case "GET":
		t, err := template.ParseFiles("views/insert.html", "views/header.html", "views/footer.html")
		checkErr(err)
		t.ExecuteTemplate(w, "content", &p)
	case "POST":
		p, err := p.Render(r.FormValue("origin"))
		checkErr(err)

		p.Id, _ = strconv.ParseInt(id, 10, 32)

		p, err = p.Update()
		checkErr(err)

		http.Redirect(w, r, "/"+strconv.FormatInt(p.Id, 10), http.StatusFound)
	}

}

func Login(w http.ResponseWriter, r *http.Request) {

	t, err := template.ParseFiles("views/login.html", "views/header.html", "views/footer.html")
	checkErr(err)

	switch r.Method {
	case "GET":
		session, _ := sess.Get(r, "auth")
		auth := session.Values["auth"]
		if auth == nil {
			t.ExecuteTemplate(w, "content", nil)
		} else {
			http.Redirect(w, r, "/", http.StatusFound)
		}

	case "POST":
		password := r.FormValue("password")
		if password == conf.Password {
			session, _ := sess.Get(r, "auth")
			session.Values["auth"] = conf.Password
			session.Save(r, w)
			http.Redirect(w, r, "/", http.StatusFound)
		} else {
			http.Redirect(w, r, "/login", http.StatusFound)
		}
	}

}

func checkErr(err error) {
	if err != nil {
		//log.Fatal(err)
		log.Println(err)
	}
}
