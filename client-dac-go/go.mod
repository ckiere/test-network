module github.com/ckiere/test-network/client-dac-go

go 1.15

require (
	github.com/dbogatov/dac-lib v1.0.0
	github.com/dbogatov/fabric-amcl v0.0.0-20190731091901-c69f438d7884
	github.com/golang/protobuf v1.3.3
	github.com/hyperledger/fabric-protos-go v0.0.0-20200707132912-fee30f3ccd23
	github.com/hyperledger/fabric-sdk-go v1.0.0
	github.com/pkg/errors v0.8.1
)

replace github.com/hyperledger/fabric-sdk-go v1.0.0 => ./internal-fabric-sdk-go
