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
	install -Dm644 packaging/sinty-input.xml $(DESTDIR)$(PREFIX)/share/ibus/component/sinty-input.xml
	install -d $(DESTDIR)$(PREFIX)/share/sinty-input/dict
	install -m644 dict/*.sdict.gz dict/NOTICE $(DESTDIR)$(PREFIX)/share/sinty-input/dict/

dict:
	mkdir -p $(SOURCES)
	curl -fsSL -o $(SOURCES)/SKK-JISYO.L.json https://raw.githubusercontent.com/skk-dev/dict/master/json/SKK-JISYO.L.json
	curl -fsSL -o $(SOURCES)/cedict.txt.gz https://www.mdbg.net/chinese/export/cedict/cedict_1_0_ts_utf-8_mdbg.txt.gz
	curl -fsSL -o $(SOURCES)/dict.txt https://raw.githubusercontent.com/fxsjy/jieba/master/jieba/dict.txt
	curl -fsSL -o $(SOURCES)/Unihan.zip https://www.unicode.org/Public/UCD/latest/ucd/Unihan.zip
	cd $(SOURCES) && unzip -o -q Unihan.zip Unihan_Readings.txt
	curl -fsSL -o $(SOURCES)/wubi86.dict.yaml https://raw.githubusercontent.com/rime/rime-wubi/master/wubi86.dict.yaml
	curl -fsSL -o $(SOURCES)/xinshijiwubi.dict.yaml https://raw.githubusercontent.com/Pioneer-Linzi/xinshijiwubi/master/xinshijiwubi.dict.yaml
	for t in wubi98 cangjie3 cangjie5 quick5 array30 zhengma; do \
		curl -fsSL -o $(SOURCES)/$$t.txt https://raw.githubusercontent.com/fcitx/fcitx5-table-extra/master/tables/$$t.txt || exit 1; \
	done
	for f in jyut6ping3.chars.dict.yaml jyut6ping3.words.dict.yaml essay-cantonese.txt; do \
		curl -fsSL -o $(SOURCES)/$$f https://raw.githubusercontent.com/rime/rime-cantonese/main/$$f || exit 1; \
	done
	for f in STCharacters STPhrases TSCharacters TSPhrases TWVariants TWPhrases HKVariants; do \
		curl -fsSL -o $(SOURCES)/$$f.txt https://raw.githubusercontent.com/BYVoid/OpenCC/master/data/dictionary/$$f.txt || exit 1; \
	done
	go run ./internal/dict/gen -sources $(SOURCES) -out dict
