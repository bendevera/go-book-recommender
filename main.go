package main 

import (
	"net/http"
	"database/sql"
	"fmt"
	"log"
	"encoding/json"
	_ "github.com/lib/pq"
	"html/template"
	"strconv"
)

const (
	host	= "localhost"
	port	= 5432
	dbname 	= "book_recommender"
)


type Book struct {
	ID 		int 			`json:"id"`
	BookID  int 			`json:"book_id"`
	Title 	string 			`json:"title"`
	Authors string			`json:"author"`
	PubYear sql.NullInt64	`json:"pub_year"`
	Rating 	sql.NullFloat64 `json:"rating"`
	ImgURL 	string			`json:"img_url"`

}

type Recommendation struct {
	ID 					int 			`json:"id"`
	ParentBookID		int 			`json:"parent_id"`
	RecommendationID	int				`json:"rec_id"`
	BookData			Book 			`json:"book_data"`
	Rank 				sql.NullFloat64 `json:"rank"`
}

type Page struct {
	Books 	[]Book 
	Recs 	[]Recommendation
	RefBook	Book
	Numbers	[]int 
}

func main() {
	// attempts to connect to the postgres database
	psqlInfo := fmt.Sprintf("host=%s port=%d dbname=%s sslmode=disable", host, port, dbname)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// ensures connection was successful
	err = db.Ping()
	if err != nil {
		panic(err)
	}
	fmt.Println("Successfully Connected!")

	// instantiates mux server
	mux := http.NewServeMux()

	// TEMPLATE ROUTES 
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// grabs "page" query string parameter
		pages, ok := r.URL.Query()["page"]
		// gets SQL query for that page
		sqlStatement, page := getSqlStatement(pages, ok)
		log.Println(" / with page #: " + strconv.Itoa(page))
		rows, err := db.Query(sqlStatement)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// generates splice of books for that page
		var books []Book
		for rows.Next() {
			var book Book 
			err := rows.Scan(&book.ID, &book.BookID, &book.Title, &book.Authors, &book.PubYear, &book.Rating, &book.ImgURL)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			books = append(books, book)
		}
		err = rows.Err()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// instantiates page struct for pagination numbers and books
		var CurrPage Page 
		CurrPage.Numbers = getNumbers(page)
		CurrPage.Books = books
		tpl := template.Must(template.ParseFiles("./templates/index.html"))
		tpl.Execute(w, CurrPage)
	})
	mux.HandleFunc("/recommend", func(w http.ResponseWriter, r *http.Request) {
		// gets ref book id
		bookIds, ok := r.URL.Query()["book_id"]
		if !ok {
			return
		}
		book_id := bookIds[0]
		log.Println(" /recommend with book_id: " + string(book_id))

		// gets recommendation using ref book
		rows, err := db.Query("SELECT id, recommendation_id, parent_book_id, rank FROM recommendation WHERE parent_book_id=$1;", book_id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		// generates splice of recommendations
		var recs []Recommendation
		// need to track books I already add to splice
		// because there were 2 methods recs were made so
		// some are duplicated
		memory := make(map[int]bool)
		for rows.Next() {
			var rec Recommendation
			err = rows.Scan(&rec.ID, &rec.RecommendationID, &rec.ParentBookID, &rec.Rank)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			// makes sure rec isn't already in recs splice
			if val, ok := memory[rec.RecommendationID]; !ok {
				memory[rec.RecommendationID] = !val
				// curr Book query
				sqlStatement := `SELECT id, book_id, title, authors, pub_year, avg_rating, img_url FROM book WHERE book_id=$1;`
				var book_data Book

				row := db.QueryRow(sqlStatement, rec.RecommendationID)
				err := row.Scan(&book_data.ID, &book_data.BookID, &book_data.Title, &book_data.Authors, &book_data.PubYear, &book_data.Rating, &book_data.ImgURL);
				switch err {
				case sql.ErrNoRows:
					fmt.Println("no rows were returned")
				case nil:
					rec.BookData = book_data
				default:
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				recs = append(recs, rec)
			}
		}

		err = rows.Err()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// grabs ref book (for title and header of the page)
		var book Book 
		sqlStatement := `SELECT id, book_id, title, authors, pub_year, avg_rating, img_url FROM book WHERE id=$1;`
		row := db.QueryRow(sqlStatement, book_id)
		err = row.Scan(&book.ID, &book.BookID, &book.Title, &book.Authors, &book.PubYear, &book.Rating, &book.ImgURL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// instantiates Page struct for the rec template
		var RecPage Page 
		RecPage.Recs = recs
		RecPage.RefBook = book
		tpl := template.Must(template.ParseFiles("./templates/recommend.html"))
		tpl.Execute(w, RecPage)
	})

	// API ROUTES
	mux.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		message := "Welcome to the book recomender Go API!"
		js, err := json.Marshal(message)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	})
	mux.HandleFunc("/api/books", func(w http.ResponseWriter, r *http.Request) {
		book_ids, ok := r.URL.Query()["book_id"]
		if ok {
			book_id := book_ids[0]
			var book Book 
			sqlStatement := `SELECT id, book_id, title, authors, pub_year, avg_rating, img_url FROM book WHERE id=$1;`
			row := db.QueryRow(sqlStatement, book_id)
			err := row.Scan(&book.ID, &book.BookID, &book.Title, &book.Authors, &book.PubYear, &book.Rating, &book.ImgURL)

			switch err {
			case sql.ErrNoRows:
				fmt.Println("did not find book with id: " + string(book_id))
			case nil:
				fmt.Println("query with book id: " + string(book_id))
			default:
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return 
			}

			js, err := json.Marshal(book)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return 
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(js)
		} else {
			sqlStatement := `SELECT id, book_id, title, authors, pub_year, avg_rating, img_url FROM book LIMIT 10;`
			rows, err := db.Query(sqlStatement)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			var books []Book
			for rows.Next() {
				var book Book 
				err := rows.Scan(&book.ID, &book.BookID, &book.Title, &book.Authors, &book.PubYear, &book.Rating, &book.ImgURL)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				books = append(books, book)
			}
			err = rows.Err()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			js, err := json.Marshal(books)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(js)
		}
	})
	mux.HandleFunc("/api/recommend", func(w http.ResponseWriter, r *http.Request) {
		book_ids, ok := r.URL.Query()["book_id"]
		if !ok {
			return
		}
		types, ok := r.URL.Query()["type"]
		if !ok {
			return
		}
		book_id := book_ids[0]
		recType := types[0]
		log.Println("book_id is: " + string(book_id))
		log.Println("type is : " + string(recType))
		// Recommendations query
		rows, err := db.Query("SELECT id, recommendation_id, parent_book_id, rank FROM recommendation WHERE parent_book_id=$1 AND type=$2;", book_id, recType)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var recs []Recommendation
		for rows.Next() {
			var rec Recommendation
			err = rows.Scan(&rec.ID, &rec.RecommendationID, &rec.ParentBookID, &rec.Rank)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			// curr Book query
			sqlStatement := `SELECT id, book_id, title, authors, pub_year, avg_rating, img_url FROM book WHERE book_id=$1;`
			var book_data Book

			row := db.QueryRow(sqlStatement, rec.RecommendationID)
			err := row.Scan(&book_data.ID, &book_data.BookID, &book_data.Title, &book_data.Authors, &book_data.PubYear, &book_data.Rating, &book_data.ImgURL);
			switch err {
			case sql.ErrNoRows:
				fmt.Println("no rows were returned")
			case nil:
				rec.BookData = book_data
			default:
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			recs = append(recs, rec)
		}

		err = rows.Err()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		js, err := json.Marshal(recs)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	})

	fs := http.FileServer(http.Dir("./assets/"))
	mux.Handle("/assets/", http.StripPrefix("/assets/", fs))
	log.Fatal(http.ListenAndServe(":3000", mux))
}

func getSqlStatement(pages []string, ok bool) (string, int) {
	if !ok {
		return `SELECT id, book_id, title, authors, pub_year, avg_rating, img_url FROM book LIMIT 10;`, 1
	} else {
		page, err := strconv.Atoi(pages[0])
		if err != nil {
			return `SELECT id, book_id, title, authors, pub_year, avg_rating, img_url FROM book LIMIT 10;`, 1
		} else {
			startId := (page-1) * 10
			endId := page * 10
			return `SELECT id, book_id, title, authors, pub_year, avg_rating, img_url FROM book WHERE id < ` + strconv.Itoa(endId) + ` AND id > ` + strconv.Itoa(startId) + `;`, page
		}
	}
}

func getNumbers(page int) []int {
	if page == 1 {
		return []int{1, 2, 3, 4}
	} else {
		return []int{page-1, page, page+1, page+2}
	}
}