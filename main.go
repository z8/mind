package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	_ "github.com/bmizerany/pq"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/russross/blackfriday"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var sess = sessions.NewCookieStore([]byte("auth"))

type Config struct {
	DbName   string
	DbConfig string
	Password string
}

type Post struct {
	Id       int64
	Title    string
	Category string
	Time     int64
	Content  string
	Origin   string
}

type Category struct {
	Id   int
	Name string
}

func main() {

	// Get Config
	getConfig()

	// Routing
	http.Handle("/style/", http.FileServer(http.Dir("static")))
	http.Handle("/script/", http.FileServer(http.Dir("static")))

	r := mux.NewRouter()
	r.HandleFunc("/insert", InsertHandler).Methods("GET", "POST")
	r.HandleFunc("/{id:[0-9]+}", SelectHandler).Methods("GET")
	r.HandleFunc("/{id:[0-9]+}/update", UpdateHandler).Methods("GET", "POST")
	r.HandleFunc("/login", LoginHandler).Methods("GET", "POST")
	r.HandleFunc("/", IndexHandler).Methods("GET")

	http.Handle("/", r)

	err := http.ListenAndServe(":8001", nil)
	checkErr(err)
}

var C Config

func getConfig() {

	configFile, err := ioutil.ReadFile("./config.json")
	checkErr(err)

	json.Unmarshal(configFile, &C)

}

func IndexHandler(w http.ResponseWriter, r *http.Request) {

	type Post struct {
		Id       int
		Title    string
		unixTime int64
		Time     string
	}

	Posts := []Post{}

	// Connect database
	db, err := sql.Open(C.DbName, C.DbConfig)
	checkErr(err)
	defer db.Close()

	rows, err := db.Query("SELECT id, title, time FROM post ORDER BY time DESC OFFSET 0 LIMIT 10")
	checkErr(err)

	for rows.Next() {
		var p Post
		err = rows.Scan(&p.Id, &p.Title, &p.unixTime)
		checkErr(err)
		p.Time = time.Unix(p.unixTime, 0).UTC().Format("2006-01-02")

		Posts = append(Posts, p)
	}

	t, err := template.ParseFiles("views/index.html", "views/header.html", "views/footer.html")
	checkErr(err)
	t.ExecuteTemplate(w, "content", &Posts)

}

func InsertHandler(w http.ResponseWriter, r *http.Request) {

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
		p, err := Render(r.FormValue("origin"))
		checkErr(err)

		p, err = p.Insert()
		checkErr(err)

		http.Redirect(w, r, "/"+strconv.FormatInt(p.Id, 10), http.StatusFound)
		//fmt.Fprintf(w, p.Origin)

	}
}

func UpdateHandler(w http.ResponseWriter, r *http.Request) {

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
		t.ExecuteTemplate(w, "content", nil)
	case "POST":
		p, err := Render(r.FormValue("origin"))
		checkErr(err)

		p.Id, _ = strconv.ParseInt(id, 10, 32)

		p, err = p.Update()
		checkErr(err)

		http.Redirect(w, r, "/"+strconv.FormatInt(p.Id, 10), http.StatusFound)
		//fmt.Fprintf(w, p.Origin)

	}

}

func SelectHandler(w http.ResponseWriter, r *http.Request) {
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

	// Connect database
	db, err := sql.Open(C.DbName, C.DbConfig)
	if err != nil {
		checkErr(err)
	}
	defer db.Close()

	var p Post
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

func LoginHandler(w http.ResponseWriter, r *http.Request) {

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
		if password == C.Password {
			session, _ := sess.Get(r, "auth")
			session.Values["auth"] = C.Password
			session.Save(r, w)
			http.Redirect(w, r, "/", http.StatusFound)
		} else {
			http.Redirect(w, r, "/login", http.StatusFound)
		}
	}

}

func Render(s string) (p Post, e error) {

	lines := strings.Split(s, "\n")
	recording := true
	c := bytes.Buffer{}

	for i := 0; i < len(lines); i++ {
		if recording && strings.Index(lines[i], "---") == 0 {
			recording = false
		} else if recording {
			if strings.Index(lines[i], "#") == 0 {
				p.Title = strings.Trim(strings.Split(lines[i], "#")[1], " \r\n")
			} else if strings.Index(lines[i], "- category:") == 0 {
				p.Category = strings.Trim(strings.Split(lines[i], ":")[1], " \r\n")
			} else if strings.Index(lines[i], "- time:") == 0 {
				s := strings.Trim(strings.Split(lines[i], ":")[1], " \r\n")
				t, err := time.Parse(`2006-01-02`, s) // 字符串格式化方式
				if err != nil {
					panic(err)
				}
				p.Time = t.Unix()
			}
		} else {
			c.WriteString(lines[i])
			c.WriteString("\n")
		}
	}

	content := c.String()
	p.Content = string(blackfriday.MarkdownCommon([]byte(content)))

	if p.Category == "" {
		p.Category = "uncategories"
	}
	if p.Time == 0 {
		// 当前时间
	}

	p.Origin = s

	if p.Title == "" || p.Category == "" || p.Time == 0 {
		e = errors.New("Title or Category or Time empty!")
	} else {
		e = nil
	}

	return p, e

}

