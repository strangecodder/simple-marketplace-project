.PHONY: proto gen
proto:
	protoc \
		--proto_path=proto \
		--go_out=./gen --go_opt=module=simple-marketplace-project/gen \
		--go-grpc_out=./gen --go-grpc_opt=module=simple-marketplace-project/gen \
		proto/listing/v1/listing.proto \
		proto/order/v1/order.proto
gen:
	protoc \
		-I proto \
		-I proto/third_party \
		--go_out=gen --go_opt=paths=source_relative \
		--go-grpc_out=gen --go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=gen --grpc-gateway_opt=paths=source_relative \
		proto/payment/v1/payment.proto