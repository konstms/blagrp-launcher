"""Рисует иконку лаунчера и собирает ресурсы Windows (иконка + манифест) в COFF-файл .syso,
который Go подхватывает при сборке. Запуск: python3 tools/mkres.py launcher/ 386"""
import io, struct, sys
from PIL import Image, ImageDraw, ImageFont

out_dir, arch = sys.argv[1], sys.argv[2]
FONT = "/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf"

def draw(size):
    S = 512
    im = Image.new("RGBA", (S, S), (0, 0, 0, 0))
    d = ImageDraw.Draw(im)
    # тёмно-синяя плашка со скруглением и янтарной полосой снизу
    d.rounded_rectangle((16, 16, S - 16, S - 16), radius=110, fill=(14, 24, 44, 255))
    d.rounded_rectangle((16, 16, S - 16, S - 16), radius=110, outline=(245, 165, 36, 255), width=18)
    f = ImageFont.truetype(FONT, 300)
    t = "Б"
    bb = d.textbbox((0, 0), t, font=f)
    d.text(((S - (bb[2] - bb[0])) / 2 - bb[0], (S - (bb[3] - bb[1])) / 2 - bb[1] - 30), t, font=f, fill=(255, 255, 255, 255))
    f2 = ImageFont.truetype(FONT, 78)
    t2 = "RP"
    bb = d.textbbox((0, 0), t2, font=f2)
    d.text(((S - (bb[2] - bb[0])) / 2 - bb[0], 360), t2, font=f2, fill=(245, 165, 36, 255))
    return im.resize((size, size), Image.LANCZOS)

sizes = [16, 24, 32, 48, 64, 128, 256]
pngs = []
for s in sizes:
    b = io.BytesIO()
    draw(s).save(b, "PNG")
    pngs.append(b.getvalue())
draw(256).save(out_dir + "/icon.png")

manifest = b"""<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0">
  <assemblyIdentity type="win32" name="BlagRP.Launcher" version="1.0.0.0"/>
  <trustInfo xmlns="urn:schemas-microsoft-com:asm.v3"><security><requestedPrivileges>
    <requestedExecutionLevel level="asInvoker" uiAccess="false"/>
  </requestedPrivileges></security></trustInfo>
  <application xmlns="urn:schemas-microsoft-com:asm.v3"><windowsSettings>
    <dpiAware xmlns="http://schemas.microsoft.com/SMI/2005/WindowsSettings">true</dpiAware>
  </windowsSettings></application>
  <compatibility xmlns="urn:schemas-microsoft-com:compatibility.v1"><application>
    <supportedOS Id="{8e0f7a12-bfb3-4fe8-b9a5-48fd50a15a9a}"/>
  </application></compatibility>
</assembly>
"""

grp = struct.pack("<HHH", 0, 1, len(sizes))
for i, (s, p) in enumerate(zip(sizes, pngs)):
    w = 0 if s >= 256 else s
    grp += struct.pack("<BBBBHHIH", w, w, 0, 0, 1, 32, len(p), i + 1)

# ресурсы: тип -> [(id, данные)]
res = {3: [(i + 1, p) for i, p in enumerate(pngs)], 14: [(1, grp)], 24: [(1, manifest)]}
LANG = 0x0409

# --- раскладка дерева каталогов ресурсов ---
def dir_size(n):
    return 16 + 8 * n

types = sorted(res)
off = dir_size(len(types))
type_dir_off = {}
for t in types:
    type_dir_off[t] = off
    off += dir_size(len(res[t]))
name_dir_off = {}
for t in types:
    for rid, _ in res[t]:
        name_dir_off[(t, rid)] = off
        off += dir_size(1)
entry_off = {}
for t in types:
    for rid, _ in res[t]:
        entry_off[(t, rid)] = off
        off += 16
data_off = {}
for t in types:
    for rid, data in res[t]:
        off = (off + 7) & ~7
        data_off[(t, rid)] = off
        off += len(data)
raw = bytearray(off)

def put_dir(at, entries):
    struct.pack_into("<IIHHHH", raw, at, 0, 0, 0, 0, 0, len(entries))
    for i, (ident, target, is_dir) in enumerate(entries):
        struct.pack_into("<II", raw, at + 16 + 8 * i, ident, target | (0x80000000 if is_dir else 0))

put_dir(0, [(t, type_dir_off[t], True) for t in types])
relocs = []
for t in types:
    put_dir(type_dir_off[t], [(rid, name_dir_off[(t, rid)], True) for rid, _ in res[t]])
    for rid, data in res[t]:
        put_dir(name_dir_off[(t, rid)], [(LANG, entry_off[(t, rid)], False)])
        e = entry_off[(t, rid)]
        struct.pack_into("<IIII", raw, e, data_off[(t, rid)], len(data), 0, 0)
        relocs.append(e)
        raw[data_off[(t, rid)]:data_off[(t, rid)] + len(data)] = data

machine, rtype = (0x14C, 7) if arch == "386" else (0x8664, 3)
hdr_size = 20 + 40
reloc_off = hdr_size + len(raw)
sym_off = reloc_off + 10 * len(relocs)
coff = struct.pack("<HHIIIHH", machine, 1, 0, sym_off, 1, 0, 0)
coff += struct.pack("<8sIIIIIIHHI", b".rsrc", 0, 0, len(raw), hdr_size, reloc_off, 0, len(relocs), 0, 0x40000040)
coff += bytes(raw)
for r in relocs:
    coff += struct.pack("<IIH", r, 0, rtype)
coff += struct.pack("<8sIhHBB", b".rsrc", 0, 1, 0, 3, 0)
coff += struct.pack("<I", 4)
open(f"{out_dir}/rsrc_windows_{arch}.syso", "wb").write(coff)
print("ok", len(coff), "bytes")
