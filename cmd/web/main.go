package main

import (
	"database/sql"
	"flag"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/obrikash/snippetbox/internal/models"
)

type application struct {
	errorLog *log.Logger
	infoLog  *log.Logger
    snippets *models.SnippetModel
    templateCache map[string]*template.Template
}

func main() {
	addr := flag.String("addr", ":4000", "HTTP network address")
    dsn := flag.String("dsn", "web:pass/snippetbox?parseTime=true", "MySQL data source name")

	flag.Parse()

	infoFile, err := os.OpenFile("/tmp/info.log", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		log.Fatal(err)
	}

	defer infoFile.Close()

	errorFile, err := os.OpenFile("/tmp/error.log", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		log.Fatal(err)
	}

	defer errorFile.Close()
	infoWriter := io.MultiWriter(infoFile, os.Stdout)
	errorWriter := io.MultiWriter(errorFile, os.Stderr)

	infoLog := log.New(infoWriter, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(errorWriter, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

    db, err := openDB(*dsn)
    if err != nil {
        errorLog.Fatal(err)
    }
    defer db.Close()

    templateCache, err := newTemplateCache()
    if err != nil {
        errorLog.Fatal(err)
    }

     
	app := &application{
		infoLog:  infoLog,
		errorLog: errorLog,
        snippets: &models.SnippetModel{DB: db},
        templateCache: templateCache,
	}

	srv := &http.Server{
		Addr:     *addr,
		ErrorLog: errorLog,
		Handler:  app.routes(),
	}

	infoLog.Printf("Starting server on %s", *addr)
	err = srv.ListenAndServe()
	errorLog.Fatal(err)
}

func openDB(dsn string) (*sql.DB, error) {
    db, err := sql.Open("mysql", dsn)
    if err != nil {
        return nil, err
    }

    if err = db.Ping(); err != nil {
        return nil, err
    }

    return db, nil
}

