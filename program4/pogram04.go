package program4

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/kakaba2009/golang/global"
	"github.com/kakaba2009/golang/program2"
	"github.com/kakaba2009/golang/program3"
)

type ConfigFile = global.ConfigFile

func ReadSubPage(job chan string, dir string, config ConfigFile, wg *sync.WaitGroup) {
	fmt.Println("ReadSubPage ... ")
	defer wg.Done()
	for data := range job {
		links := strings.Split(data, "|")
		url := links[0]
		if strings.Contains(url, "http:") || strings.Contains(url, "https:") {
			continue
		}
		name := links[1]
		if program3.IsDownloaded(dir, name) {
			fmt.Println(url + " already downloaded, skip ...")
			continue
		}
		res, err := http.Get(config.Url + url)
		if err != nil {
			log.Println(err)
			continue
		}
		doc, err := goquery.NewDocumentFromReader(res.Body)
		if err != nil {
			log.Println(err)
			continue
		}
		content := doc.Find("p").Text()
		err = program3.WriteFile(dir, name, string(content))
		if err != nil {
			log.Println(err)
		}
		res.Body.Close()
	}
}

func ReadMainPage(link string, dir string, config ConfigFile) error {
	var wg sync.WaitGroup

	fmt.Println("ReadMainPage ... ")
	job := make(chan string)

	res, err := http.Get(link)
	if err != nil {
		log.Println(err)
		return err
	}

	wg.Add(1)
	go program2.FindLinks(res, job, &wg)

	threads := config.Threads
	wg.Add(threads)
	for i := 1; i <= threads; i++ {
		go ReadSubPage(job, dir, config, &wg)
	}

	wg.Wait()
	return res.Body.Close()
}

func Download(config ConfigFile) error {
	fmt.Println("Start to download ... ")
	dir := time.Now().Format("2006-01-02")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.Mkdir(dir, 0755)
	}
	return ReadMainPage(config.Url, dir, config)
}

func saveJson() error {
	message := ConfigFile{
		Url:      "https://www.secretchina.com",
		Threads:  5,
		Interval: 1,
	}
	b, err := json.Marshal(message)
	if err != nil {
		fmt.Print(err)
		return err
	}
	return os.WriteFile("program4/config.json", b, 0755)
}

func Main() error {
	pwd, _ := os.Getwd()
	fmt.Println(pwd)

	file := "program4/config.json"

	if len(os.Args) >= 2 {
		// Use config file from command line
		file = os.Args[1]
		fmt.Println("Use config file " + file)
	}

	conFile, err := os.ReadFile(file)
	if err != nil {
		fmt.Print(err)
		return err
	}
	var config ConfigFile
	err = json.Unmarshal(conFile, &config)
	fmt.Println(config)
	for {
		err = Download(config)
		if err != nil {
			fmt.Print(err)
			return err
		}
		fmt.Println("Sleep " + strconv.Itoa(config.Interval) + " minutes")
		time.Sleep(time.Minute * time.Duration(config.Interval))
	}
}
