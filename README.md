file-glance
============

基于go的简单文件服务器, 并提供相关文件内容搜索功能


### 参数说明
```
$ go run main.go -h

Usage:
    a file server which support search
Flags:
  -p string
        static file server port (default "8888")
  -timeout duration
        the file search timeout (default 1s)
  -type string
        search between the type of files; eg -type=txt,md,py
```