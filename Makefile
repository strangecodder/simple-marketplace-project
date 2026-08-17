.PHONY: proto
proto:
	protoc \
		--proto_path=proto \
		--go_out=./gen --go_opt=module=simple-marketplace-project/gen \
		--go-grpc_out=./gen --go-grpc_opt=module=simple-marketplace-project/gen \
		proto/listing/v1/listing.proto \
		proto/order/v1/order.proto