from zoneinfo import ZoneInfo, ZoneInfoNotFoundError, available_timezones, TZPATH, reset_tzpath
import datetime as dt_mod

# ===== TZPATH =====
print(type(TZPATH).__name__)          # tuple

# ===== available_timezones =====
avail = available_timezones()
print(type(avail).__name__)           # set
print("UTC" in avail)                 # True
print("America/New_York" in avail)    # True
print("Europe/London" in avail)       # True
print(len(avail) > 100)               # True

# ===== ZoneInfo construction =====
utc = ZoneInfo("UTC")
print(type(utc).__name__)             # ZoneInfo
print(utc.key)                        # UTC
print(str(utc))                       # UTC
print(repr(utc))                      # zoneinfo.ZoneInfo(key='UTC')

# ===== Caching =====
z1 = ZoneInfo("UTC")
z2 = ZoneInfo("UTC")
print(z1 is z2)                       # True

# ===== no_cache — never cached =====
z3 = ZoneInfo.no_cache("UTC")
z4 = ZoneInfo.no_cache("UTC")
print(z3 is z4)                       # False

# ===== clear_cache — full clear =====
ZoneInfo.clear_cache()
z5 = ZoneInfo("UTC")
z6 = ZoneInfo("UTC")
print(z5 is z6)                       # True (re-cached after clear)

# ===== ZoneInfoNotFoundError =====
try:
    ZoneInfo("Not/A/Real/Zone/xyz")
except ZoneInfoNotFoundError:
    print("ZoneInfoNotFoundError")
except Exception as e:
    print(type(e).__name__)

# ===== tzinfo methods — UTC is deterministic =====
d = dt_mod.datetime(2023, 6, 15, 12, 0, 0)
print(utc.utcoffset(d))               # 0:00:00
print(utc.tzname(d))                  # UTC
print(utc.dst(d))                     # 0:00:00

# ===== ZoneInfo as tzinfo in datetime =====
dt_aware = dt_mod.datetime(2023, 6, 15, 12, 0, 0, tzinfo=ZoneInfo("UTC"))
print(dt_aware.isoformat())           # 2023-06-15T12:00:00+00:00
print(dt_aware.tzname())              # UTC
print(dt_aware.utcoffset())           # 0:00:00

# ===== Non-UTC zones — key access only =====
ny = ZoneInfo("America/New_York")
print(ny.key)                         # America/New_York
print(type(ny).__name__)              # ZoneInfo

london = ZoneInfo("Europe/London")
print(london.key)                     # Europe/London
print(type(london).__name__)          # ZoneInfo

tokyo = ZoneInfo("Asia/Tokyo")
print(tokyo.key)                      # Asia/Tokyo

# ===== clear_cache with only_keys =====
ZoneInfo.clear_cache()
ny_a = ZoneInfo("America/New_York")
ZoneInfo.clear_cache(only_keys=["UTC"])   # only clear UTC; NY stays
ny_b = ZoneInfo("America/New_York")
print(ny_a is ny_b)                   # True (NY was not cleared)

# ===== reset_tzpath =====
reset_tzpath([])
print(type(TZPATH).__name__)          # tuple

# ===== ZoneInfoNotFoundError is subclass of KeyError =====
print(issubclass(ZoneInfoNotFoundError, KeyError))   # True

print('done')
