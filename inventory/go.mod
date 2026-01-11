module github.com/you-humble/rocket-maintenance/inventory

go 1.25.4

replace github.com/you-humble/rocket-maintenance/shared => ../shared

replace github.com/you-humble/rocket-maintenance/platform => ../platform

require (
	github.com/brianvoe/gofakeit/v7 v7.14.0
	github.com/caarlos0/env/v11 v11.3.1
	github.com/google/uuid v1.6.0
	github.com/joho/godotenv v1.5.1
	github.com/samber/lo v1.52.0
	github.com/stretchr/testify v1.11.1
	github.com/you-humble/rocket-maintenance/platform v0.0.0-00010101000000-000000000000
	github.com/you-humble/rocket-maintenance/shared v0.0.0-00010101000000-000000000000
	google.golang.org/grpc v1.78.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/golang/snappy v1.0.0 // indirect
	github.com/klauspost/compress v1.16.7 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.1 // indirect
	golang.org/x/crypto v0.44.0 // indirect
	golang.org/x/sync v0.18.0 // indirect
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	go.mongodb.org/mongo-driver/v2 v2.4.1
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251029180050-ab9386a59fda // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
