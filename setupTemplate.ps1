#rename to {YOUR NAME}_setup.ps1 when you're going to use it,
#that way the it's properly ignored (.gitignore)
go build .\server.go
$env:NODB="" #Have this set to any value to not use the DB
$env:DBHOST=""
$env:DBPORT=""
$env:DBUSER=""
$env:DBPASS=""
$env:DBNAME=""
.\server.exe  #assumes that server.go is already compiled