func (p Post) Insert() (r Post, e error) {

	// Connect database
	db, err := sql.Open(C.DbName, C.DbConfig)
	if err != nil {
		e = err
	}
	defer db.Close()

	c := Category{0, p.Category}

	c, err = c.Select()

	if c.Id == 0 {
		c.Name = p.Category
		c, err = c.Insert()
		checkErr(err)
	}

	err = db.QueryRow("INSERT INTO post(title, categoryid, time, content, origin) VALUES($1, $2, $3, $4, $5) RETURNING id;", &p.Title, &c.Id, &p.Time, &p.Content, &p.Origin).Scan(&p.Id)
	if err != nil {
		e = err
	}

	return p, e
}

func (p Post) Update() (r Post, e error) {

	// Connect database
	db, err := sql.Open(C.DbName, C.DbConfig)
	if err != nil {
		e = err
	}
	defer db.Close()

	if p.Id == 0 {
		e = errors.New("Post Id is Must")
	}

	c := Category{0, p.Category}

	c, err = c.Select()

	if c.Id == 0 {
		c.Name = p.Category
		c, err = c.Insert()
		checkErr(err)
	}

	stmt, err := db.Prepare("UPDATE post SET title=$1, categoryid=$2, time=$3, content=$4, origin=$5 WHERE id = $6")
	if err != nil {
		e = err
	}

	_, err = stmt.Exec(&p.Title, &c.Id, &p.Time, &p.Content, &p.Origin, &p.Id)
	if err != nil {
		e = err
	}

	return p, e
}

func (p Post) Select() (r Post, e error) {

	if p.Id == 0 {
		e = errors.New("Post Id is Must")
	}

	// Connect database
	db, err := sql.Open(C.DbName, C.DbConfig)
	if err != nil {
		e = err
	}
	defer db.Close()

	queryPost := db.QueryRow("SELECT post.id, post.title, category.name, post.time, post.content, post.origin FROM post INNER JOIN category ON post.categoryid = category.id WHERE post.id = $1", &p.Id)
	queryPost.Scan(&r.Id, &r.Title, &r.Category, &r.Time, &r.Content, &r.Origin)

	return r, e
}

func (c Category) Insert() (r Category, e error) {

	// Connect database
	db, err := sql.Open(C.DbName, C.DbConfig)
	if err != nil {
		e = err
	}
	defer db.Close()

	err = db.QueryRow("INSERT INTO category(name) VALUES($1) RETURNING id, name;", &c.Name).Scan(&r.Id, &r.Name)
	if err != nil {
		e = err
	}

	return r, e

}

func (c Category) Update() (r Category, e error) {

	if c.Id == 0 {
		e = errors.New("category id is must")
	}

	// Connect database
	db, err := sql.Open(C.DbName, C.DbConfig)
	if err != nil {
		e = err
	}
	defer db.Close()

	stmt, err := db.Prepare("UPDATE category SET name=$1 WHERE id = $2")
	if err != nil {
		e = err
	}

	_, err = stmt.Exec(&c.Name, &c.Id)
	if err != nil {
		e = err
	}

	return c, e

}

func (c Category) Select() (r Category, e error) {

	// Connect database
	db, err := sql.Open(C.DbName, C.DbConfig)
	if err != nil {
		e = err
	}
	defer db.Close()

	if c.Id == 0 {
		err := db.QueryRow("SELECT id, name FROM category WHERE name = $1;", &c.Name).Scan(&r.Id, &r.Name)
		if err != nil {
			e = err
		}
	} else {
		err := db.QueryRow("SELECT id, name FROM category WHERE id = $1;", &c.Id).Scan(&r.Id, &r.Name)
		if err != nil {
			e = err
		}
	}

	return r, e

}

func checkErr(err error) {

	if err != nil {
		//log.Fatal(err)
		log.Println(err)
	}

}

/*

# Hello Mind
- category: mind
- time: 2013-03-13

---

This is a test post.

*/

/*

CREATE TABLE category (
	id SERIAL PRIMARY KEY,
	name VARCHAR(100)
);

CREATE TABLE post (
	id SERIAL PRIMARY KEY,
	title VARCHAR(100) NOT NULL,
	categoryid INTEGER NOT NULL REFERENCES category(id),
	time INTEGER NOT NULL,
	content TEXT,
	origin TEXT NOT NULL
);

*/
