from Crypto.Util.number import getPrime, bytes_to_long
FLAG = "NBCTF{CENSURÉ}".encode("utf-8")

p = getPrime(1024)
q = getPrime(1024)
N = e = p*q
φ = (p-1)*(q-1)
d = pow(e, -1, φ)

m = bytes_to_long(FLAG)
c = pow(m, e, N)

out = ''
out += f"N = {N}\n"
out += f"e = {pow(d, -1, φ)}\n"
out += f"c = {c}"

open("output.txt", "w") \
	.write(out)
