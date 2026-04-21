import statistics
import calendar
import pprint
import html

# statistics
print(statistics.mean([1, 2, 3, 4]))
print(statistics.fmean([1, 2, 3, 4]))
print(statistics.median([1, 3, 5]))
print(statistics.median([1, 3, 5, 7]))
print(statistics.median_low([1, 3, 5, 7]))
print(statistics.median_high([1, 3, 5, 7]))
print(statistics.mode([1, 1, 2, 3]))
print(statistics.multimode([1, 1, 2, 2, 3]))
print(round(statistics.pvariance([1, 2, 3, 4, 5]), 4))
print(round(statistics.variance([1, 2, 3, 4, 5]), 4))
print(round(statistics.pstdev([1, 2, 3, 4, 5]), 4))
print(round(statistics.stdev([1, 2, 3, 4, 5]), 4))
print(round(statistics.geometric_mean([1, 2, 4, 8]), 4))
print(round(statistics.harmonic_mean([1, 2, 4]), 4))
print(statistics.quantiles([1, 2, 3, 4, 5, 6, 7, 8, 9], n=4))

# calendar
print(calendar.isleap(2020), calendar.isleap(2021))
print(calendar.leapdays(2000, 2021))
print(calendar.weekday(2024, 1, 1))
mr = calendar.monthrange(2024, 2)
print(mr[0], mr[1])
print(calendar.month_name[1], calendar.month_abbr[1])
print(calendar.day_name[0], calendar.day_abbr[0])
print(calendar.MONDAY, calendar.SUNDAY)
weeks = calendar.monthcalendar(2024, 2)
print(len(weeks), weeks[0], weeks[-1])
print(calendar.timegm((2024, 1, 1, 0, 0, 0, 0, 0, 0)))

# pprint
data = {"b": 2, "a": 1, "c": [1, 2, 3]}
print(pprint.pformat(data))
print(pprint.pformat([1, 2, 3]))
print(pprint.pformat(list(range(20)), width=20))
print(pprint.saferepr("hello"))
print(pprint.isreadable({"a": 1}))

# html
print(html.escape("<a href='x'>&</a>"))
print(html.escape("quotes \" and '", quote=False))
print(html.unescape("&lt;tag&gt; &amp; &quot;text&quot;"))
print(html.unescape("&#65; &#x42; &copy; &hellip;"))
