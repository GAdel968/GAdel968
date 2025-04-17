package packages

import (
	"database/sql"
	"net/http"
	"net/url"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/sessions"
)

const PORT = "8000"
const URL = "localhost"
const DB_ADRESS = "localhost:3306"
const DB_NAME = "forum2"
const DB_USER = "forum2"
const DB_PASSWORD = "123"

var (
	key   = []byte("super-secret-key")
	store = sessions.NewCookieStore(key)
)

func PostTopic(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	db, _ := sql.Open("mysql", DB_USER+":"+DB_PASSWORD+"@tcp("+DB_ADRESS+")/"+DB_NAME)
	defer db.Close()

	username, ok := session.Values["username"].(string)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	var userID string
	// Obtenir l'id_user avec l'username du cookie.
	_ = db.QueryRow("SELECT id_user FROM users WHERE username = ?", username).Scan(&userID)
	content := r.FormValue("content")
	title := r.FormValue("title")
	datePost := time.Now().Format("2006-01-02 15:04:05")

	// Ajouter un topic dans la table topics.
	insert1, _ := db.Exec("INSERT INTO topics (title) VALUES (?)", title)
	topicID, _ := insert1.LastInsertId()
	// Obtenir l'id_topic du dernier topic.
	_ = db.QueryRow("SELECT id_topic FROM topics ORDER BY id_topic DESC LIMIT 1").Scan(&topicID)
	// Ajouter un message dans la table message avec le dernier id_topic qui vient d'être créé.
	_, _ = db.Exec("INSERT INTO messages (content, id_user, date_created, id_topic) VALUES (?, ?, ?, ?)",
		content, userID, datePost, topicID)

	http.Redirect(w, r, "/topic?t="+strconv.Itoa(int(topicID)), http.StatusSeeOther)
}

func PostMessage(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	db, _ := sql.Open("mysql", DB_USER+":"+DB_PASSWORD+"@tcp("+DB_ADRESS+")/"+DB_NAME)
	defer db.Close()

	username, ok := session.Values["username"].(string)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	var userID string
	// Obtenir l'id_user avec l'username du cookie.
	_ = db.QueryRow("SELECT id_user FROM users WHERE username = ?", username).Scan(&userID)
	content := r.FormValue("content")
	datePost := time.Now().Format("2006-01-02 15:04:05")

	parsedURL, _ := url.Parse(r.URL.String())
	queryValues, _ := url.ParseQuery(parsedURL.RawQuery)
	topicID := queryValues.Get("t")
	// Ajouter un message avec le même id_topic que celui de l'URL.
	_, _ = db.Exec("INSERT INTO messages (content, id_user, date_created, id_topic) VALUES (?, ?, ?, ?)",
		content, userID, datePost, topicID)

	http.Redirect(w, r, "/topic?t="+topicID, http.StatusSeeOther)
}
