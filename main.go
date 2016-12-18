package main

import (
	"context"
	"flag"
	"fmt"
	"golang.org/x/sync/errgroup"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var port = flag.String("p", "8888", "static file server port")
var timeout = flag.Duration("timeout", 1000*time.Millisecond, "the file search timeout")
var filetype = flag.String("type", "", "search between the type of files; eg -type=txt,md,py")

func main() {

	flag.Usage = func() {
		fmt.Println("Usage:")
		fmt.Printf("    a file server which support search \n")
		fmt.Println("Flags:")
		flag.PrintDefaults()
	}

	flag.Parse()
	server_port := *port
	println("start static file server :" + server_port)
	mux := http.NewServeMux()
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	//开启静态文件服务
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(wd))))

	//搜索服务
	mux.HandleFunc("/search/", func(writer http.ResponseWriter, req *http.Request) {
		rf := req.RequestURI
		query := strings.Replace(rf, "/search/", "", 1)
		querys := strings.Split(query, "/")

		filenames := []string{}
		for _, q := range querys {
			strings.TrimSpace(q)
			if q == "" || strings.TrimSpace(q) == "" {
				continue
			}
			filenames = append(filenames, q)
		}

		//设置带超时的上下文
		ctx, _ := context.WithTimeout(context.Background(), *timeout)
		results, err := search(ctx, wd, filenames, *filetype)
		if err != nil {
			writer.Write([]byte("search error"))
			return
		}
		res := strings.Join(results, "\n")
		writer.Write([]byte(res))

	})
	log.Fatal(http.ListenAndServe(":"+server_port, mux))
}

//文件名搜索
func search(ctx context.Context, root string, filenames []string, filetype string) ([]string, error) {
	g, ctx := errgroup.WithContext(ctx)

	log.Printf("search root : %s \n", root)
	log.Printf("search file : %v\n", filenames)

	paths := make(chan string, 100)

	g.Go(func() error {
		defer close(paths)
		//遍历
		return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.Mode().IsRegular() {
				return nil
			}

			//如果当前文件名不匹配

			if info.IsDir() {
				return nil
			}

			if strings.HasSuffix(info.Name(), ".tar.gz") {
				return nil
			}

			//匹配文件后缀
			if filetype != "" {
				isfind := false
				fts := strings.Split(filetype, ",")
				for _, f := range fts {
					if !(strings.HasSuffix(info.Name(), "."+f)) {
						continue
					}
					isfind = true
				}

				if !isfind {
					return nil
				}
			}

			select {
			case paths <- path:
			case <-ctx.Done():
				return ctx.Err()
			}
			return nil
		})
	})

	c := make(chan string, 100)

	//文件内容搜索
	for path := range paths {
		p := path
		g.Go(func() error {
			data, err := ioutil.ReadFile(p)
			if err != nil {
				return err
			}
			if !checkContentExist(data, filenames) {
				return nil
			}

			output := strings.Replace(p, root, "", 1)

			select {
			case c <- output:
			case <-ctx.Done():
				return ctx.Err()
			}
			return nil
		})
	}

	go func() {
		g.Wait()
		close(c)
	}()

	var m []string
	for r := range c {
		m = append(m, r)
	}
	return m, g.Wait()
}

//检查文件是否存在
func checkFileExist(searchs []string, fileName string) bool {
	for _, s := range searchs {
		if strings.Contains(fileName, s) {
			return true
		}
	}
	return false
}

func checkContentExist(data []byte, searchs []string) bool {
	for _, s := range searchs {
		if strings.Contains(string(data), s) {
			return true
		}
	}
	return false
}
