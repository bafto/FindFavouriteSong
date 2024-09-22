set -e pipefail

go generate
go build .
