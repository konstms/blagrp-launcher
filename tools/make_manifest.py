"""Собирает manifest.json для ветки pack.

python3 tools/make_manifest.py <pack_dir> <launcher.exe> <версия лаунчера>

В <pack_dir> должны лежать files.json (результат build_pack.py) и папка o/ с объектами.
Скрипт добавит туда сжатый лаунчер, его копию BlagRP-Launcher.exe и manifest.json.
"""
import gzip, hashlib, json, os, shutil, sys, time

pack, exe, ver = sys.argv[1], sys.argv[2], sys.argv[3]
files = json.load(open(os.path.join(pack, "files.json"), encoding="utf-8"))["files"]

def put_obj(data):
    h = hashlib.sha256(data).hexdigest()
    p = os.path.join(pack, "o", h[:2], h + ".gz")
    if not os.path.exists(p):
        os.makedirs(os.path.dirname(p), exist_ok=True)
        open(p, "wb").write(gzip.compress(data, 9, mtime=0))
    return h, os.path.getsize(p)

exe_data = open(exe, "rb").read()
eh, ez = put_obj(exe_data)
shutil.copyfile(exe, os.path.join(pack, "BlagRP-Launcher.exe"))

loader = next(f for f in files if f["p"] == "vorbisFile.dll")
# номер сборки: +1, если список файлов изменился
build = 1
old_files = None
try:
    old = json.load(open(os.path.join(pack, "manifest.json"), encoding="utf-8"))
    build = int(old.get("build", 0)) or 13
    old_files = old.get("files")
except Exception:
    pass
built = None
if old_files != files:
    build = build + 1 if old_files is not None else build
    built = time.strftime("%d.%m.%Y %H:%M", time.gmtime(time.time() + 9 * 3600))
else:
    built = old.get("built") or time.strftime("%d.%m.%Y %H:%M", time.gmtime(time.time() + 9 * 3600))
manifest = {
    "version": int(time.time()),
    "build": build,
    "built": built,
    "server": "87.228.5.28:7777",
    "launcher": {"v": ver, "h": eh, "s": len(exe_data), "z": ez},
    # оригинальный vorbisFile.dll игры (65536 байт) сохраняем как vorbisHooked.dll — его подгружает ASI-загрузчик
    "renames": [{"from": "vorbisFile.dll", "to": "vorbisHooked.dll", "if_not": loader["h"], "if_size": 65536}],
    "files": files,
}
json.dump(manifest, open(os.path.join(pack, "manifest.json"), "w", encoding="utf-8"), ensure_ascii=False, separators=(",", ":"))
used = {f["h"] for f in files} | {eh}
print("build", build, built, "|", len(files), "files;", sum(f["z"] for f in files) // 2**20, "MB to download; objects:", len(used))
