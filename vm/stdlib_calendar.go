package vm

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/tamnd/goipy/object"
)

// calFWD is the global first weekday (0=Monday … 6=Sunday).
var calFWD = 0

// calDayOffset returns how many padding days precede the 1st of the month in
// a calendar whose leftmost column is weekday fwd (Mon=0).
func calDayOffset(year, month, fwd int) int {
	first := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	wd := (int(first.Weekday()) + 6) % 7 // Go Sun=0 → Mon=0
	return (wd - fwd + 7) % 7
}

// calIterDays4 returns every (year, month, day, weekday) cell that appears in
// the calendar view of the given month, including adjacent-month padding.
func calIterDays4(year, month, fwd int) [][4]int {
	first := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	offset := calDayOffset(year, month, fwd)
	start := first.AddDate(0, 0, -offset)

	last := time.Date(year, time.Month(month+1), 0, 0, 0, 0, 0, time.UTC)
	lastWd := (int(last.Weekday()) + 6) % 7
	endPad := (fwd + 6 - lastWd + 7) % 7
	end := last.AddDate(0, 0, endPad)

	var result [][4]int
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		dwd := (int(d.Weekday()) + 6) % 7
		result = append(result, [4]int{d.Year(), int(d.Month()), d.Day(), dwd})
	}
	return result
}

// calMonthCalFWD generates a month-calendar matrix respecting fwd.
// Each row is a 7-element slice; out-of-month days are 0.
func calMonthCalFWD(year, month, fwd int) [][]int {
	days4 := calIterDays4(year, month, fwd)
	var weeks [][]int
	row := make([]int, 0, 7)
	for _, d := range days4 {
		day := 0
		if d[1] == month {
			day = d[2]
		}
		row = append(row, day)
		if len(row) == 7 {
			weeks = append(weeks, row)
			row = make([]int, 0, 7)
		}
	}
	return weeks
}

func calCenterStr(s string, width int) string {
	if len(s) >= width {
		return s
	}
	total := width - len(s)
	left := total / 2
	right := total - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}

func calWeekHeader(n, fwd int) string {
	parts := make([]string, 7)
	for k := 0; k < 7; k++ {
		wd := (fwd + k) % 7
		name := calDayAbbr[wd]
		if n < len(name) {
			name = name[:n]
		}
		for len(name) < n {
			name += " "
		}
		parts[k] = name
	}
	return strings.Join(parts, " ")
}

func calFormatDay(day, w int) string {
	if day == 0 {
		return strings.Repeat(" ", w)
	}
	s := fmt.Sprintf("%d", day)
	for len(s) < w {
		s = " " + s
	}
	return s
}

func calFormatWeekRow(week []int, w int) string {
	parts := make([]string, len(week))
	for j, d := range week {
		parts[j] = calFormatDay(d, w)
	}
	return strings.Join(parts, " ")
}

func calFormatMonth(year, month, w, l, fwd int) string {
	if w < 2 {
		w = 2
	}
	if l < 1 {
		l = 1
	}
	colWidth := 7*w + 6
	header := calCenterStr(fmt.Sprintf("%s %d", calMonthName[month], year), colWidth)
	wh := calWeekHeader(w, fwd)
	sep := strings.Repeat("\n", l)
	weeks := calMonthCalFWD(year, month, fwd)
	lines := []string{header, wh}
	for _, week := range weeks {
		lines = append(lines, calFormatWeekRow(week, w))
	}
	return strings.Join(lines, sep) + "\n"
}

