/* 将 SQL 中的 post 转为 .md 文件
 */

package main

import (
	"database/sql"
	"encoding/json"
	"os"
	//"errors"
	_ "github.com/bmizerany/pq"
	"io/ioutil"
	"log"
	"strconv"
	"time"
)

const CONFIG_NAME = "config.json"

type Post struct {
	Id         int64
	Title      string
	Category   string
	CategoryId int
	Time       int64
	Content    string
	Origin     string
}

type Config struct {
	Password string
	DbName   string
	DbConfig string
}

func getConfig() Config {
	var config Config

	configFile, err := ioutil.ReadFile(CONFIG_NAME)
	checkErr(err)

	json.Unmarshal(configFile, &config)
	return config
}

var conf = getConfig()

func main() {

	// Connect database
	db, err := sql.Open(conf.DbName, conf.DbConfig)
	checkErr(err)
	defer db.Close()

	sql := "SELECT id, title, time, origin FROM post ORDER BY time DESC"

	rows, err := db.Query(sql)
	checkErr(err)

	posts := []Post{}

	for rows.Next() {
		var p Post
		err = rows.Scan(&p.Id, &p.Title, &p.Time, &p.Origin)
		checkErr(err)
		//p.Time = time.Unix(unixTime, 0).UTC().Format("2006-01-02")

		posts = append(posts, p)
	}

	for i := 0; i < len(posts); i++ {
		postId := strconv.FormatInt(posts[i].Id, 10)
		postTime := time.Unix(posts[i].Time, 0).UTC().Format("20060102")

		fileName := "utility/posts/" + postId + "_" + postTime + "_" + posts[i].Title + ".md"
		fout, err := os.Create(fileName)
		defer fout.Close()
		checkErr(err)

		fout.WriteString(posts[i].Origin)
	}

}

func checkErr(err error) {
	if err != nil {
		log.Println(err)
	}
}
