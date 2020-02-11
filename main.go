package main 

import (
	"net/http"
	"database/sql"
	"fmt"
	"log"
	"encoding/json"
	_ "github.com/lib/pq"
	"html/template"
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

func main() {
	psqlInfo := fmt.Sprintf("host=%s port=%d dbname=%s sslmode=disable", host, port, dbname)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		panic(err)
	}
	fmt.Println("Successfully Connected!")

	mux := http.NewServeMux()

	// TEMPLATE ROUTES 
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tpl := template.Must(template.ParseFiles("./templates/index.html"))
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
		tpl.Execute(w, books)
	})
	mux.HandleFunc("/recommend", func(w http.ResponseWriter, r *http.Request) {
		bookIds, ok := r.URL.Query()["book_id"]
		if !ok {
			return
		}
		// types, ok := r.URL.Query()["type"]
		// if !ok {
		// 	return
		// }
		book_id := bookIds[0]
		// recType := types[0]
		log.Println("book_id is: " + string(book_id))
		// log.Println("type is : " + string(recType))
		// Recommendations query
		rows, err := db.Query("SELECT id, recommendation_id, parent_book_id, rank FROM recommendation WHERE parent_book_id=$1;", book_id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var recs []Recommendation
		memory := make(map[int]bool)
		for rows.Next() {
			var rec Recommendation
			err = rows.Scan(&rec.ID, &rec.RecommendationID, &rec.ParentBookID, &rec.Rank)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if val, ok := memory[rec.RecommendationID]; !ok {
				fmt.Println("rec id: ", rec.RecommendationID)
				fmt.Println("value: ", val)
				memory[rec.RecommendationID] = true
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

		tpl := template.Must(template.ParseFiles("./templates/recommend.html"))
		tpl.Execute(w, recs)
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