func calFormatYear(year, w, l, c, m, fwd int) string {
	if w < 2 {
		w = 2
	}
	if l < 1 {
		l = 1
	}
	if c < 1 {
		c = 1
	}
	if m < 1 {
		m = 1
	}
	colWidth := 7*w + 6
	gap := strings.Repeat(" ", c)
	sep := strings.Repeat("\n", l)

	blocks := make([][]string, 12)
	for mo := 1; mo <= 12; mo++ {
		weeks := calMonthCalFWD(year, mo, fwd)
		hdr := calCenterStr(calMonthName[mo], colWidth)
		wh := calWeekHeader(w, fwd)
		var lines []string
		lines = append(lines, hdr)
		lines = append(lines, wh)
		for _, week := range weeks {
			lines = append(lines, calFormatWeekRow(week, w))
		}
		blocks[mo-1] = lines
	}

	var sb strings.Builder
	yearW := m*colWidth + (m-1)*c
	sb.WriteString(calCenterStr(fmt.Sprintf("%d", year), yearW))
	sb.WriteString("\n\n")

	for row := 0; row < 12; row += m {
		maxH := 0
		for k := row; k < row+m && k < 12; k++ {
			if len(blocks[k]) > maxH {
				maxH = len(blocks[k])
			}
		}
		for k := row; k < row+m && k < 12; k++ {
			for len(blocks[k]) < maxH {
				blocks[k] = append(blocks[k], strings.Repeat(" ", colWidth))
			}
		}
		for line := 0; line < maxH; line++ {
			var parts []string
			for k := row; k < row+m && k < 12; k++ {
				parts = append(parts, blocks[k][line])
			}
			sb.WriteString(strings.Join(parts, gap))
			sb.WriteString(sep)
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

var calHTMLWdNames = []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"}

func calHTMLDay(day, wd int) string {
	if day == 0 {
		return `<td class="noday">&nbsp;</td>`
	}
	return fmt.Sprintf(`<td class="%s">%d</td>`, calHTMLWdNames[wd], day)
}

func calHTMLFormatMonth(year, month, fwd int, withyear bool) string {
	var sb strings.Builder
	sb.WriteString(`<table border="0" cellpadding="0" cellspacing="0" class="month">` + "\n")
	sb.WriteString(`<tr><th colspan="7" class="month">`)
	if withyear {
		sb.WriteString(fmt.Sprintf("%s %d", calMonthName[month], year))
	} else {
		sb.WriteString(calMonthName[month])
	}
	sb.WriteString("</th></tr>\n<tr>")
	for k := 0; k < 7; k++ {
		wd := (fwd + k) % 7
		cls := "weekday"
		if wd >= 5 {
			cls = "weekend"
		}
		sb.WriteString(fmt.Sprintf(`<th class="%s">%s</th>`, cls, calDayAbbr[wd]))
	}
	sb.WriteString("</tr>\n")
	for _, week := range calMonthCalFWD(year, month, fwd) {
		sb.WriteString("<tr>")
		for j, day := range week {
			wd := (fwd + j) % 7
			sb.WriteString(calHTMLDay(day, wd))
		}
		sb.WriteString("</tr>\n")
	}
	sb.WriteString("</table>\n")
	return sb.String()
}

func calHTMLFormatYear(year, width, fwd int) string {
	var sb strings.Builder
	sb.WriteString(`<table border="0" cellpadding="0" cellspacing="0" class="year">` + "\n")
	sb.WriteString(fmt.Sprintf(`<tr><th colspan="%d" class="year">%d</th></tr>`, width, year) + "\n")
	for row := 1; row <= 12; row += width {
		sb.WriteString("<tr>")
		for k := row; k < row+width && k <= 12; k++ {
			sb.WriteString("<td>")
			sb.WriteString(calHTMLFormatMonth(year, k, fwd, false))
			sb.WriteString("</td>")
		}
		sb.WriteString("</tr>\n")
	}
	sb.WriteString("</table>\n")
	return sb.String()
}

// fillCalendarInst populates the Calendar instance methods in inst.Dict.
func (i *Interp) fillCalendarInst(inst *object.Instance, fwd int) {
	inst.Dict.SetStr("firstweekday", object.IntFromBig(big.NewInt(int64(fwd))))

	inst.Dict.SetStr("iterweekdays", &object.BuiltinFunc{Name: "iterweekdays", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= 7 {
				return nil, false, nil
			}
			wd := (fwd + idx) % 7
			idx++
			return object.IntFromBig(big.NewInt(int64(wd))), true, nil
		}}, nil
	}})

	inst.Dict.SetStr("itermonthdays", &object.BuiltinFunc{Name: "itermonthdays", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "itermonthdays() requires year, month")
		}
		y, _ := toInt64(a[0])
		mo, _ := toInt64(a[1])
		days4 := calIterDays4(int(y), int(mo), fwd)
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(days4) {
				return nil, false, nil
			}
			d := days4[idx]
			idx++
			day := 0
			if d[1] == int(mo) {
				day = d[2]
			}
			return object.IntFromBig(big.NewInt(int64(day))), true, nil
		}}, nil
	}})

	inst.Dict.SetStr("itermonthdays2", &object.BuiltinFunc{Name: "itermonthdays2", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "itermonthdays2() requires year, month")
		}
		y, _ := toInt64(a[0])
		mo, _ := toInt64(a[1])
		days4 := calIterDays4(int(y), int(mo), fwd)
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(days4) {
				return nil, false, nil
			}
			d := days4[idx]
			idx++
			day := 0
			if d[1] == int(mo) {
				day = d[2]
			}
			return &object.Tuple{V: []object.Object{
				object.IntFromBig(big.NewInt(int64(day))),
				object.IntFromBig(big.NewInt(int64(d[3]))),
			}}, true, nil
		}}, nil
	}})

	inst.Dict.SetStr("itermonthdays3", &object.BuiltinFunc{Name: "itermonthdays3", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "itermonthdays3() requires year, month")
		}
		y, _ := toInt64(a[0])
		mo, _ := toInt64(a[1])
		days4 := calIterDays4(int(y), int(mo), fwd)
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(days4) {
				return nil, false, nil
			}
			d := days4[idx]
			idx++
			return &object.Tuple{V: []object.Object{
				object.IntFromBig(big.NewInt(int64(d[0]))),
				object.IntFromBig(big.NewInt(int64(d[1]))),
				object.IntFromBig(big.NewInt(int64(d[2]))),
			}}, true, nil
		}}, nil
	}})

	inst.Dict.SetStr("itermonthdays4", &object.BuiltinFunc{Name: "itermonthdays4", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "itermonthdays4() requires year, month")
		}
		y, _ := toInt64(a[0])
		mo, _ := toInt64(a[1])
		days4 := calIterDays4(int(y), int(mo), fwd)
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(days4) {
				return nil, false, nil
			}
			d := days4[idx]
			idx++
			return &object.Tuple{V: []object.Object{
				object.IntFromBig(big.NewInt(int64(d[0]))),
				object.IntFromBig(big.NewInt(int64(d[1]))),
				object.IntFromBig(big.NewInt(int64(d[2]))),
				object.IntFromBig(big.NewInt(int64(d[3]))),
			}}, true, nil
		}}, nil
	}})

	inst.Dict.SetStr("monthdayscalendar", &object.BuiltinFunc{Name: "monthdayscalendar", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "monthdayscalendar() requires year, month")
		}
		y, _ := toInt64(a[0])
		mo, _ := toInt64(a[1])
		weeks := calMonthCalFWD(int(y), int(mo), fwd)
		out := make([]object.Object, len(weeks))
		for k, week := range weeks {
			row := make([]object.Object, len(week))
			for j, d := range week {
				row[j] = object.IntFromBig(big.NewInt(int64(d)))
			}
			out[k] = &object.List{V: row}
		}
		return &object.List{V: out}, nil
	}})

	inst.Dict.SetStr("monthdays2calendar", &object.BuiltinFunc{Name: "monthdays2calendar", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "monthdays2calendar() requires year, month")
		}
		y, _ := toInt64(a[0])
		mo, _ := toInt64(a[1])
		days4 := calIterDays4(int(y), int(mo), fwd)
		var weeks []object.Object
		row := []object.Object{}
		for _, d := range days4 {
			day := 0
			if d[1] == int(mo) {
				day = d[2]
			}
			tup := &object.Tuple{V: []object.Object{
				object.IntFromBig(big.NewInt(int64(day))),
				object.IntFromBig(big.NewInt(int64(d[3]))),
			}}
			row = append(row, tup)
			if len(row) == 7 {
				weeks = append(weeks, &object.List{V: row})
				row = []object.Object{}
			}
		}
		return &object.List{V: weeks}, nil
	}})

	inst.Dict.SetStr("yeardayscalendar", &object.BuiltinFunc{Name: "yeardayscalendar", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "yeardayscalendar() requires year")
		}
		y, _ := toInt64(a[0])
		width := 3
		if len(a) > 1 {
			if v, ok := toInt64(a[1]); ok {
				width = int(v)
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("width"); ok {
				if n, ok2 := toInt64(v); ok2 {
					width = int(n)
				}
			}
		}
		var rows []object.Object
		for mo := 1; mo <= 12; mo += width {
			var rowMons []object.Object
			for k := mo; k < mo+width && k <= 12; k++ {
				weeks := calMonthCalFWD(int(y), k, fwd)
				var wObjs []object.Object
				for _, week := range weeks {
					var dayObjs []object.Object
					for _, d := range week {
						dayObjs = append(dayObjs, object.IntFromBig(big.NewInt(int64(d))))
					}
					wObjs = append(wObjs, &object.List{V: dayObjs})
				}
				rowMons = append(rowMons, &object.List{V: wObjs})
			}
			rows = append(rows, &object.List{V: rowMons})
		}
		return &object.List{V: rows}, nil
	}})

	inst.Dict.SetStr("yeardays2calendar", &object.BuiltinFunc{Name: "yeardays2calendar", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "yeardays2calendar() requires year")
		}
		y, _ := toInt64(a[0])
		width := 3
		if len(a) > 1 {
			if v, ok := toInt64(a[1]); ok {
				width = int(v)
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("width"); ok {
				if n, ok2 := toInt64(v); ok2 {
					width = int(n)
				}
			}
		}
		var rows []object.Object
		for mo := 1; mo <= 12; mo += width {
			var rowMons []object.Object
			for k := mo; k < mo+width && k <= 12; k++ {
				days4 := calIterDays4(int(y), k, fwd)
				var wObjs []object.Object
				row := []object.Object{}
				for _, d := range days4 {
					day := 0
					if d[1] == k {
						day = d[2]
					}
					tup := &object.Tuple{V: []object.Object{
						object.IntFromBig(big.NewInt(int64(day))),
						object.IntFromBig(big.NewInt(int64(d[3]))),
					}}
					row = append(row, tup)
					if len(row) == 7 {
						wObjs = append(wObjs, &object.List{V: row})
						row = []object.Object{}
					}
				}
				rowMons = append(rowMons, &object.List{V: wObjs})
			}
			rows = append(rows, &object.List{V: rowMons})
		}
		return &object.List{V: rows}, nil
	}})
}

