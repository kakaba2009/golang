package program3

import (
	"crypto/md5"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/kakaba2009/golang/program2"
)

var hp string = "https://www.secretchina.com"

func IsDownloaded(dir string, name string) bool {
	full := HashName(name, dir)
	if _, err := os.Stat(full); os.IsNotExist(err) {
		return false
	}
	return true
}

func HashName(name string, dir string) string {
	md5s := md5.Sum([]byte(name))
	hash := fmt.Sprintf("%x", md5s)
	full := dir + "/" + hash + ".txt"
	return full
}

func WriteFile(dir string, name string, content string) error {
	full := HashName(name, dir)
	f, err := os.Create(full)
	if err != nil {
		log.Println(err)
		return err
	}
	defer f.Close()

	_, err = f.WriteString(content)
	if err != nil {
		log.Println(err)
		return err
	}
	fmt.Println("WriteFile done")
	return nil
}

func ReadSubPage(job chan string, dir string, wg *sync.WaitGroup) {
	fmt.Println("ReadSubPage ... ")
	defer wg.Done()
	for data := range job {
		links := strings.Split(data, "|")
		url := links[0]
		if strings.Contains(url, hp) || strings.Contains(url, "http:") || strings.Contains(url, "https:") {
			continue
		}
		name := links[1]
		if IsDownloaded(dir, name) {
			fmt.Println(url + " already downloaded, skip ...")
			continue
		}
		res, err := http.Get(hp + url)
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
		err = WriteFile(dir, name, string(content))
		if err != nil {
			log.Println(err)
		}
		res.Body.Close()
	}
}

func ReadMainPage(link string, dir string) error {
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

	threads := 5
	wg.Add(threads)
	for i := 1; i <= threads; i++ {
		go ReadSubPage(job, dir, &wg)
	}

	wg.Wait()
	return res.Body.Close()
}

func Download() error {
	fmt.Println("Start to download ... ")
	dir := time.Now().Format("2006-01-02")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.Mkdir(dir, 0755)
	}
	return ReadMainPage(hp, dir)
}

func Main() error {
	for {
		err := Download()
		if err != nil {
			log.Println(err)
			return err
		}
		fmt.Println("Sleep ...")
		time.Sleep(time.Minute)
	}
}
