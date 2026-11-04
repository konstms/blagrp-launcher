# Собирает сборку для лаунчера из эталонной папки игры.
import os, sys, hashlib, gzip, json, fnmatch
GAME, OUT = sys.argv[1], sys.argv[2]
INCLUDE = [
  "modloader.asi", "modloader/modloader.ini", "modloader/.data/**",
  "modloader/V_idlewood/**", "modloader/gai/**", "modloader/store_24_7/**", "modloader/hospital_fix/**",
  "modloader/RealVision/**", "modloader/Trees_Birch/**", "modloader/HD_Vegetation/**",
  "CLEO.asi", "cleo/*.cleo", "cleo/no_stat_popups.cs",
  "SilentPatchSA.asi", "SilentPatchSA.ini", "neo/*", "stream.ini", "data/gta.dat",
  "models/grass/*", "vorbisFile.dll", "scripts/global.ini",
]
EXCLUDE = ["*.log", "modloader/.data/cache/**", "modloader/RealVision/models/Mobile_details.txd",
           "modloader/RealVision/models/texdb.txt", "*.bak*", "*.0"]
def match(rel, pats):
    for p in pats:
        if p.endswith("/**"):
            if rel.startswith(p[:-3] + "/"): return True
        elif fnmatch.fnmatchcase(rel, p) and ("/" in p or "/" not in rel): return True
    return False
files = []
for root, dirs, fs in os.walk(GAME):
    for f in fs:
        full = os.path.join(root, f)
        rel = os.path.relpath(full, GAME).replace(os.sep, "/")
        if match(rel, INCLUDE) and not match(rel, EXCLUDE):
            files.append(rel)
files.sort()
out = []
for rel in files:
    data = open(os.path.join(GAME, rel), "rb").read()
    h = hashlib.sha256(data).hexdigest()
    obj = os.path.join(OUT, "o", h[:2], h + ".gz")
    if not os.path.exists(obj):
        os.makedirs(os.path.dirname(obj), exist_ok=True)
        with open(obj + ".tmp", "wb") as fo:
            fo.write(gzip.compress(data, 6, mtime=0))
        os.replace(obj + ".tmp", obj)
    out.append({"p": rel, "s": len(data), "h": h, "z": os.path.getsize(obj)})
json.dump({"files": out}, open(os.path.join(OUT, "files.json"), "w"), ensure_ascii=False, indent=0)
print(len(out), "files", sum(f["s"] for f in out) // 2**20, "MB raw", sum(f["z"] for f in out) // 2**20, "MB gz")
