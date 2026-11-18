package services

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

func newScheduleRulesTestTemplate(frequencyType models.TransactionScheduleFrequencyType, frequency string, timezoneName string, startDate string, endDate string) *models.TransactionTemplate {
	location := time.UTC

	if timezoneName != "" {
		location, _ = time.LoadLocation(timezoneName)
	}

	createdUnixTime := time.Date(2023, 12, 15, 10, 0, 0, 0, location).Unix()

	template := &models.TransactionTemplate{
		TemplateId:                 1001,
		Uid:                        123,
		TemplateType:               models.TRANSACTION_TEMPLATE_TYPE_SCHEDULE,
		Type:                       models.TRANSACTION_TYPE_EXPENSE,
		CategoryId:                 234,
		AccountId:                  345,
		Amount:                     456,
		ScheduledFrequencyType:     frequencyType,
		ScheduledFrequency:         frequency,
		ScheduledTimezoneName:      timezoneName,
		ScheduledTimezoneUtcOffset: utils.GetTimezoneOffsetMinutes(createdUnixTime, location),
		ScheduledAt:                utils.GetScheduledTransactionScheduledAt(createdUnixTime, location),
		CreatedUnixTime:            createdUnixTime,
	}

	if startDate != "" {
		startTime, _ := utils.ParseFromLongDateFirstTimeInTimezone(startDate, location)
		startUnixTime := startTime.Unix()
		template.ScheduledStartTime = &startUnixTime
	}

	if endDate != "" {
		endTime, _ := utils.ParseFromLongDateLastTimeInTimezone(endDate, location)
		endUnixTime := endTime.Unix()
		template.ScheduledEndTime = &endUnixTime
	}

	return template
}

// getScheduleRulesTestOccurrence returns whether the helper says the template fires on the utc day
func getScheduleRulesTestOccurrence(template *models.TransactionTemplate, todayFirstUnixTimeInUTC int64) (bool, *time.Time) {
	location := GetScheduledTransactionTemplateLocation(template)
	transactionTime := utils.GetScheduledTransactionFinalTime(todayFirstUnixTimeInUTC, template.ScheduledAt, template.CreatedUnixTime, location)

	if transactionTime == nil {
		return false, nil
	}

	frequencyValues, err := utils.StringArrayToInt64Array(strings.Split(template.ScheduledFrequency, ","))

	if err != nil {
		return false, nil
	}

	return IsScheduledTransactionTemplateOccurrence(template, frequencyValues, *transactionTime, location), transactionTime
}

// assertScheduleRulesMatchScheduler asserts the helper returns the same result as the scheduler for every day in the range, and returns the local dates of the occurrences
func assertScheduleRulesMatchScheduler(t *testing.T, template *models.TransactionTemplate, fromDate time.Time, days int) []string {
	occurrenceDates := make([]string, 0)
	location := GetScheduledTransactionTemplateLocation(template)

	for i := 0; i < days; i++ {
		today := fromDate.AddDate(0, 0, i)
		todayFirstUnixTimeInUTC := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC).Unix()
		startTimeInUTC := time.Unix(todayFirstUnixTimeInUTC+int64(template.ScheduledAt)*60, 0).UTC()

		transaction, _ := Transactions.createTransactionByScheduledTransactionTemplate(core.NewNullContext(), template, startTimeInUTC, todayFirstUnixTimeInUTC)
		occurrence, transactionTime := getScheduleRulesTestOccurrence(template, todayFirstUnixTimeInUTC)

		assert.Equal(t, transaction != nil, occurrence, "frequency type %d \"%s\" on %s", template.ScheduledFrequencyType, template.ScheduledFrequency, today.Format("2006-01-02"))

		if transaction != nil && transactionTime != nil {
			assert.Equal(t, utils.GetMinTransactionTimeFromUnixTime(transactionTime.Unix()), transaction.TransactionTime)
		}

		if occurrence {
			occurrenceDates = append(occurrenceDates, transactionTime.In(location).Format("2006-01-02"))
		}
	}

	return occurrenceDates
}

func TestIsScheduledTransactionTemplateOccurrence_MatchesScheduler(t *testing.T) {
	fromDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	timezones := []string{"", "Asia/Shanghai", "America/New_York", "Pacific/Kiritimati"}
	cases := []struct {
		frequencyType models.TransactionScheduleFrequencyType
		frequency     string
		startDate     string
		endDate       string
	}{
		{models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_DAILY, "0", "", ""},
		{models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_DAILY, "0", "2024-03-10", "2024-11-03"},
		{models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_WEEKLY, "0,3,6", "", ""},
		{models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_WEEKLY, "1", "2024-02-01", "2024-08-31"},
		{models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_MONTHLY, "1,15,31", "", ""},
		{models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_MONTHLY, "-1", "", ""},
		{models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_MONTHLY, "-3,-31,29", "2024-02-01", "2025-03-31"},
		{models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_YEARLY, "0229,1225", "", ""},
		{models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_YEARLY, "0101", "2024-06-01", ""},
		{models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_EVERY_N_DAYS, "3", "2024-01-05", ""},
		{models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_EVERY_N_DAYS, "10", "2024-02-20", "2024-12-31"},
		{models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_EVERY_N_DAYS, "7", "", ""},
		{models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_EVERY_N_DAYS, "2,3", "2024-01-01", ""},
	}

	for _, timezone := range timezones {
		for _, testCase := range cases {
			template := newScheduleRulesTestTemplate(testCase.frequencyType, testCase.frequency, timezone, testCase.startDate, testCase.endDate)
			assertScheduleRulesMatchScheduler(t, template, fromDate, 400)
		}
	}
}

