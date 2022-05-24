package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"sync"
)

const URL = "https://learnku.com/go?filter=created_at_1_month&order=score&l=y"

var wg sync.WaitGroup

func main() {
	wg.Add(1)

	go fetchArticles()

	wg.Wait()

	fmt.Println("完成抓取....")
}

func fetchArticles() {

	defer wg.Done()

	req, _ := http.NewRequest("GET", URL, nil)

	resp, _ := http.DefaultClient.Do(req)

	// 读取网页内容
	content, _ := ioutil.ReadAll(resp.Body)

	// 二进制文件转成网页内容
	respBody := string(content)

	// 正则表达式匹配
	reg := regexp.MustCompile(`<span class="topic-title">(?s:(.*?))</span>`)

	if reg == nil {
		fmt.Println("regex err")
		return
	}

	result := reg.FindAllStringSubmatch(respBody, -1)

	for _, values := range result {

		// 这里可以去除html元素和空格
		title := trimHtml(values[1])

		fmt.Println(title)
	}
}

// 去除html元素标签
func trimHtml(src string) string {

	//将HTML标签全转换成小写
	re, _ := regexp.Compile("\\<[\\S\\s]+?\\>")
	src = re.ReplaceAllStringFunc(src, strings.ToLower)

	//去除STYLE
	re, _ = regexp.Compile("\\<style[\\S\\s]+?\\</style\\>")
	src = re.ReplaceAllString(src, "")

	//去除SCRIPT
	re, _ = regexp.Compile("\\<script[\\S\\s]+?\\</script\\>")
	src = re.ReplaceAllString(src, "")

	//去除所有尖括号内的HTML代码，并换成换行符
	re, _ = regexp.Compile("\\<[\\S\\s]+?\\>")
	src = re.ReplaceAllString(src, "\n")

	//去除连续的换行符
	re, _ = regexp.Compile("\\s{2,}")
	src = re.ReplaceAllString(src, "\n")

	return strings.TrimSpace(src)
}
