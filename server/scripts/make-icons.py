"""Génère les icônes PNG d'AURORA (Centre de paquets et menu principal DSM).
Dessin identique au logo web : un « A » en arc d'aurore sur fond de salle obscure.
Usage : python3 -I scripts/make-icons.py packaging/synology/icons
Les PNG produits sont versionnés : la construction du paquet ne dépend pas de Pillow."""
import sys
from pathlib import Path
from PIL import Image, ImageDraw

out = Path(sys.argv[1])
out.mkdir(parents=True, exist_ok=True)
S = 1024  # dessin sur-échantillonné puis réduit (anticrénelage)

def lerp(a, b, t):
    return tuple(round(a[i] + (b[i] - a[i]) * t) for i in range(3))

def draw(size):
    img = Image.new('RGBA', (S, S), (0, 0, 0, 0))
    d = ImageDraw.Draw(img)
    r = S * 0.22
    d.rounded_rectangle([0, 0, S - 1, S - 1], radius=r, fill=(11, 12, 15, 255))
    # Halo d'aurore discret en haut à gauche.
    glow = Image.new('RGBA', (S, S), (0, 0, 0, 0))
    gd = ImageDraw.Draw(glow)
    for i in range(60):
        t = i / 60
        gd.ellipse([-S * 0.4 + i * 4, -S * 0.5 + i * 4, S * 0.9 - i * 4, S * 0.45 - i * 4],
                   fill=(126, 232, 198, int(3 + 2 * t)))
    mask = Image.new('L', (S, S), 0)
    ImageDraw.Draw(mask).rounded_rectangle([0, 0, S - 1, S - 1], radius=r, fill=255)
    img = Image.composite(Image.alpha_composite(img, glow), img, mask)
    # Lettre A : dégradé vert profond -> menthe pâle, tracée par segments épais.
    strokes = Image.new('RGBA', (S, S), (0, 0, 0, 0))
    sd = ImageDraw.Draw(strokes)
    w = int(S * 0.085)
    pts = [(S * 0.19, S * 0.75), (S * 0.5, S * 0.19), (S * 0.81, S * 0.75)]
    c0, c1 = (47, 191, 152), (212, 250, 236)
    steps = 200
    for seg in range(2):
        (x0, y0), (x1, y1) = pts[seg], pts[seg + 1]
        for k in range(steps):
            t = k / steps
            x, y = x0 + (x1 - x0) * t, y0 + (y1 - y0) * t
            col = lerp(c0, c1, 1 - (y - S * 0.19) / (S * 0.56))
            sd.ellipse([x - w / 2, y - w / 2, x + w / 2, y + w / 2], fill=col + (255,))
    # Arc de l'aurore (barre du A).
    aw = int(S * 0.06)
    for k in range(steps):
        t = k / steps
        x = S * 0.31 + S * 0.38 * t
        y = S * 0.585 - S * 0.09 * (1 - (2 * t - 1) ** 2)
        sd.ellipse([x - aw / 2, y - aw / 2, x + aw / 2, y + aw / 2], fill=lerp(c0, c1, t) + (200,))
    img = Image.alpha_composite(img, strokes)
    return img.resize((size, size), Image.LANCZOS)

draw(64).save(out / 'PACKAGE_ICON.PNG')
draw(256).save(out / 'PACKAGE_ICON_256.PNG')
for s in (16, 24, 32, 48, 64, 72, 256):
    draw(s).save(out / f'aurora_{s}.png')
print('icônes générées dans', out)
