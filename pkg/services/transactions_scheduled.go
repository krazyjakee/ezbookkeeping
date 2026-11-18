package services

import (
	"strings"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// CreateScheduledTransactions saves all scheduled transactions that should be created now
func (s *TransactionService) CreateScheduledTransactions(c core.Context, currentUnixTime int64, interval time.Duration) error {
	var allTemplates []*models.TransactionTemplate
	intervalMinute := int(interval / time.Minute)
	currentTime := time.Unix(currentUnixTime, 0)
	currentMinute := (currentTime.Minute() / intervalMinute) * intervalMinute

	startTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), currentTime.Hour(), currentMinute, 0, 0, time.Local)
	startTimeInUTC := startTime.In(time.UTC)

	minutesElapsedOfDayInUtc := startTimeInUTC.Hour()*60 + startTimeInUTC.Minute()
	secondsElapsedOfDayInUtc := minutesElapsedOfDayInUtc * 60
	todayFirstTimeInUTC := startTimeInUTC.Add(time.Duration(-secondsElapsedOfDayInUtc) * time.Second)
	todayFirstUnixTimeInUTC := todayFirstTimeInUTC.Unix()

	minScheduledAt := minutesElapsedOfDayInUtc
	maxScheduledAt := minScheduledAt + intervalMinute

	for i := 0; i < s.UserDataDBCount(); i++ {
		var templates []*models.TransactionTemplate
		err := s.UserDataDBByIndex(i).NewSession(c).Where("deleted=?"+
			" AND template_type=?"+
			" AND (scheduled_frequency_type=? OR scheduled_frequency_type=? OR scheduled_frequency_type=? OR scheduled_frequency_type=? OR scheduled_frequency_type=?)"+
			" AND (scheduled_start_time IS NULL OR scheduled_start_time<=?)"+
			" AND (scheduled_end_time IS NULL OR scheduled_end_time>=?)"+
			" AND scheduled_at>=?"+
			" AND scheduled_at<?",
			false,
			models.TRANSACTION_TEMPLATE_TYPE_SCHEDULE,
			models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_WEEKLY, models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_MONTHLY, models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_DAILY, models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_YEARLY, models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_EVERY_N_DAYS,
			startTime.Unix(),
			startTime.Unix(),
			minScheduledAt,
			maxScheduledAt).Find(&templates)

		if err != nil {
			return err
		}

		allTemplates = append(allTemplates, templates...)
	}

	if len(allTemplates) < 1 {
		return nil
	}

	log.Infof(c, "[transactions.CreateScheduledTransactions] should process %d scheduled transaction templates now (scheduled at from %d to %d)", len(allTemplates), minScheduledAt, maxScheduledAt)

	successCount := 0
	skipCount := 0
	failedCount := 0

	for i := 0; i < len(allTemplates); i++ {
		template := allTemplates[i]
		transaction, tagIds := s.createTransactionByScheduledTransactionTemplate(c, template, startTimeInUTC, todayFirstUnixTimeInUTC)

		if transaction == nil {
			skipCount++
			continue
		}

		err := s.CreateTransaction(c, transaction, tagIds, nil)

		if err == nil {
			successCount++
			log.Infof(c, "[transactions.CreateScheduledTransactions] transaction template \"id:%d\" has created a new trasaction \"id:%d\"", template.TemplateId, transaction.TransactionId)
		} else {
			failedCount++
			log.Errorf(c, "[transactions.CreateScheduledTransactions] transaction template \"id:%d\" failed to create new trasaction, because %s", template.TemplateId, err.Error())
		}
	}

	log.Infof(c, "[transactions.CreateScheduledTransactions] %d transactions has been created successfully, %d templates does not need to create transactions and %d transactions failed to create", successCount, skipCount, failedCount)

	return nil
}