func TestIsScheduledTransactionTemplateOccurrence_Weekly(t *testing.T) {
	template := newScheduleRulesTestTemplate(models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_WEEKLY, "1,5", "", "", "")
	dates := assertScheduleRulesMatchScheduler(t, template, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 14)
	assert.Equal(t, []string{"2024-01-01", "2024-01-05", "2024-01-08", "2024-01-12"}, dates)
}

func TestIsScheduledTransactionTemplateOccurrence_MonthlyNegativeDays(t *testing.T) {
	template := newScheduleRulesTestTemplate(models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_MONTHLY, "-1", "", "", "")
	dates := assertScheduleRulesMatchScheduler(t, template, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 121)
	assert.Equal(t, []string{"2024-01-31", "2024-02-29", "2024-03-31", "2024-04-30"}, dates)

	template = newScheduleRulesTestTemplate(models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_MONTHLY, "-30", "", "", "")
	dates = assertScheduleRulesMatchScheduler(t, template, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 91)
	assert.Equal(t, []string{"2024-01-02", "2024-03-02"}, dates)
}

func TestIsScheduledTransactionTemplateOccurrence_Yearly(t *testing.T) {
	template := newScheduleRulesTestTemplate(models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_YEARLY, "0229", "", "", "")
	dates := assertScheduleRulesMatchScheduler(t, template, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 1100)
	assert.Equal(t, []string{"2024-02-29"}, dates)
}

func TestIsScheduledTransactionTemplateOccurrence_EveryNDays(t *testing.T) {
	template := newScheduleRulesTestTemplate(models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_EVERY_N_DAYS, "10", "", "2024-01-05", "2024-01-31")
	dates := assertScheduleRulesMatchScheduler(t, template, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 60)
	assert.Equal(t, []string{"2024-01-05", "2024-01-15", "2024-01-25"}, dates)

	template.ScheduledStartTime = nil
	occurrence, _ := getScheduleRulesTestOccurrence(template, time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC).Unix())
	assert.False(t, occurrence)
}

func TestIsScheduledTransactionTemplateOccurrence_DailyWithStartAndEnd(t *testing.T) {
	template := newScheduleRulesTestTemplate(models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_DAILY, "0", "", "2024-01-03", "2024-01-05")
	dates := assertScheduleRulesMatchScheduler(t, template, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 10)
	assert.Equal(t, []string{"2024-01-03", "2024-01-04", "2024-01-05"}, dates)
}

func TestIsScheduledTransactionTemplateOccurrence_InvalidFrequency(t *testing.T) {
	transactionTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	template := newScheduleRulesTestTemplate(models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_DISABLED, "1", "", "", "")
	assert.False(t, IsScheduledTransactionTemplateOccurrence(template, []int64{1}, transactionTime, time.UTC))

	template.ScheduledFrequencyType = models.TransactionScheduleFrequencyType(255)
	assert.False(t, IsScheduledTransactionTemplateOccurrence(template, []int64{1}, transactionTime, time.UTC))

	template.ScheduledFrequencyType = models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_DAILY
	assert.False(t, IsScheduledTransactionTemplateOccurrence(template, nil, transactionTime, time.UTC))
}

func TestIsScheduledTransactionTemplateOccurrence_DoesNotModifyFrequencyValues(t *testing.T) {
	template := newScheduleRulesTestTemplate(models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_MONTHLY, "-1", "", "", "")
	frequencyValues := []int64{-1}
	assert.True(t, IsScheduledTransactionTemplateOccurrence(template, frequencyValues, time.Date(2024, 1, 31, 10, 0, 0, 0, time.UTC), time.UTC))
	assert.Equal(t, []int64{-1}, frequencyValues)
	assert.True(t, IsScheduledTransactionTemplateOccurrence(template, frequencyValues, time.Date(2024, 2, 29, 10, 0, 0, 0, time.UTC), time.UTC))
}

func TestGetScheduledTransactionTemplateLocation(t *testing.T) {
	template := &models.TransactionTemplate{ScheduledTimezoneUtcOffset: 480}
	location := GetScheduledTransactionTemplateLocation(template)
	_, offset := time.Date(2024, 1, 1, 0, 0, 0, 0, location).Zone()
	assert.Equal(t, 480*60, offset)

	template.ScheduledTimezoneName = "Europe/London"
	assert.Equal(t, "Europe/London", GetScheduledTransactionTemplateLocation(template).String())

	template.ScheduledTimezoneName = "Invalid/Timezone"
	_, offset = time.Date(2024, 1, 1, 0, 0, 0, 0, GetScheduledTransactionTemplateLocation(template)).Zone()
	assert.Equal(t, 480*60, offset)
}
