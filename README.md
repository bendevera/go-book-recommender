# book_recommender_web_server

## dump local postgres db into remote server db
`pg_dump -h localhost -C book_recommender | psql -h <serverip> -d book_recommender -U postgres`

## had to change /etc/postgresql/10/main/pg_hba.conf first

record that was added then commented out after completing dump:

`host   all             postgres        0.0.0.0/0               trust`