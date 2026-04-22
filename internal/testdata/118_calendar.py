import calendar

# ===== Constants =====
print(calendar.MONDAY)     # 0
print(calendar.TUESDAY)    # 1
print(calendar.WEDNESDAY)  # 2
print(calendar.THURSDAY)   # 3
print(calendar.FRIDAY)     # 4
print(calendar.SATURDAY)   # 5
print(calendar.SUNDAY)     # 6

print(calendar.JANUARY)    # 1
print(calendar.FEBRUARY)   # 2
print(calendar.MARCH)      # 3
print(calendar.APRIL)      # 4
print(calendar.MAY)        # 5
print(calendar.JUNE)       # 6
print(calendar.JULY)       # 7
print(calendar.AUGUST)     # 8
print(calendar.SEPTEMBER)  # 9
print(calendar.OCTOBER)    # 10
print(calendar.NOVEMBER)   # 11
print(calendar.DECEMBER)   # 12

# ===== Day enum =====
print(calendar.Day.MONDAY.value)    # 0
print(calendar.Day.TUESDAY.value)   # 1
print(calendar.Day.SUNDAY.value)    # 6
print(calendar.Day.MONDAY.name)     # MONDAY

# ===== Month enum =====
print(calendar.Month.JANUARY.value)   # 1
print(calendar.Month.DECEMBER.value)  # 12
print(calendar.Month.JANUARY.name)    # JANUARY

# ===== firstweekday / setfirstweekday =====
print(calendar.firstweekday())  # 0
calendar.setfirstweekday(6)
print(calendar.firstweekday())  # 6
calendar.setfirstweekday(0)
print(calendar.firstweekday())  # 0

# ===== weekheader =====
print(calendar.weekheader(2))   # Mo Tu We Th Fr Sa Su
print(calendar.weekheader(3))   # Mon Tue Wed Thu Fri Sat Sun

# ===== isleap =====
print(calendar.isleap(2024))   # True
print(calendar.isleap(2023))   # False
print(calendar.isleap(2000))   # True
print(calendar.isleap(1900))   # False

# ===== leapdays =====
print(calendar.leapdays(2000, 2024))  # 6

# ===== weekday =====
print(calendar.weekday(2024, 1, 1))  # 0  (Monday)
print(calendar.weekday(2024, 1, 7))  # 6  (Sunday)
print(calendar.weekday(2024, 2, 29)) # 3  (Thursday)

# ===== monthrange — use int() to avoid enum repr =====
wd, nd = calendar.monthrange(2024, 1)
print(int(wd), nd)  # 0 31
wd, nd = calendar.monthrange(2024, 2)
print(int(wd), nd)  # 3 29
wd, nd = calendar.monthrange(2023, 2)
print(int(wd), nd)  # 2 28

# ===== monthcalendar =====
mc = calendar.monthcalendar(2024, 1)
print(mc[0])   # [1, 2, 3, 4, 5, 6, 7]
print(mc[-1])  # [29, 30, 31, 0, 0, 0, 0]
print(len(mc)) # 5

# ===== month =====
m_str = calendar.month(2024, 1)
lines = m_str.splitlines()
print(len(lines))              # 7
print(lines[0].strip())        # January 2024
print(lines[1])                # Mo Tu We Th Fr Sa Su
print(lines[2])                #  1  2  3  4  5  6  7

# ===== Calendar class =====
c = calendar.Calendar()
print(type(c).__name__)                          # Calendar
print(list(c.iterweekdays()))                    # [0, 1, 2, 3, 4, 5, 6]
print(list(c.itermonthdays(2024, 1))[:7])        # [1, 2, 3, 4, 5, 6, 7]

# itermonthdays2: first element is (day, weekday) - weekday may be int or Day enum
d2 = list(c.itermonthdays2(2024, 1))[0]
print(int(d2[0]), int(d2[1]))                    # 1 0

# itermonthdays3: first element is (year, month, day)
d3 = list(c.itermonthdays3(2024, 1))[0]
print(d3[0], d3[1], d3[2])                       # 2024 1 1

# itermonthdays4: first element is (year, month, day, weekday)
d4 = list(c.itermonthdays4(2024, 1))[0]
print(d4[0], d4[1], d4[2], int(d4[3]))           # 2024 1 1 0

# Calendar with Sunday as first weekday
c6 = calendar.Calendar(6)
print([int(x) for x in c6.iterweekdays()])  # [6, 0, 1, 2, 3, 4, 5]

# monthdayscalendar
mc2 = c.monthdayscalendar(2024, 1)
print(mc2[0])   # [1, 2, 3, 4, 5, 6, 7]
print(len(mc2)) # 5

# monthdays2calendar
md2 = c.monthdays2calendar(2024, 1)
t = md2[0][0]
print(int(t[0]), int(t[1]))  # 1 0

# yeardayscalendar
yc = c.yeardayscalendar(2024, width=3)
print(len(yc))     # 4

# yeardays2calendar
yc2 = c.yeardays2calendar(2024, width=3)
print(len(yc2))    # 4

# ===== TextCalendar =====
tc = calendar.TextCalendar()
print(type(tc).__name__)               # TextCalendar
fm = tc.formatmonth(2024, 1)
flines = fm.splitlines()
print(flines[0].strip())               # January 2024
print(flines[1])                       # Mo Tu We Th Fr Sa Su

wh = tc.formatweekheader(2)
print(wh)                              # Mo Tu We Th Fr Sa Su

fy = tc.formatyear(2024)
print(fy.splitlines()[0].strip())      # 2024

# ===== HTMLCalendar =====
hc = calendar.HTMLCalendar()
print(type(hc).__name__)               # HTMLCalendar
hm = hc.formatmonth(2024, 1)
print(hm[:6])                          # <table
print('<th' in hm)                     # True

hy = hc.formatyear(2024)
print(hy[:6])                          # <table

# ===== IllegalMonthError =====
try:
    calendar.monthrange(2024, 13)
except ValueError:
    print("IllegalMonthError caught")  # IllegalMonthError caught

# ===== IllegalWeekdayError =====
try:
    calendar.setfirstweekday(7)
except ValueError:
    print("IllegalWeekdayError caught") # IllegalWeekdayError caught

# ===== LocaleTextCalendar =====
ltc = calendar.LocaleTextCalendar()
print(type(ltc).__name__)              # LocaleTextCalendar

# ===== LocaleHTMLCalendar =====
lhc = calendar.LocaleHTMLCalendar()
print(type(lhc).__name__)              # LocaleHTMLCalendar

print('done')
