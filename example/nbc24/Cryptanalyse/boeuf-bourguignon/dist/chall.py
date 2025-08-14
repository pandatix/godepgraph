FLAG = "NBCTF{CENSURÉ}".encode('utf-8')

INGREDIENTS = ["🧈", "🧅", "🧄", "🥓", "🥩", "🥕", "🍷", "🍄", "🌿"]
NB_INGREDIENTS = len(INGREDIENTS)

def boeuf_bourguignon():
	# Conversion du FLAG en base 10 depuis la base 256
	# (car, dans une bytestring ("chaine d'octet"), chaque caractère ("octet"),
	# peut prendre 2^8=256 valeurs différentes, nous sommes donc en base 256)
	base_10 = 0
	for i,v in enumerate(FLAG):
		base_10 += v*256**i

	# Préparation d'un boeuf bourguignon
	mon_boeuf_bourguignon = ""
	while base_10 > 0:
		mon_boeuf_bourguignon += INGREDIENTS[base_10 % NB_INGREDIENTS]
		base_10 //= NB_INGREDIENTS

	return mon_boeuf_bourguignon

open("output.txt", "w") \
	.write(f'out = "{boeuf_bourguignon()}"')
