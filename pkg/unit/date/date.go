package date

import (
	"fmt"
	"time"
)

func GetNowDay() string {
	return time.Now().Format("2006-01-02")
}

// GetBeforeDay 返回多少天之前的时间 固定格式 2006-01-02
func GetBeforeDay(i int) string {
	return time.Now().AddDate(0, 0, i).Format("2006-01-02")
}

// GetBeforeMonth 返回多少月之前的时间 固定格式 2006-01-02
func GetBeforeMonth(i int) string {
	return time.Now().AddDate(0, i, 0).Format("2006-01-02")
}

// GetDateBeforeMonth 返回特指日期多少月之前的时间 固定格式 2006-01-02
func GetDateBeforeMonth(date string, i int) (string, error) {
	layout := "2006-01-02"
	t, err := time.Parse(layout, date)
	if err != nil {
		return "", err
	}
	return t.AddDate(0, i, 0).Format("2006-01-02"), nil
}

// GetDateBeforeYear 返回特指日期多少月之前的时间 固定格式 2006-01-02
func GetDateBeforeYear(date string, i int) (string, error) {
	layout := "2006-01-02"
	t, err := time.Parse(layout, date)
	if err != nil {
		return "", err
	}
	return t.AddDate(i, 0, 0).Format("2006-01-02"), nil
}

// IfDateBigNow 判断时间在当前时间之前
func IfDateBigNow(dateStart string) (bool, error) {
	layout := "2006-01-02"
	t, err := time.Parse(layout, dateStart)
	if err != nil {
		return false, err
	}
	return time.Now().After(t), nil
}

// ChangeDateLayout date layout change: "02/01/2006" -> "2006-01-02"
func ChangeDateLayout(oldTime string) (string, error) {
	layout := "02/01/2006"
	t, err := time.Parse(layout, oldTime)
	if err != nil {
		return "", fmt.Errorf("Failed ChangeDateLayout error %s ", err.Error())
	}
	return t.Format("2006-01-02"), nil
}

// GetDateSubPeriodUnit 两个日期之间相差单位. "2022-10-01", "2023-10-1" -> year
func GetDateSubPeriodUnit(dateStart, dateEnd string) (string, error) {
	layout := "2006-01-02"
	t0, err := time.Parse(layout, dateStart)
	if err != nil {
		return "", err
	}
	t1, err := time.Parse(layout, dateEnd)
	if err != nil {
		return "", err
	}
	dateDiffer := t1.Sub(t0)
	if dateDiffer > time.Hour*24*360 {
		return "year", nil
	}
	if dateDiffer >= time.Hour*24*28 {
		return "month", nil
	}
	return "", fmt.Errorf("unrecognized time period")
}

// GetDateSubPeriodDays 两个时间之间相差天数.
func GetDateSubPeriodDays(dateStart, dateEnd string) (float64, error) {
	var day float64
	layout := "2006-01-02"
	t0, err := time.Parse(layout, dateStart)
	if err != nil {
		return day, err
	}
	t1, err := time.Parse(layout, dateEnd)
	if err != nil {
		return day, err
	}
	day = t1.Sub(t0).Hours() / 24
	return day, nil
}

// GetDateRangeYearToMonth 获取时间期间的月份
func GetDateRangeYearToMonth(dateStart, dateEnd string) ([]string, error) {
	var dateRangeMonth []string
	layout := "2006-01-02"
	t0, err := time.Parse(layout, dateStart)
	if err != nil {
		return dateRangeMonth, err
	}
	t1, err := time.Parse(layout, dateEnd)
	if err != nil {
		return dateRangeMonth, err
	}

	dateRangeMonth = append(dateRangeMonth, dateStart)
	for d := t0.AddDate(0, 1, 0); d.After(t1) == false; d = d.AddDate(0, 1, 0) {
		dateRangeMonth = append(dateRangeMonth, fmt.Sprintf("%s%s", d.Format("2006-01"), "-01"))
	}
	dateRangeMonth = append(dateRangeMonth, dateEnd)
	return dateRangeMonth, nil
}

// GetDateRangeToDay 获取日期范围日期 list. req: ("2022-10-02", "2022-12-02")
func GetDateRangeToDay(dateStart, dateEnd string) []string {
	var dateList []string
	layout := "2006-01-02"
	start, _ := time.Parse(layout, dateStart)
	end, _ := time.Parse(layout, dateEnd)
	for d := start; d.After(end) == false; d = d.AddDate(0, 0, 1) {
		d.Format("2006-01-02")
		dateList = append(dateList, d.Format("2006-01-02"))
	}
	return dateList
}
