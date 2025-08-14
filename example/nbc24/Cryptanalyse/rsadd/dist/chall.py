# Tu auras besoin de ça : https://pycryptodome.readthedocs.io/en/latest/src/installation.html
from Crypto.Util.number import bytes_to_long, long_to_bytes, getPrime
from math import log

# Rajoute des zeros pour avoir un msg de la même taille que n ~ lors du chiffrement
def pad(msg, n):
	count = log(n, 256)

	while(len(msg) < count):
		msg += b"\x00"

	return msg

FLAG = "NBCTF{CENSURÉ}".encode("utf-8")
assert FLAG == long_to_bytes(bytes_to_long(FLAG))

# Génération de la clef publique
p, q = getPrime(1024), getPrime(1024)
n = p*q
e = 0x10001

# Chiffrement
m = bytes_to_long(pad(FLAG, n))
c = (m*e) % n

# Export des données dans output.txt
out = ""
out += f"n = {n}\n"
out += f"e = {e}\n"
out += f"c = {c}"
open("output.txt", "w").write(out)
