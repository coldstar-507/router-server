module github.com/coldstar-507/router-server

go 1.23.2

require (
	github.com/coldstar-507/utils/http_utils v0.0.0
	github.com/coldstar-507/utils/utils v0.0.0
)

require go.mongodb.org/mongo-driver v1.17.1 // indirect

replace (
	github.com/coldstar-507/utils/http_utils => ../utils/http_utils
	github.com/coldstar-507/utils/utils => ../utils/utils
)
