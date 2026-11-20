// Copyright 2026 Bob Benjy and contributors. Licensed under Apache-2.0. See LICENSE.

// Shared Chabad.org helpers for the hand-written commands: location
// resolution, the zmanim and candle-lighting webservices, and the
// daily-study RSS feed. All requests go through the generated client
// (flags.newClient), so transport, caching, rate limiting, and the verify
// mock all apply.

package cli

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/mvanhorn/printing-press-library/library/productivity/chabad/internal/client"
	"github.com/mvanhorn/printing-press-library/library/productivity/chabad/internal/config"
	"github.com/mvanhorn/printing-press-library/library/productivity/chabad/internal/jcal"
)

const (
	pathGetLocations   = "/WebServices/RemoteCall/Get_Locations"
	pathGetZmanim      = "/webservices/zmanim/zmanim/Get_Zmanim"
	pathCandleWeeks    = "/webservices/Zmanim/CandleLighting/Load_CandleLightingWeeks"
	pathDailyStudyRSS  = "/tools/rss/dailystudy_rss.xml"
	maxZmanimChunkDays = 150
)

// chabadLocation is a resolved Chabad.org location.
type chabadLocation struct {
	ID   string `json:"id"`
	Type int    `json:"type"` // 1 = Chabad.org city id, 2 = US ZIP
	Name string `json:"name"`
}

func (l chabadLocation) key() string { return fmt.Sprintf("%d-%s", l.Type, l.ID) }

var (
	reZIP    = regexp.MustCompile(`^\d{5}$`)
	reDigits = regexp.MustCompile(`^\d+$`)
	reTags   = regexp.MustCompile(`<[^>]*>`)
	reSpaces = regexp.MustCompile(`\s+`)
)

// Shared help text. Each command declares its flags inline (so static
// SKILL/README verification can see them) and reuses these strings.
const (
	locationFlagHelp       = "Location: a US ZIP (11213), a Chabad.org city id (city:247), or a city name (\"Los Angeles\"); defaults to the location saved with 'locations use'"
	candleMinutesHelp      = "Light candles this many minutes before sunset instead of the location's Chabad.org default (18; 40 in Jerusalem)"
	shabbatEndsMinutesHelp = "End Shabbat/Yom Tov this many minutes after sunset instead of Chabad.org's calculated time"
)

// locationSuggestion is one Get_Locations autocomplete row.
type locationSuggestion struct {
	Title    string `json:"Title"`
	Value    string `json:"Value"`
	ItemType string `json:"ItemType"`
}

func searchLocations(ctx context.Context, c *client.Client, query string) ([]locationSuggestion, error) {
	data, err := c.Get(ctx, pathGetLocations, map[string]string{"location": query})
	if err != nil {
		return nil, err
	}
	var env struct {
		Suggestions []locationSuggestion `json:"Suggestions"`
	}
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("parsing Chabad.org location search: %w", err)
	}
	return env.Suggestions, nil
}

// resolveChabadLocation turns --location input (or the saved default) into a
// Chabad.org location id and type.
func resolveChabadLocation(ctx context.Context, flags *rootFlags, c *client.Client, input string) (chabadLocation, error) {
	in := strings.TrimSpace(input)
	if in == "" {
		saved, ok, err := config.LoadDefaultLocation(flags.configPath)
		if err != nil {
			return chabadLocation{}, configErr(fmt.Errorf("reading saved location: %w", err))
		}
		if !ok {
			return chabadLocation{}, usageErr(fmt.Errorf("no location given: pass --location 11213 (a ZIP, city name, or city:<id>), or save a default with 'chabad-pp-cli locations use 11213'"))
		}
		return chabadLocation{ID: saved.ID, Type: saved.Type, Name: saved.Name}, nil
	}
	lower := strings.ToLower(in)
	switch {
	case strings.HasPrefix(lower, "zip:"):
		return chabadLocation{ID: strings.TrimSpace(in[4:]), Type: 2}, nil
	case strings.HasPrefix(lower, "city:"):
		return chabadLocation{ID: strings.TrimSpace(in[5:]), Type: 1}, nil
	case reZIP.MatchString(in):
		return chabadLocation{ID: in, Type: 2}, nil
	case reDigits.MatchString(in):
		return chabadLocation{ID: in, Type: 1}, nil
	}
	sugg, err := searchLocations(ctx, c, in)
	if err != nil {
		return chabadLocation{}, err
	}
	for _, s := range sugg {
		t, _ := strconv.Atoi(s.ItemType)
		if (t == 1 || t == 2) && s.Value != "" {
			return chabadLocation{ID: s.Value, Type: t, Name: stripControl(s.Title)}, nil
		}
	}
	return chabadLocation{}, notFoundErr(fmt.Errorf("no Chabad.org location matches %q; try 'chabad-pp-cli locations --query %q'", in, in))
}

