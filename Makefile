BIN := sinty-input
PREFIX ?= /usr
SOURCES ?= sources

.PHONY: all build static test vet install clean dict

all: build

build:
	go build -o $(BIN) ./cmd/sinty-input

static:
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o $(BIN) ./cmd/sinty-input

test:
	go test ./...

vet:
	go vet ./...

install: static
	install -Dm755 $(BIN) $(DESTDIR)$(PREFIX)/libexec/$(BIN)
	install -Dm644 packaging/dev.sinty.Input1.service $(DESTDIR)$(PREFIX)/share/dbus-1/services/dev.sinty.Input1.service

dict:
	mkdir -p $(SOURCES)
	curl -fsSL -o $(SOURCES)/SKK-JISYO.L.json https://raw.githubusercontent.com/skk-dev/dict/master/json/SKK-JISYO.L.json
	curl -fsSL -o $(SOURCES)/cedict.txt.gz https://www.mdbg.net/chinese/export/cedict/cedict_1_0_ts_utf-8_mdbg.txt.gz
	curl -fsSL -o $(SOURCES)/dict.txt https://raw.githubusercontent.com/fxsjy/jieba/master/jieba/dict.txt
	curl -fsSL -o $(SOURCES)/Unihan.zip https://www.unicode.org/Public/UCD/latest/ucd/Unihan.zip
	cd $(SOURCES) && unzip -o -q Unihan.zip Unihan_Readings.txt
	go run ./internal/dict/gen -skk $(SOURCES)/SKK-JISYO.L.json -cedict $(SOURCES)/cedict.txt.gz -words $(SOURCES)/dict.txt -unihan $(SOURCES)/Unihan_Readings.txt

clean:
	rm -f $(BIN)
