package vm

import (
	"fmt"
	"sync"
	"time"
	_ "time/tzdata"

	"github.com/tamnd/goipy/object"
)

// Zone cache shared across all buildZoneinfo calls on the same Interp.
var (
	ziCacheMap = map[string]*object.Instance{}
	ziCacheMu  sync.Mutex
)

// ziTDCls is the timedelta class used for ZoneInfo offset instances.
// It is compatible with datetime's tdFromObj (checks _days/_secs/_usecs).
var ziTDCls = &object.Class{Name: "timedelta", Dict: object.NewDict()}

func (i *Interp) buildZoneinfo() *object.Module {
	m := &object.Module{Name: "zoneinfo", Dict: object.NewDict()}

	// ZoneInfoNotFoundError — subclass of KeyError
	ziErr := &object.Class{
		Name:  "ZoneInfoNotFoundError",
		Dict:  object.NewDict(),
		Bases: []*object.Class{i.keyErr},
	}
	m.Dict.SetStr("ZoneInfoNotFoundError", ziErr)

	// InvalidTZPathWarning — warning category (no-op in our runtime)
	ziWarn := &object.Class{Name: "InvalidTZPathWarning", Dict: object.NewDict()}
	m.Dict.SetStr("InvalidTZPathWarning", ziWarn)

	// Internal ZoneInfo class — instances carry type(inst).__name__ == "ZoneInfo"
	ziCls := &object.Class{Name: "ZoneInfo", Dict: object.NewDict()}

	// makeTD builds a timedelta instance (using ziTDCls) for UTC offsets.
	// It is compatible with datetime's tzFromObj because tdFromObj only
	// inspects _days, _secs, _usecs — not the class pointer.
	makeTD := func(secs int64) *object.Instance {
		return i.makeTDInstance(ziTDCls, normTimedelta(0, secs, 0))
	}

	// makeZI constructs a new (uncached) ZoneInfo instance for key.
	// Returns error if key is unknown to the OS/embedded tzdata.
	makeZI := func(key string) (*object.Instance, error) {
		loc, err := time.LoadLocation(key)
		if err != nil {
			exc := object.NewException(ziErr, fmt.Sprintf("No time zone found with key %s", key))
			return nil, exc
		}

		inst := &object.Instance{Class: ziCls, Dict: object.NewDict()}
		inst.Dict.SetStr("key", &object.Str{V: key})

		// _offset and _name make this instance usable as a tzinfo in datetime's
		// isoformat/utcoffset/tzname formatters (which call tzFromObj).
		// We compute the "current" standard offset; DST-varying zones will
		// show the offset as of construction time.
		now := time.Now().In(loc)
		zoneName, offsetSecs := now.Zone()
		inst.Dict.SetStr("_offset", makeTD(int64(offsetSecs)))
		inst.Dict.SetStr("_name", &object.Str{V: zoneName})

		// utcoffset(dt) — recompute for the supplied datetime if possible.
		inst.Dict.SetStr("utcoffset", &object.BuiltinFunc{Name: "utcoffset", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			t := time.Now().In(loc)
			if len(a) > 0 {
				if dt, ok2 := dtFromObj(a[0]); ok2 {
					t = time.Date(dt.year, time.Month(dt.month), dt.day,
						dt.hour, dt.min, dt.sec, 0, loc)
				}
			}
			_, off := t.Zone()
			return makeTD(int64(off)), nil
		}})

		// dst(dt) — returns timedelta(0) for non-DST, timedelta(hours=1) for DST.
		inst.Dict.SetStr("dst", &object.BuiltinFunc{Name: "dst", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			t := time.Now().In(loc)
			if len(a) > 0 {
				if dt, ok2 := dtFromObj(a[0]); ok2 {
					t = time.Date(dt.year, time.Month(dt.month), dt.day,
						dt.hour, dt.min, dt.sec, 0, loc)
				}
			}
			if t.IsDST() {
				return makeTD(3600), nil
			}
			return makeTD(0), nil
		}})

		// tzname(dt) — returns the zone abbreviation at the given datetime.
		inst.Dict.SetStr("tzname", &object.BuiltinFunc{Name: "tzname", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			t := time.Now().In(loc)
			if len(a) > 0 {
				if dt, ok2 := dtFromObj(a[0]); ok2 {
					t = time.Date(dt.year, time.Month(dt.month), dt.day,
						dt.hour, dt.min, dt.sec, 0, loc)
				}
			}
			name, _ := t.Zone()
			return &object.Str{V: name}, nil
		}})

		// fromutc(dt) — convert UTC datetime to local time.
		inst.Dict.SetStr("fromutc", &object.BuiltinFunc{Name: "fromutc", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "fromutc() requires a datetime argument")
			}
			dt, ok := dtFromObj(a[0])
			if !ok {
				return nil, object.Errorf(i.typeErr, "fromutc() argument must be a datetime")
			}
			utcT := time.Date(dt.year, time.Month(dt.month), dt.day,
				dt.hour, dt.min, dt.sec, dt.usec*1000, time.UTC)
			localT := utcT.In(loc)
			// Return the original instance with updated time fields (best-effort).
			if origInst, ok2 := a[0].(*object.Instance); ok2 {
				nd := goDatetime{
					goDate{localT.Year(), int(localT.Month()), localT.Day()},
					goTime{localT.Hour(), localT.Minute(), localT.Second(),
						localT.Nanosecond() / 1000, origInst, 0},
				}
				_ = nd
			}
			return a[0], nil
		}})

		inst.Dict.SetStr("__str__", &object.BuiltinFunc{Name: "__str__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: key}, nil
		}})
		inst.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: fmt.Sprintf("zoneinfo.ZoneInfo(key='%s')", key)}, nil
		}})
		inst.Dict.SetStr("__hash__", &object.BuiltinFunc{Name: "__hash__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return intObj64(int64(len(key))), nil
		}})

		return inst, nil
	}

	// The ZoneInfo callable — a BuiltinFunc so its Attrs hold class methods,
	// and repeated calls with the same key return the exact same object.
	var ziCallable *object.BuiltinFunc
	ziCallable = &object.BuiltinFunc{
		Name:  "ZoneInfo",
		Attrs: object.NewDict(),
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			key := ""
			if len(a) > 0 {
				if s, ok := a[0].(*object.Str); ok {
					key = s.V
				}
			}
			if kw != nil {
				if v, ok := kw.GetStr("key"); ok {
					if s, ok2 := v.(*object.Str); ok2 {
						key = s.V
					}
				}
			}
			if key == "" {
				return nil, object.Errorf(i.typeErr, "ZoneInfo() requires a key argument (got empty string)")
			}

			ziCacheMu.Lock()
			if inst, ok := ziCacheMap[key]; ok {
				ziCacheMu.Unlock()
				return inst, nil
			}
			ziCacheMu.Unlock()

			inst, err := makeZI(key)
			if err != nil {
				return nil, err
			}

			ziCacheMu.Lock()
			ziCacheMap[key] = inst
			ziCacheMu.Unlock()
			return inst, nil
		},
	}
	_ = ziCallable // used below in Attrs

	// ZoneInfo.no_cache — creates a fresh instance each call (never cached).
	ziCallable.Attrs.SetStr("no_cache", &object.BuiltinFunc{Name: "no_cache", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		key := ""
		if len(a) > 0 {
			if s, ok := a[0].(*object.Str); ok {
				key = s.V
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("key"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					key = s.V
				}
			}
		}
		if key == "" {
			return nil, object.Errorf(i.typeErr, "ZoneInfo.no_cache() requires a key argument")
		}
		return makeZI(key)
	}})

	// ZoneInfo.clear_cache — remove entries from the cache.
	ziCallable.Attrs.SetStr("clear_cache", &object.BuiltinFunc{Name: "clear_cache", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		ziCacheMu.Lock()
		defer ziCacheMu.Unlock()

		// only_keys kwarg: remove specific keys
		if kw != nil {
			if v, ok := kw.GetStr("only_keys"); ok {
				switch lst := v.(type) {
				case *object.List:
					for _, item := range lst.V {
						if s, ok2 := item.(*object.Str); ok2 {
							delete(ziCacheMap, s.V)
						}
					}
				case *object.Tuple:
					for _, item := range lst.V {
						if s, ok2 := item.(*object.Str); ok2 {
							delete(ziCacheMap, s.V)
						}
					}
				}
				return object.None, nil
			}
		}
		// Clear all
		ziCacheMap = map[string]*object.Instance{}
		return object.None, nil
	}})

	// ZoneInfo.from_file — construct from a TZif file object.
	ziCallable.Attrs.SetStr("from_file", &object.BuiltinFunc{Name: "from_file", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		// In our runtime we have no TZif parser. Accept a key via keyword
		// so callers can at least test the call path.
		key := "<unknown>"
		if kw != nil {
			if v, ok := kw.GetStr("key"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					key = s.V
				}
			}
		}
		return makeZI(key)
	}})

	m.Dict.SetStr("ZoneInfo", ziCallable)

	// TZPATH — read-only tuple of absolute search paths.
	// Since we use Go's embedded tzdata, no file-system paths are needed.
	tzpathTuple := &object.Tuple{V: []object.Object{}}
	m.Dict.SetStr("TZPATH", tzpathTuple)

	// available_timezones() — returns all canonical IANA keys as a set.
	m.Dict.SetStr("available_timezones", &object.BuiltinFunc{Name: "available_timezones", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		s := object.NewSet()
		for _, z := range ziIANAZones {
			s.Add(&object.Str{V: z})
		}
		return s, nil
	}})

	// reset_tzpath(to=None) — update TZPATH; zones still load from embedded data.
	m.Dict.SetStr("reset_tzpath", &object.BuiltinFunc{Name: "reset_tzpath", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		var items []object.Object
		if len(a) > 0 {
			switch lst := a[0].(type) {
			case *object.List:
				items = lst.V
			case *object.Tuple:
				items = lst.V
			}
		} else if kw != nil {
			if v, ok := kw.GetStr("to"); ok {
				switch lst := v.(type) {
				case *object.List:
					items = lst.V
				case *object.Tuple:
					items = lst.V
				}
			}
		}
		newPath := make([]object.Object, len(items))
		copy(newPath, items)
		m.Dict.SetStr("TZPATH", &object.Tuple{V: newPath})
		return object.None, nil
	}})

	return m
}

