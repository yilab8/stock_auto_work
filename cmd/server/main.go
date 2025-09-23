package main

import (
	"flag"
	"html/template"
	"log"
	"net/http"

	"github.com/yilab8/stock_auto_work/internal/revenue"
	"github.com/yilab8/stock_auto_work/internal/server"
)

func main() {
	addr := flag.String("addr", ":8080", "http 監聽位址")
	flag.Parse()

	tmpl, err := template.ParseFiles("web/template/index.html")
	if err != nil {
		log.Fatalf("載入樣板失敗: %v", err)
	}

	app := server.NewApp(&revenue.Service{}, tmpl)

	log.Printf("服務啟動於 %s", *addr)
	if err := http.ListenAndServe(*addr, app); err != nil {
		log.Fatalf("服務停止: %v", err)
	}
}
