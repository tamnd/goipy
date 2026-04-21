import statistics
import calendar
import pprint
import html

# --- statistics ------------------------------------------------------------

# 1) mean of int list that divides evenly.
print(statistics.mean([2, 4, 6]))

# 2) mean of int list that doesn't divide evenly returns float.
print(statistics.mean([1, 2, 3, 4]))

# 3) mean of floats.
print(statistics.mean([1.5, 2.5, 3.5]))

# 4) fmean always returns float.
print(statistics.fmean([2, 4, 6]))

# 5) fmean of a range.
print(statistics.fmean(range(10)))

# 6) median odd-length int list returns int.
print(statistics.median([1, 3, 5]))

# 7) median even-length int list returns float.
print(statistics.median([1, 2, 3, 4]))

# 8) median_low picks lower middle.
print(statistics.median_low([1, 2, 3, 4]))

# 9) median_high picks upper middle.
print(statistics.median_high([1, 2, 3, 4]))

# 10) median of a single element.
print(statistics.median([7]))

# 11) mode picks the most frequent.
print(statistics.mode([1, 1, 2, 3, 3, 3]))

# 12) mode of strings.
print(statistics.mode(["a", "b", "a", "c"]))

# 13) multimode returns all ties.
print(statistics.multimode([1, 1, 2, 2, 3]))

# 14) multimode of empty returns [].
print(statistics.multimode([]))

# 15) pvariance of constant is 0.
print(statistics.pvariance([5, 5, 5]))

# 16) variance of two distinct values.
print(statistics.variance([1, 3]))

# 17) pstdev is sqrt(pvariance).
print(round(statistics.pstdev([1, 2, 3, 4, 5]), 4))

# 18) stdev is sqrt(variance).
print(round(statistics.stdev([1, 2, 3, 4, 5]), 4))

# 19) geometric_mean of powers of 2.
print(round(statistics.geometric_mean([1, 2, 4, 8, 16]), 4))

# 20) harmonic_mean.
print(round(statistics.harmonic_mean([1, 2, 4]), 4))

# 21) quantiles with default n=4.
print(statistics.quantiles([1, 2, 3, 4, 5, 6, 7, 8, 9]))

# 22) quantiles n=10.
print([round(q, 2) for q in statistics.quantiles(list(range(1, 11)), n=10)])

# 23) quantiles with method="inclusive".
print([round(q, 2) for q in statistics.quantiles([1, 2, 3, 4, 5], method="inclusive")])

# 24) mean on tuple input.
print(statistics.mean((10, 20, 30)))

# 25) mean with negative values.
print(statistics.mean([-1, 0, 1]))

# 26) pvariance of symmetric data.
print(statistics.pvariance([-2, -1, 0, 1, 2]))

# 27) stdev on large scale.
print(round(statistics.stdev([100, 200, 300]), 4))

# 28) mode prefers first-seen on tie.
print(statistics.mode([1, 2, 1, 2]))

# 29) multimode preserves order.
print(statistics.multimode(["b", "a", "b", "a"]))

# 30) mean of booleans (True=1, False=0).
print(statistics.mean([True, False, True, True]))

# --- calendar --------------------------------------------------------------

# 31) 2000 is a leap year (div by 400).
print(calendar.isleap(2000))

# 32) 1900 is not leap (div by 100 not 400).
print(calendar.isleap(1900))

# 33) 2024 is leap.
print(calendar.isleap(2024))

# 34) 2023 is not leap.
print(calendar.isleap(2023))

# 35) leapdays between two years.
print(calendar.leapdays(2000, 2025))

# 36) leapdays on empty range.
print(calendar.leapdays(2001, 2001))

# 37) weekday of 2024-01-01 (Monday).
print(calendar.weekday(2024, 1, 1))

# 38) weekday of 2020-02-29 (Saturday).
print(calendar.weekday(2020, 2, 29))

# 39) monthrange for Jan 2024.
mr = calendar.monthrange(2024, 1)
print(mr[0], mr[1])

# 40) monthrange for Feb 2024 (leap).
mr = calendar.monthrange(2024, 2)
print(mr[0], mr[1])

# 41) monthrange for Feb 2023 (non-leap).
mr = calendar.monthrange(2023, 2)
print(mr[0], mr[1])