// diasporaDefaultLocation is Crown Heights (ZIP 11213), used by commands whose
// output only depends on Israel vs. Diaspora when no location is known.
var diasporaDefaultLocation = chabadLocation{ID: "11213", Type: 2, Name: "Brooklyn, NY 11213"}

// resolveLocationOrDiaspora resolves --location or the saved default; when
// neither exists it returns the Diaspora default with usedDefault=true. Real
// errors (bad config, failed lookups) are returned, not hidden.
func resolveLocationOrDiaspora(ctx context.Context, flags *rootFlags, c *client.Client, input string) (chabadLocation, bool, error) {
	if strings.TrimSpace(input) == "" {
		if _, ok, err := config.LoadDefaultLocation(flags.configPath); err != nil {
			return chabadLocation{}, false, configErr(fmt.Errorf("reading saved location: %w", err))
		} else if !ok {
			return diasporaDefaultLocation, true, nil
		}
	}
	loc, err := resolveChabadLocation(ctx, flags, c, input)
	return loc, false, err
}

// zmanimOptions are the shared calculation overrides.
type zmanimOptions struct {
	CandleMinutes      int
	ShabbatEndsMinutes int
	AllOpinions        bool
}

func (o zmanimOptions) apply(params map[string]string) {
	if o.CandleMinutes > 0 {
		params["bdef"] = "0"
		params["before"] = strconv.Itoa(o.CandleMinutes)
	}
	if o.ShabbatEndsMinutes > 0 {
		params["shabbosends"] = "min"
		params["after"] = strconv.Itoa(o.ShabbatEndsMinutes)
	}
	if o.AllOpinions {
		params["additional"] = "true"
	}
}

func (o zmanimOptions) validate() error {
	if o.CandleMinutes < 0 || o.CandleMinutes > 120 {
		return usageErr(fmt.Errorf("--candle-minutes must be 0 (Chabad.org default) to 120"))
	}
	if o.ShabbatEndsMinutes < 0 || o.ShabbatEndsMinutes > 180 {
		return usageErr(fmt.Errorf("--shabbat-ends-minutes must be 0 (Chabad.org default) to 180"))
	}
	return nil
}

// zmanEntry is one halachic time.
type zmanEntry struct {
	Type        string     `json:"type"`
	Slug        string     `json:"slug"`
	Title       string     `json:"title"`
	HebrewTitle string     `json:"hebrew_title,omitempty"`
	Time        string     `json:"time"`
	At          *time.Time `json:"at,omitempty"`
	Opinion     string     `json:"opinion,omitempty"`
	Default     bool       `json:"default"`
}

// zmanimDay is one day of Chabad.org zmanim.
type zmanimDay struct {
	Date      string      `json:"date"`
	Weekday   string      `json:"weekday"`
	Hebrew    string      `json:"hebrew_date"`
	Reading   string      `json:"torah_reading,omitempty"`
	Holiday   string      `json:"holiday,omitempty"`
	IsHoliday bool        `json:"is_holiday"`
	Zmanim    []zmanEntry `json:"zmanim"`

	day       time.Time // Gregorian date at UTC midnight
	offset    int       // location UTC offset in minutes for this date
	hasOffset bool
}

// find returns the default (or first) entry of a zman type.
func (d zmanimDay) find(zmanType string) (zmanEntry, bool) {
	var first *zmanEntry
	for i := range d.Zmanim {
		z := d.Zmanim[i]
		if !strings.EqualFold(z.Type, zmanType) {
			continue
		}
		if z.Default {
			return z, true
		}
		if first == nil {
			first = &d.Zmanim[i]
		}
	}
	if first != nil {
		return *first, true
	}
	return zmanEntry{}, false
}

