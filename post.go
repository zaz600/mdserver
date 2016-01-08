package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"strings"
	"sync"

	"github.com/russross/blackfriday"
)

type post struct {
	Title   string
	Body    template.HTML
	ModTime int64
}

type postArray struct {
	Items map[string]post
	sync.RWMutex
}

func newPostArray() *postArray {
	p := postArray{}
	p.Items = make(map[string]post)
	return &p
}

// Get Загружает markdown-файл и конвертирует его в HTML
// Возвращает объект типа Post
// Если путь не существует или является каталогом, то возвращаем ошибку
func (p *postArray) Get(md string) (post, int, error) {
	info, err := os.Stat(md)
	if err != nil {
		if os.IsNotExist(err) {
			// файл не существует
			return post{}, 404, err
		}
		return post{}, 500, err
	}
	if info.IsDir() {
		// не файл, а папка
		return post{}, 404, fmt.Errorf("dir")
	}
	val, ok := p.Items[md]
	if !ok || (ok && val.ModTime != info.ModTime().UnixNano()) {
		p.RLock()
		defer p.RUnlock()
		fileread, _ := ioutil.ReadFile(md)
		lines := strings.Split(string(fileread), "\n")
		title := string(lines[0])
		body := strings.Join(lines[1:len(lines)], "\n")
		body = string(blackfriday.MarkdownCommon([]byte(body)))
		p.Items[md] = post{title, template.HTML(body), info.ModTime().UnixNano()}
	}
	post := p.Items[md]
	return post, 200, nil
}
