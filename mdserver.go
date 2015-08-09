package main

import (
	"fmt"
	"github.com/bmizerany/pat"
	"github.com/russross/blackfriday"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
)

type Post struct {
	Title string
	Body  template.HTML
}

type CacheEntry struct {
	ModTime int64
	title   string
	Body    template.HTML
}

var (
	// компилируем шаблоны, если не удалось, то выходим
	post_template  = template.Must(template.ParseFiles(path.Join("templates", "layout.html"), path.Join("templates", "post.html")))
	error_template = template.Must(template.ParseFiles(path.Join("templates", "layout.html"), path.Join("templates", "error.html")))

	cache map[string]CacheEntry = make(map[string]CacheEntry)
)

func main() {
	// для отдачи сервером статичных файлов из папки public/static
	fs := noDirListing(http.FileServer(http.Dir("./public/static")))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	uploads := noDirListing(http.FileServer(http.Dir("./public/uploads")))
	http.Handle("/uploads/", http.StripPrefix("/uploads/", uploads))

	mux := pat.New()
	mux.Get("/:page", http.HandlerFunc(postHandler))
	mux.Get("/:page/", http.HandlerFunc(postHandler))
	mux.Get("/", http.HandlerFunc(postHandler))

	http.Handle("/", mux)
	log.Println("Listening...")
	http.ListenAndServe(":3000", nil)
}

func postHandler(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	// Извлекаем параметр
	// Например, в http://127.0.0.1:3000/p1 page = "p1"
	// в http://127.0.0.1:3000/ page = ""
	page := params.Get(":page")
	// Путь к файлу (без расширения)
	// Например, posts/p1
	p := path.Join("posts", page)

	var post_md string
	if page != "" {
		// если page не пусто, то считаем, что запрашивается файл
		// получим posts/p1.md
		post_md = p + ".md"
	} else {
		// если page пусто, то выдаем главную
		post_md = p + "/index.md"
	}

	post, status, err := load_post(post_md)
	if err != nil {
		errorHandler(w, r, status)
		return
	}
	if err := post_template.ExecuteTemplate(w, "layout", post); err != nil {
		log.Println(err.Error())
		errorHandler(w, r, 500)
	}
}

// Загружает markdown-файл и конвертирует его в HTML
// Возвращает объект типа Post
// Если путь не существует или является каталогом, то возвращаем ошибку
func load_post(md string) (Post, int, error) {
	info, err := os.Stat(md)
	if err != nil {
		if os.IsNotExist(err) {
			// файл не существует
			return Post{}, http.StatusNotFound, err
		}
	}
	if info.IsDir() {
		// не файл, а папка
		return Post{}, http.StatusNotFound, fmt.Errorf("dir")
	}
	val, ok := cache[md]
	if !ok || (ok && val.ModTime != info.ModTime().UnixNano()) {
		fileread, _ := ioutil.ReadFile(md)
		lines := strings.Split(string(fileread), "\n")
		title := string(lines[0])
		body := strings.Join(lines[1:len(lines)], "\n")
		body = string(blackfriday.MarkdownCommon([]byte(body)))
		cache[md] = CacheEntry{info.ModTime().UnixNano(), title, template.HTML(body)}
	}
	post := Post{cache[md].title, cache[md].Body}
	return post, 200, nil
}

func errorHandler(w http.ResponseWriter, r *http.Request, status int) {
	w.WriteHeader(status)
	if err := error_template.ExecuteTemplate(w, "layout", map[string]interface{}{"Error": http.StatusText(status), "Status": status}); err != nil {
		log.Println(err.Error())
		http.Error(w, http.StatusText(500), 500)
		return
	}
}

// обертка для http.FileServer, чтобы она не выдавала список файлов
// например, если открыть http://127.0.0.1:3000/static/,
// то будет видно список файлов внутри каталога.
// noDirListing - вернет 404 ошибку в этом случае.
func noDirListing(h http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") || r.URL.Path == "" {
			http.NotFound(w, r)
			return
		}
		h.ServeHTTP(w, r)
	})
}