// zmanimResult is a parsed Get_Zmanim response.
type zmanimResult struct {
	Location   string      `json:"location"`
	LocationID string      `json:"location_id"`
	Details    string      `json:"details,omitempty"`
	Days       []zmanimDay `json:"days"`
}

type rawZmanItem struct {
	ZmanType           string  `json:"ZmanType"`
	Title              string  `json:"Title"`
	Zman               string  `json:"Zman"`
	Date               string  `json:"Date"`
	Default            bool    `json:"Default"`
	OpinionDescription *string `json:"OpinionDescription"`
}

type rawZmanimDay struct {
	DisplayDate string  `json:"DisplayDate"`
	Parsha      *string `json:"Parsha"`
	HolidayName *string `json:"HolidayName"`
	IsHoliday   bool    `json:"IsHoliday"`
	IsDstActive *bool   `json:"IsDstActive"`
	TimeGroups  []struct {
		ZmanType    string        `json:"ZmanType"`
		Title       string        `json:"Title"`
		HebrewTitle string        `json:"HebrewTitle"`
		Items       []rawZmanItem `json:"Items"`
	} `json:"TimeGroups"`
	Zmanim []struct {
		EssentialZmanType string `json:"EssentialZmanType"`
		Zman              string `json:"Zman"`
	} `json:"Zmanim"`
}

type rawZmanimResponse struct {
	LocationName    string         `json:"LocationName"`
	LocationID      string         `json:"LocationId"`
	LocationDetails string         `json:"LocationDetails"`
	Days            []rawZmanimDay `json:"Days"`
}

// zmanInfo maps Chabad.org zman types to a slug and an English label.
var zmanInfo = map[string][2]string{
	"AlosHashachar":         {"dawn", "Dawn (Alot Hashachar)"},
	"EarliestTefillin":      {"misheyakir", "Earliest Tallit and Tefillin (Misheyakir)"},
	"NetzHachamah":          {"sunrise", "Sunrise (Hanetz Hachamah)"},
	"LatestShema":           {"latest-shema", "Latest Shema"},
	"LatestTefillah":        {"latest-shacharit", "Latest Shacharit"},
	"BedikatChametz":        {"bedikat-chametz", "Search for chametz"},
	"LastEatingChametzTime": {"last-eating-chametz", "Last time to eat chametz"},
	"BurnChametzTime":       {"burn-chametz", "Burn chametz by"},
	"Chatzos":               {"midday", "Midday (Chatzot Hayom)"},
	"MinchahGedolah":        {"mincha-gedolah", "Earliest Mincha (Mincha Gedolah)"},
	"MinchahKetanah":        {"mincha-ketanah", "Mincha Ketanah"},
	"PlagHaminchah":         {"plag", "Plag Hamincha"},
	"CandleLighting":        {"candle-lighting", "Candle lighting"},
	"Shkiah":                {"sunset", "Sunset (Shkiah)"},
	"Tzeis":                 {"nightfall", "Nightfall (Tzeit Hakochavim)"},
	"FastEnds":              {"fast-ends", "Fast ends (nightfall)"},
	"ShabbatEndTime":        {"shabbat-ends", "Shabbat / holiday ends"},
	"ShabbosEnds":           {"shabbat-ends", "Shabbat / holiday ends"},
	"ChatzosNight":          {"midnight", "Midnight (Chatzot HaLailah)"},
	"ShaahZmanit":           {"shaah-zmanit", "Proportional hour (Shaah Zmanit)"},
}

