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

const CONFIG_NAME = "config.json"

type Config struct {
	Password string
	DbName   string
	DbConfig string
}

type Category struct {
	Id   int
	Name string
}

type Post struct {
	Id         int64
	Title      string
	Category   string
	CategoryId int
	Time       int64
	Content    string
	Origin     string
}

type Data struct {
	Title    string
	Category string
	Time     int64
	Content  string
	Origin   string
}

func getConfig() Config {
	var config Config

	configFile, err := ioutil.ReadFile(CONFIG_NAME)
	checkErr(err)

	json.Unmarshal(configFile, &config)
	return config
}

var conf = getConfig()
var sess = sessions.NewCookieStore([]byte("auth"))

/** Render
 * 将 string 转为 Data
 */
func Render(s string) (d Data, e error) {

	lines := strings.Split(s, "\n")
	recording := true
	c := bytes.Buffer{}

	for i := 0; i < len(lines); i++ {
		if recording && strings.Index(lines[i], "---") == 0 {
			recording = false
		} else if recording {
			if strings.Index(lines[i], "#") == 0 {
				d.Title = strings.Trim(strings.Split(lines[i], "#")[1], " \r\n")
			} else if strings.Index(lines[i], "- category:") == 0 {
				d.Category = strings.Trim(strings.Split(lines[i], ":")[1], " \r\n")
			} else if strings.Index(lines[i], "- time:") == 0 {
				s := strings.Trim(strings.Split(lines[i], ":")[1], " \r\n")
				t, err := time.Parse(`2006-01-02`, s) // 字符串格式化方式
				if err != nil {
					e = err
				}
				d.Time = t.Unix()
			}
		} else {
			c.WriteString(lines[i])
			c.WriteString("\n")
		}
	}

	content := c.String()
	d.Content = string(blackfriday.MarkdownCommon([]byte(content)))

	d.Origin = s

	return d, e
}

/** Post Render
 * 将 Data 转为 Post
 */
func (post Post) Render(s string) (r Post, err error) {
	data, err := Render(s)
	r.Title = data.Title
	r.Category = data.Category
	r.Time = data.Time
	r.Content = data.Content
	r.Origin = data.Origin
	return r, err
}

func List(year int, categoryName string, pageId int) *[]Post {
	posts := []Post{}

	// Connect database
	db, err := sql.Open(conf.DbName, conf.DbConfig)
	checkErr(err)
	defer db.Close()

	categorySQL := ""
	if categoryName != "" {
		categorySQL = " AND category='" + categoryName + "' "
	} else {
		categorySQL = " "
	}

	dateSQL := ""
	if year != 0 {
		dateSQL = " AND TO_CHAR(TO_TIMESTAMP(time), 'YYYY')='" + strconv.Itoa(year) + "' "
	} else {
		dateSQL = " "
	}

	pageSQL := ""
	if pageId != 0 {
		pageSQL = " OFFSET " + strconv.Itoa((pageId-1)*10) + " LIMIT 10 "
	} else {
		pageSQL = " "
	}

	sql := "SELECT id, title, time FROM post WHERE 1=1" + categorySQL + dateSQL + "ORDER BY time DESC" + pageSQL

	//rows, err := db.Query("SELECT id, title, time FROM post ORDER BY time DESC OFFSET 0 LIMIT 10")
	rows, err := db.Query(sql)
	checkErr(err)

	for rows.Next() {
		var p Post
		err = rows.Scan(&p.Id, &p.Title, &p.Time)
		checkErr(err)
		//p.Time = time.Unix(unixTime, 0).UTC().Format("2006-01-02")

		posts = append(posts, p)
	}

	return &posts
}

func (post Post) Insert() (r Post, err error) {
	// Connect database
	db, err := sql.Open(conf.DbName, conf.DbConfig)
	defer db.Close()

	err = db.QueryRow("INSERT INTO post(title, category, categoryid, time, content, origin) VALUES($1, $2, $3, $4, $5, $6) RETURNING id;", &post.Title, &post.Category, &post.CategoryId, &post.Time, &post.Content, &post.Origin).Scan(&post.Id)

	return post, err
}

func (post Post) Select() (r Post, err error) {
	if post.Id == 0 {
		err = errors.New("post Id 不可为空")
	}

	// Connect database
	db, err := sql.Open(conf.DbName, conf.DbConfig)
	defer db.Close()

	queryPost := db.QueryRow("SELECT id, title, category, categoryid, time, content, origin FROM post WHERE id = $1", &post.Id)
	queryPost.Scan(&r.Id, &r.Title, &r.Category, &r.CategoryId, &r.Time, &r.Content, &r.Origin)

	return r, err
}

func (p Post) Update() (r Post, e error) {

	// Connect database
	db, err := sql.Open(conf.DbName, conf.DbConfig)
	defer db.Close()

	if p.Id == 0 {
		e = errors.New("Post Id is Must")
	}

	c := Category{0, p.Category}

	c, err = c.Select()

	if c.Id == 0 {
		c.Name = p.Category
		c, err = c.Insert()
		if err != nil {
			e = err
		}
	}

	stmt, err := db.Prepare("UPDATE post SET title=$1, category=$2, categoryid=$3, time=$4, content=$5, origin=$6 WHERE id=$7")
	if err != nil {
		e = err
	}

	_, err = stmt.Exec(&p.Title, &p.Category, &c.Id, &p.Time, &p.Content, &p.Origin, &p.Id)
	if err != nil {
		e = err
	}

	return p, e
}

