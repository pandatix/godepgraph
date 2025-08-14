FLAG = "NBCTF{CENSURÉ}".encode("utf-8")
FLAG = bin(int(FLAG.hex(), 16))[2:]

def is_binary(e):
    return all([ee in ['0', '1'] for ee in e])

def get_pk():
    pk = [2]
    for i in range(len(FLAG) - 1):
        pk.append((pk[i] * 2) + 1)

    return pk

def encrypt(pt, pk):
    assert is_binary(pt)
    Σ = 0

    for (bit, nb) in zip(pt, pk):
        if bit == '1':
            Σ += nb

    return Σ

pk = get_pk()
ct = encrypt(FLAG, pk)

out = ""
out += f"{ct = }\n"
out += f"{pk = }"

open("output.txt", "w")\
    .write(out)
