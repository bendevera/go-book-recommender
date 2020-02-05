package main 

import (
	"database/sql"
	"fmt"
	"os"
	_ "github.com/lib/pq"
)

const (
	host	= "localhost"
	port	= 5432
	dbname 	= "book_recommender"
)


type Book struct {
	ID 		int
	title 	string
	authors string
}

type Recommendation struct {
	ID 					int 
	parent_book_id 		int 
	recommendation_id	int
	book_data			Book
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

	book_id := os.Args[1]
	rows, err := db.Query("SELECT id, recommendation_id, parent_book_id FROM recommendation WHERE parent_book_id=$1;", book_id)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var rec Recommendation
		err = rows.Scan(&rec.ID, &rec.recommendation_id, &rec.parent_book_id)
		if err != nil {
			panic(err)
		}
		sqlStatement := `SELECT id, title, authors FROM book WHERE book_id=$1;`
		var book_data Book

		row := db.QueryRow(sqlStatement, rec.recommendation_id)
		err := row.Scan(&book_data.ID, &book_data.title, &book_data.authors);
		switch err {
		case sql.ErrNoRows:
			fmt.Println("now rows were returned")
		case nil:
			rec.book_data = book_data
		default:
			panic(err)
		}
		fmt.Println(rec)
	}

	err = rows.Err()
	if err != nil {
		panic(err)
	}
}