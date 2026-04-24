"""Tests for the extended time module."""
import time

# --- struct_time construction ---
st = time.gmtime(0)
print(type(st).__name__)     # struct_time
print(st.tm_year)            # 1970
print(st.tm_mon)             # 1
print(st.tm_mday)            # 1
print(st.tm_hour)            # 0
print(st.tm_min)             # 0
print(st.tm_sec)             # 0
print(st.tm_wday)            # 3  (Thursday)
print(st.tm_yday)            # 1
print(st.tm_isdst)           # 0

# --- index access ---
print(st[0])   # 1970
print(st[1])   # 1
print(st[2])   # 1
print(st[3])   # 0
print(st[4])   # 0
print(st[5])   # 0
print(st[6])   # 3
print(st[7])   # 1
print(st[8])   # 0

# --- negative index ---
print(st[-1])  # 0 (tm_isdst)
print(st[-9])  # 1970 (tm_year)

# --- gmtime with custom timestamp ---
st2 = time.gmtime(86400)   # 1970-01-02
print(st2.tm_year)   # 1970
print(st2.tm_mday)   # 2
print(st2.tm_yday)   # 2

# --- localtime returns struct_time ---
lt = time.localtime()
print(type(lt).__name__)      # struct_time
print(lt.tm_year > 2000)      # True

# --- mktime round-trip ---
epoch_st = time.gmtime(0)
# mktime uses local time, so round-trip through localtime instead
lt_now = time.localtime()
ts = time.mktime(lt_now)
print(abs(ts - time.time()) < 5)   # True

# --- asctime ---
print(time.asctime(time.gmtime(0)))  # Thu Jan  1 00:00:00 1970

# --- ctime ---
# ctime uses local time so just check it's a string
c = time.ctime(0)
print(isinstance(c, str))   # True
print(len(c) > 0)           # True

# --- strftime ---
print(time.strftime("%Y-%m-%d", time.gmtime(0)))        # 1970-01-01
print(time.strftime("%H:%M:%S", time.gmtime(0)))        # 00:00:00
print(time.strftime("%A", time.gmtime(0)))              # Thursday
print(time.strftime("%a", time.gmtime(0)))              # Thu
print(time.strftime("%B", time.gmtime(0)))              # January
print(time.strftime("%b", time.gmtime(0)))              # Jan
print(time.strftime("%%", time.gmtime(0)))              # %
print(time.strftime("%j", time.gmtime(0)))              # 001
print(time.strftime("%d", time.gmtime(86400)))          # 02
print(time.strftime("%m", time.gmtime(86400*31)))       # 02

# --- strptime ---
parsed = time.strptime("2024-01-15", "%Y-%m-%d")
print(parsed.tm_year)   # 2024
print(parsed.tm_mon)    # 1
print(parsed.tm_mday)   # 15

parsed2 = time.strptime("12:30:45", "%H:%M:%S")
print(parsed2.tm_hour)  # 12
print(parsed2.tm_min)   # 30
print(parsed2.tm_sec)   # 45

# --- timezone / altzone / daylight / tzname ---
print(isinstance(time.timezone, int))   # True
print(isinstance(time.altzone, int))    # True
print(time.daylight in (0, 1))          # True
print(isinstance(time.tzname, tuple))   # True
print(len(time.tzname) == 2)            # True
print(isinstance(time.tzname[0], str))  # True
print(isinstance(time.tzname[1], str))  # True

# --- process_time ---
pt = time.process_time()
print(isinstance(pt, float))  # True
print(pt >= 0)                # True

pt_ns = time.process_time_ns()
print(isinstance(pt_ns, int))  # True
print(pt_ns >= 0)              # True

# --- thread_time ---
tt = time.thread_time()
print(isinstance(tt, float))  # True
print(tt >= 0)                # True

tt_ns = time.thread_time_ns()
print(isinstance(tt_ns, int))  # True
print(tt_ns >= 0)              # True

# --- get_clock_info ---
info = time.get_clock_info("time")
print(isinstance(info.implementation, str))   # True
print(isinstance(info.monotonic, bool))       # True
print(isinstance(info.adjustable, bool))      # True
print(isinstance(info.resolution, float))     # True

info2 = time.get_clock_info("monotonic")
print(info2.monotonic)    # True

# --- CLOCK constants ---
print(time.CLOCK_REALTIME == 0)          # True
print(time.CLOCK_MONOTONIC == 1)         # True
print(time.CLOCK_PROCESS_CPUTIME_ID == 2) # True
print(time.CLOCK_THREAD_CPUTIME_ID == 3)  # True
