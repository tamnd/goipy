import colorsys


def f(x):
    return round(x, 10)


def test_rgb_to_yiq():
    # red
    y, i, q = colorsys.rgb_to_yiq(1.0, 0.0, 0.0)
    print(f(y), f(i), f(q))
    # green
    y, i, q = colorsys.rgb_to_yiq(0.0, 1.0, 0.0)
    print(f(y), f(i), f(q))
    # blue
    y, i, q = colorsys.rgb_to_yiq(0.0, 0.0, 1.0)
    print(f(y), f(i), f(q))
    # white
    y, i, q = colorsys.rgb_to_yiq(1.0, 1.0, 1.0)
    print(f(y), f(i), f(q))
    # black
    y, i, q = colorsys.rgb_to_yiq(0.0, 0.0, 0.0)
    print(f(y), f(i), f(q))
    # mixed
    y, i, q = colorsys.rgb_to_yiq(0.5, 0.5, 0.5)
    print(f(y), f(i), f(q))
    print('test_rgb_to_yiq ok')


def test_yiq_to_rgb():
    # roundtrip red
    r, g, b = colorsys.yiq_to_rgb(0.3, 0.6, 0.21)
    print(f(r), f(g), f(b))
    # roundtrip white
    r, g, b = colorsys.yiq_to_rgb(1.0, 0.0, 0.0)
    print(f(r), f(g), f(b))
    # roundtrip black
    r, g, b = colorsys.yiq_to_rgb(0.0, 0.0, 0.0)
    print(f(r), f(g), f(b))
    # clamp: out-of-range input
    r, g, b = colorsys.yiq_to_rgb(0.0, 0.6, 0.0)
    print(f(r), f(g), f(b))
    print('test_yiq_to_rgb ok')


def test_rgb_to_hls():
    # red → (0.0, 0.5, 1.0)
    h, l, s = colorsys.rgb_to_hls(1.0, 0.0, 0.0)
    print(f(h), f(l), f(s))
    # green → (1/3, 0.5, 1.0)
    h, l, s = colorsys.rgb_to_hls(0.0, 1.0, 0.0)
    print(f(h), f(l), f(s))
    # blue → (2/3, 0.5, 1.0)
    h, l, s = colorsys.rgb_to_hls(0.0, 0.0, 1.0)
    print(f(h), f(l), f(s))
    # white → (0.0, 1.0, 0.0)
    h, l, s = colorsys.rgb_to_hls(1.0, 1.0, 1.0)
    print(f(h), f(l), f(s))
    # black → (0.0, 0.0, 0.0)
    h, l, s = colorsys.rgb_to_hls(0.0, 0.0, 0.0)
    print(f(h), f(l), f(s))
    # grey 50%
    h, l, s = colorsys.rgb_to_hls(0.5, 0.5, 0.5)
    print(f(h), f(l), f(s))
    print('test_rgb_to_hls ok')


def test_hls_to_rgb():
    # hue=0 (red), l=0.5, s=1.0
    r, g, b = colorsys.hls_to_rgb(0.0, 0.5, 1.0)
    print(f(r), f(g), f(b))
    # hue=1/3 (green)
    r, g, b = colorsys.hls_to_rgb(1.0/3, 0.5, 1.0)
    print(f(r), f(g), f(b))
    # hue=2/3 (blue)
    r, g, b = colorsys.hls_to_rgb(2.0/3, 0.5, 1.0)
    print(f(r), f(g), f(b))
    # s=0 (grey)
    r, g, b = colorsys.hls_to_rgb(0.0, 0.5, 0.0)
    print(f(r), f(g), f(b))
    # white
    r, g, b = colorsys.hls_to_rgb(0.0, 1.0, 0.0)
    print(f(r), f(g), f(b))
    print('test_hls_to_rgb ok')


def test_rgb_to_hsv():
    # red → (0.0, 1.0, 1.0)
    h, s, v = colorsys.rgb_to_hsv(1.0, 0.0, 0.0)
    print(f(h), f(s), f(v))
    # green → (1/3, 1.0, 1.0)
    h, s, v = colorsys.rgb_to_hsv(0.0, 1.0, 0.0)
    print(f(h), f(s), f(v))
    # blue → (2/3, 1.0, 1.0)
    h, s, v = colorsys.rgb_to_hsv(0.0, 0.0, 1.0)
    print(f(h), f(s), f(v))
    # white → (0.0, 0.0, 1.0)
    h, s, v = colorsys.rgb_to_hsv(1.0, 1.0, 1.0)
    print(f(h), f(s), f(v))
    # black → (0.0, 0.0, 0.0)
    h, s, v = colorsys.rgb_to_hsv(0.0, 0.0, 0.0)
    print(f(h), f(s), f(v))
    # grey 50% → (0.0, 0.0, 0.5)
    h, s, v = colorsys.rgb_to_hsv(0.5, 0.5, 0.5)
    print(f(h), f(s), f(v))
    print('test_rgb_to_hsv ok')


def test_hsv_to_rgb():
    # hue=0 (red)
    r, g, b = colorsys.hsv_to_rgb(0.0, 1.0, 1.0)
    print(f(r), f(g), f(b))
    # hue=1/3 (green)
    r, g, b = colorsys.hsv_to_rgb(1.0/3, 1.0, 1.0)
    print(f(r), f(g), f(b))
    # hue=2/3 (blue)
    r, g, b = colorsys.hsv_to_rgb(2.0/3, 1.0, 1.0)
    print(f(r), f(g), f(b))
    # s=0 (grey)
    r, g, b = colorsys.hsv_to_rgb(0.0, 0.0, 0.5)
    print(f(r), f(g), f(b))
    # black
    r, g, b = colorsys.hsv_to_rgb(0.0, 0.0, 0.0)
    print(f(r), f(g), f(b))
    # cyan hue=0.5
    r, g, b = colorsys.hsv_to_rgb(0.5, 1.0, 1.0)
    print(f(r), f(g), f(b))
    print('test_hsv_to_rgb ok')


def test_roundtrips():
    colors = [
        (1.0, 0.0, 0.0),
        (0.0, 1.0, 0.0),
        (0.0, 0.0, 1.0),
        (0.5, 0.25, 0.75),
        (0.2, 0.8, 0.4),
    ]
    for r, g, b in colors:
        # HSV roundtrip
        h, s, v = colorsys.rgb_to_hsv(r, g, b)
        r2, g2, b2 = colorsys.hsv_to_rgb(h, s, v)
        print(f(r2), f(g2), f(b2))

        # HLS roundtrip
        h, l, s = colorsys.rgb_to_hls(r, g, b)
        r2, g2, b2 = colorsys.hls_to_rgb(h, l, s)
        print(f(r2), f(g2), f(b2))

        # YIQ roundtrip
        y, i, q = colorsys.rgb_to_yiq(r, g, b)
        r2, g2, b2 = colorsys.yiq_to_rgb(y, i, q)
        print(f(r2), f(g2), f(b2))

    print('test_roundtrips ok')


test_rgb_to_yiq()
test_yiq_to_rgb()
test_rgb_to_hls()
test_hls_to_rgb()
test_rgb_to_hsv()
test_hsv_to_rgb()
test_roundtrips()