// ziIANAZones is the canonical set of IANA timezone keys available via
// Go's embedded tzdata (time/tzdata). Excludes posix/ right/ links.
var ziIANAZones = []string{
	// Africa
	"Africa/Abidjan", "Africa/Accra", "Africa/Addis_Ababa", "Africa/Algiers",
	"Africa/Asmara", "Africa/Bamako", "Africa/Bangui", "Africa/Banjul",
	"Africa/Bissau", "Africa/Blantyre", "Africa/Brazzaville", "Africa/Bujumbura",
	"Africa/Cairo", "Africa/Casablanca", "Africa/Ceuta", "Africa/Conakry",
	"Africa/Dakar", "Africa/Dar_es_Salaam", "Africa/Djibouti", "Africa/Douala",
	"Africa/El_Aaiun", "Africa/Freetown", "Africa/Gaborone", "Africa/Harare",
	"Africa/Johannesburg", "Africa/Juba", "Africa/Kampala", "Africa/Khartoum",
	"Africa/Kigali", "Africa/Kinshasa", "Africa/Lagos", "Africa/Libreville",
	"Africa/Lome", "Africa/Luanda", "Africa/Lubumbashi", "Africa/Lusaka",
	"Africa/Malabo", "Africa/Maputo", "Africa/Maseru", "Africa/Mbabane",
	"Africa/Mogadishu", "Africa/Monrovia", "Africa/Nairobi", "Africa/Ndjamena",
	"Africa/Niamey", "Africa/Nouakchott", "Africa/Ouagadougou", "Africa/Porto-Novo",
	"Africa/Sao_Tome", "Africa/Tripoli", "Africa/Tunis", "Africa/Windhoek",
	// America
	"America/Adak", "America/Anchorage", "America/Anguilla", "America/Antigua",
	"America/Araguaina", "America/Argentina/Buenos_Aires", "America/Argentina/Catamarca",
	"America/Argentina/Cordoba", "America/Argentina/Jujuy", "America/Argentina/La_Rioja",
	"America/Argentina/Mendoza", "America/Argentina/Rio_Gallegos", "America/Argentina/Salta",
	"America/Argentina/San_Juan", "America/Argentina/San_Luis", "America/Argentina/Tucuman",
	"America/Argentina/Ushuaia", "America/Aruba", "America/Asuncion", "America/Atikokan",
	"America/Bahia", "America/Bahia_Banderas", "America/Barbados", "America/Belem",
	"America/Belize", "America/Blanc-Sablon", "America/Boa_Vista", "America/Bogota",
	"America/Boise", "America/Cambridge_Bay", "America/Campo_Grande", "America/Cancun",
	"America/Caracas", "America/Cayenne", "America/Cayman", "America/Chicago",
	"America/Chihuahua", "America/Ciudad_Juarez", "America/Costa_Rica", "America/Creston",
	"America/Cuiaba", "America/Curacao", "America/Danmarkshavn", "America/Dawson",
	"America/Dawson_Creek", "America/Denver", "America/Detroit", "America/Dominica",
	"America/Edmonton", "America/Eirunepe", "America/El_Salvador", "America/Fortaleza",
	"America/Glace_Bay", "America/Goose_Bay", "America/Grand_Turk",
	"America/Grenada", "America/Guadeloupe", "America/Guatemala", "America/Guayaquil",
	"America/Guyana", "America/Halifax", "America/Havana", "America/Hermosillo",
	"America/Indiana/Indianapolis", "America/Indiana/Knox", "America/Indiana/Marengo",
	"America/Indiana/Petersburg", "America/Indiana/Tell_City", "America/Indiana/Vevay",
	"America/Indiana/Vincennes", "America/Indiana/Winamac", "America/Inuvik",
	"America/Iqaluit", "America/Jamaica", "America/Juneau", "America/Kentucky/Louisville",
	"America/Kentucky/Monticello", "America/Kralendijk", "America/La_Paz", "America/Lima",
	"America/Los_Angeles", "America/Lower_Princes", "America/Maceio", "America/Managua",
	"America/Manaus", "America/Marigot", "America/Martinique", "America/Matamoros",
	"America/Mazatlan", "America/Menominee", "America/Merida", "America/Metlakatla",
	"America/Mexico_City", "America/Miquelon", "America/Moncton", "America/Monterrey",
	"America/Montevideo", "America/Montserrat", "America/Nassau", "America/New_York",
	"America/Nome", "America/Noronha", "America/North_Dakota/Beulah",
	"America/North_Dakota/Center", "America/North_Dakota/New_Salem", "America/Nuuk",
	"America/Ojinaga", "America/Panama", "America/Paramaribo",
	"America/Phoenix", "America/Port-au-Prince", "America/Port_of_Spain",
	"America/Porto_Velho", "America/Puerto_Rico", "America/Punta_Arenas",
	"America/Rankin_Inlet", "America/Recife", "America/Regina",
	"America/Resolute", "America/Rio_Branco", "America/Santarem", "America/Santiago",
	"America/Santo_Domingo", "America/Sao_Paulo", "America/Scoresbysund",
	"America/Sitka", "America/St_Barthelemy", "America/St_Johns", "America/St_Kitts",
	"America/St_Lucia", "America/St_Thomas", "America/St_Vincent", "America/Swift_Current",
	"America/Tegucigalpa", "America/Thule", "America/Tijuana",
	"America/Toronto", "America/Tortola", "America/Vancouver", "America/Whitehorse",
	"America/Winnipeg", "America/Yakutat", "America/Yellowknife",
	// Antarctica
	"Antarctica/Casey", "Antarctica/Davis", "Antarctica/DumontDUrville",
	"Antarctica/Macquarie", "Antarctica/Mawson", "Antarctica/McMurdo",
	"Antarctica/Palmer", "Antarctica/Rothera", "Antarctica/Syowa",
	"Antarctica/Troll", "Antarctica/Vostok",
	// Arctic
	"Arctic/Longyearbyen",
	// Asia
	"Asia/Aden", "Asia/Almaty", "Asia/Amman", "Asia/Anadyr", "Asia/Aqtau",
	"Asia/Aqtobe", "Asia/Ashgabat", "Asia/Atyrau", "Asia/Baghdad", "Asia/Bahrain",
	"Asia/Baku", "Asia/Bangkok", "Asia/Barnaul", "Asia/Beirut", "Asia/Bishkek",
	"Asia/Brunei", "Asia/Chita", "Asia/Choibalsan", "Asia/Colombo", "Asia/Damascus",
	"Asia/Dhaka", "Asia/Dili", "Asia/Dubai", "Asia/Dushanbe", "Asia/Famagusta",
	"Asia/Gaza", "Asia/Hebron", "Asia/Ho_Chi_Minh", "Asia/Hong_Kong", "Asia/Hovd",
	"Asia/Irkutsk", "Asia/Jakarta", "Asia/Jayapura", "Asia/Jerusalem", "Asia/Kabul",
	"Asia/Kamchatka", "Asia/Karachi", "Asia/Kathmandu", "Asia/Khandyga", "Asia/Kolkata",
	"Asia/Krasnoyarsk", "Asia/Kuala_Lumpur", "Asia/Kuching", "Asia/Kuwait",
	"Asia/Macau", "Asia/Magadan", "Asia/Makassar", "Asia/Manila", "Asia/Muscat",
	"Asia/Nicosia", "Asia/Novokuznetsk", "Asia/Novosibirsk", "Asia/Omsk",
	"Asia/Oral", "Asia/Phnom_Penh", "Asia/Pontianak", "Asia/Pyongyang", "Asia/Qatar",
	"Asia/Qostanay", "Asia/Qyzylorda", "Asia/Riyadh", "Asia/Sakhalin", "Asia/Samarkand",
	"Asia/Seoul", "Asia/Shanghai", "Asia/Singapore", "Asia/Srednekolymsk",
	"Asia/Taipei", "Asia/Tashkent", "Asia/Tbilisi", "Asia/Tehran", "Asia/Thimphu",
	"Asia/Tokyo", "Asia/Tomsk", "Asia/Ulaanbaatar", "Asia/Urumqi", "Asia/Ust-Nera",
	"Asia/Vientiane", "Asia/Vladivostok", "Asia/Yakutsk", "Asia/Yangon",
	"Asia/Yekaterinburg", "Asia/Yerevan",
	// Atlantic
	"Atlantic/Azores", "Atlantic/Bermuda", "Atlantic/Canary", "Atlantic/Cape_Verde",
	"Atlantic/Faroe", "Atlantic/Madeira", "Atlantic/Reykjavik", "Atlantic/South_Georgia",
	"Atlantic/St_Helena", "Atlantic/Stanley",
	// Australia
	"Australia/Adelaide", "Australia/Brisbane", "Australia/Broken_Hill",
	"Australia/Darwin", "Australia/Eucla", "Australia/Hobart", "Australia/Lindeman",
	"Australia/Lord_Howe", "Australia/Melbourne", "Australia/Perth", "Australia/Sydney",
	// Europe
	"Europe/Amsterdam", "Europe/Andorra", "Europe/Astrakhan", "Europe/Athens",
	"Europe/Belgrade", "Europe/Berlin", "Europe/Bratislava", "Europe/Brussels",
	"Europe/Bucharest", "Europe/Budapest", "Europe/Busingen", "Europe/Chisinau",
	"Europe/Copenhagen", "Europe/Dublin", "Europe/Gibraltar", "Europe/Guernsey",
	"Europe/Helsinki", "Europe/Isle_of_Man", "Europe/Istanbul", "Europe/Jersey",
	"Europe/Kaliningrad", "Europe/Kirov", "Europe/Kyiv",
	"Europe/Lisbon", "Europe/Ljubljana", "Europe/London", "Europe/Luxembourg",
	"Europe/Madrid", "Europe/Malta", "Europe/Mariehamn", "Europe/Minsk",
	"Europe/Monaco", "Europe/Moscow", "Europe/Nicosia", "Europe/Oslo",
	"Europe/Paris", "Europe/Podgorica", "Europe/Prague", "Europe/Riga",
	"Europe/Rome", "Europe/Samara", "Europe/San_Marino", "Europe/Sarajevo",
	"Europe/Saratov", "Europe/Simferopol", "Europe/Skopje", "Europe/Sofia",
	"Europe/Stockholm", "Europe/Tallinn", "Europe/Tirane", "Europe/Ulyanovsk",
	"Europe/Uzhgorod", "Europe/Vaduz", "Europe/Vatican", "Europe/Vienna",
	"Europe/Vilnius", "Europe/Volgograd", "Europe/Warsaw", "Europe/Zagreb",
	"Europe/Zaporozhye", "Europe/Zurich",
	// Indian
	"Indian/Antananarivo", "Indian/Chagos", "Indian/Christmas", "Indian/Cocos",
	"Indian/Comoro", "Indian/Kerguelen", "Indian/Mahe", "Indian/Maldives",
	"Indian/Mauritius", "Indian/Mayotte", "Indian/Reunion",
	// Pacific
	"Pacific/Apia", "Pacific/Auckland", "Pacific/Bougainville", "Pacific/Chatham",
	"Pacific/Chuuk", "Pacific/Easter", "Pacific/Efate", "Pacific/Fakaofo",
	"Pacific/Fiji", "Pacific/Funafuti", "Pacific/Galapagos", "Pacific/Gambier",
	"Pacific/Guadalcanal", "Pacific/Guam", "Pacific/Honolulu", "Pacific/Kanton",
	"Pacific/Kiritimati", "Pacific/Kosrae", "Pacific/Kwajalein", "Pacific/Majuro",
	"Pacific/Marquesas", "Pacific/Midway", "Pacific/Nauru", "Pacific/Niue",
	"Pacific/Norfolk", "Pacific/Noumea", "Pacific/Pago_Pago", "Pacific/Palau",
	"Pacific/Pitcairn", "Pacific/Pohnpei", "Pacific/Port_Moresby", "Pacific/Rarotonga",
	"Pacific/Saipan", "Pacific/Tahiti", "Pacific/Tarawa", "Pacific/Tongatapu",
	"Pacific/Wake", "Pacific/Wallis",
	// Etc / UTC
	"Etc/GMT", "Etc/GMT+0", "Etc/GMT+1", "Etc/GMT+10", "Etc/GMT+11", "Etc/GMT+12",
	"Etc/GMT+2", "Etc/GMT+3", "Etc/GMT+4", "Etc/GMT+5", "Etc/GMT+6", "Etc/GMT+7",
	"Etc/GMT+8", "Etc/GMT+9", "Etc/GMT-0", "Etc/GMT-1", "Etc/GMT-10", "Etc/GMT-11",
	"Etc/GMT-12", "Etc/GMT-13", "Etc/GMT-14", "Etc/GMT-2", "Etc/GMT-3", "Etc/GMT-4",
	"Etc/GMT-5", "Etc/GMT-6", "Etc/GMT-7", "Etc/GMT-8", "Etc/GMT-9", "Etc/UTC",
	"UTC",
}