# 42) month_name length is 13 (index 0 is empty).
print(len(calendar.month_name), calendar.month_name[0], calendar.month_name[12])

# 43) month_abbr covers 12 months.
print(calendar.month_abbr[1], calendar.month_abbr[12])

# 44) day_name is 7 long.
print(len(calendar.day_name), calendar.day_name[0], calendar.day_name[6])

# 45) day_abbr short names.
print(calendar.day_abbr[0], calendar.day_abbr[6])

# 46) MONDAY=0, SUNDAY=6.
print(calendar.MONDAY, calendar.TUESDAY, calendar.SUNDAY)

# 47) monthcalendar for Feb 2024 has 5 weeks.
weeks = calendar.monthcalendar(2024, 2)
print(len(weeks))

# 48) monthcalendar pads with zeros at the edges.
print(weeks[0][:3], weeks[-1][-3:])

# 49) monthcalendar for Feb 2023.
w2 = calendar.monthcalendar(2023, 2)
print(len(w2), sum(sum(row) for row in w2))

# 50) timegm at the epoch.
print(calendar.timegm((1970, 1, 1, 0, 0, 0, 0, 0, 0)))

# 51) timegm on 2024-01-01.
print(calendar.timegm((2024, 1, 1, 0, 0, 0)))

# 52) timegm round-trip via weekday.
ts = calendar.timegm((2024, 6, 15, 12, 0, 0))
print(ts)

# 53) isleap on year 0 (div by 400).
print(calendar.isleap(0))

# 54) weekday of 2000-01-01 (Saturday).
print(calendar.weekday(2000, 1, 1))

# --- pprint ----------------------------------------------------------------

# 55) simple dict sorts keys.
print(pprint.pformat({"b": 1, "a": 2}))

# 56) simple list fits on one line.
print(pprint.pformat([1, 2, 3]))

# 57) empty containers.
print(pprint.pformat([]))
print(pprint.pformat({}))
print(pprint.pformat(()))

# 58) single-element tuple has trailing comma.
print(pprint.pformat((1,)))

# 59) wide list breaks across lines.
out = pprint.pformat(list(range(10)), width=20)
print(out.count("\n") >= 9)

# 60) nested dict.
print(pprint.pformat({"a": [1, 2], "b": {"c": 3}}))

# 61) saferepr on a string.
print(pprint.saferepr("hi"))

# 62) saferepr on an int.
print(pprint.saferepr(42))

# 63) isreadable on primitives.
print(pprint.isreadable([1, 2, {"a": 1}]))

# 64) pprint writes to stdout with newline.
pprint.pprint({"x": 1})

# 65) pformat respects sort_dicts=False.
d = {}
d["b"] = 1
d["a"] = 2
print(pprint.pformat(d, sort_dicts=False))

# 66) pformat on a bool.
print(pprint.pformat(True))

# 67) pformat on None.
print(pprint.pformat(None))

# 68) pformat on bytes.
print(pprint.pformat(b"abc"))

# --- html ------------------------------------------------------------------

# 69) escape basic special chars.
print(html.escape("<p>hi</p>"))

# 70) escape with quote=False leaves quotes alone.
print(html.escape("a \" b ' c", quote=False))

# 71) escape & first so it doesn't double-escape.
print(html.escape("a & <b>"))

# 72) escape an empty string.
print(html.escape(""))

# 73) unescape named entities.
print(html.unescape("&lt;&gt;&amp;&quot;&apos;"))

# 74) unescape decimal numeric reference.
print(html.unescape("&#65;&#66;&#67;"))

# 75) unescape hex numeric reference.
print(html.unescape("&#x41;&#x42;&#x43;"))

# 76) unescape leaves unknown entities alone.
print(html.unescape("&unknown;"))

# 77) unescape mixed text.
print(html.unescape("foo &amp; bar &lt;baz&gt;"))

# 78) escape+unescape round-trips printable ASCII text.
s = "a < b & c > d"
print(html.unescape(html.escape(s)) == s)

# 79) escape handles non-ascii unchanged.
print(html.escape("café"))

# 80) unescape common typographic entities.
print(html.unescape("&hellip; &mdash; &copy;"))
