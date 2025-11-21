package translator

import (
	"time"

	"github.com/go-playground/locales/currency"
)

func T(key any, params ...string) (string, error) {
	return Get().T(key, params...)
}
func C(key interface{}, num float64, digits uint64, param string) (string, error) {
	return Get().C(key, num, digits, param)
}
func O(key interface{}, num float64, digits uint64, param string) (string, error) {
	return Get().O(key, num, digits, param)
}
func R(key interface{}, num1, num2 float64, digits1, digits2 uint64, param1, param2 string) (string, error) {
	return Get().R(key, num1, digits1, num2, digits2, param1, param2)
}

func FmtNumber(num float64, v uint64) string {
	return Get().FmtNumber(num, v)
}
func FmtPercent(num float64, v uint64) string {
	return Get().FmtPercent(num, v)
}
func FmtCurrency(num float64, v uint64, currency currency.Type) string {
	return Get().FmtCurrency(num, v, currency)
}
func FmtAccounting(num float64, v uint64, currency currency.Type) string {
	return Get().FmtAccounting(num, v, currency)
}

func MonthWide(time time.Month) string {
	return Get().MonthWide(time)
}
func MonthAbbreviated(time time.Month) string {
	return Get().MonthAbbreviated(time)
}
func MonthNarrow(time time.Month) string {
	return Get().MonthNarrow(time)
}

func FmtDateFull(time time.Time) string {
	return Get().FmtDateFull(time)
}
func FmtDateLong(time time.Time) string {
	return Get().FmtDateLong(time)
}
func FmtDateMedium(time time.Time) string {
	return Get().FmtDateMedium(time)
}
func FmtDateShort(time time.Time) string {
	return Get().FmtDateShort(time)
}

func WeekdayWide(time time.Weekday) string {
	return Get().WeekdayWide(time)
}
func WeekdayAbbreviated(time time.Weekday) string {
	return Get().WeekdayAbbreviated(time)
}
func WeekdayShort(time time.Weekday) string {
	return Get().WeekdayShort(time)
}
func WeekdayNarrow(time time.Weekday) string {
	return Get().WeekdayNarrow(time)
}

func FmtTimeFull(time time.Time) string {
	return Get().FmtTimeFull(time)
}
func FmtTimeLong(time time.Time) string {
	return Get().FmtTimeLong(time)
}
func FmtTimeMedium(time time.Time) string {
	return Get().FmtTimeMedium(time)
}
func FmtTimeShort(time time.Time) string {
	return Get().FmtTimeShort(time)
}

func Add(key, text string, override bool) {
	Get().Add(key, text, override)
}

func AddDictionary(name string, dict map[string]string) {
	t := Get(name)
	for k, tt := range dict {
		t.Add(k, tt, false)
	}
}
