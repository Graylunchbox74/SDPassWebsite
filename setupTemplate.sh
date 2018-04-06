#rename to {YOUR NAME}_setup.ps1 when you're going to use it,
#that way the it's properly ignored (.gitignore)
go build server.go
export NODB=1 #Have this set to any value to not use the DB
export DBHOST=localhost
export DBPORT=5432
export DBUSER=username
export DBPASS=password
export DBNAME=dbname
./server #assumes that server.go is already compiled