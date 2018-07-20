
all:
	go get -u github.com/xiaomingfuckeasylife/job/db
	go get github.com/astaxie/beego
	go build -o backend main.go

format:
	go fmt ./...

clean:
	rm -rf *.8 *.o *.out *.6