// zmanAliases resolves user input (slug, type, or common name) to a type.
var zmanAliases = map[string]string{
	"dawn": "AlosHashachar", "alos": "AlosHashachar", "alot": "AlosHashachar",
	"misheyakir": "EarliestTefillin", "tefillin": "EarliestTefillin", "tallit": "EarliestTefillin",
	"sunrise": "NetzHachamah", "netz": "NetzHachamah", "hanetz": "NetzHachamah",
	"latest-shema": "LatestShema", "shema": "LatestShema",
	"latest-shacharit": "LatestTefillah", "shacharit": "LatestTefillah", "tefillah": "LatestTefillah",
	"midday": "Chatzos", "chatzos": "Chatzos", "chatzot": "Chatzos", "noon": "Chatzos",
	"mincha-gedolah": "MinchahGedolah", "earliest-mincha": "MinchahGedolah", "mincha": "MinchahGedolah",
	"mincha-ketanah": "MinchahKetanah",
	"plag":           "PlagHaminchah", "plag-hamincha": "PlagHaminchah",
	"candle-lighting": "CandleLighting", "candles": "CandleLighting", "candlelighting": "CandleLighting",
	"sunset": "Shkiah", "shkiah": "Shkiah", "shkia": "Shkiah",
	"nightfall": "Tzeis", "tzeis": "Tzeis", "tzeit": "Tzeis",
	"fast-ends": "FastEnds", "fast-end": "FastEnds",
	"shabbat-ends": "ShabbatEndTime", "havdalah": "ShabbatEndTime", "shabbos-ends": "ShabbatEndTime",
	"midnight": "ChatzosNight", "chatzos-night": "ChatzosNight",
	"bedikat-chametz": "BedikatChametz", "last-eating-chametz": "LastEatingChametzTime", "burn-chametz": "BurnChametzTime",
}

func resolveZmanType(s string) (string, bool) {
	k := strings.ToLower(strings.TrimSpace(s))
	k = strings.ReplaceAll(k, " ", "-")
	if t, ok := zmanAliases[k]; ok {
		return t, true
	}
	for t := range zmanInfo {
		if strings.EqualFold(t, s) {
			return t, true
		}
	}
	return "", false
}

func zmanSlug(t string) string {
	if info, ok := zmanInfo[t]; ok {
		return info[0]
	}
	return strings.ToLower(t)
}

func zmanTitle(t string) string {
	if info, ok := zmanInfo[t]; ok {
		return info[1]
	}
	return t
}

// stripHTML turns Chabad.org snippets into plain text.
func stripHTML(s string) string {
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = reTags.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	return strings.TrimSpace(reSpaces.ReplaceAllString(stripControl(s), " "))
}

