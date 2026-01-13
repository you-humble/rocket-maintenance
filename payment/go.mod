module github.com/you-humble/rocket-maintenance/payment

go 1.25.4

replace github.com/you-humble/rocket-maintenance/shared => ../shared

replace github.com/you-humble/rocket-maintenance/platform => ../platform

require (
	github.com/caarlos0/env/v11 v11.3.1
	github.com/google/uuid v1.6.0
	github.com/joho/godotenv v1.5.1
	github.com/stretchr/testify v1.11.1
	github.com/you-humble/rocket-maintenance/platform v0.0.0-00010101000000-000000000000
	github.com/you-humble/rocket-maintenance/shared v0.0.0-00010101000000-000000000000
	google.golang.org/grpc v1.78.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.1 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251029180050-ab9386a59fda // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
