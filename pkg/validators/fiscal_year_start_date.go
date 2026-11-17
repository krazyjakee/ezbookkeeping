package validators

import (
	"github.com/go-playground/validator/v10"

	"github.com/mayswind/ezbookkeeping/pkg/core"
)

// ValidateFiscalYearStart validates if a fiscal year start date is valid
func ValidateFiscalYearStart(fl validator.FieldLevel) bool {
	date, ok := fl.Field().Interface().(core.FiscalYearStart)
	if !ok {
		return false
	}

	_, _, err := date.GetMonthDay()
	return err == nil
}