// stripControl drops control characters (except tab and newline) so remote
// text cannot carry terminal escape sequences or ICS line breaks.
func stripControl(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\t' || r == '\n' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}

// parseClock reads "5:11 AM" into minutes after midnight.
func parseClock(s string) (int, bool) {
	t, err := time.Parse("3:04 PM", strings.TrimSpace(s))
	if err != nil {
		return 0, false
	}
	return t.Hour()*60 + t.Minute(), true
}

func parseDisplayDate(s string) (time.Time, bool) {
	t, err := time.Parse("1/2/2006", strings.TrimSpace(s))
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func strPtr(p *string) string {
	if p == nil {
		return ""
	}
	return stripControl(strings.TrimSpace(*p))
}

func hebrewDateFor(day time.Time) string {
	return jcal.FromTime(day, false).String()
}

var reGMTOffset = regexp.MustCompile(`([+-]?\d+(?:\.\d+)?)\s*GMT`)

// stdOffsetMinutes reads the standard UTC offset ("-5 GMT", "+5.5 GMT") from
// Chabad.org's LocationDetails text.
func stdOffsetMinutes(details string) (int, bool) {
	m := reGMTOffset.FindStringSubmatch(stripHTML(details))
	if m == nil {
		return 0, false
	}
	h, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, false
	}
	return int(h * 60), true
}

// absoluteZman converts a location wall-clock time on day to an instant.
// (Chabad.org's own /Date()/ values are serialized in the server's time zone,
// so they are not used for instants.) Midnight after 12 AM belongs to the next
// day.
// afterMidnightZman lists night-side zman types; "ends" is the candle feed's
// Shabbat/Yom Tov end row.
var afterMidnightZman = map[string]bool{
	"ChatzosNight": true, "Tzeis": true, "FastEnds": true,
	"ShabbatEndTime": true, "ShabbosEnds": true, "ends": true,
	"candle-lighting": true, // never in the morning; after-midnight means the next date
}

func absoluteZman(day time.Time, offsetMinutes int, zmanType, clock string) (*time.Time, bool) {
	mins, ok := parseClock(clock)
	if !ok {
		return nil, false
	}
	// Night-side times printed before noon belong to the following night
	// (midnight always; nightfall/Shabbat end at high latitudes in summer).
	if afterMidnightZman[zmanType] && mins < 12*60 {
		mins += 24 * 60
	}
	at := day.Add(time.Duration(mins-offsetMinutes) * time.Minute).UTC()
	return &at, true
}

func parseZmanimResponse(data json.RawMessage, allOpinions bool) (*zmanimResult, error) {
	var raw rawZmanimResponse
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing Chabad.org zmanim: %w", err)
	}
	res := &zmanimResult{
		Location:   stripControl(raw.LocationName),
		LocationID: raw.LocationID,
		Details:    stripHTML(raw.LocationDetails),
		Days:       make([]zmanimDay, 0, len(raw.Days)),
	}
	stdOffset, hasStd := stdOffsetMinutes(raw.LocationDetails)
	for _, rd := range raw.Days {
		dt, ok := parseDisplayDate(rd.DisplayDate)
		if !ok {
			continue
		}
		day := zmanimDay{
			Date:      dt.Format("2006-01-02"),
			Weekday:   dt.Weekday().String(),
			Hebrew:    hebrewDateFor(dt),
			Reading:   strPtr(rd.Parsha),
			Holiday:   strPtr(rd.HolidayName),
			IsHoliday: rd.IsHoliday,
			Zmanim:    make([]zmanEntry, 0, 16),
			day:       dt,
		}
		// Instants need the day's DST flag, which only the detailed
		// (short-range) response carries.
		if hasStd && rd.IsDstActive != nil {
			day.offset, day.hasOffset = stdOffset, true
			if *rd.IsDstActive {
				day.offset += 60
			}
		}
		for _, g := range rd.TimeGroups {
			for i, it := range g.Items {
				if !allOpinions && len(g.Items) > 1 && !it.Default && i > 0 {
					continue
				}
				t := it.ZmanType
				if t == "" {
					t = g.ZmanType
				}
				e := zmanEntry{
					Type:        t,
					Slug:        zmanSlug(t),
					Title:       firstNonEmpty(stripHTML(it.Title), zmanTitle(t)),
					HebrewTitle: stripControl(g.HebrewTitle),
					Time:        stripControl(strings.TrimSpace(it.Zman)),
					Default:     it.Default || len(g.Items) == 1,
					Opinion:     stripHTML(strPtr(it.OpinionDescription)),
				}
				if !allOpinions {
					e.Opinion = ""
				}
				day.Zmanim = append(day.Zmanim, e)
			}
		}
		if len(rd.TimeGroups) == 0 {
			for _, z := range rd.Zmanim {
				day.Zmanim = append(day.Zmanim, zmanEntry{
					Type:    z.EssentialZmanType,
					Slug:    zmanSlug(z.EssentialZmanType),
					Title:   zmanTitle(z.EssentialZmanType),
					Time:    stripControl(strings.TrimSpace(z.Zman)),
					Default: true,
				})
			}
		}
		if !allOpinions {
			day.Zmanim = dedupeDefaults(day.Zmanim)
		}
		if day.hasOffset {
			for i := range day.Zmanim {
				if day.Zmanim[i].Type == "ShaahZmanit" {
					continue
				}
				if at, ok := absoluteZman(day.day, day.offset, day.Zmanim[i].Type, day.Zmanim[i].Time); ok {
					day.Zmanim[i].At = at
				}
			}
		}
		res.Days = append(res.Days, day)
	}
	return res, nil
}

// dedupeDefaults keeps one entry per zman type, preferring the default opinion.
func dedupeDefaults(in []zmanEntry) []zmanEntry {
	out := make([]zmanEntry, 0, len(in))
	idx := map[string]int{}
	for _, z := range in {
		if i, ok := idx[z.Type]; ok {
			if z.Default && !out[i].Default {
				out[i] = z
			}
			continue
		}
		idx[z.Type] = len(out)
		out = append(out, z)
	}
	return out
}

func firstNonEmpty(xs ...string) string {
	for _, x := range xs {
		if strings.TrimSpace(x) != "" {
			return x
		}
	}
	return ""
}

