# flip_car.cs: Mouse4 (боковая кнопка мыши, «назад») за рулём - поставить машину на колёса
import struct
src=open('asm4.py').read().split('op(0x0000)')[0]
exec(src)
op(0x0000)
lab("TOP")
op(0x0001,0)
IF(); op(0x0256,PCHAR); op(0x004D,J("TOP"))
IF(); op(0x0AB0,5); op(0x004D,J("TOP"))           # VK_XBUTTON1 = Mouse4
IF(); op(0x00DF,PACTOR); op(0x004D,J("REL"))
op(0x03C0,PACTOR,L(0))
op(0x046C,L(0),L(8))
IF(); op(0x003C,PACTOR,L(8)); op(0x004D,J("REL"))   # только водитель
op(0x00AA,L(0),L(1),L(2),L(3))
op(0x0174,L(0),L(4))
op(0x000B,L(3),0.6)
op(0x00AB,L(0),L(1),L(2),L(3))
op(0x0175,L(0),L(4))
lab("REL")
op(0x0001,0)
IF(); op(0x0AB0,5); op(0x004D,J("TOP"))
op(0x0002,J("REL"))
data=bytearray(b''.join(code))
for pos,n in fix: data[pos:pos+4]=struct.pack("<i",-labels[n])
open("flip_car.cs","wb").write(data); print(len(data),labels)
