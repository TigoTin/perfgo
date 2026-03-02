$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -o bin/perfgo-linux-amd64 .