func (category Category) Insert() (r Category, err error) {
	// Connect database
	db, err := sql.Open(conf.DbName, conf.DbConfig)
	defer db.Close()

	err = db.QueryRow("INSERT INTO category(name) VALUES($1) RETURNING id;", &category.Name).Scan(&category.Id)

	return category, err
}

func (category Category) Select() (r Category, err error) {
	// Connect database
	db, err := sql.Open(conf.DbName, conf.DbConfig)
	defer db.Close()

	if category.Id == 0 {
		err = db.QueryRow("SELECT id, name FROM category WHERE name = $1;", &category.Name).Scan(&r.Id, &r.Name)
	} else {
		err = db.QueryRow("SELECT id, name FROM category WHERE id = $1;", &category.Id).Scan(&r.Id, &r.Name)
	}

	return r, err
}

/** CONTROLLERS
 *
 */

// 模板函数
func tmpTimeFormat(unixTime int64) string {
	return time.Unix(unixTime, 0).UTC().Format("2006-01-02")
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	var posts = List(0, "", 1)

	t, err := template.New("").Funcs(template.FuncMap{"tmpTimeFormat": tmpTimeFormat}).ParseFiles("views/list.html", "views/header.html", "views/footer.html")
	checkErr(err)

	var templateData struct {
		PageName string
		Posts    *[]Post
	}

	templateData.Posts = posts

	t.ExecuteTemplate(w, "content", &templateData)
}

func CategoryHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	categoryName := vars["categoryName"]
	pageId := r.FormValue("p")

	p := 1
	if pageId != "" {
		p, _ = strconv.Atoi(pageId)
	}

	var posts = List(0, categoryName, p)

	t, err := template.New("").Funcs(template.FuncMap{"tmpTimeFormat": tmpTimeFormat}).ParseFiles("views/list.html", "views/header.html", "views/footer.html")
	checkErr(err)

	var templateData struct {
		PageName string
		Posts    *[]Post
	}

	templateData.PageName = strings.ToUpper(categoryName)
	templateData.Posts = posts

	t.ExecuteTemplate(w, "content", &templateData)
}

func DateHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	date := vars["date"]

	d, _ := strconv.Atoi(date)
	var posts = List(d, "", 0)

	t, err := template.New("").Funcs(template.FuncMap{"tmpTimeFormat": tmpTimeFormat}).ParseFiles("views/list.html", "views/header.html", "views/footer.html")
	checkErr(err)

	var templateData struct {
		PageName string
		Posts    *[]Post
	}

	templateData.PageName = strings.ToUpper(date)
	templateData.Posts = posts

	t.ExecuteTemplate(w, "content", &templateData)

}

func RecentHandler(w http.ResponseWriter, r *http.Request) {
	pageId := r.FormValue("p")

	p := 1
	if pageId != "" {
		p, _ = strconv.Atoi(pageId)
	}

	var posts = List(0, "", p)

	t, err := template.New("").Funcs(template.FuncMap{"tmpTimeFormat": tmpTimeFormat}).ParseFiles("views/list.html", "views/header.html", "views/footer.html")

	checkErr(err)
	t.ExecuteTemplate(w, "content", &posts)
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
		var p Post
		var c Category

		p, err = p.Render(r.FormValue("origin"))
		checkErr(err)

		c.Name = p.Category
		if p.Category == "" {
			p.Category = "note"
		}

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

func main() {
	http.Handle("/styles/", http.FileServer(http.Dir("statics")))
	http.Handle("/scripts/", http.FileServer(http.Dir("statics")))

	r := mux.NewRouter()

	// Post
	r.HandleFunc("/insert", InsertHandler).Methods("GET", "POST")
	r.HandleFunc("/{id:[0-9]+}", SelectHandler).Methods("GET")
	r.HandleFunc("/{id:[0-9]+}/update", UpdateHandler).Methods("GET", "POST")
	// Category
	r.HandleFunc("/go/{categoryName}", CategoryHandler).Methods("GET")
	// Date
	r.HandleFunc("/date/{date}", DateHandler).Methods("GET")
	// Auth
	r.HandleFunc("/login", LoginHandler).Methods("GET", "POST")
	// Home
	r.HandleFunc("/recent", RecentHandler).Methods("GET")
	r.HandleFunc("/", IndexHandler).Methods("GET")

	http.Handle("/", r)

	err := http.ListenAndServe(":8001", nil)
	checkErr(err)
}

func checkErr(err error) {
	if err != nil {
		log.Println(err)
	}
}

/*
CREATE TABLE category (id SERIAL PRIMARY KEY,name VARCHAR(100));
CREATE TABLE post (id SERIAL PRIMARY KEY,title VARCHAR(100) NOT NULL,category VARCHAR(100) NOT NULL,categoryid INTEGER NOT NULL REFERENCES category(id),time INTEGER NOT NULL,content TEXT,origin TEXT NOT NULL);
*/