// fetchZmanim calls Get_Zmanim for [from, to], splitting long ranges.
func fetchZmanim(ctx context.Context, c *client.Client, loc chabadLocation, from, to time.Time, opt zmanimOptions) (*zmanimResult, error) {
	if to.Before(from) {
		from, to = to, from
	}
	var merged *zmanimResult
	for start := from; !start.After(to); start = start.AddDate(0, 0, maxZmanimChunkDays) {
		end := start.AddDate(0, 0, maxZmanimChunkDays-1)
		if end.After(to) {
			end = to
		}
		params := map[string]string{
			"locationid":   loc.ID,
			"locationtype": strconv.Itoa(loc.Type),
			"startdate":    start.Format("2006-01-02"),
			"enddate":      end.Format("2006-01-02"),
		}
		opt.apply(params)
		data, err := c.Get(ctx, pathGetZmanim, params)
		if err != nil {
			return nil, err
		}
		part, err := parseZmanimResponse(data, opt.AllOpinions)
		if err != nil {
			return nil, err
		}
		if merged == nil {
			merged = part
		} else {
			merged.Days = append(merged.Days, part.Days...)
		}
	}
	if merged == nil {
		merged = &zmanimResult{Days: make([]zmanimDay, 0)}
	}
	if merged.Location == "" {
		merged.Location = loc.Name
	}
	return merged, nil
}

// candleEvent is one candle-lighting or Shabbat/holiday-end row.
type candleEvent struct {
	Date                 string     `json:"date"`
	Weekday              string     `json:"weekday"`
	Hebrew               string     `json:"hebrew_date"`
	Label                string     `json:"label"`
	Kind                 string     `json:"kind"`
	Time                 string     `json:"time"`
	At                   *time.Time `json:"at,omitempty"`
	FromPreexistingFlame bool       `json:"from_preexisting_flame,omitempty"`
	Opinion              string     `json:"opinion,omitempty"`
	ShabbatMevarchim     bool       `json:"shabbat_mevarchim,omitempty"`
	OmerDay              int        `json:"omer_day,omitempty"`
}

// candleGroup is one Shabbat or holiday block.
type candleGroup struct {
	Title  string        `json:"title"`
	Type   string        `json:"type"`
	Parsha string        `json:"parsha,omitempty"`
	Also   string        `json:"also,omitempty"`
	Events []candleEvent `json:"events"`
}

// candleResult is a parsed Load_CandleLightingWeeks response.
type candleResult struct {
	Location     string        `json:"location"`
	Details      string        `json:"details,omitempty"`
	CandleOffset int           `json:"candle_offset_minutes,omitempty"`
	Groups       []candleGroup `json:"groups"`
}

type rawCandleDay struct {
	WhenToLightString         string `json:"WhenToLightString"`
	LightFromPreexistingFlame bool   `json:"LightFromPreexistingFlame"`
	PrimaryZman               struct {
		Name                string   `json:"Name"`
		CalculatedTimeZone  *float64 `json:"CalculatedTimeZone"`
		ZmanTimeString      string   `json:"ZmanTimeString"`
		ZmanDateString      string   `json:"ZmanDateString"`
		JewishDateFormatted string   `json:"JewishDateFormatted"`
		OpinionDescription  *string  `json:"OpinionDescription"`
	} `json:"PrimaryZman"`
	JewishDate struct {
		IsShabbosMevorchim bool `json:"IsShabbosMevorchim"`
		DayOfOmer          int  `json:"DayOfOmer"`
	} `json:"JewishDate"`
}

type rawCandleResponse struct {
	LocationName         string `json:"LocationName"`
	LocationDetails      string `json:"LocationDetails"`
	CandleLightingOffset int    `json:"CandleLightingOffset"`
	Groups               []struct {
		GroupHeaderTitle    string         `json:"GroupHeaderTitle"`
		GroupHeaderType     string         `json:"GroupHeaderType"`
		GroupSubHeaderTitle *string        `json:"GroupSubHeaderTitle"`
		Days                []rawCandleDay `json:"Days"`
	} `json:"Groups"`
}

func parshaFromTitle(s string) string {
	const marker = "Torah Reading:"
	if i := strings.Index(s, marker); i >= 0 {
		return strings.TrimSpace(s[i+len(marker):])
	}
	return ""
}

