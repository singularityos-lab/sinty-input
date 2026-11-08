# sinty-input

The input method service of Sinty OS. It lets you type languages that need
more than one key per character, inside every application, without an
external input method framework.

## Engines

| Name | Language | Typing |
|---|---|---|
| `pinyin` | Chinese (simplified) | Pinyin without tones; `v` for `ü`, `'` to split syllables. Whole sentences are segmented into the most likely words. |
| `kana` | Japanese | Romaji to hiragana. Space converts to kanji, including verbs with their okurigana; katakana is always offered. `nn` or `n'` types `ん`. |
| `hangul` | Korean | Dubeolsik (two-set) layout, with syllable composition and compound vowels and finals. |
| `telex` | Vietnamese | Telex: `s f r x j` for the tones, `z` to remove one, `aa ee oo aw ow uw dd` for the letters. |
| `simlish` | Simlish | Every word turns into Simlish when you finish it; `hello` is `sul sul`, `goodbye` is `dag dag`. |

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

```sh
gdbus call --session --dest dev.sinty.Input1 --object-path /dev/sinty/Input1 \
  --method dev.sinty.Input1.SelectEngine pinyin
```

## Building

```sh
make
make test
make install DESTDIR=...
```

The dictionaries in `internal/dict/data` are built into the binary.
`make dict` downloads their sources and generates them again; see
[NOTICE](internal/dict/data/NOTICE) for where they come from and their
licenses.

## License

GPL-3.0-only. The dictionaries keep their own licenses, listed in
[NOTICE](internal/dict/data/NOTICE).
