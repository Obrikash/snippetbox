package main

import (
	"crypto/tls"
	"database/sql"
	"flag"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/alexedwards/scs/mysqlstore"
	"github.com/alexedwards/scs/v2"
	"github.com/go-playground/form/v4"
	_ "github.com/go-sql-driver/mysql"
	"github.com/obrikash/snippetbox/internal/models"
)

type application struct {
	errorLog       *log.Logger
	infoLog        *log.Logger
	snippets       models.SnippetModelInterface
	userModel      models.UserModelInterface
	templateCache  map[string]*template.Template
	formDecoder    *form.Decoder
	sessionManager *scs.SessionManager
	debugMode      bool
}

func main() {
	addr := flag.String("addr", "localhost:4000", "HTTP network address")
	dsn := flag.String("dsn", "web:pass/snippetbox?parseTime=true", "MySQL data source name")
	debug := flag.Bool("debug", false, "Debug mode (True or False)")

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

	formDecoder := form.NewDecoder()

	sessionManager := scs.New()
	sessionManager.Store = mysqlstore.New(db)
	sessionManager.Lifetime = 12 * time.Hour
	sessionManager.Cookie.Secure = true

	app := &application{
		infoLog:        infoLog,
		errorLog:       errorLog,
		snippets:       &models.SnippetModel{DB: db},
		userModel:      &models.UserModel{DB: db},
		templateCache:  templateCache,
		formDecoder:    formDecoder,
		sessionManager: sessionManager,
		debugMode:      *debug,
	}

	tlsConfig := &tls.Config{
		CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
	}

	srv := &http.Server{
		Addr:         *addr,
		ErrorLog:     errorLog,
		Handler:      app.routes(),
		TLSConfig:    tlsConfig,
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	infoLog.Printf("Starting server on %s", *addr)
	err = srv.ListenAndServeTLS("./tls/cert.pem", "./tls/key.pem")
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