func parseCandleResponse(data json.RawMessage) (*candleResult, error) {
	var raw rawCandleResponse
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing Chabad.org candle-lighting times: %w", err)
	}
	res := &candleResult{
		Location:     stripControl(raw.LocationName),
		Details:      stripHTML(raw.LocationDetails),
		CandleOffset: raw.CandleLightingOffset,
		Groups:       make([]candleGroup, 0, len(raw.Groups)),
	}
	for _, g := range raw.Groups {
		sub := strPtr(g.GroupSubHeaderTitle)
		grp := candleGroup{
			Title:  stripHTML(g.GroupHeaderTitle),
			Type:   strings.ToLower(firstNonEmpty(g.GroupHeaderType, "holiday")),
			Parsha: firstNonEmpty(parshaFromTitle(g.GroupHeaderTitle), parshaFromTitle(sub)),
			Events: make([]candleEvent, 0, len(g.Days)),
		}
		if sub != "" && parshaFromTitle(sub) == "" {
			grp.Also = stripHTML(sub)
		}
		for _, d := range g.Days {
			dt, ok := parseDisplayDate(d.PrimaryZman.ZmanDateString)
			if !ok {
				continue
			}
			kind := "ends"
			if strings.Contains(strings.ToLower(d.PrimaryZman.Name), "candle") || strings.HasPrefix(strings.ToLower(d.WhenToLightString), "light") {
				kind = "candle-lighting"
			}
			ev := candleEvent{
				Date:                 dt.Format("2006-01-02"),
				Weekday:              dt.Weekday().String(),
				Hebrew:               firstNonEmpty(stripControl(d.PrimaryZman.JewishDateFormatted), hebrewDateFor(dt)),
				Label:                stripControl(strings.TrimSpace(d.WhenToLightString)),
				Kind:                 kind,
				Time:                 stripControl(strings.TrimSpace(d.PrimaryZman.ZmanTimeString)),
				FromPreexistingFlame: d.LightFromPreexistingFlame,
				Opinion:              stripHTML(strPtr(d.PrimaryZman.OpinionDescription)),
				ShabbatMevarchim:     d.JewishDate.IsShabbosMevorchim,
				OmerDay:              d.JewishDate.DayOfOmer,
			}
			// CalculatedTimeZone is the location's UTC offset (hours, DST
			// included) for this date.
			if tz := d.PrimaryZman.CalculatedTimeZone; tz != nil {
				if at, ok := absoluteZman(dt, int(*tz*60), ev.Kind, ev.Time); ok {
					ev.At = at
				}
			}
			grp.Events = append(grp.Events, ev)
		}
		res.Groups = append(res.Groups, grp)
	}
	return res, nil
}

// fetchCandles calls Load_CandleLightingWeeks.
func fetchCandles(ctx context.Context, c *client.Client, loc chabadLocation, from time.Time, weeks int, opt zmanimOptions) (*candleResult, error) {
	if weeks < 1 {
		weeks = 1
	}
	params := map[string]string{
		"locationid":   loc.ID,
		"locationtype": strconv.Itoa(loc.Type),
		"startdate":    from.Format("2006-01-02"),
		"limit":        strconv.Itoa(weeks),
	}
	opt.AllOpinions = false
	opt.apply(params)
	data, err := c.Get(ctx, pathCandleWeeks, params)
	if err != nil {
		return nil, err
	}
	res, err := parseCandleResponse(data)
	if err != nil {
		return nil, err
	}
	if res.Location == "" {
		res.Location = loc.Name
	}
	return res, nil
}

// studyItem is one daily-study track.
type studyItem struct {
	Track  string `json:"track"`
	Title  string `json:"title"`
	Lesson string `json:"lesson"`
	Link   string `json:"link"`
}

// studyDay is the Chabad.org daily study for one date.
type studyDay struct {
	Date   string      `json:"date"`
	Hebrew string      `json:"hebrew_date"`
	Items  []studyItem `json:"items"`
}

var studyTracks = []struct {
	prefix string
	track  string
}{
	{"Daily Chumash", "chumash"},
	{"Daily Tehilim - Psalms (Hebrew)", ""}, // duplicate of the English entry
	{"Daily Tehilim", "tehillim"},
	{"Daily Tanya", "tanya"},
	{"Hayom Yom", "hayom-yom"},
	{"Daily Mitzvah", "sefer-hamitzvot"},
	{"Daily Rambam - 1 Chapter", "rambam-1"},
	{"Daily Rambam - 3 Chapter", "rambam-3"},
}

