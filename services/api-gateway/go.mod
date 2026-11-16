module github.com/sinergiinvestasiproperti/erp/services/api-gateway

go 1.23

require (
	github.com/google/uuid v1.6.0
	github.com/segmentio/kafka-go v0.4.47
	github.com/sinergiinvestasiproperti/erp/services/shared v0.0.0
)

require (
	github.com/klauspost/compress v1.15.9 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
)

replace github.com/sinergiinvestasiproperti/erp/services/shared => ../shared
