package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
)

const URL = "https://learnku.com/go"

var wg sync.WaitGroup

func main() {

	wg.Add(1)

	go fetchArticles()

	wg.Wait()

	fmt.Println("main over")
}

func fetchArticles() {

	defer wg.Done()

	req, _ := http.NewRequest("GET", URL, nil)

	resp, _ := http.DefaultClient.Do(req)

	// 读取网页内容
	content, _ := ioutil.ReadAll(resp.Body)

	// 二进制文件转成网页内容
	respBody := string(content)

	fmt.Println(respBody)
}
