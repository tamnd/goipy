from datetime import (date, time, datetime, timedelta, timezone, tzinfo,
                      MINYEAR, MAXYEAR)
import datetime as dt_mod

# --- module constants ---
print(MINYEAR)
print(MAXYEAR)
print(type(dt_mod.UTC).__name__)

# ===== timedelta =====

# constructor & normalization
td = timedelta(days=1, seconds=3600, microseconds=500, milliseconds=100,
               minutes=5, hours=2, weeks=1)
print(td.days)
print(td.seconds)
print(td.microseconds)
print(td.total_seconds())

# overflow normalization
td2 = timedelta(seconds=90061, microseconds=1000001)
print(td2.days)
print(td2.seconds)
print(td2.microseconds)

# negative normalization
td3 = timedelta(days=-1, seconds=3600)
print(td3.days, td3.seconds)

# arithmetic
print(timedelta(days=1) + timedelta(hours=12))
print(timedelta(days=2) - timedelta(hours=6))
print(timedelta(hours=3) * 4)
print(3 * timedelta(hours=3))
print(timedelta(days=1) / 2)
print(timedelta(days=7) // timedelta(days=3))
print(timedelta(days=7) % timedelta(days=3))
print(-timedelta(days=1, seconds=30))
print(+timedelta(days=2))
print(abs(timedelta(days=-3, seconds=7200)))

# comparisons
print(timedelta(days=1) == timedelta(hours=24))
print(timedelta(days=1) != timedelta(hours=23))
print(timedelta(days=1) < timedelta(days=2))
print(timedelta(days=2) > timedelta(days=1))
print(timedelta(days=1) <= timedelta(hours=24))
print(timedelta(days=1) >= timedelta(hours=24))

# bool
print(bool(timedelta(0)))
print(bool(timedelta(days=1)))

# class attrs
print(timedelta.min.days)
print(timedelta.max.days)
print(timedelta.resolution)

# str/repr
print(str(timedelta(days=1)))
print(str(timedelta(days=1, seconds=3600, microseconds=500)))
print(str(timedelta(days=-1)))
print(str(timedelta(days=0, seconds=0)))

# ===== date =====

d = date(2023, 6, 15)
print(d.year, d.month, d.day)
print(d.isoformat())
print(str(d))
print(repr(d))
print(d.weekday())
print(d.isoweekday())
print(d.toordinal())
print(date.fromordinal(d.toordinal()))
ic = d.isocalendar()
print(ic.year, ic.week, ic.weekday)
print(d.timetuple()[:6])
print(d.ctime())
print(d.strftime('%Y/%m/%d'))
print(d.replace(year=2024, month=1))
print(d + timedelta(days=10))
print(d - timedelta(days=5))
diff = d - date(2023, 1, 1)
print(diff.days)
print(date.fromisoformat('2023-06-15'))
print(date.fromisocalendar(2023, 24, 4))
print(date.fromtimestamp(0))

# date comparisons
print(date(2023, 1, 1) < date(2023, 6, 15))
print(date(2023, 6, 15) == date(2023, 6, 15))
print(date(2023, 6, 15) != date(2023, 6, 14))
print(date(2023, 12, 31) > date(2023, 6, 15))

# class attrs
print(date.min)
print(date.max)
print(date.resolution)

# ===== time =====

t = time(14, 30, 45, 123456)
print(t.hour, t.minute, t.second, t.microsecond)
print(t.fold)
print(t.tzinfo)
print(t.isoformat())
print(t.isoformat(timespec='hours'))
print(t.isoformat(timespec='minutes'))
print(t.isoformat(timespec='seconds'))
print(t.isoformat(timespec='milliseconds'))
print(t.isoformat(timespec='microseconds'))
print(t.strftime('%H:%M:%S'))
print(t.replace(hour=15, microsecond=0))
print(t.utcoffset())
print(t.dst())
print(t.tzname())
print(time.fromisoformat('14:30:45'))
print(time.fromisoformat('14:30:45.123456'))
print(time.fromisoformat('14:30'))

# time with timezone
tz_utc = timezone.utc
t_tz = time(12, 0, 0, tzinfo=tz_utc)
print(t_tz.isoformat())
print(t_tz.utcoffset())
print(t_tz.tzname())

# comparisons
print(time(10, 0) < time(12, 0))
print(time(12, 0) == time(12, 0))
print(time(12, 0) != time(12, 1))
print(time(15, 0) > time(12, 0))
print(bool(time(0, 0, 0)))
print(bool(time(1, 0, 0)))

print(time.min)
print(time.max)
print(time.resolution)

# ===== timezone =====

utc = timezone.utc
print(utc.utcoffset(None))
print(utc.tzname(None))
print(utc.dst(None))

tz_pos = timezone(timedelta(hours=5, minutes=30))
print(tz_pos.utcoffset(None))
print(tz_pos.tzname(None))

tz_neg = timezone(timedelta(hours=-5), 'EST')
print(tz_neg.utcoffset(None))
print(tz_neg.tzname(None))
print(tz_neg.dst(None))

print(timezone.min.utcoffset(None))
print(timezone.max.utcoffset(None))

# ===== datetime =====

dt = datetime(2023, 6, 15, 14, 30, 45, 123456)
print(dt.year, dt.month, dt.day)
print(dt.hour, dt.minute, dt.second, dt.microsecond)
print(dt.fold)
print(dt.tzinfo)

# date/time extraction
print(dt.date())
print(dt.time())

# isoformat
print(dt.isoformat())
print(dt.isoformat(sep=' '))
print(dt.isoformat(sep='T', timespec='seconds'))
print(dt.isoformat(timespec='milliseconds'))
print(str(dt))

# ctime
print(dt.ctime())

# weekday
print(dt.weekday())
print(dt.isoweekday())

# isocalendar
ic2 = dt.isocalendar()
print(ic2.year, ic2.week, ic2.weekday)

# timetuple
print(dt.timetuple()[:6])

# toordinal
print(dt.toordinal())
print(datetime.fromordinal(dt.toordinal()).date())

# strftime
print(dt.strftime('%Y-%m-%d %H:%M:%S'))
print(dt.strftime('%A %B %d %Y'))

# replace
print(dt.replace(year=2024, month=1, day=1, hour=0, minute=0, second=0, microsecond=0))

# arithmetic
dt2 = dt + timedelta(days=1, hours=6)
print(dt2.date())
dt3 = dt - timedelta(days=10)
print(dt3.date())
diff2 = dt - datetime(2023, 1, 1)
print(diff2.days)

# combine
d3 = date(2023, 11, 1)
t3 = time(8, 30, 0)
print(datetime.combine(d3, t3))

# fromisoformat
print(datetime.fromisoformat('2023-06-15T14:30:45'))
print(datetime.fromisoformat('2023-06-15 14:30:45.123456'))

# strptime
parsed = datetime.strptime('2023-06-15 14:30:45', '%Y-%m-%d %H:%M:%S')
print(parsed)
parsed2 = datetime.strptime('15/06/2023', '%d/%m/%Y')
print(parsed2.date())

# fromtimestamp with UTC tz
dt_utc = datetime.fromtimestamp(0, tz=timezone.utc)
print(dt_utc.year, dt_utc.month, dt_utc.day)
print(dt_utc.hour, dt_utc.minute, dt_utc.second)
print(dt_utc.isoformat())

# utcfromtimestamp
dt_utcf = datetime.utcfromtimestamp(86400)
print(dt_utcf.year, dt_utcf.month, dt_utcf.day)
print(dt_utcf.hour)

# datetime with timezone
dt_tz = datetime(2023, 6, 15, 12, 0, 0, tzinfo=timezone.utc)
print(dt_tz.isoformat())
print(dt_tz.utcoffset())
print(dt_tz.tzname())
print(dt_tz.dst())
print(dt_tz.timestamp())
print(dt_tz.utctimetuple()[:6])
print(dt_tz.timetz())

# fromisocalendar
print(datetime.fromisocalendar(2023, 24, 4))

# today/now return datetime (check type)
print(type(datetime.today()).__name__)
print(type(datetime.now()).__name__)
print(type(datetime.utcnow()).__name__)

# comparisons
print(datetime(2023, 1, 1) < datetime(2023, 6, 15))
print(datetime(2023, 6, 15) == datetime(2023, 6, 15))
print(datetime(2023, 6, 15) != datetime(2023, 6, 14))
print(datetime(2023, 12, 31) > datetime(2023, 6, 15))

print(datetime.min)
print(datetime.max)
print(datetime.resolution)

print('done')
