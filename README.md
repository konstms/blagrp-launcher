# Лаунчер «Благовещенск RP»

Лёгкий лаунчер для КРМП-сервера «Благовещенск RP» (open.mp / SA-MP 0.3.7).

**Скачать лаунчер:** [BlagRP-Launcher.exe](https://github.com/konstms/blagrp-launcher/raw/pack/BlagRP-Launcher.exe)

Что делает:
- находит GTA San Andreas 1.0 US с SA-MP на компьютере (или даёт выбрать папку);
- ставит поверх неё сборку сервера и докачивает только изменённые файлы;
- оригинальные файлы игры, которые заменяет, сохраняет в `_blagrp_backup`;
- показывает онлайн сервера и подключает к нему одной кнопкой;
- обновляется сам.

Нужно у игрока: GTA San Andreas 1.0 US и SA-MP 0.3.7 (https://sa-mp.mp/downloads/).
Саму игру и SA-MP лаунчер не раздаёт.

## Устройство

- `launcher/` — исходники на Go, только стандартная библиотека (Win32 через syscall).
- `tools/build_pack.py` — собирает объекты сборки из эталонной папки игры.
- `tools/make_manifest.py` — пишет `manifest.json` и кладёт лаунчер в сборку.
- `tools/mkres.py` — иконка и манифест Windows (`.syso`).
- Ветка `pack` — то, что качают игроки: `manifest.json` и объекты `o/xx/<sha256>.gz`.

## Сборка лаунчера

```
python3 tools/mkres.py launcher 386
cd launcher && go test ./... && GOOS=windows GOARCH=386 go build -ldflags "-H windowsgui -s -w" -o BlagRP-Launcher.exe .
```

## Как выпустить обновление сборки

1. На эталонном ПК: `python build_pack.py "<папка игры>" out` (список файлов — в начале скрипта).
2. `python3 tools/make_manifest.py out BlagRP-Launcher.exe <версия>`.
3. Скопировать новые объекты и `manifest.json` в ветку `pack`, закоммитить и запушить.
   Лаунчеры игроков скачают только изменённые файлы.
