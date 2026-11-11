# sinty-input

The input method service of Sinty OS. It lets you type languages that need
more than one key per character, inside every application, without an
external input method framework.

## Engines

| Name | Language | Typing |
|---|---|---|
| `pinyin` | Chinese | Pinyin without tones; `v` for `ü`, `'` to split syllables. Whole sentences are segmented into the most likely words. |
| `shuangpin-xiaohe`, `shuangpin-ziranma`, `shuangpin-microsoft`, `shuangpin-sogou`, `shuangpin-abc`, `shuangpin-ziguang` | Chinese | Shuangpin: two keys for every syllable, with the layouts of Xiaohe, Ziranma, Microsoft, Sogou, Intelligent ABC and Ziguang. |
| `zhuyin` | Chinese (traditional) | Zhuyin (Bopomofo) on the standard layout; `3 4 6 7` type the tones, which end a syllable. Digits pick candidates after Up or Down. |
| `jyutping` | Cantonese | Jyutping without tones. |
| `wubi86`, `wubi98`, `wubi-ng` | Chinese (simplified) | Wubi 86, 98 and New Generation. A full code with one candidate is typed at once, `z` matches any key. |
| `cangjie3`, `cangjie5` | Chinese (traditional) | Cangjie 3 and 5, with the radicals shown while typing; `*` matches any key. |
| `quick` | Chinese (traditional) | Quick (Sucheng). |
| `array30` | Chinese (traditional) | Array 30. |
| `zhengma` | Chinese (simplified) | Zhengma. |
| `table-NAME` | | Every other dictionary of type `table` found in the dictionary folders. |
| `kana` | Japanese | Romaji typed as hiragana, with kanji candidates shown while typing, including verbs with their okurigana; the hiragana and katakana forms are always offered. `nn` or `n'` types `ん`. |
| `hangul` | Korean | Dubeolsik (two-set) layout, with syllable composition and compound vowels and finals. |
| `telex` | Vietnamese | Telex: `s f r x j` for the tones, `z` to remove one, `aa ee oo aw ow uw dd` for the letters. |
| `bijoy` | Bengali | Bijoy layout: keys type letters directly, `g` joins consonants (`g` twice shows the hasanta), `d c C` are typed before the consonant, `A` after it for the reph, `g` before a vowel sign types the vowel. |
| `simlish` | Simlish | Every word turns into Simlish when you finish it; `hello` is `sul sul`, `goodbye` is `dag dag`. |

Phonetic and table methods remember the candidates you choose and offer
them first next time; a sentence built from several choices is learned as a
new phrase. What they learn stays in
`~/.local/share/sinty-input/user`, one dictionary per method, and can be
listed, edited, exported, imported and cleared over D-Bus. Every Chinese
method can convert what it types to simplified or traditional characters,
with the Taiwan and Hong Kong variants, and every Chinese, Japanese and
Korean method types full width punctuation unless it is turned off. These
options are kept in `~/.config/sinty-input/config.json`.

While candidates are shown, Space or Down selects the next one, Up the
previous one, Page Up and Page Down turn the page, a digit picks from the
page, Enter commits and Escape goes back to what was typed. Keys with Ctrl,
Alt or Super are never taken.

## D-Bus

The service owns `dev.sinty.Input1` on the session bus at
`/dev/sinty/Input1` and is started on demand.

| Member | Signature | |
|---|---|---|
| `ListEngines()` | out `aas` | Name, label, symbol and language of every engine. |
| `SelectEngine(name)` | in `s` | An empty name turns the engines off. Switching commits what is being typed. |
| `ProcessKey(keysym, keycode, state, release)` | in `uuub`, out `b` | False when the key belongs to the application. |
| `FocusIn()`, `FocusOut()` | | Losing focus commits what is being typed. |
| `Reset()` | | Drops what is being typed. |
| `SetSurrounding(text, cursor)` | in `su` | |
| `SelectCandidate(index)` | in `i` | Index on the current page. |
| `ChangePage(next)` | in `b` | |
| `Engine` | property `s` | The active engine. |
| `Commit(text)` | signal `s` | |
| `Preedit(text, cursor)` | signal `si` | |
| `Candidates(items, labels, selected, previous, next, auxiliary)` | signal `asasibbs` | Empty `items` hides the list. |
| `Methods()` | out `a(ssssa{sv})` | Name, label, symbol, language and options of every engine: `kind`, `dictionary`, `dictionaries`, `conversion`, `conversions`, `fullwidth`, `learn`, `learned`. |
| `Configure(name, options)` | in `sa{sv}` | Sets `conversion` (`none`, `simplified`, `traditional`, `taiwan`, `hongkong`), `fullwidth`, `learn` or `dictionary`. |
| `Dictionaries()` | out `a(sssssb)` | Name, type, language, title, license and whether it is the user's. |
| `UserEntries(name)` | in `s`, out `a(ssi)` | What a method has learned: code, text and weight. |
| `UserSet(name, code, text, weight)`, `UserRemove(name, code, text)`, `UserClear(name)` | | Edits what a method has learned. |
| `UserExport(name, path)`, `UserImport(name, path)` | in `ss`, out of import `u` | Saves to or merges from a dictionary file. |
| `AddDictionary(path, name, language, license)` | in `ssss`, out `s` | Imports a Rime, fcitx or sinty table into the user's dictionaries and returns the new method's name. |
| `RemoveDictionary(name)` | in `s` | Deletes one of the user's dictionaries. |
| `Reload()` | | Looks for dictionaries again, after one is added or removed. |

