package main

import (
	"bytes"
	"encoding/json"
	"github.com/russross/blackfriday"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	PATH = "utility/posts"
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

func main() {
	var wg sync.WaitGroup
	tokens := make(chan int, runtime.NumCPU())

	err := filepath.Walk(PATH, func(path string, info os.FileInfo, err error) error {
		tokens <- 1 // 获取令牌
		wg.Add(1)

		go func() {
			if strings.HasSuffix(info.Name(), ".md") {
				fileData, err := ioutil.ReadFile(path)
				checkErr(err)
				//defer fileData.Close()

				origin := string(fileData)

				var p Post
				p, err = p.Render(origin)
				checkErr(err)

				log.Println(p.Title)
			}

			wg.Done()
			<-tokens
		}()

		wg.Wait()
		return nil
	})

	checkErr(err)
}

func checkErr(err error) {
	if err != nil {
		log.Println(err)
	}
}