// fillTextCalendarInst adds TextCalendar-specific methods to inst.Dict.
func (i *Interp) fillTextCalendarInst(inst *object.Instance, fwd int) {
	inst.Dict.SetStr("formatweekday", &object.BuiltinFunc{Name: "formatweekday", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "formatweekday() requires day, width")
		}
		day, _ := toInt64(a[0])
		width, _ := toInt64(a[1])
		name := calDayAbbr[int(day)%7]
		if int(width) < len(name) {
			name = name[:width]
		}
		for len(name) < int(width) {
			name += " "
		}
		return &object.Str{V: name}, nil
	}})

	inst.Dict.SetStr("formatweekheader", &object.BuiltinFunc{Name: "formatweekheader", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		w := 2
		if len(a) > 0 {
			if v, ok := toInt64(a[0]); ok {
				w = int(v)
			}
		}
		return &object.Str{V: calWeekHeader(w, fwd)}, nil
	}})

	inst.Dict.SetStr("formatday", &object.BuiltinFunc{Name: "formatday", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "formatday() requires day, weekday[, width]")
		}
		day, _ := toInt64(a[0])
		w := 2
		if len(a) > 2 {
			if v, ok := toInt64(a[2]); ok {
				w = int(v)
			}
		}
		return &object.Str{V: calFormatDay(int(day), w)}, nil
	}})

	inst.Dict.SetStr("formatweek", &object.BuiltinFunc{Name: "formatweek", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "formatweek() requires theweek")
		}
		w := 2
		if len(a) > 1 {
			if v, ok := toInt64(a[1]); ok {
				w = int(v)
			}
		}
		var days []int
		switch v := a[0].(type) {
		case *object.List:
			for _, item := range v.V {
				if tup, ok := item.(*object.Tuple); ok && len(tup.V) >= 1 {
					d, _ := toInt64(tup.V[0])
					days = append(days, int(d))
				} else if n, ok := toInt64(item); ok {
					days = append(days, int(n))
				}
			}
		}
		return &object.Str{V: calFormatWeekRow(days, w)}, nil
	}})

	inst.Dict.SetStr("formatmonthname", &object.BuiltinFunc{Name: "formatmonthname", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "formatmonthname() requires year, month")
		}
		y, _ := toInt64(a[0])
		mo, _ := toInt64(a[1])
		colWidth := 0
		withyear := true
		if len(a) > 2 {
			if v, ok := toInt64(a[2]); ok {
				colWidth = int(v)
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("width"); ok {
				if n, ok2 := toInt64(v); ok2 {
					colWidth = int(n)
				}
			}
			if v, ok := kw.GetStr("withyear"); ok {
				withyear = object.Truthy(v)
			}
		}
		var name string
		if withyear {
			name = fmt.Sprintf("%s %d", calMonthName[mo], y)
		} else {
			name = calMonthName[mo]
		}
		if colWidth == 0 {
			colWidth = 7*2 + 6
		}
		return &object.Str{V: calCenterStr(name, colWidth)}, nil
	}})

	inst.Dict.SetStr("formatmonth", &object.BuiltinFunc{Name: "formatmonth", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "formatmonth() requires year, month")
		}
		y, _ := toInt64(a[0])
		mo, _ := toInt64(a[1])
		w, l := 0, 0
		if len(a) > 2 {
			if v, ok := toInt64(a[2]); ok {
				w = int(v)
			}
		}
		if len(a) > 3 {
			if v, ok := toInt64(a[3]); ok {
				l = int(v)
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("w"); ok {
				if n, ok2 := toInt64(v); ok2 {
					w = int(n)
				}
			}
			if v, ok := kw.GetStr("l"); ok {
				if n, ok2 := toInt64(v); ok2 {
					l = int(n)
				}
			}
		}
		return &object.Str{V: calFormatMonth(int(y), int(mo), w, l, fwd)}, nil
	}})

	inst.Dict.SetStr("prmonth", &object.BuiltinFunc{Name: "prmonth", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "prmonth() requires year, month")
		}
		y, _ := toInt64(a[0])
		mo, _ := toInt64(a[1])
		w, l := 0, 0
		if len(a) > 2 {
			if v, ok := toInt64(a[2]); ok {
				w = int(v)
			}
		}
		if len(a) > 3 {
			if v, ok := toInt64(a[3]); ok {
				l = int(v)
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("w"); ok {
				if n, ok2 := toInt64(v); ok2 {
					w = int(n)
				}
			}
			if v, ok := kw.GetStr("l"); ok {
				if n, ok2 := toInt64(v); ok2 {
					l = int(n)
				}
			}
		}
		fmt.Print(calFormatMonth(int(y), int(mo), w, l, fwd))
		return object.None, nil
	}})

	inst.Dict.SetStr("formatyear", &object.BuiltinFunc{Name: "formatyear", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "formatyear() requires year")
		}
		y, _ := toInt64(a[0])
		w, l, c, m := 2, 1, 6, 3
		if len(a) > 1 {
			if v, ok := toInt64(a[1]); ok {
				w = int(v)
			}
		}
		if len(a) > 2 {
			if v, ok := toInt64(a[2]); ok {
				l = int(v)
			}
		}
		if len(a) > 3 {
			if v, ok := toInt64(a[3]); ok {
				c = int(v)
			}
		}
		if len(a) > 4 {
			if v, ok := toInt64(a[4]); ok {
				m = int(v)
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("w"); ok {
				if n, ok2 := toInt64(v); ok2 {
					w = int(n)
				}
			}
			if v, ok := kw.GetStr("l"); ok {
				if n, ok2 := toInt64(v); ok2 {
					l = int(n)
				}
			}
			if v, ok := kw.GetStr("c"); ok {
				if n, ok2 := toInt64(v); ok2 {
					c = int(n)
				}
			}
			if v, ok := kw.GetStr("m"); ok {
				if n, ok2 := toInt64(v); ok2 {
					m = int(n)
				}
			}
		}
		return &object.Str{V: calFormatYear(int(y), w, l, c, m, fwd)}, nil
	}})

	inst.Dict.SetStr("pryear", &object.BuiltinFunc{Name: "pryear", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "pryear() requires year")
		}
		y, _ := toInt64(a[0])
		w, l, c, m := 2, 1, 6, 3
		if len(a) > 1 {
			if v, ok := toInt64(a[1]); ok {
				w = int(v)
			}
		}
		if len(a) > 2 {
			if v, ok := toInt64(a[2]); ok {
				l = int(v)
			}
		}
		if len(a) > 3 {
			if v, ok := toInt64(a[3]); ok {
				c = int(v)
			}
		}
		if len(a) > 4 {
			if v, ok := toInt64(a[4]); ok {
				m = int(v)
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("w"); ok {
				if n, ok2 := toInt64(v); ok2 {
					w = int(n)
				}
			}
			if v, ok := kw.GetStr("l"); ok {
				if n, ok2 := toInt64(v); ok2 {
					l = int(n)
				}
			}
			if v, ok := kw.GetStr("c"); ok {
				if n, ok2 := toInt64(v); ok2 {
					c = int(n)
				}
			}
			if v, ok := kw.GetStr("m"); ok {
				if n, ok2 := toInt64(v); ok2 {
					m = int(n)
				}
			}
		}
		fmt.Print(calFormatYear(int(y), w, l, c, m, fwd))
		return object.None, nil
	}})
}

