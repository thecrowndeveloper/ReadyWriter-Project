package main

import (
	"database/sql"
	"fmt"
	"github.com/gorilla/sessions"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
	"html/template"
	"net/http"
	"strings"
)

type Chapter struct {
	ID      int
	Title   string
	Content string
}

type Earning struct {
	ID          int
	ChapterID   int
	Amount      float64
	Currency    string
	EarningType string
	Status      string
	CreatedAt   string
}

type Reading struct {
	ChapterID    int
	ChapterTitle string
	BookID       int
	BookTitle    string
}

type SavedBook struct {
	BookID    int
	BookTitle string
}

type Book struct {
	ID           int
	Title        string
	ChapterCount int
	UserID       int
}

type ChapterPage struct {
	Chapter     Chapter
	PreviousID  int
	NextID      int
	HasPrevious bool
	HasNext     bool
	IsOwner     bool
}

var db *sql.DB

func home(w http.ResponseWriter, r *http.Request) {

	userID := getCurrentUser(r)

	var userName string
	var userRole string

	if userID != 0 {

		err := db.QueryRow(
			"SELECT name, role FROM users WHERE id = $1",
			userID,
		).Scan(&userName, &userRole)

		if err != nil {
			userName = ""
			userRole = ""
		}
	}

	data := struct {
		UserName string
		UserRole string
	}{
		UserName: userName,
		UserRole: userRole,
	}

	tmpl, err := template.ParseFiles("templates/index.html")

	if err != nil {
		http.Error(w, "Unable to load page", http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, data)
}

func books(w http.ResponseWriter, r *http.Request) {

	rows, err := db.Query(`
    SELECT
        books.id,
        books.title,
        COUNT(chapters.id) AS chapter_count,
books.user_id
    FROM books
    LEFT JOIN chapters
        ON books.id = chapters.book_id
        AND chapters.status = 'published'
    GROUP BY books.id, books.title, books.user_id
    ORDER BY books.id DESC
`)

	if err != nil {
		fmt.Println("Scan error:", err)
		http.Error(w, "Unable to read book", http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	var books []Book

	for rows.Next() {

		var book Book

		err := rows.Scan(
			&book.ID,
			&book.Title,
			&book.ChapterCount,
			&book.UserID,
		)
		if err != nil {
			http.Error(w, "Unable to read book", http.StatusInternalServerError)
			return
		}

		books = append(books, book)
	}

	tmpl, err := template.ParseFiles("templates/books.html")

	if err != nil {
		http.Error(w, "Unable to load books page", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, books)

	if err != nil {
		http.Error(w, "Unable to display books page", http.StatusInternalServerError)
		return
	}
}

func book(w http.ResponseWriter, r *http.Request) {

	id := r.URL.Query().Get("id")

	var currentBook Book

	err := db.QueryRow(
		"SELECT id, title FROM books WHERE id = $1",
		id,
	).Scan(&currentBook.ID, &currentBook.Title)

	if err != nil {
		http.Error(w, "Book not found", http.StatusNotFound)
		return
	}

	rows, err := db.Query(
		"SELECT id, title, content FROM chapters WHERE book_id = $1 AND status = 'published' ORDER BY id ASC",
		id,
	)

	if err != nil {
		http.Error(w, "Unable to load chapters", http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	var chapters []Chapter

	for rows.Next() {

		var chapter Chapter

		err := rows.Scan(
			&chapter.ID,
			&chapter.Title,
			&chapter.Content,
		)

		if err != nil {
			http.Error(w, "Unable to read chapter", http.StatusInternalServerError)
			return
		}

		chapters = append(chapters, chapter)
	}

	userID := getCurrentUser(r)

	isSaved := false

	if userID != 0 {

		err = db.QueryRow(
			"SELECT EXISTS (SELECT 1 FROM saved_books WHERE user_id = $1 AND book_id = $2)",
			userID,
			id,
		).Scan(&isSaved)

		if err != nil {
			http.Error(w, "Unable to check saved book", http.StatusInternalServerError)
			return
		}
	}

	data := struct {
		Book     Book
		Chapters []Chapter
		IsSaved  bool
	}{
		Book:     currentBook,
		Chapters: chapters,
		IsSaved:  isSaved,
	}

	tmpl, err := template.ParseFiles("templates/book.html")

	if err != nil {
		http.Error(w, "Unable to load book page", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, data)

	if err != nil {
		http.Error(w, "Unable to display book page", http.StatusInternalServerError)
		return
	}
}

func jobs(w http.ResponseWriter, r *http.Request) {

	userID := getCurrentUser(r)

	var userName string

	if userID != 0 {
		err := db.QueryRow(
			"SELECT name FROM users WHERE id = $1",
			userID,
		).Scan(&userName)

		if err != nil {
			http.Error(w, "Unable to load user", http.StatusInternalServerError)
			return
		}
	}

	data := struct {
		UserName string
	}{
		UserName: userName,
	}

	tmpl, err := template.ParseFiles("templates/jobs.html")

	if err != nil {
		http.Error(w, "Unable to load jobs page", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, data)

	if err != nil {
		http.Error(w, "Unable to display jobs page", http.StatusInternalServerError)
		return
	}
}

func community(w http.ResponseWriter, r *http.Request) {

	userID := getCurrentUser(r)

	var userName string

	if userID != 0 {
		err := db.QueryRow(
			"SELECT name FROM users WHERE id = $1",
			userID,
		).Scan(&userName)

		if err != nil {
			http.Error(w, "Unable to load user", http.StatusInternalServerError)
			return
		}
	}

	data := struct {
		UserName string
	}{
		UserName: userName,
	}

	tmpl, err := template.ParseFiles("templates/community.html")

	if err != nil {
		http.Error(w, "Unable to load community page", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, data)

	if err != nil {
		http.Error(w, "Unable to display community page", http.StatusInternalServerError)
		return
	}
}

func drafts(w http.ResponseWriter, r *http.Request) {

	userID := getCurrentUser(r)

	if userID == 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	rows, err := db.Query(
		`
SELECT
    chapters.id,
    chapters.title,
    chapters.content
FROM chapters
JOIN books
    ON chapters.book_id = books.id
WHERE chapters.status = 'draft'
AND books.user_id = $1
ORDER BY chapters.id DESC
`,
		userID,
	)

	if err != nil {
		http.Error(w, "Unable to load drafts", http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	var drafts []Chapter

	for rows.Next() {

		var draft Chapter

		err := rows.Scan(
			&draft.ID,
			&draft.Title,
			&draft.Content,
		)

		if err != nil {
			http.Error(w, "Unable to read draft", http.StatusInternalServerError)
			return
		}

		drafts = append(drafts, draft)
	}

	tmpl, err := template.ParseFiles("templates/drafts.html")

	if err != nil {
		http.Error(w, "Unable to load drafts page", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, drafts)

	if err != nil {
		http.Error(w, "Unable to display drafts", http.StatusInternalServerError)
		return
	}
}

func chapter(w http.ResponseWriter, r *http.Request) {

	userID := getCurrentUser(r)

	id := r.URL.Query().Get("id")

	var ch Chapter

	var bookID int

	var ownerID int

	err := db.QueryRow(
		`
	SELECT
    chapters.id,
    chapters.title,
    chapters.content,
    books.id,
    books.user_id
	FROM chapters
	JOIN books
		ON chapters.book_id = books.id
	WHERE chapters.id = $1
AND chapters.status = 'published'
	`,
		id,
	).Scan(
		&ch.ID,
		&ch.Title,
		&ch.Content,
		&bookID,
		&ownerID,
	)
	if err != nil {
		http.Error(w, "Chapter not found", http.StatusNotFound)
		return
	}

	isOwner := userID == ownerID

	if userID != 0 && !isOwner {

		_, err = db.Exec(
			`
    INSERT INTO reading_history (user_id, chapter_id)
    VALUES ($1, $2)
    ON CONFLICT (user_id, chapter_id)
    DO UPDATE SET read_at = CURRENT_TIMESTAMP
    `,
			userID,
			ch.ID,
		)
		if err != nil {
			http.Error(w, "Unable to save reading history", http.StatusInternalServerError)
			return
		}
	}

	var previousID int

	err = db.QueryRow(
		"SELECT id FROM chapters WHERE book_id = $1 AND id < $2 AND status = 'published' ORDER BY id DESC LIMIT 1",
		bookID,
		id,
	).Scan(&previousID)
	hasPrevious := err == nil

	var nextID int

	err = db.QueryRow(
		"SELECT id FROM chapters WHERE book_id = $1 AND id > $2 AND status = 'published' ORDER BY id ASC LIMIT 1",
		bookID,
		id,
	).Scan(&nextID)

	hasNext := err == nil

	pageData := ChapterPage{
		Chapter:     ch,
		PreviousID:  previousID,
		NextID:      nextID,
		HasPrevious: hasPrevious,
		HasNext:     hasNext,
		IsOwner:     isOwner,
	}

	tmpl, err := template.ParseFiles("templates/chapter.html")

	if err != nil {
		http.Error(w, "Unable to load chapter page", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, pageData)

	if err != nil {
		http.Error(w, "Unable to display chapter", http.StatusInternalServerError)
		return
	}
}

func edit(w http.ResponseWriter, r *http.Request) {

	userID := getCurrentUser(r)

	if userID == 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	id := r.URL.Query().Get("id")

	var ch Chapter

	err := db.QueryRow(
		`
    SELECT
        chapters.id,
        chapters.title,
        chapters.content
    FROM chapters
    JOIN books
        ON chapters.book_id = books.id
    WHERE chapters.id = $1
    AND books.user_id = $2
    `,
		id,
		userID,
	).Scan(&ch.ID, &ch.Title, &ch.Content)

	if err != nil {
		http.Error(w, "Chapter not found", http.StatusNotFound)
		return
	}

	tmpl, err := template.ParseFiles("templates/edit.html")

	if err != nil {
		http.Error(w, "Unable to load edit page", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, ch)

	if err != nil {
		http.Error(w, "Unable to display edit page", http.StatusInternalServerError)
		return
	}
}

func update(w http.ResponseWriter, r *http.Request) {

	userID := getCurrentUser(r)

	if userID == 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.FormValue("id")
	title := r.FormValue("title")
	content := r.FormValue("content")
	action := r.FormValue("action")

	var oldStatus string

	err := db.QueryRow(
		"SELECT status FROM chapters WHERE id = $1",
		id,
	).Scan(&oldStatus)

	if err != nil {
		http.Error(w, "Chapter not found", http.StatusNotFound)
		return
	}

	status := "published"

	if action == "draft" {
		status = "draft"
	}

	_, err = db.Exec(
		`
    UPDATE chapters
    SET title = $1, content = $2, status = $3
    WHERE id = $4
    AND EXISTS (
        SELECT 1
        FROM books
        WHERE books.id = chapters.book_id
        AND books.user_id = $5
    )
    `,
		title,
		content,
		status,
		id,
		userID,
	)

	if err != nil {
		http.Error(w, "Unable to update chapter", http.StatusInternalServerError)
		return
	}

	if status == "published" {

		wordCount := len(strings.Fields(content))

		activityType := "updated"

		if oldStatus == "draft" {
			activityType = "published"
		}

		_, err = db.Exec(
			`
        INSERT INTO writing_activity (
            user_id,
            chapter_id,
            word_count,
            activity_type
        )
        VALUES ($1, $2, $3, $4)
        `,
			userID,
			id,
			wordCount,
			activityType,
		)

		if err != nil {
			http.Error(w, "Unable to record writing activity", http.StatusInternalServerError)
			return
		}
	}

	if status == "draft" {
		http.Redirect(w, r, "/drafts", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/chapter?id="+id, http.StatusSeeOther)
}

func publishDraft(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := getCurrentUser(r)

	if userID == 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	draftID := r.FormValue("id")

	var chapterID int

	err := db.QueryRow(
		`
        SELECT chapters.id
        FROM chapters
        JOIN books
            ON chapters.book_id = books.id
        WHERE chapters.id = $1
        AND chapters.status = 'draft'
        AND books.user_id = $2
        `,
		draftID,
		userID,
	).Scan(&chapterID)

	if err != nil {
		http.Error(w, "Draft not found", http.StatusNotFound)
		return
	}

	_, err = db.Exec(
		`
        UPDATE chapters
        SET status = 'published'
        WHERE id = $1
        `,
		chapterID,
	)

	if err != nil {
		http.Error(w, "Unable to publish draft", http.StatusInternalServerError)
		return
	}

	_, err = db.Exec(
		`
        INSERT INTO earnings (
            user_id,
            chapter_id,
            amount,
            currency,
            earning_type,
            status
        )
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (chapter_id, earning_type) DO NOTHING
        `,
		userID,
		chapterID,
		1.00,
		"USD",
		"chapter_base",
		"pending",
	)

	if err != nil {
		http.Error(w, "Unable to record earning", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/drafts", http.StatusSeeOther)
}

func delete(w http.ResponseWriter, r *http.Request) {

	userID := getCurrentUser(r)

	if userID == 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	id := r.URL.Query().Get("id")

	_, err := db.Exec(
		`
    DELETE FROM chapters
    WHERE id = $1
    AND EXISTS (
        SELECT 1
        FROM books
        WHERE books.id = chapters.book_id
        AND books.user_id = $2
    )
    `,
		id,
		userID,
	)

	if err != nil {
		http.Error(w, "Unable to delete chapter", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/books", http.StatusSeeOther)
}

func write(w http.ResponseWriter, r *http.Request) {

	userID := getCurrentUser(r)

	if userID == 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	var role string

	err := db.QueryRow(
		"SELECT role FROM users WHERE id = $1",
		userID,
	).Scan(&role)

	if err != nil {
		http.Error(w, "Unable to load user", http.StatusInternalServerError)
		return
	}

	if role != "writer" {
		http.Error(w, "Only writers can access the writing studio", http.StatusForbidden)
		return
	}

	rows, err := db.Query(
		"SELECT id, title FROM books WHERE user_id = $1",
		userID,
	)

	if err != nil {
		http.Error(w, "Unable to load books", http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	var books []Book

	for rows.Next() {

		var book Book

		err := rows.Scan(&book.ID, &book.Title)

		if err != nil {
			http.Error(w, "Unable to read book", http.StatusInternalServerError)
			return
		}

		books = append(books, book)
	}

	tmpl, err := template.ParseFiles("templates/write.html")

	if err != nil {
		http.Error(w, "Unable to load writing page", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, books)

	if err != nil {
		http.Error(w, "Unable to display writing page", http.StatusInternalServerError)
		return
	}
}

func publish(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := getCurrentUser(r)

	if userID == 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	var role string

	err := db.QueryRow(
		"SELECT role FROM users WHERE id = $1",
		userID,
	).Scan(&role)

	if err != nil {
		http.Error(w, "Unable to load user", http.StatusInternalServerError)
		return
	}

	if role != "writer" {
		http.Error(w, "Only writers can publish chapters", http.StatusForbidden)
		return
	}

	title := r.FormValue("title")
	content := r.FormValue("content")
	bookID := r.FormValue("book_id")

	wordCount := len(strings.Fields(content))

	var ownerID int

	err = db.QueryRow(
		"SELECT user_id FROM books WHERE id = $1",
		bookID,
	).Scan(&ownerID)

	if err != nil {
		http.Error(w, "Book not found", http.StatusNotFound)
		return
	}

	if ownerID != userID {
		http.Error(w, "You do not have permission to use this book", http.StatusForbidden)
		return
	}
	action := r.FormValue("action")

	status := "published"

	if action == "draft" {
		status = "draft"
	}

	var chapterID int

	err = db.QueryRow(
		`
    INSERT INTO chapters (
        title,
        content,
        book_id,
        status
    )
    VALUES ($1, $2, $3, $4)
    RETURNING id
    `,
		title,
		content,
		bookID,
		status,
	).Scan(&chapterID)

	if err != nil {
		http.Error(w, "Unable to save chapter", http.StatusInternalServerError)
		return
	}

	if status == "published" {

		_, err = db.Exec(
			`
        INSERT INTO writing_activity (
            user_id,
            chapter_id,
            word_count,
            activity_type
        )
        VALUES ($1, $2, $3, $4)
        `,
			userID,
			chapterID,
			wordCount,
			"published",
		)

		if err != nil {
			http.Error(w, "Unable to record writing activity", http.StatusInternalServerError)
			return
		}
	}

	if err != nil {
		http.Error(w, "Unable to save chapter", http.StatusInternalServerError)
		return
	}

	message := "Your chapter has been published successfully."
	titleMessage := "Chapter published."

	if status == "draft" {
		message = "Your chapter has been saved safely as a draft."
		titleMessage = "Draft saved."
	}

	data := struct {
		Label       string
		Title       string
		Message     string
		PrimaryURL  string
		PrimaryText string
	}{
		Label:       "WRITING STUDIO",
		Title:       titleMessage,
		Message:     message,
		PrimaryURL:  "/dashboard",
		PrimaryText: "View Dashboard",
	}

	tmpl, err := template.ParseFiles("templates/message.html")

	if err != nil {
		http.Error(w, "Unable to load message page", http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, data)
}

func createBook(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := getCurrentUser(r)

	if userID == 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	var role string

	err := db.QueryRow(
		"SELECT role FROM users WHERE id = $1",
		userID,
	).Scan(&role)

	if err != nil {
		http.Error(w, "Unable to load user", http.StatusInternalServerError)
		return
	}

	if role != "writer" {
		http.Error(w, "Only writers can create books", http.StatusForbidden)
		return
	}

	title := r.FormValue("title")

	_, err = db.Exec(
		"INSERT INTO books (title, user_id) VALUES ($1, $2)",
		title,
		userID,
	)
	if err != nil {
		http.Error(w, "Unable to create book", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/write", http.StatusSeeOther)
}

func saveBook(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := getCurrentUser(r)

	if userID == 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	bookID := r.FormValue("book_id")

	_, err := db.Exec(
		`
        INSERT INTO saved_books (user_id, book_id)
        VALUES ($1, $2)
        ON CONFLICT (user_id, book_id) DO NOTHING
        `,
		userID,
		bookID,
	)

	if err != nil {
		http.Error(w, "Unable to save book", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/book?id="+bookID, http.StatusSeeOther)
}

func unsaveBook(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := getCurrentUser(r)

	if userID == 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	bookID := r.FormValue("book_id")

	_, err := db.Exec(
		`
		DELETE FROM saved_books
		WHERE user_id = $1
		AND book_id = $2
		`,
		userID,
		bookID,
	)

	if err != nil {
		http.Error(w, "Unable to unsave book", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/book?id="+bookID, http.StatusSeeOther)
}

func register(w http.ResponseWriter, r *http.Request) {

	if r.Method == http.MethodGet {

		tmpl, err := template.ParseFiles("templates/register.html")

		if err != nil {
			http.Error(w, "Unable to load registration page", http.StatusInternalServerError)
			return
		}

		tmpl.Execute(w, nil)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := r.FormValue("name")
	email := r.FormValue("email")
	password := r.FormValue("password")
	role := r.FormValue("role")

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)

	if err != nil {
		http.Error(w, "Unable to secure password", http.StatusInternalServerError)
		return
	}

	_, err = db.Exec(
		"INSERT INTO users (name, email, password, role) VALUES ($1, $2, $3, $4)",
		name,
		email,
		string(hashedPassword),
		role,
	)

	if err != nil {
		http.Error(w, "Unable to create account", http.StatusInternalServerError)
		return
	}

	data := struct {
		Label       string
		Title       string
		Message     string
		PrimaryURL  string
		PrimaryText string
	}{
		Label:       "WELCOME TO READYWRITER",
		Title:       "Account created.",
		Message:     "Your ReadyWriter account is ready. You can now log in and start exploring.",
		PrimaryURL:  "/login",
		PrimaryText: "Log In",
	}

	tmpl, err := template.ParseFiles("templates/message.html")

	if err != nil {
		http.Error(w, "Unable to load message page", http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, data)
}

func login(w http.ResponseWriter, r *http.Request) {

	if r.Method == http.MethodGet {

		tmpl, err := template.ParseFiles("templates/login.html")

		if err != nil {
			http.Error(w, "Unable to load login page", http.StatusInternalServerError)
			return
		}

		tmpl.Execute(w, nil)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	var userID int
	var storedPassword string

	err := db.QueryRow(
		"SELECT id, password FROM users WHERE email = $1",
		email,
	).Scan(&userID, &storedPassword)

	if err != nil {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(storedPassword),
		[]byte(password),
	)

	if err != nil {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	session, err := store.Get(r, "readywriter-session")

	if err != nil {
		http.Error(w, "Unable to start session", http.StatusInternalServerError)
		return
	}

	session.Values["user_id"] = userID

	err = session.Save(r, w)

	if err != nil {
		http.Error(w, "Unable to save session", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

var store = sessions.NewCookieStore(
	[]byte("readywriter-secret-key"),
)

func getCurrentUser(r *http.Request) int {

	session, err := store.Get(r, "readywriter-session")

	if err != nil {
		return 0
	}

	userID, ok := session.Values["user_id"].(int)

	if !ok {
		return 0
	}

	return userID
}

func logout(w http.ResponseWriter, r *http.Request) {

	session, err := store.Get(r, "readywriter-session")

	if err != nil {
		http.Error(w, "Unable to log out", http.StatusInternalServerError)
		return
	}

	session.Values = map[interface{}]interface{}{}

	err = session.Save(r, w)

	if err != nil {
		http.Error(w, "Unable to log out", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func dashboard(w http.ResponseWriter, r *http.Request) {

	userID := getCurrentUser(r)

	if userID == 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	var totalEarned float64

	var activeDays int

	err := db.QueryRow(
		`
    SELECT COALESCE(SUM(amount), 0)
    FROM earnings
    WHERE user_id = $1
    `,
		userID,
	).Scan(&totalEarned)

	if err != nil {
		http.Error(w, "Unable to load earnings", http.StatusInternalServerError)
		return
	}

	err = db.QueryRow(
		`
    SELECT COUNT(DISTINCT DATE(created_at))
    FROM writing_activity
    WHERE user_id = $1
    `,
		userID,
	).Scan(&activeDays)

	if err != nil {
		http.Error(w, "Unable to load writing activity", http.StatusInternalServerError)
		return
	}

	var userName string
	var userRole string

	err = db.QueryRow(
		"SELECT name, role FROM users WHERE id = $1",
		userID,
	).Scan(&userName, &userRole)

	if err != nil {
		http.Error(w, "Unable to load user", http.StatusInternalServerError)
		return
	}

	rows, err := db.Query(
		"SELECT id, title FROM books WHERE user_id = $1 ORDER BY id DESC",
		userID,
	)

	if err != nil {
		http.Error(w, "Unable to load books", http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	var books []Book

	for rows.Next() {

		var book Book

		err := rows.Scan(
			&book.ID,
			&book.Title,
		)

		if err != nil {
			http.Error(w, "Unable to load books", http.StatusInternalServerError)
			return
		}

		books = append(books, book)
	}

	readingRows, err := db.Query(
		`
        SELECT
            chapters.id,
            chapters.title,
            books.id,
            books.title
        FROM reading_history
        JOIN chapters
            ON reading_history.chapter_id = chapters.id
        JOIN books
            ON chapters.book_id = books.id
        WHERE reading_history.user_id = $1
        AND chapters.status = 'published'
        ORDER BY reading_history.read_at DESC
        LIMIT 5
        `,
		userID,
	)

	if err != nil {
		http.Error(w, "Unable to load reading history", http.StatusInternalServerError)
		return
	}

	defer readingRows.Close()

	var readings []Reading

	for readingRows.Next() {

		var reading Reading

		err := readingRows.Scan(
			&reading.ChapterID,
			&reading.ChapterTitle,
			&reading.BookID,
			&reading.BookTitle,
		)

		if err != nil {
			http.Error(w, "Unable to read history", http.StatusInternalServerError)
			return
		}

		readings = append(readings, reading)
	}

	savedRows, err := db.Query(
		`
        SELECT
            books.id,
            books.title
        FROM saved_books
        JOIN books
            ON saved_books.book_id = books.id
        WHERE saved_books.user_id = $1
        ORDER BY saved_books.saved_at DESC
        `,
		userID,
	)

	if err != nil {
		http.Error(w, "Unable to load saved books", http.StatusInternalServerError)
		return
	}

	defer savedRows.Close()

	var savedBooks []SavedBook

	for savedRows.Next() {

		var savedBook SavedBook

		err := savedRows.Scan(
			&savedBook.BookID,
			&savedBook.BookTitle,
		)

		if err != nil {
			http.Error(w, "Unable to read saved books", http.StatusInternalServerError)
			return
		}

		savedBooks = append(savedBooks, savedBook)
	}

	data := struct {
		UserName    string
		UserRole    string
		Books       []Book
		Readings    []Reading
		SavedBooks  []SavedBook
		TotalEarned float64
		ActiveDays  int
	}{
		UserName:    userName,
		UserRole:    userRole,
		Books:       books,
		Readings:    readings,
		SavedBooks:  savedBooks,
		TotalEarned: totalEarned,
		ActiveDays:  activeDays,
	}

	tmpl, err := template.ParseFiles("templates/dashboard.html")

	if err != nil {
		http.Error(w, "Unable to load dashboard", http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, data)
}

func becomeWriter(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := getCurrentUser(r)

	if userID == 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	_, err := db.Exec(
		"UPDATE users SET role = 'writer' WHERE id = $1",
		userID,
	)

	if err != nil {
		http.Error(w, "Unable to become a writer", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func earnings(w http.ResponseWriter, r *http.Request) {

	userID := getCurrentUser(r)

	if userID == 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	var totalEarned float64
	var pending float64
	var paid float64

	err := db.QueryRow(
		`
    SELECT
        COALESCE(SUM(amount), 0),
        COALESCE(SUM(CASE WHEN status = 'pending' THEN amount ELSE 0 END), 0),
        COALESCE(SUM(CASE WHEN status = 'paid' THEN amount ELSE 0 END), 0)
    FROM earnings
    WHERE user_id = $1
    `,
		userID,
	).Scan(
		&totalEarned,
		&pending,
		&paid,
	)

	if err != nil {
		http.Error(w, "Unable to calculate earnings", http.StatusInternalServerError)
		return
	}

	rows, err := db.Query(
		`
        SELECT
            id,
            chapter_id,
            amount,
            currency,
            earning_type,
            status,
            created_at
        FROM earnings
        WHERE user_id = $1
        ORDER BY created_at DESC
        `,
		userID,
	)

	if err != nil {
		http.Error(w, "Unable to load earnings", http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	var earnings []Earning

	for rows.Next() {

		var earning Earning

		err := rows.Scan(
			&earning.ID,
			&earning.ChapterID,
			&earning.Amount,
			&earning.Currency,
			&earning.EarningType,
			&earning.Status,
			&earning.CreatedAt,
		)

		if err != nil {
			http.Error(w, "Unable to read earnings", http.StatusInternalServerError)
			return
		}

		earnings = append(earnings, earning)
	}

	tmpl, err := template.ParseFiles("templates/earnings.html")

	if err != nil {
		http.Error(w, "Unable to load earnings page", http.StatusInternalServerError)
		return
	}

	data := struct {
		Earnings    []Earning
		TotalEarned float64
		Pending     float64
		Paid        float64
	}{
		Earnings:    earnings,
		TotalEarned: totalEarned,
		Pending:     pending,
		Paid:        paid,
	}

	err = tmpl.Execute(w, data)

	if err != nil {
		http.Error(w, "Unable to display earnings", http.StatusInternalServerError)
		return
	}

	if err != nil {
		http.Error(w, "Unable to display earnings", http.StatusInternalServerError)
		return
	}
}

func main() {

	var err error

	db, err = sql.Open(
		"postgres",
		"dbname=readywriter user=postgres password=readywriter123 sslmode=disable",
	)

	if err != nil {
		panic(err)
	}

	err = db.Ping()

	if err != nil {
		panic(err)
	}

	http.HandleFunc("/", home)
	http.HandleFunc("/books", books)
	http.HandleFunc("/book", book)
	http.HandleFunc("/jobs", jobs)
	http.HandleFunc("/community", community)
	http.HandleFunc("/drafts", drafts)
	http.HandleFunc("/write", write)
	http.HandleFunc("/create-book", createBook)
	http.HandleFunc("/save-book", saveBook)
	http.HandleFunc("/unsave-book", unsaveBook)
	http.HandleFunc("/publish", publish)
	http.HandleFunc("/publish-draft", publishDraft)
	http.HandleFunc("/chapter", chapter)
	http.HandleFunc("/edit", edit)
	http.HandleFunc("/update", update)
	http.HandleFunc("/delete", delete)
	http.HandleFunc("/register", register)
	http.HandleFunc("/login", login)
	http.HandleFunc("/logout", logout)
	http.HandleFunc("/dashboard", dashboard)
	http.HandleFunc("/earnings", earnings)
	http.HandleFunc("/become-writer", becomeWriter)
	fs := http.FileServer(http.Dir("./static"))

	http.Handle("/static/", http.StripPrefix("/static/", fs))

	println("ReadyWriter is running at http://localhost:8080")

	http.ListenAndServe(":8080", nil)
}