```sh
gdbus call --session --dest dev.sinty.Input1 --object-path /dev/sinty/Input1 \
  --method dev.sinty.Input1.SelectEngine pinyin
```

## IBus

`sinty-input ibus` runs the same engines as IBus engines, named
`sinty-` followed by the engine name, so they can be used on GNOME and
other desktops that use IBus, next to engines such as ibus-rime.
`make install` puts the component in `share/ibus/component`; IBus lists
the engines after `ibus restart`, and `ibus engine sinty-pinyin` selects
one. Options and learned phrases are shared with the D-Bus service.

## Building

```sh
make
make test
make install DESTDIR=...
```

`make dict` downloads the sources of the shipped dictionaries and generates
them again into `dict`; see [NOTICE](dict/NOTICE) for where they come from
and their licenses.

## Dictionaries

Dictionaries are files, not part of the binary. They are looked up by name
in `~/.local/share/sinty-input/dict` first and then in
`sinty-input/dict` under every directory of `XDG_DATA_DIRS`
(`/usr/local/share` and `/usr/share` by default), so a file in the home
folder replaces a system one with the same name. `SINTY_INPUT_DICT_PATH`
replaces the whole search path.

A dictionary is a UTF-8 text file named `NAME.sdict`, or `NAME.sdict.gz`
when compressed with gzip. It starts with header lines, followed by one
entry per line:

```
@format sinty-dict 1
@name Wubi 86
@type table
@language zh-Hans
@keys abcdefghijklmnopqrstuvwxy
@length 4
@wildcard z
@autocommit yes
@license LGPL-3.0-only
@source https://github.com/rime/rime-wubi
a	工	99454797
a	戈
aa	式	12
```

Each entry is a code, a tab, the text it types and, optionally, a tab and
an integer weight. Higher weights are offered first, entries with the same
weight keep the order of the file, and an entry listed twice keeps its
highest weight. Lines starting with `#` and empty lines are ignored.

| Header | |
|---|---|
| `@format` | Always `sinty-dict 1`. Required, and must come first. |
| `@name` | The name shown in Settings. |
| `@type` | `phonetic`, `table` or `conversion`. |
| `@language` | A BCP 47 tag, such as `zh-Hans`, `zh-Hant`, `yue` or `ja`. |
| `@license` | An SPDX identifier. |
| `@source` | Where the data comes from. |
| `@keys` | Table: the keys that make up codes. |
| `@length` | Table: the longest code. |
| `@wildcard` | Table: a key that matches any one key. |
| `@autocommit` | Table: `yes` types the only candidate of a full code at once. |
| `@display` | Table: `key=label` pairs shown in place of the keys. |
| `@phrase` | Table: how to build the code of a new phrase, as `length:formula` rules. `4+:AaBaCaZa` takes, for phrases of four characters or more, the first key of the first, second and third character and of the last one. |

In a phonetic dictionary the code is the reading, syllables separated by
single spaces and without tones. In a conversion dictionary the code is the
text to replace and the entry is its replacement.

`sinty-input dict import` converts Rime (`.dict.yaml`) and fcitx table
files into this format. It only imports files whose license is compatible;
`LicenseRef-own` marks a table you made yourself:

```sh
sinty-input dict import -license LGPL-3.0-only -name "Wubi 86" wubi86.dict.yaml \
  ~/.local/share/sinty-input/dict/my-wubi.sdict.gz
```

## License

GPL-3.0-only. The dictionaries keep their own licenses, listed in
[NOTICE](dict/NOTICE).
