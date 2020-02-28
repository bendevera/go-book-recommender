# book_recommender_web_server

## dump local postgres db into remote server db
`pg_dump -h localhost -C book_recommender | psql -h <serverip> -d book_recommender -U postgres`