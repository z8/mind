package config

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

const CONFIG_NAME = "config.json"

type Config struct {
	DbName   string
	DbConfig string
	Password string
}

func GetConfig() Config {
	var config Config

	configFile, err := ioutil.ReadFile(CONFIG_NAME)
	checkErr(err)

	json.Unmarshal(configFile, &config)
	return config
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
