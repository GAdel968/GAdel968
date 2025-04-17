package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"sort"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/sessions"

	p "src/packages"
)

var (
	key   = []byte("super-secret-key")
	store = sessions.NewCookieStore(key)
)

type PassHash struct {
	p.Argon2ID
}

var passHash PassHash

type Post struct {
	Title    string
	Content  string
	DatePost string
	Username string
	PostID   int
	NbRep    int
}

type User struct {
	Username     string
	Email        string
	PasswordHash string
	RoleId       int
}

func homePage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		p.PostTopic(w, r)
		return
	}
	session, _ := store.Get(r, "session")
	username, ok := session.Values["username"].(string)
	db, _ := sql.Open("mysql", p.DB_USER+":"+p.DB_PASSWORD+"@tcp("+p.DB_ADRESS+")/"+p.DB_NAME)
	defer db.Close()
	// Obtenir le titre, la date de création, l'auteur, l'id du topic
	// de chaque message qui est un topic.
	rows, _ := db.Query(`SELECT t.title, MAX(m.date_created), MAX(u.username), t.id_topic, COUNT(m.id_topic)
	FROM topics t
	JOIN messages m ON t.id_topic = m.id_topic
	JOIN users u ON m.id_user = u.id_user
	GROUP BY t.id_topic, t.title`)
	defer rows.Close()
	var posts []Post
	for rows.Next() {
		var post Post
		err := rows.Scan(&post.Title, &post.DatePost, &post.Username, &post.PostID, &post.NbRep)
		if err != nil {
			fmt.Print(err)
		}
		post.NbRep = post.NbRep - 1
		posts = append(posts, post)
	}

	parsedURL, _ := url.Parse(r.URL.String())
	queryValues, _ := url.ParseQuery(parsedURL.RawQuery)
	triMode := queryValues.Get("s")

	if triMode == "2" { // Les topics récents d'abord.
		sort.Slice(posts, func(i, j int) bool {
			timeI, _ := time.Parse("2006-01-02 15:04:05", posts[i].DatePost)
			timeJ, _ := time.Parse("2006-01-02 15:04:05", posts[j].DatePost)
			return timeI.After(timeJ)
		})
	} else if triMode == "3" { // Les topics anciens d'abord.
		sort.Slice(posts, func(i, j int) bool {
			timeI, _ := time.Parse("2006-01-02 15:04:05", posts[i].DatePost)
			timeJ, _ := time.Parse("2006-01-02 15:04:05", posts[j].DatePost)
			return timeI.Before(timeJ)
		})
	} else { // Les topics avec le plus grand nombre de réponses d'abord.
		sort.Slice(posts, func(i, j int) bool {
			return posts[i].NbRep > posts[j].NbRep
		})
	}

	data := struct {
		LoggedIn bool
		Username string
		Posts    []Post
	}{
		LoggedIn: ok,
		Username: username,
		Posts:    posts,
	}
	template.Must(template.ParseFiles("templates/index.html")).Execute(w, data)
}

func loginPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		template.Must(template.ParseFiles("templates/login.html")).Execute(w, r)
		return
	}

	username := r.FormValue("username")
	plainPassword := r.FormValue("password")
	db, _ := sql.Open("mysql", p.DB_USER+":"+p.DB_PASSWORD+"@tcp("+p.DB_ADRESS+")/"+p.DB_NAME)
	defer db.Close()
	var passwordHash string
	// Obtenir le hash argon2 du mot de passe de l'utilisateur.
	_ = db.QueryRow("SELECT password_hash FROM users WHERE username = ?", username).Scan(&passwordHash)

	passHash.NewArgon2ID()
	if passwordHash != "" {
		boolPassCheck, _ := passHash.Verify(plainPassword, passwordHash)
		if boolPassCheck {
			session, _ := store.Get(r, "session")
			session.Values["username"] = username
			session.Save(r, w)

			http.Redirect(w, r, "/", http.StatusSeeOther)
		} else {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
		}
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func signupPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		template.Must(template.ParseFiles("templates/signup.html")).Execute(w, r)
		return
	}
	db, _ := sql.Open("mysql", p.DB_USER+":"+p.DB_PASSWORD+"@tcp("+p.DB_ADRESS+")/"+p.DB_NAME)

	username := r.FormValue("username")
	var countUsername int
	// Obtenir le nombre d'username similaire deja existant.
	_ = db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", username).Scan(&countUsername)
	if countUsername > 0 {
		fmt.Print("Erreur: Le nom d'utilisateur existe deja.")
		return
	}
	email := r.FormValue("email")
	plainPassword := r.FormValue("password")
	passHash.NewArgon2ID()
	passwordHash, _ := passHash.Hash(plainPassword)

	// Ajouter un utilisateur à la table users.
	_, _ = db.Exec("INSERT INTO users (username, mail, password_hash, id_role) VALUES (?, ?, ?, ?)",
		username, email, passwordHash, 3)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func logoutPage(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	delete(session.Values, "username")
	session.Save(r, w)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func topicPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		p.PostMessage(w, r)
		return
	}
	session, _ := store.Get(r, "session")
	username, ok := session.Values["username"].(string)
	db, _ := sql.Open("mysql", p.DB_USER+":"+p.DB_PASSWORD+"@tcp("+p.DB_ADRESS+")/"+p.DB_NAME)
	defer db.Close()

	parsedURL, _ := url.Parse(r.URL.String())
	queryValues, _ := url.ParseQuery(parsedURL.RawQuery)
	topicID := queryValues.Get("t")

	// Obtenir le contenu, la date de création ainsi que l'username de l'auteur
	// de chaque message qui a le même id_topic que l'id_topic de l'URL.
	rows, _ := db.Query(`SELECT messages.content, messages.date_created, users.username
		FROM messages
		JOIN users ON messages.id_user = users.id_user
		WHERE messages.id_topic = ?`, topicID)
	defer rows.Close()
	var posts []Post
	for rows.Next() {
		var post Post
		err := rows.Scan(&post.Content, &post.DatePost, &post.Username)
		if err != nil {
			fmt.Print(err)
		}
		posts = append(posts, post)
	}
	var topicTitle string
	_ = db.QueryRow("SELECT title FROM topics WHERE id_topic = ?", topicID).Scan(&topicTitle)
	// Trier selon la date du message.
	sort.Slice(posts, func(i, j int) bool {
		timeI, _ := time.Parse("2006-01-02 15:04:05", posts[i].DatePost)
		timeJ, _ := time.Parse("2006-01-02 15:04:05", posts[j].DatePost)
		return timeI.Before(timeJ)
	})
	data := struct {
		LoggedIn bool
		Username string
		Posts    []Post
		Title    string
	}{
		LoggedIn: ok,
		Username: username,
		Posts:    posts,
		Title:    topicTitle,
	}

	template.Must(template.ParseFiles("templates/topic.html")).Execute(w, data)
}

func userPage(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	username, ok := session.Values["username"].(string)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	db, _ := sql.Open("mysql", p.DB_USER+":"+p.DB_PASSWORD+"@tcp("+p.DB_ADRESS+")/"+p.DB_NAME)
	defer db.Close()
	if r.Method == http.MethodPost {
		plainPassword := r.FormValue("oldPass")
		var passwordHash string
		// Obtenir le hash argon2 du mot de passe de l'utilisateur.
		_ = db.QueryRow("SELECT password_hash FROM users WHERE username = ?", username).Scan(&passwordHash)
		passHash.NewArgon2ID()
		if passwordHash != "" {
			boolPassCheck, _ := passHash.Verify(plainPassword, passwordHash)
			if boolPassCheck {
				// Modifier le hash du mot de passe de l'utilisateur dans la bdd.
				plainPassword = r.FormValue("newPass")
				passHash.NewArgon2ID()
				passwordHash, _ = passHash.Hash(plainPassword)
				db.Exec(`UPDATE users SET password_hash = ?
				WHERE username = ?`, passwordHash, username)
			} else {
				http.Redirect(w, r, "/user", http.StatusSeeOther)
			}
		}
	}

	// Obtenir les topics créés par l'utilisateur connecté.
	rows, _ := db.Query(`SELECT t.title, MAX(m.date_created), t.id_topic, COUNT(m.id_topic)
	FROM topics t
	JOIN messages m ON t.id_topic = m.id_topic
	JOIN users u ON m.id_user = u.id_user
	WHERE u.username = ?
	GROUP BY t.id_topic, t.title`, username)
	defer rows.Close()
	var posts []Post
	for rows.Next() {
		var post Post
		err := rows.Scan(&post.Title, &post.DatePost, &post.PostID, &post.NbRep)
		if err != nil {
			fmt.Print(err)
		}
		post.NbRep = post.NbRep - 1
		posts = append(posts, post)
	}

	data := struct {
		LoggedIn bool
		Username string
		Posts    []Post
	}{
		LoggedIn: ok,
		Username: username,
		Posts:    posts,
	}
	template.Must(template.ParseFiles("templates/user.html")).Execute(w, data)
}

func main() {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/", homePage)
	http.HandleFunc("/login", loginPage)
	http.HandleFunc("/signup", signupPage)
	http.HandleFunc("/logout", logoutPage)
	http.HandleFunc("/topic", topicPage)
	http.HandleFunc("/user", userPage)

	fmt.Println("(http://" + p.URL + ":" + p.PORT + ") - Le serveur est démarré.")
	err := http.ListenAndServe(":"+p.PORT, nil)
	if err != nil {
		log.Fatal(err)
	}
}