// fillHTMLCalendarInst adds HTMLCalendar-specific methods to inst.Dict.
func (i *Interp) fillHTMLCalendarInst(inst *object.Instance, fwd int) {
	inst.Dict.SetStr("formatday", &object.BuiltinFunc{Name: "formatday", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "formatday() requires day, weekday")
		}
		day, _ := toInt64(a[0])
		wd, _ := toInt64(a[1])
		return &object.Str{V: calHTMLDay(int(day), int(wd))}, nil
	}})

	inst.Dict.SetStr("formatweekheader", &object.BuiltinFunc{Name: "formatweekheader", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		var sb strings.Builder
		sb.WriteString("<tr>")
		for k := 0; k < 7; k++ {
			wd := (fwd + k) % 7
			cls := "weekday"
			if wd >= 5 {
				cls = "weekend"
			}
			sb.WriteString(fmt.Sprintf(`<th class="%s">%s</th>`, cls, calDayAbbr[wd]))
		}
		sb.WriteString("</tr>")
		return &object.Str{V: sb.String()}, nil
	}})

	inst.Dict.SetStr("formatmonthname", &object.BuiltinFunc{Name: "formatmonthname", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "formatmonthname() requires year, month")
		}
		y, _ := toInt64(a[0])
		mo, _ := toInt64(a[1])
		withyear := true
		if len(a) > 2 {
			withyear = object.Truthy(a[2])
		}
		if kw != nil {
			if v, ok := kw.GetStr("withyear"); ok {
				withyear = object.Truthy(v)
			}
		}
		var name string
		if withyear {
			name = fmt.Sprintf("%s %d", calMonthName[mo], y)
		} else {
			name = calMonthName[mo]
		}
		return &object.Str{V: fmt.Sprintf(`<tr><th colspan="7" class="month">%s</th></tr>`, name)}, nil
	}})

	inst.Dict.SetStr("formatmonth", &object.BuiltinFunc{Name: "formatmonth", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "formatmonth() requires year, month")
		}
		y, _ := toInt64(a[0])
		mo, _ := toInt64(a[1])
		withyear := true
		if len(a) > 2 {
			withyear = object.Truthy(a[2])
		}
		if kw != nil {
			if v, ok := kw.GetStr("withyear"); ok {
				withyear = object.Truthy(v)
			}
		}
		return &object.Str{V: calHTMLFormatMonth(int(y), int(mo), fwd, withyear)}, nil
	}})

	inst.Dict.SetStr("formatyear", &object.BuiltinFunc{Name: "formatyear", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "formatyear() requires year")
		}
		y, _ := toInt64(a[0])
		width := 3
		if len(a) > 1 {
			if v, ok := toInt64(a[1]); ok {
				width = int(v)
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("width"); ok {
				if n, ok2 := toInt64(v); ok2 {
					width = int(n)
				}
			}
		}
		return &object.Str{V: calHTMLFormatYear(int(y), width, fwd)}, nil
	}})

	inst.Dict.SetStr("formatyearpage", &object.BuiltinFunc{Name: "formatyearpage", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "formatyearpage() requires year")
		}
		y, _ := toInt64(a[0])
		width := 3
		css := "calendar.css"
		encoding := "utf-8"
		if len(a) > 1 {
			if v, ok := toInt64(a[1]); ok {
				width = int(v)
			}
		}
		if len(a) > 2 {
			if s, ok := a[2].(*object.Str); ok {
				css = s.V
			}
		}
		if len(a) > 3 {
			if s, ok := a[3].(*object.Str); ok {
				encoding = s.V
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("width"); ok {
				if n, ok2 := toInt64(v); ok2 {
					width = int(n)
				}
			}
			if v, ok := kw.GetStr("css"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					css = s.V
				}
			}
			if v, ok := kw.GetStr("encoding"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					encoding = s.V
				}
			}
		}
		body := calHTMLFormatYear(int(y), width, fwd)
		var page string
		if encoding != "" {
			page = fmt.Sprintf("<?xml version=\"1.0\" encoding=\"%s\"?>\n<!DOCTYPE html PUBLIC \"-//W3C//DTD XHTML 1.0 Strict//EN\" \"http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd\">\n<html>\n<head>\n<meta http-equiv=\"Content-Type\" content=\"text/html; charset=%s\" />\n", encoding, encoding)
		} else {
			page = "<!DOCTYPE html PUBLIC \"-//W3C//DTD XHTML 1.0 Strict//EN\" \"http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd\">\n<html>\n<head>\n"
		}
		if css != "" {
			page += fmt.Sprintf("<link rel=\"stylesheet\" type=\"text/css\" href=\"%s\" />\n", css)
		}
		page += fmt.Sprintf("</head>\n<body>\n%s</body>\n</html>\n", body)
		return &object.Str{V: page}, nil
	}})
}

