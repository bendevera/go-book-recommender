package main 

import (
	"net/http"
	"database/sql"
	"fmt"
	"log"
	"encoding/json"
	_ "github.com/lib/pq"
)

const (
	host	= "localhost"
	port	= 5432
	dbname 	= "book_recommender"
)


type Book struct {
	ID 		int 	`json:"book_id"`
	Title 	string 	`json:"title"`
	Authors string	`json:"author"`
}

type Recommendation struct {
	ID 					int 	`json:"id"`
	ParentBookID		int 	`json:"parent_id"`
	RecommendationID	int		`json:"rec_id"`
	BookData			Book 	`json:"book_data"`
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
	mux.HandleFunc("/recommend", func(w http.ResponseWriter, r *http.Request) {
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
		rows, err := db.Query("SELECT id, recommendation_id, parent_book_id FROM recommendation WHERE parent_book_id=$1 AND type=$2;", book_id, recType)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// defer rows.Close()
		var recs []Recommendation
		for rows.Next() {
			var rec Recommendation
			err = rows.Scan(&rec.ID, &rec.RecommendationID, &rec.ParentBookID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			sqlStatement := `SELECT id, title, authors FROM book WHERE book_id=$1;`
			var book_data Book

			row := db.QueryRow(sqlStatement, rec.RecommendationID)
			err := row.Scan(&book_data.ID, &book_data.Title, &book_data.Authors);
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

	log.Fatal(http.ListenAndServe(":3000", mux))
}