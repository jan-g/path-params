all: build

proto: model/db.pb.go

model/db.pb.go: model/db.proto
	protoc -I=model --go_out=model model/db.proto

build: example

example: main.go proto
	go build -o example main.go

.PHONY: all proto build