// extendCalendar adds the full calendar module API to m.
func (i *Interp) extendCalendar(m *object.Module) {
	// Exception classes
	illegalMonth := &object.Class{Name: "IllegalMonthError", Dict: object.NewDict(), Bases: []*object.Class{i.valueErr}}
	illegalWeekday := &object.Class{Name: "IllegalWeekdayError", Dict: object.NewDict(), Bases: []*object.Class{i.valueErr}}
	m.Dict.SetStr("IllegalMonthError", illegalMonth)
	m.Dict.SetStr("IllegalWeekdayError", illegalWeekday)

	// Month constants JANUARY–DECEMBER
	monthConsts := []string{"JANUARY", "FEBRUARY", "MARCH", "APRIL", "MAY", "JUNE",
		"JULY", "AUGUST", "SEPTEMBER", "OCTOBER", "NOVEMBER", "DECEMBER"}
	for k, name := range monthConsts {
		m.Dict.SetStr(name, object.IntFromBig(big.NewInt(int64(k+1))))
	}

	// Day enum class — class-level attrs hold enum instances; callable via Day(n)
	dayCls := &object.Class{Name: "Day", Dict: object.NewDict()}
	dayNames := []string{"MONDAY", "TUESDAY", "WEDNESDAY", "THURSDAY", "FRIDAY", "SATURDAY", "SUNDAY"}
	dayAttrs := object.NewDict()
	for k, name := range dayNames {
		inst := &object.Instance{Class: dayCls, Dict: object.NewDict()}
		inst.Dict.SetStr("value", object.IntFromBig(big.NewInt(int64(k))))
		inst.Dict.SetStr("name", &object.Str{V: name})
		dayAttrs.SetStr(name, inst)
	}
	dayCallable := &object.BuiltinFunc{Name: "Day", Attrs: dayAttrs, Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "Day() requires value")
		}
		n, _ := toInt64(a[0])
		if n < 0 || n > 6 {
			return nil, object.NewException(illegalWeekday, fmt.Sprintf("Day(%d) out of range", n))
		}
		inst := &object.Instance{Class: dayCls, Dict: object.NewDict()}
		inst.Dict.SetStr("value", object.IntFromBig(big.NewInt(n)))
		inst.Dict.SetStr("name", &object.Str{V: dayNames[n]})
		return inst, nil
	}}
	m.Dict.SetStr("Day", dayCallable)

	// Month enum class
	monthEnumCls := &object.Class{Name: "Month", Dict: object.NewDict()}
	monthAttrs := object.NewDict()
	for k, name := range monthConsts {
		inst := &object.Instance{Class: monthEnumCls, Dict: object.NewDict()}
		inst.Dict.SetStr("value", object.IntFromBig(big.NewInt(int64(k+1))))
		inst.Dict.SetStr("name", &object.Str{V: name})
		monthAttrs.SetStr(name, inst)
	}
	monthCallable := &object.BuiltinFunc{Name: "Month", Attrs: monthAttrs, Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "Month() requires value")
		}
		n, _ := toInt64(a[0])
		if n < 1 || n > 12 {
			return nil, object.NewException(illegalMonth, fmt.Sprintf("Month(%d) out of range", n))
		}
		inst := &object.Instance{Class: monthEnumCls, Dict: object.NewDict()}
		inst.Dict.SetStr("value", object.IntFromBig(big.NewInt(n)))
		inst.Dict.SetStr("name", &object.Str{V: monthConsts[n-1]})
		return inst, nil
	}}
	m.Dict.SetStr("Month", monthCallable)

	// Calendar, TextCalendar, HTMLCalendar, Locale* class objects
	calCls := &object.Class{Name: "Calendar", Dict: object.NewDict()}
	txtCls := &object.Class{Name: "TextCalendar", Dict: object.NewDict(), Bases: []*object.Class{calCls}}
	htmlCls := &object.Class{Name: "HTMLCalendar", Dict: object.NewDict(), Bases: []*object.Class{calCls}}
	localeTxtCls := &object.Class{Name: "LocaleTextCalendar", Dict: object.NewDict(), Bases: []*object.Class{txtCls}}
	localeHtmlCls := &object.Class{Name: "LocaleHTMLCalendar", Dict: object.NewDict(), Bases: []*object.Class{htmlCls}}

	parseFWD := func(a []object.Object, kw *object.Dict) int {
		fwd := calFWD
		if kw != nil {
			if v, ok := kw.GetStr("firstweekday"); ok {
				if n, ok2 := toInt64(v); ok2 {
					fwd = int(n)
				}
			}
		}
		if len(a) > 0 {
			if n, ok := toInt64(a[0]); ok {
				fwd = int(n)
			}
		}
		return fwd
	}

	m.Dict.SetStr("Calendar", &object.BuiltinFunc{Name: "Calendar", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		fwd := parseFWD(a, kw)
		inst := &object.Instance{Class: calCls, Dict: object.NewDict()}
		i.fillCalendarInst(inst, fwd)
		return inst, nil
	}})
	m.Dict.SetStr("TextCalendar", &object.BuiltinFunc{Name: "TextCalendar", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		fwd := parseFWD(a, kw)
		inst := &object.Instance{Class: txtCls, Dict: object.NewDict()}
		i.fillCalendarInst(inst, fwd)
		i.fillTextCalendarInst(inst, fwd)
		return inst, nil
	}})
	m.Dict.SetStr("HTMLCalendar", &object.BuiltinFunc{Name: "HTMLCalendar", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		fwd := parseFWD(a, kw)
		inst := &object.Instance{Class: htmlCls, Dict: object.NewDict()}
		i.fillCalendarInst(inst, fwd)
		i.fillHTMLCalendarInst(inst, fwd)
		return inst, nil
	}})
	m.Dict.SetStr("LocaleTextCalendar", &object.BuiltinFunc{Name: "LocaleTextCalendar", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		fwd := parseFWD(a, kw)
		inst := &object.Instance{Class: localeTxtCls, Dict: object.NewDict()}
		i.fillCalendarInst(inst, fwd)
		i.fillTextCalendarInst(inst, fwd)
		return inst, nil
	}})
	m.Dict.SetStr("LocaleHTMLCalendar", &object.BuiltinFunc{Name: "LocaleHTMLCalendar", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		fwd := parseFWD(a, kw)
		inst := &object.Instance{Class: localeHtmlCls, Dict: object.NewDict()}
		i.fillCalendarInst(inst, fwd)
		i.fillHTMLCalendarInst(inst, fwd)
		return inst, nil
	}})

	// Module-level functions
	m.Dict.SetStr("firstweekday", &object.BuiltinFunc{Name: "firstweekday", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.IntFromBig(big.NewInt(int64(calFWD))), nil
	}})
	m.Dict.SetStr("setfirstweekday", &object.BuiltinFunc{Name: "setfirstweekday", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "setfirstweekday() requires weekday")
		}
		n, ok := toInt64(a[0])
		if !ok || n < 0 || n > 6 {
			return nil, object.NewException(illegalWeekday, fmt.Sprintf("bad weekday %v; must be 0 (Monday) to 6 (Sunday)", a[0]))
		}
		calFWD = int(n)
		return object.None, nil
	}})
	m.Dict.SetStr("weekheader", &object.BuiltinFunc{Name: "weekheader", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		n := 2
		if len(a) > 0 {
			if v, ok := toInt64(a[0]); ok {
				n = int(v)
			}
		}
		return &object.Str{V: calWeekHeader(n, calFWD)}, nil
	}})
	m.Dict.SetStr("month", &object.BuiltinFunc{Name: "month", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "month() requires year, month")
		}
		y, _ := toInt64(a[0])
		mo, _ := toInt64(a[1])
		w, l := 0, 0
		if len(a) > 2 {
			if v, ok := toInt64(a[2]); ok {
				w = int(v)
			}
		}
		if len(a) > 3 {
			if v, ok := toInt64(a[3]); ok {
				l = int(v)
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("w"); ok {
				if n, ok2 := toInt64(v); ok2 {
					w = int(n)
				}
			}
			if v, ok := kw.GetStr("l"); ok {
				if n, ok2 := toInt64(v); ok2 {
					l = int(n)
				}
			}
		}
		return &object.Str{V: calFormatMonth(int(y), int(mo), w, l, calFWD)}, nil
	}})
	m.Dict.SetStr("prmonth", &object.BuiltinFunc{Name: "prmonth", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "prmonth() requires year, month")
		}
		y, _ := toInt64(a[0])
		mo, _ := toInt64(a[1])
		w, l := 0, 0
		if len(a) > 2 {
			if v, ok := toInt64(a[2]); ok {
				w = int(v)
			}
		}
		if len(a) > 3 {
			if v, ok := toInt64(a[3]); ok {
				l = int(v)
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("w"); ok {
				if n, ok2 := toInt64(v); ok2 {
					w = int(n)
				}
			}
			if v, ok := kw.GetStr("l"); ok {
				if n, ok2 := toInt64(v); ok2 {
					l = int(n)
				}
			}
		}
		fmt.Print(calFormatMonth(int(y), int(mo), w, l, calFWD))
		return object.None, nil
	}})
	m.Dict.SetStr("calendar", &object.BuiltinFunc{Name: "calendar", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "calendar() requires year")
		}
		y, _ := toInt64(a[0])
		w, l, c, mn := 2, 1, 6, 3
		if len(a) > 1 {
			if v, ok := toInt64(a[1]); ok {
				w = int(v)
			}
		}
		if len(a) > 2 {
			if v, ok := toInt64(a[2]); ok {
				l = int(v)
			}
		}
		if len(a) > 3 {
			if v, ok := toInt64(a[3]); ok {
				c = int(v)
			}
		}
		if len(a) > 4 {
			if v, ok := toInt64(a[4]); ok {
				mn = int(v)
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("w"); ok {
				if n, ok2 := toInt64(v); ok2 {
					w = int(n)
				}
			}
			if v, ok := kw.GetStr("l"); ok {
				if n, ok2 := toInt64(v); ok2 {
					l = int(n)
				}
			}
			if v, ok := kw.GetStr("c"); ok {
				if n, ok2 := toInt64(v); ok2 {
					c = int(n)
				}
			}
			if v, ok := kw.GetStr("m"); ok {
				if n, ok2 := toInt64(v); ok2 {
					mn = int(n)
				}
			}
		}
		return &object.Str{V: calFormatYear(int(y), w, l, c, mn, calFWD)}, nil
	}})
	m.Dict.SetStr("prcal", &object.BuiltinFunc{Name: "prcal", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "prcal() requires year")
		}
		y, _ := toInt64(a[0])
		w, l, c, mn := 2, 1, 6, 3
		if len(a) > 1 {
			if v, ok := toInt64(a[1]); ok {
				w = int(v)
			}
		}
		if len(a) > 2 {
			if v, ok := toInt64(a[2]); ok {
				l = int(v)
			}
		}
		if len(a) > 3 {
			if v, ok := toInt64(a[3]); ok {
				c = int(v)
			}
		}
		if len(a) > 4 {
			if v, ok := toInt64(a[4]); ok {
				mn = int(v)
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("w"); ok {
				if n, ok2 := toInt64(v); ok2 {
					w = int(n)
				}
			}
			if v, ok := kw.GetStr("l"); ok {
				if n, ok2 := toInt64(v); ok2 {
					l = int(n)
				}
			}
			if v, ok := kw.GetStr("c"); ok {
				if n, ok2 := toInt64(v); ok2 {
					c = int(n)
				}
			}
			if v, ok := kw.GetStr("m"); ok {
				if n, ok2 := toInt64(v); ok2 {
					mn = int(n)
				}
			}
		}
		fmt.Print(calFormatYear(int(y), w, l, c, mn, calFWD))
		return object.None, nil
	}})

	// Override monthcalendar to respect calFWD (global firstweekday).
	m.Dict.SetStr("monthcalendar", &object.BuiltinFunc{Name: "monthcalendar", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "monthcalendar() requires year, month")
		}
		y, _ := toInt64(a[0])
		mo, _ := toInt64(a[1])
		if mo < 1 || mo > 12 {
			return nil, object.NewException(illegalMonth, fmt.Sprintf("bad month number %d; must be 1-12", mo))
		}
		weeks := calMonthCalFWD(int(y), int(mo), calFWD)
		out := make([]object.Object, len(weeks))
		for k, week := range weeks {
			row := make([]object.Object, len(week))
			for j, d := range week {
				row[j] = object.IntFromBig(big.NewInt(int64(d)))
			}
			out[k] = &object.List{V: row}
		}
		return &object.List{V: out}, nil
	}})

	// Override monthrange to use IllegalMonthError.
	m.Dict.SetStr("monthrange", &object.BuiltinFunc{Name: "monthrange", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "monthrange() requires year, month")
		}
		y, _ := toInt64(a[0])
		mo, _ := toInt64(a[1])
		if mo < 1 || mo > 12 {
			return nil, object.NewException(illegalMonth, fmt.Sprintf("bad month number %d; must be 1-12", mo))
		}
		first := time.Date(int(y), time.Month(mo), 1, 0, 0, 0, 0, time.UTC)
		w := (int(first.Weekday()) + 6) % 7
		days := daysInMonth(int(y), int(mo))
		return &object.Tuple{V: []object.Object{
			object.IntFromBig(big.NewInt(int64(w))),
			object.IntFromBig(big.NewInt(int64(days))),
		}}, nil
	}})
}
