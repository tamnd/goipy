package vm

import (
	"math"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildColorsys() *object.Module {
	m := &object.Module{Name: "colorsys", Dict: object.NewDict()}

	mkf := func(f float64) *object.Float { return &object.Float{V: f} }

	getF := func(o object.Object) float64 {
		switch v := o.(type) {
		case *object.Float:
			return v.V
		case *object.Int:
			return float64(v.Int64())
		}
		return 0
	}

	clamp01 := func(v float64) float64 {
		if v < 0 {
			return 0
		}
		if v > 1 {
			return 1
		}
		return v
	}

	fmod1 := func(v float64) float64 {
		v = math.Mod(v, 1.0)
		if v < 0 {
			v += 1.0
		}
		return v
	}

	// _v helper for hls_to_rgb
	const (
		oneSixth = 1.0 / 6.0
		oneThird = 1.0 / 3.0
		twoThird = 2.0 / 3.0
	)
	_v := func(m1, m2, hue float64) float64 {
		hue = math.Mod(hue, 1.0)
		if hue < 0 {
			hue += 1.0
		}
		if hue < oneSixth {
			return m1 + (m2-m1)*hue*6.0
		}
		if hue < 0.5 {
			return m2
		}
		if hue < twoThird {
			return m1 + (m2-m1)*(twoThird-hue)*6.0
		}
		return m1
	}

	m.Dict.SetStr("rgb_to_yiq", &object.BuiltinFunc{Name: "rgb_to_yiq", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		r, g, b := getF(a[0]), getF(a[1]), getF(a[2])
		y := 0.30*r + 0.59*g + 0.11*b
		ii := 0.74*(r-y) - 0.27*(b-y)
		q := 0.48*(r-y) + 0.41*(b-y)
		return &object.Tuple{V: []object.Object{mkf(y), mkf(ii), mkf(q)}}, nil
	}})

	m.Dict.SetStr("yiq_to_rgb", &object.BuiltinFunc{Name: "yiq_to_rgb", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		y, ii, q := getF(a[0]), getF(a[1]), getF(a[2])
		r := y + 0.9468822170900693*ii + 0.6235565819861433*q
		g := y - 0.27478764629897834*ii - 0.6356910791873801*q
		b := y - 1.1085450346420322*ii + 1.7090069284064666*q
		return &object.Tuple{V: []object.Object{mkf(clamp01(r)), mkf(clamp01(g)), mkf(clamp01(b))}}, nil
	}})

	m.Dict.SetStr("rgb_to_hls", &object.BuiltinFunc{Name: "rgb_to_hls", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		r, g, b := getF(a[0]), getF(a[1]), getF(a[2])
		maxc := math.Max(math.Max(r, g), b)
		minc := math.Min(math.Min(r, g), b)
		sumc := maxc + minc
		rangec := maxc - minc
		l := sumc / 2.0
		if minc == maxc {
			return &object.Tuple{V: []object.Object{mkf(0.0), mkf(l), mkf(0.0)}}, nil
		}
		var s float64
		if l <= 0.5 {
			s = rangec / sumc
		} else {
			s = rangec / (2.0 - sumc)
		}
		rc := (maxc - r) / rangec
		gc := (maxc - g) / rangec
		bc := (maxc - b) / rangec
		var h float64
		if r == maxc {
			h = bc - gc
		} else if g == maxc {
			h = 2.0 + rc - bc
		} else {
			h = 4.0 + gc - rc
		}
		h = fmod1(h / 6.0)
		return &object.Tuple{V: []object.Object{mkf(h), mkf(l), mkf(s)}}, nil
	}})

	m.Dict.SetStr("hls_to_rgb", &object.BuiltinFunc{Name: "hls_to_rgb", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		h, l, s := getF(a[0]), getF(a[1]), getF(a[2])
		if s == 0.0 {
			return &object.Tuple{V: []object.Object{mkf(l), mkf(l), mkf(l)}}, nil
		}
		var m2 float64
		if l <= 0.5 {
			m2 = l * (1.0 + s)
		} else {
			m2 = l + s - l*s
		}
		m1 := 2.0*l - m2
		return &object.Tuple{V: []object.Object{mkf(_v(m1, m2, h+oneThird)), mkf(_v(m1, m2, h)), mkf(_v(m1, m2, h-oneThird))}}, nil
	}})

	m.Dict.SetStr("rgb_to_hsv", &object.BuiltinFunc{Name: "rgb_to_hsv", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		r, g, b := getF(a[0]), getF(a[1]), getF(a[2])
		maxc := math.Max(math.Max(r, g), b)
		minc := math.Min(math.Min(r, g), b)
		v := maxc
		if minc == maxc {
			return &object.Tuple{V: []object.Object{mkf(0.0), mkf(0.0), mkf(v)}}, nil
		}
		s := (maxc - minc) / maxc
		rc := (maxc - r) / (maxc - minc)
		gc := (maxc - g) / (maxc - minc)
		bc := (maxc - b) / (maxc - minc)
		var h float64
		if r == maxc {
			h = bc - gc
		} else if g == maxc {
			h = 2.0 + rc - bc
		} else {
			h = 4.0 + gc - rc
		}
		h = fmod1(h / 6.0)
		return &object.Tuple{V: []object.Object{mkf(h), mkf(s), mkf(v)}}, nil
	}})

	m.Dict.SetStr("hsv_to_rgb", &object.BuiltinFunc{Name: "hsv_to_rgb", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		h, s, v := getF(a[0]), getF(a[1]), getF(a[2])
		if s == 0.0 {
			return &object.Tuple{V: []object.Object{mkf(v), mkf(v), mkf(v)}}, nil
		}
		idx := int(h * 6.0)
		f := h*6.0 - float64(idx)
		p := v * (1.0 - s)
		q := v * (1.0 - s*f)
		t := v * (1.0 - s*(1.0-f))
		idx = idx % 6
		var ro, go_, bo float64
		switch idx {
		case 0:
			ro, go_, bo = v, t, p
		case 1:
			ro, go_, bo = q, v, p
		case 2:
			ro, go_, bo = p, v, t
		case 3:
			ro, go_, bo = p, q, v
		case 4:
			ro, go_, bo = t, p, v
		default: // 5
			ro, go_, bo = v, p, q
		}
		return &object.Tuple{V: []object.Object{mkf(ro), mkf(go_), mkf(bo)}}, nil
	}})

	return m
}
