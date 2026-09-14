package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"

	_ "github.com/lib/pq"
)

type Chapter struct {
	ID      int
	Title   string
	Content string
}

type Book struct {
	ID    int
	Title string
}

type ChapterPage struct {
	Chapter     Chapter
	PreviousID  int
	NextID      int
	HasPrevious bool
	HasNext     bool
}

var db *sql.DB

func home(w http.ResponseWriter, r *http.Request) {

	tmpl, err := template.ParseFiles("templates/index.html")

	if err != nil {
		http.Error(w, "Unable to load page", http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, nil)
}

func books(w http.ResponseWriter, r *http.Request) {

    rows, err := db.Query("SELECT id, title FROM books ORDER BY id DESC")

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

func drafts(w http.ResponseWriter, r *http.Request) {

    rows, err := db.Query(
        "SELECT id, title, content FROM chapters WHERE status = 'draft' ORDER BY id DESC",
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

	id := r.URL.Query().Get("id")

	var ch Chapter

	err := db.QueryRow(
		"SELECT id, title, content FROM chapters WHERE id = $1",
		id,
	).Scan(&ch.ID, &ch.Title, &ch.Content)

	if err != nil {
		http.Error(w, "Chapter not found", http.StatusNotFound)
		return
	}

	var previousID int

	err = db.QueryRow(
    "SELECT id FROM chapters WHERE id < $1 AND status = 'published' ORDER BY id DESC LIMIT 1",
    id,
).Scan(&previousID)

	hasPrevious := err == nil

	var nextID int

	err = db.QueryRow(
    "SELECT id FROM chapters WHERE id > $1 AND status = 'published' ORDER BY id ASC LIMIT 1",
    id,
).Scan(&nextID)

	hasNext := err == nil

	pageData := ChapterPage{
		Chapter:     ch,
		PreviousID:  previousID,
		NextID:      nextID,
		HasPrevious: hasPrevious,
		HasNext:     hasNext,
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

	id := r.URL.Query().Get("id")

	var ch Chapter

	err := db.QueryRow(
		"SELECT id, title, content FROM chapters WHERE id = $1",
		id,
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

    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    id := r.FormValue("id")
    title := r.FormValue("title")
    content := r.FormValue("content")
    action := r.FormValue("action")

    status := "published"

    if action == "draft" {
        status = "draft"
    }

    _, err := db.Exec(
        "UPDATE chapters SET title = $1, content = $2, status = $3 WHERE id = $4",
        title,
        content,
        status,
        id,
    )

    if err != nil {
        http.Error(w, "Unable to update chapter", http.StatusInternalServerError)
        return
    }

    if status == "draft" {
        http.Redirect(w, r, "/drafts", http.StatusSeeOther)
        return
    }

    http.Redirect(w, r, "/chapter?id="+id, http.StatusSeeOther)
}

func delete(w http.ResponseWriter, r *http.Request) {

	id := r.URL.Query().Get("id")

	_, err := db.Exec(
		"DELETE FROM chapters WHERE id = $1",
		id,
	)

	if err != nil {
		http.Error(w, "Unable to delete chapter", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/books", http.StatusSeeOther)
}

func write(w http.ResponseWriter, r *http.Request) {

	rows, err := db.Query("SELECT id, title FROM books")

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

    title := r.FormValue("title")
    content := r.FormValue("content")
    bookID := r.FormValue("book_id")
    action := r.FormValue("action")

    status := "published"

    if action == "draft" {
        status = "draft"
    }

    _, err := db.Exec(
        "INSERT INTO chapters (title, content, book_id, status) VALUES ($1, $2, $3, $4)",
        title,
        content,
        bookID,
        status,
    )

    if err != nil {
        http.Error(w, "Unable to save chapter", http.StatusInternalServerError)
        return
    }

    if status == "draft" {
        fmt.Fprintln(w, "Chapter saved as draft!")
        return
    }

    fmt.Fprintln(w, "Chapter published successfully!")
}

func createBook(w http.ResponseWriter, r *http.Request) {

    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    title := r.FormValue("title")

    _, err := db.Exec(
        "INSERT INTO books (title) VALUES ($1)",
        title,
    )

    if err != nil {
        http.Error(w, "Unable to create book", http.StatusInternalServerError)
        return
    }

    http.Redirect(w, r, "/write", http.StatusSeeOther)
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
	http.HandleFunc("/drafts", drafts)
	http.HandleFunc("/write", write)
	http.HandleFunc("/create-book", createBook)
	http.HandleFunc("/publish", publish)
	http.HandleFunc("/chapter", chapter)
	http.HandleFunc("/edit", edit)
	http.HandleFunc("/update", update)
	http.HandleFunc("/delete", delete)

	fs := http.FileServer(http.Dir("./static"))

	http.Handle("/static/", http.StripPrefix("/static/", fs))

	println("ReadyWriter is running at http://localhost:8080")

	http.ListenAndServe(":8080", nil)
}
