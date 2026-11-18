package services

import (
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// GetScheduledTransactionTemplateLocation returns the timezone which the scheduler uses for the scheduled transaction template
// (the IANA timezone of the template if it can be loaded, otherwise the fixed utc offset of the template)
func GetScheduledTransactionTemplateLocation(template *models.TransactionTemplate) *time.Location {
	location := time.FixedZone("Template Timezone", int(template.ScheduledTimezoneUtcOffset)*60)

	if len(template.ScheduledTimezoneName) > 0 {
		if loadedLocation, err := time.LoadLocation(template.ScheduledTimezoneName); err == nil {
			location = loadedLocation
		}
	}

	return location
}

// IsScheduledTransactionTemplateOccurrence returns whether the scheduler creates a transaction of the scheduled transaction template
// at the specified transaction time, i.e. whether the local date of the transaction time matches the frequency of the template and
// the transaction time is within the start time and the end time (both inclusive) of the template.
// The transaction time must be the final transaction time returned by utils.GetScheduledTransactionFinalTime in the template timezone,
// and frequencyValues must be the parsed values of the scheduled frequency of the template (they are not modified).
// This is a pure function, it does not access the database.
func IsScheduledTransactionTemplateOccurrence(template *models.TransactionTemplate, frequencyValues []int64, transactionTime time.Time, location *time.Location) bool {
	if len(frequencyValues) < 1 {
		return false
	}

	matched := false

	switch template.ScheduledFrequencyType {
	case models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_DAILY:
		matched = true
	case models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_WEEKLY:
		for i := 0; i < len(frequencyValues); i++ {
			if frequencyValues[i] == int64(transactionTime.Weekday()) {
				matched = true
				break
			}
		}
	case models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_MONTHLY:
		maxDayInMonth := int64(utils.GetMaxDayOfMonth(transactionTime.Year(), transactionTime.Month()))

		for i := 0; i < len(frequencyValues); i++ {
			value := frequencyValues[i]

			if value < 0 {
				value = maxDayInMonth + value + 1
			}

			if value == int64(transactionTime.Day()) {
				matched = true
				break
			}
		}
	case models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_YEARLY:
		for i := 0; i < len(frequencyValues); i++ {
			if frequencyValues[i] == int64(transactionTime.Month())*100+int64(transactionTime.Day()) {
				matched = true
				break
			}
		}
	case models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_EVERY_N_DAYS:
		if template.ScheduledStartTime == nil || len(frequencyValues) != 1 || frequencyValues[0] <= 0 {
			return false
		}

		startDate := time.Unix(*template.ScheduledStartTime, 0).In(location)
		daysDiff := utils.GetDaysBetweenDates(startDate, transactionTime)
		matched = daysDiff >= 0 && int64(daysDiff)%frequencyValues[0] == 0
	default:
		return false
	}

	if !matched {
		return false
	}

	if template.ScheduledStartTime != nil && *template.ScheduledStartTime > transactionTime.Unix() {
		return false
	}

	if template.ScheduledEndTime != nil && *template.ScheduledEndTime < transactionTime.Unix() {
		return false
	}

	return true
}