var studyTrackNames = []string{"chumash", "tehillim", "tanya", "hayom-yom", "sefer-hamitzvot", "rambam-1", "rambam-3"}

func studyTrackFor(title string) string {
	for _, t := range studyTracks {
		if strings.HasPrefix(title, t.prefix) {
			return t.track
		}
	}
	return "other"
}

func cleanLesson(track, desc string) string {
	d := stripHTML(desc)
	for _, p := range []string{"Today's Lessons:", "Today's Lesson:", "Today's Psalms:", "Today's Tehillim:"} {
		d = strings.TrimSpace(strings.TrimPrefix(d, p))
	}
	if track == "hayom-yom" {
		return "Today's Hayom Yom entry"
	}
	return d
}

func parseStudyRSS(data []byte, date time.Time) (*studyDay, error) {
	var rss struct {
		Channel struct {
			Title string `xml:"title"`
			Items []struct {
				Title       string `xml:"title"`
				Link        string `xml:"link"`
				Description string `xml:"description"`
			} `xml:"item"`
		} `xml:"channel"`
	}
	if err := xml.Unmarshal(data, &rss); err != nil {
		return nil, fmt.Errorf("parsing Chabad.org daily-study feed: %w", err)
	}
	day := &studyDay{Date: date.Format("2006-01-02"), Items: make([]studyItem, 0, 8)}
	if i := strings.LastIndex(rss.Channel.Title, " - "); i >= 0 {
		day.Hebrew = strings.TrimSpace(rss.Channel.Title[i+3:])
	}
	if day.Hebrew == "" {
		day.Hebrew = hebrewDateFor(date)
	} else {
		day.Hebrew = normalizeRSSHebrew(day.Hebrew)
	}
	for _, it := range rss.Channel.Items {
		title := strings.TrimSpace(it.Title)
		track := studyTrackFor(title)
		if track == "" {
			continue
		}
		day.Items = append(day.Items, studyItem{
			Track:  track,
			Title:  stripControl(title),
			Lesson: cleanLesson(track, it.Description),
			Link:   strings.TrimSpace(it.Link),
		})
	}
	return day, nil
}

// normalizeRSSHebrew turns "Tevet 01, 5787" into "Tevet 1, 5787".
func normalizeRSSHebrew(s string) string {
	parts := strings.Fields(strings.ReplaceAll(s, ",", " "))
	if len(parts) < 3 {
		return s
	}
	dayStr := parts[len(parts)-2]
	n, err := strconv.Atoi(dayStr)
	if err != nil {
		return s
	}
	month := strings.Join(parts[:len(parts)-2], " ")
	return fmt.Sprintf("%s %d, %s", month, n, parts[len(parts)-1])
}

// fetchStudy reads the daily-study feed for one date.
func fetchStudy(ctx context.Context, c *client.Client, date time.Time) (*studyDay, error) {
	// The feed is XML; tell the generated client not to treat it as an HTML
	// error page, and ask for XML explicitly.
	headers := map[string]string{
		client.HTMLResponseHeader: "true",
		"Accept":                  "application/rss+xml, text/xml;q=0.9, */*;q=0.1",
	}
	data, err := c.GetWithHeaders(ctx, pathDailyStudyRSS, map[string]string{"tdate": date.Format("2006-01-02")}, headers)
	if err != nil {
		return nil, err
	}
	return parseStudyRSS(data, date)
}

// todayDate returns the current calendar date (UTC midnight of the local date).
func todayDate() time.Time {
	n := time.Now()
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
}

// parseDateArg reads YYYY-MM-DD (also "today"/"tomorrow").
func parseDateArg(s string) (time.Time, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "", "today":
		return todayDate(), nil
	case "tomorrow":
		return todayDate().AddDate(0, 0, 1), nil
	case "yesterday":
		return todayDate().AddDate(0, 0, -1), nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, usageErr(fmt.Errorf("invalid date %q: use YYYY-MM-DD", s))
	}
	return t, nil
}
