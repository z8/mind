package model

import (
	"bytes"
	"config"
	"database/sql"
	"errors"
	_ "github.com/bmizerany/pq"
	"github.com/russross/blackfriday"
	"log"
	"strings"
	"time"
)

var conf = config.GetConfig()

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

func (post Post) Render(s string) (r Post, err error) {
	data, err := Render(s)
	post.Title = data.Title
	post.Category = data.Category
	post.Time = data.Time
	post.Content = data.Content
	post.Origin = data.Origin
	return post, err
}

func List(int) *[]Post {
	posts := []Post{}

	// Connect database
	db, err := sql.Open(conf.DbName, conf.DbConfig)
	checkErr(err)
	defer db.Close()

	rows, err := db.Query("SELECT id, title, time FROM post ORDER BY time DESC OFFSET 0 LIMIT 10")
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

func checkErr(err error) {
	if err != nil {
		log.Println(err)
	}
}