func (s *TransactionService) createTransactionByScheduledTransactionTemplate(c core.Context, template *models.TransactionTemplate, startTimeInUTC time.Time, todayFirstUnixTimeInUTC int64) (*models.Transaction, []int64) {
	if template.ScheduledFrequencyType == models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_DISABLED {
		log.Infof(c, "[transactions.CreateScheduledTransactions] transaction template \"id:%d\" disabled scheduled transaction frequency", template.TemplateId)
		return nil, nil
	}

	if (template.ScheduledFrequencyType != models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_WEEKLY &&
		template.ScheduledFrequencyType != models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_MONTHLY &&
		template.ScheduledFrequencyType != models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_DAILY &&
		template.ScheduledFrequencyType != models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_YEARLY &&
		template.ScheduledFrequencyType != models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_EVERY_N_DAYS) ||
		template.ScheduledFrequency == "" {
		log.Warnf(c, "[transactions.CreateScheduledTransactions] transaction template \"id:%d\" has invalid scheduled transaction frequency", template.TemplateId)
		return nil, nil
	}

	frequencyValues, err := utils.StringArrayToInt64Array(strings.Split(template.ScheduledFrequency, ","))

	if err != nil {
		log.Warnf(c, "[transactions.CreateScheduledTransactions] transaction template \"id:%d\" has invalid scheduled transaction frequency, because %s", template.TemplateId, err.Error())
		return nil, nil
	}

	templateTimezone := time.FixedZone("Template Timezone", int(template.ScheduledTimezoneUtcOffset)*60)

	if len(template.ScheduledTimezoneName) > 0 {
		location, err := time.LoadLocation(template.ScheduledTimezoneName)

		if err != nil {
			log.Warnf(c, "[transactions.CreateScheduledTransactions] transaction template \"id:%d\" cannot load timezone \"%s\", because %s", template.TemplateId, template.ScheduledTimezoneName, err.Error())
		} else {
			templateTimezone = location
		}
	}

	transactionTime := utils.GetScheduledTransactionFinalTime(todayFirstUnixTimeInUTC, template.ScheduledAt, template.CreatedUnixTime, templateTimezone)

	if transactionTime == nil {
		log.Warnf(c, "[transactions.CreateScheduledTransactions] transaction template \"id:%d\" does not need to create transaction, the scheduled local date does not exist", template.TemplateId)
		return nil, nil
	}

	transactionTimezoneUtcOffset := utils.GetTimezoneOffsetMinutes(transactionTime.Unix(), templateTimezone)

	if template.ScheduledFrequencyType == models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_MONTHLY {
		maxDayInMonth := utils.GetMaxDayOfMonth(transactionTime.Year(), transactionTime.Month())

		for i := 0; i < len(frequencyValues); i++ {
			if frequencyValues[i] < 0 {
				frequencyValues[i] = int64(maxDayInMonth) + frequencyValues[i] + 1
			}
		}
	}

	frequencyValueSet := utils.ToSet(frequencyValues)

	if template.ScheduledFrequencyType == models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_WEEKLY && !frequencyValueSet[int64(transactionTime.Weekday())] {
		log.Infof(c, "[transactions.CreateScheduledTransactions] transaction template \"id:%d\" does not need to create transaction, today is %s", template.TemplateId, startTimeInUTC.Weekday())
		return nil, nil
	} else if template.ScheduledFrequencyType == models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_MONTHLY && !frequencyValueSet[int64(transactionTime.Day())] {
		log.Infof(c, "[transactions.CreateScheduledTransactions] transaction template \"id:%d\" does not need to create transaction, today is %d of month", template.TemplateId, startTimeInUTC.Day())
		return nil, nil
	} else if template.ScheduledFrequencyType == models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_YEARLY && !frequencyValueSet[int64(transactionTime.Month())*100+int64(transactionTime.Day())] {
		log.Infof(c, "[transactions.CreateScheduledTransactions] transaction template \"id:%d\" does not need to create transaction, today is %d-%d of year", template.TemplateId, startTimeInUTC.Month(), startTimeInUTC.Day())
		return nil, nil
	} else if template.ScheduledFrequencyType == models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_EVERY_N_DAYS {
		if template.ScheduledStartTime == nil || len(frequencyValues) != 1 || frequencyValues[0] <= 0 {
			log.Warnf(c, "[transactions.CreateScheduledTransactions] transaction template \"id:%d\" has invalid scheduled transaction frequency for every N days", template.TemplateId)
			return nil, nil
		}

		n := frequencyValues[0]
		startDate := time.Unix(*template.ScheduledStartTime, 0).In(templateTimezone)
		daysDiff := utils.GetDaysBetweenDates(startDate, *transactionTime)

		if daysDiff < 0 || int64(daysDiff)%n != 0 {
			log.Infof(c, "[transactions.CreateScheduledTransactions] transaction template \"id:%d\" does not need to create transaction, days diff is %d with interval %d", template.TemplateId, daysDiff, n)
			return nil, nil
		}
	}

	if template.ScheduledStartTime != nil && *template.ScheduledStartTime > transactionTime.Unix() {
		log.Infof(c, "[transactions.CreateScheduledTransactions] transaction template \"id:%d\" does not need to create transaction, now is earlier than the start time %d", template.TemplateId, *template.ScheduledStartTime)
		return nil, nil
	}

	if template.ScheduledEndTime != nil && *template.ScheduledEndTime < transactionTime.Unix() {
		log.Infof(c, "[transactions.CreateScheduledTransactions] transaction template \"id:%d\" does not need to create transaction, now is later than the end time %d", template.TemplateId, *template.ScheduledEndTime)
		return nil, nil
	}

	var transactionDbType models.TransactionDbType

	if template.Type == models.TRANSACTION_TYPE_EXPENSE {
		transactionDbType = models.TRANSACTION_DB_TYPE_EXPENSE
	} else if template.Type == models.TRANSACTION_TYPE_INCOME {
		transactionDbType = models.TRANSACTION_DB_TYPE_INCOME
	} else if template.Type == models.TRANSACTION_TYPE_TRANSFER {
		transactionDbType = models.TRANSACTION_DB_TYPE_TRANSFER_OUT
	} else {
		log.Warnf(c, "[transactions.CreateScheduledTransactions] transaction template \"id:%d\" has invalid transaction type", template.TemplateId)
		return nil, nil
	}

	transaction := &models.Transaction{
		Uid:               template.Uid,
		Type:              transactionDbType,
		CategoryId:        template.CategoryId,
		TransactionTime:   utils.GetMinTransactionTimeFromUnixTime(transactionTime.Unix()),
		TimezoneUtcOffset: transactionTimezoneUtcOffset,
		AccountId:         template.AccountId,
		Amount:            template.Amount,
		HideAmount:        template.HideAmount,
		Comment:           template.Comment,
		CreatedIp:         c.ClientIP(),
		ScheduledCreated:  true,
	}

	if template.Type == models.TRANSACTION_TYPE_TRANSFER {
		transaction.RelatedAccountId = template.RelatedAccountId
		transaction.RelatedAccountAmount = template.RelatedAccountAmount
	}

	tagIds := template.GetTagIds()
	return transaction, tagIds
}
