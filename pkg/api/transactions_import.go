package api

import (
	"encoding/json/v2"
	"io"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/converters"
	"github.com/mayswind/ezbookkeeping/pkg/converters/converter"
	"github.com/mayswind/ezbookkeeping/pkg/converters/datatable"
	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/duplicatechecker"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

const transactionImportResultExpiration = 1 * time.Hour

// TransactionParseImportCustomFileDataHandler returns the parsed file data by request parameters for current user
func (a *TransactionsApi) TransactionParseImportCustomFileDataHandler(c *core.WebContext) (any, *errs.Error) {
	uid := c.GetCurrentUid()
	form, err := c.MultipartForm()

	if err != nil {
		log.Errorf(c, "[transactions.TransactionParseImportCustomFileDataHandler] failed to get multi-part form data for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.ErrParameterInvalid
	}

	fileTypes := form.Value["fileType"]

	if len(fileTypes) < 1 || fileTypes[0] == "" {
		return nil, errs.ErrImportFileTypeIsEmpty
	}

	fileType := fileTypes[0]

	if !converters.IsCustomFileFormatFileType(fileType) {
		return nil, errs.Or(err, errs.ErrImportFileTypeNotSupported)
	}

	fileEncodings := form.Value["fileEncoding"]
	fileEncoding := ""

	if len(fileEncodings) > 0 {
		fileEncoding = fileEncodings[0]
	}

	dataParser, err := converters.CreateNewCustomFileFormatTransactionDataParser(fileType, fileEncoding)

	if err != nil {
		return nil, errs.Or(err, errs.ErrImportFileTypeNotSupported)
	}

	importFiles := form.File["file"]

	if len(importFiles) < 1 {
		log.Warnf(c, "[transactions.TransactionParseImportCustomFileDataHandler] there is no import file in request for user \"uid:%d\"", uid)
		return nil, errs.ErrNoFilesUpload
	}

	if importFiles[0].Size < 1 {
		log.Warnf(c, "[transactions.TransactionParseImportCustomFileDataHandler] the size of import file in request is zero for user \"uid:%d\"", uid)
		return nil, errs.ErrUploadedFileEmpty
	}

	if importFiles[0].Size > int64(a.CurrentConfig().MaxImportFileSize) {
		log.Warnf(c, "[transactions.TransactionParseImportCustomFileDataHandler] the upload file size \"%d\" exceeds the maximum size \"%d\" of import file for user \"uid:%d\"", importFiles[0].Size, a.CurrentConfig().MaxImportFileSize, uid)
		return nil, errs.ErrExceedMaxUploadFileSize
	}

	importFile, err := importFiles[0].Open()

	if err != nil {
		log.Errorf(c, "[transactions.TransactionParseImportCustomFileDataHandler] failed to get import file from request for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.ErrOperationFailed
	}

	defer importFile.Close()
	fileData, err := io.ReadAll(importFile)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionParseImportCustomFileDataHandler] failed to read import file data for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	allLines, err := dataParser.ParseDataLines(c, fileData)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionParseImportCustomFileDataHandler] failed to parse import file data for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	return allLines, nil
}

// TransactionParseImportFileHandler returns the parsed transaction data by request parameters for current user
func (a *TransactionsApi) TransactionParseImportFileHandler(c *core.WebContext) (any, *errs.Error) {
	uid := c.GetCurrentUid()
	form, err := c.MultipartForm()

	if err != nil {
		log.Errorf(c, "[transactions.TransactionParseImportFileHandler] failed to get multi-part form data for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.ErrParameterInvalid
	}

	clientTimezone, err := c.GetClientTimezone()

	if err != nil {
		log.Warnf(c, "[transactions.TransactionParseImportFileHandler] cannot get client timezone, because %s", err.Error())
		return nil, errs.ErrClientTimezoneOffsetInvalid
	}

	fileTypes := form.Value["fileType"]

	if len(fileTypes) < 1 || fileTypes[0] == "" {
		return nil, errs.ErrImportFileTypeIsEmpty
	}

	fileType := fileTypes[0]

	textualOptions := form.Value["options"]
	textualOption := ""

	if len(textualOptions) > 0 {
		textualOption = textualOptions[0]
	}

	additionalOptions := converter.ParseImporterOptions(a.CurrentConfig(), textualOption)

	if fileType == "ai_txt" || fileType == "ai_image" {
		aiAdditionalPrompts := form.Value["aiPrompt"]
		aiAdditionalPrompt := ""

		if len(aiAdditionalPrompts) > 0 {
			aiAdditionalPrompt = aiAdditionalPrompts[0]
			additionalOptions = additionalOptions.WithAIAdditionalPrompt(aiAdditionalPrompt)
		}
	}

	var dataImporter converter.TransactionDataImporter

	if converters.IsCustomFileFormatFileType(fileType) {
		fileEncodings := form.Value["fileEncoding"]
		fileEncoding := ""

		if len(fileEncodings) > 0 {
			fileEncoding = fileEncodings[0]
		}

		columnMappings := form.Value["columnMapping"]

		if len(columnMappings) < 1 || columnMappings[0] == "" {
			return nil, errs.ErrImportFileColumnMappingInvalid
		}

		var columnIndexMapping = map[datatable.TransactionDataTableColumn]int{}
		err = json.Unmarshal([]byte(columnMappings[0]), &columnIndexMapping)

		if err != nil {
			log.Errorf(c, "[transactions.TransactionParseImportFileHandler] failed to parse column mapping for user \"uid:%d\", because %s", uid, err.Error())
			return nil, errs.ErrImportFileColumnMappingInvalid
		}

		transactionTypeMappings := form.Value["transactionTypeMapping"]

		if len(transactionTypeMappings) < 1 || transactionTypeMappings[0] == "" {
			return nil, errs.ErrImportFileTransactionTypeMappingInvalid
		}

		var transactionTypeNameMapping = map[string]models.TransactionType{}
		err = json.Unmarshal([]byte(transactionTypeMappings[0]), &transactionTypeNameMapping)

		if err != nil {
			log.Errorf(c, "[transactions.TransactionParseImportFileHandler] failed to parse transaction type mapping for user \"uid:%d\", because %s", uid, err.Error())
			return nil, errs.ErrImportFileTransactionTypeMappingInvalid
		}

		hasHeaderLines := form.Value["hasHeaderLine"]
		hasHeaderLine := false

		if len(hasHeaderLines) > 0 {
			hasHeaderLine = hasHeaderLines[0] == "true"
		}

		timeFormats := form.Value["timeFormat"]

		if len(timeFormats) < 1 || timeFormats[0] == "" {
			return nil, errs.ErrImportFileTransactionTimeFormatInvalid
		}

		timezoneFormats := form.Value["timezoneFormat"]
		timezoneFormat := ""

		if len(timezoneFormats) > 0 {
			timezoneFormat = timezoneFormats[0]
		}

		amountDecimalSeparators := form.Value["amountDecimalSeparator"]
		amountDecimalSeparator := ""

		if len(amountDecimalSeparators) > 0 {
			amountDecimalSeparator = amountDecimalSeparators[0]
		}

		amountDigitGroupingSymbols := form.Value["amountDigitGroupingSymbol"]
		amountDigitGroupingSymbol := ""

		if len(amountDigitGroupingSymbols) > 0 {
			amountDigitGroupingSymbol = amountDigitGroupingSymbols[0]
		}

		geoLocationSeparators := form.Value["geoSeparator"]
		geoLocationSeparator := ""

		if len(geoLocationSeparators) > 0 {
			geoLocationSeparator = geoLocationSeparators[0]
		}

		geoLocationOrders := form.Value["geoOrder"]
		geoLocationOrder := ""

		if len(geoLocationOrders) > 0 {
			geoLocationOrder = geoLocationOrders[0]
		}

		transactionTagSeparators := form.Value["tagSeparator"]
		transactionTagSeparator := ""

		if len(transactionTagSeparators) > 0 {
			transactionTagSeparator = transactionTagSeparators[0]
		}

		dataImporter, err = converters.CreateNewCustomTransactionDataImporter(fileType, fileEncoding, columnIndexMapping, transactionTypeNameMapping, hasHeaderLine, timeFormats[0], timezoneFormat, amountDecimalSeparator, amountDigitGroupingSymbol, geoLocationSeparator, geoLocationOrder, transactionTagSeparator)
	} else {
		dataImporter, err = converters.GetTransactionDataImporter(fileType)
	}

	if err != nil {
		return nil, errs.Or(err, errs.ErrImportFileTypeNotSupported)
	}

	if fileType == "ai_txt" && (a.CurrentConfig().TextRecognitionLLMConfig == nil || a.CurrentConfig().TextRecognitionLLMConfig.LLMProvider == "" || !a.CurrentConfig().TransactionFromAITextRecognition) {
		return nil, errs.ErrLargeLanguageModelProviderNotEnabled
	}

	if fileType == "ai_image" && (a.CurrentConfig().ReceiptImageRecognitionLLMConfig == nil || a.CurrentConfig().ReceiptImageRecognitionLLMConfig.LLMProvider == "" || !a.CurrentConfig().TransactionFromAIImageRecognition) {
		return nil, errs.ErrLargeLanguageModelProviderNotEnabled
	}

	importFiles := form.File["file"]

	if len(importFiles) < 1 {
		log.Warnf(c, "[transactions.TransactionParseImportFileHandler] there is no import file in request for user \"uid:%d\"", uid)
		return nil, errs.ErrNoFilesUpload
	}

	if importFiles[0].Size < 1 {
		log.Warnf(c, "[transactions.TransactionParseImportFileHandler] the size of import file in request is zero for user \"uid:%d\"", uid)
		return nil, errs.ErrUploadedFileEmpty
	}

	maxImportFileSize := int64(a.CurrentConfig().MaxImportFileSize)

	if fileType == "ai_image" {
		maxImportFileSize = int64(a.CurrentConfig().MaxAIRecognitionPictureFileSize)
	}

	if importFiles[0].Size > maxImportFileSize {
		log.Warnf(c, "[transactions.TransactionParseImportFileHandler] the upload file size \"%d\" exceeds the maximum size \"%d\" of import file for user \"uid:%d\"", importFiles[0].Size, maxImportFileSize, uid)
		return nil, errs.ErrExceedMaxUploadFileSize
	}

	if fileType == "ai_image" {
		fileExtension := utils.GetFileNameExtension(importFiles[0].Filename)
		contentType := utils.GetImageContentType(fileExtension)

		if contentType == "" {
			log.Warnf(c, "[transactions.TransactionParseImportFileHandler] the file extension \"%s\" of image in request is not supported for user \"uid:%d\"", fileExtension, uid)
			return nil, errs.ErrImageTypeNotSupported
		}

		additionalOptions = additionalOptions.WithAIImageContentType(contentType)
	}

	importFile, err := importFiles[0].Open()

	if err != nil {
		log.Errorf(c, "[transactions.TransactionParseImportFileHandler] failed to get import file from request for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.ErrOperationFailed
	}

	defer importFile.Close()
	fileData, err := io.ReadAll(importFile)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionParseImportFileHandler] failed to read import file data for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	user, err := a.users.GetUserById(c, uid)

	if err != nil {
		if !errs.IsCustomError(err) {
			log.Errorf(c, "[transactions.TransactionParseImportFileHandler] failed to get user, because %s", err.Error())
		}

		return nil, errs.ErrUserNotFound
	}

	if user.FeatureRestriction.Contains(core.USER_FEATURE_RESTRICTION_TYPE_IMPORT_TRANSACTION) {
		return nil, errs.ErrNotPermittedToPerformThisAction
	}

	if fileType == "ai_txt" && user.FeatureRestriction.Contains(core.USER_FEATURE_RESTRICTION_TYPE_CREATE_TRANSACTION_FROM_AI_TEXT_RECOGNITION) {
		return nil, errs.ErrNotPermittedToPerformThisAction
	}

	if fileType == "ai_image" && user.FeatureRestriction.Contains(core.USER_FEATURE_RESTRICTION_TYPE_CREATE_TRANSACTION_FROM_AI_IMAGE_RECOGNITION) {
		return nil, errs.ErrNotPermittedToPerformThisAction
	}

	accounts, err := a.accounts.GetAllAccountsByUid(c, user.Uid)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionParseImportFileHandler] failed to get accounts for user \"uid:%d\", because %s", user.Uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	accountMap := a.accounts.GetVisibleAccountNameMapByList(accounts)

	categories, err := a.transactionCategories.GetAllCategoriesByUid(c, user.Uid, 0, -1)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionParseImportFileHandler] failed to get categories for user \"uid:%d\", because %s", user.Uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	expenseCategoryMap, incomeCategoryMap, transferCategoryMap := a.transactionCategories.GetVisibleSubCategoryNameMapByList(categories)

	tags, err := a.transactionTags.GetAllTagsByUid(c, user.Uid)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionParseImportFileHandler] failed to get tags for user \"uid:%d\", because %s", user.Uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	tagMap := a.transactionTags.GetVisibleTagNameMapByList(tags)

	parsedTransactions, _, _, _, _, _, err := dataImporter.ParseImportedData(c, user, fileData, clientTimezone, additionalOptions, accountMap, expenseCategoryMap, incomeCategoryMap, transferCategoryMap, tagMap)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionParseImportFileHandler] failed to parse imported data for user \"uid:%d\", because %s", user.Uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	parsedTransactionRespsList := parsedTransactions.ToImportTransactionResponseList()

	if len(parsedTransactionRespsList) < 1 {
		return nil, errs.ErrNoDataToImport
	}

	parsedTransactionResps := &models.ImportTransactionResponsePageWrapper{
		Items:      parsedTransactionRespsList,
		TotalCount: int64(len(parsedTransactionRespsList)),
	}

	return parsedTransactionResps, nil
}

// TransactionImportHandler imports transactions by request parameters for current user
func (a *TransactionsApi) TransactionImportHandler(c *core.WebContext) (any, *errs.Error) {
	var transactionImportReq models.TransactionImportRequest
	err := c.ShouldBindJSON(&transactionImportReq)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionImportHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	clientTimezone, err := c.GetClientTimezone()

	if err != nil {
		log.Warnf(c, "[transactions.TransactionImportHandler] cannot get client timezone, because %s", err.Error())
		return nil, errs.ErrClientTimezoneOffsetInvalid
	}

	uid := c.GetCurrentUid()

	if a.CurrentConfig().EnableDuplicateSubmissionsCheck && transactionImportReq.ClientSessionId != "" {
		found, remark := a.GetSubmissionRemark(duplicatechecker.DUPLICATE_CHECKER_TYPE_IMPORT_TRANSACTIONS, uid, transactionImportReq.ClientSessionId)

		if found {
			var result *models.TransactionImportResponse

			if json.Unmarshal([]byte(remark), &result) == nil && result != nil && result.Finished {
				return a.getTransactionImportResponse(result)
			}

			return nil, errs.ErrRepeatedRequest
		}
	}

	newTransactionTagIdsMap := make(map[int][]int64, len(transactionImportReq.Transactions))

	for i := 0; i < len(transactionImportReq.Transactions); i++ {
		transactionCreateReq := transactionImportReq.Transactions[i]
		tagIds, err := utils.StringArrayToInt64Array(transactionCreateReq.TagIds)

		if err != nil {
			log.Warnf(c, "[transactions.TransactionImportHandler] parse tag ids failed of transaction \"index:%d\", because %s", i, err.Error())
			return nil, errs.ErrTransactionTagIdInvalid
		}

		if len(tagIds) > models.MaximumTagsCountOfTransaction {
			return nil, errs.ErrTransactionHasTooManyTags
		}

		if transactionCreateReq.Type < models.TRANSACTION_TYPE_MODIFY_BALANCE || transactionCreateReq.Type > models.TRANSACTION_TYPE_TRANSFER {
			log.Warnf(c, "[transactions.TransactionImportHandler] transaction type of transaction \"index:%d\" is invalid", i)
			return nil, errs.ErrTransactionTypeInvalid
		}

		if transactionCreateReq.Type == models.TRANSACTION_TYPE_MODIFY_BALANCE && transactionCreateReq.CategoryId != 0 {
			log.Warnf(c, "[transactions.TransactionImportHandler] balance modification transaction \"index:%d\" cannot set category id", i)
			return nil, errs.ErrBalanceModificationTransactionCannotSetCategory
		}

		if transactionCreateReq.Type != models.TRANSACTION_TYPE_TRANSFER && transactionCreateReq.DestinationAccountId != 0 {
			log.Warnf(c, "[transactions.TransactionImportHandler] non-transfer transaction \"index:%d\" destination account cannot be set", i)
			return nil, errs.ErrTransactionDestinationAccountCannotBeSet
		} else if transactionCreateReq.Type == models.TRANSACTION_TYPE_TRANSFER && transactionCreateReq.SourceAccountId == transactionCreateReq.DestinationAccountId {
			log.Warnf(c, "[transactions.TransactionImportHandler] transfer transaction \"index:%d\" source account must not be destination account", i)
			return nil, errs.ErrTransactionSourceAndDestinationIdCannotBeEqual
		}

		if transactionCreateReq.Type != models.TRANSACTION_TYPE_TRANSFER && transactionCreateReq.DestinationAmount != 0 {
			log.Warnf(c, "[transactions.TransactionImportHandler] non-transfer transaction \"index:%d\" destination amount cannot be set", i)
			return nil, errs.ErrTransactionDestinationAmountCannotBeSet
		}

		newTransactionTagIdsMap[i] = tagIds
	}

	user, err := a.users.GetUserById(c, uid)

	if err != nil {
		if !errs.IsCustomError(err) {
			log.Errorf(c, "[transactions.TransactionImportHandler] failed to get user, because %s", err.Error())
		}

		return nil, errs.ErrUserNotFound
	}

	if user.FeatureRestriction.Contains(core.USER_FEATURE_RESTRICTION_TYPE_IMPORT_TRANSACTION) {
		return nil, errs.ErrNotPermittedToPerformThisAction
	}

	newTransactions := make([]*models.Transaction, len(transactionImportReq.Transactions))

	for i := 0; i < len(transactionImportReq.Transactions); i++ {
		transactionCreateReq := transactionImportReq.Transactions[i]
		transaction := a.createNewTransactionModel(uid, transactionCreateReq, c.ClientIP())
		newTransactions[i] = transaction
	}

	allUsedAccounts, err := a.getTransactionUsedAccounts(c, uid, newTransactions)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionImportHandler] failed to get transaction used accounts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	for i := 0; i < len(newTransactions); i++ {
		transaction := newTransactions[i]
		transactionEditable := user.CanEditTransactionByTransactionTime(transaction.TransactionTime, clientTimezone, allUsedAccounts[transaction.AccountId], allUsedAccounts[transaction.RelatedAccountId])

		if !transactionEditable {
			log.Warnf(c, "[transactions.TransactionImportHandler] transaction \"index:%d\" is not editable for user \"uid:%d\"", i, uid)
			return nil, errs.ErrCannotCreateTransactionWithThisTransactionTime
		}
	}

	result := &models.TransactionImportResponse{
		TotalCount: len(newTransactions),
	}

	if a.CurrentConfig().EnableDuplicateSubmissionsCheck && transactionImportReq.ClientSessionId != "" {
		initialRemark, err := json.Marshal(result)

		if err != nil {
			log.Warnf(c, "[transactions.TransactionImportHandler] failed to marshal initial import result for user \"uid:%d\", because %s", uid, err.Error())
			return nil, errs.ErrOperationFailed
		}

		found, remark := a.GetOrSetSubmissionRemarkWithCustomExpiration(duplicatechecker.DUPLICATE_CHECKER_TYPE_IMPORT_TRANSACTIONS, uid, transactionImportReq.ClientSessionId, string(initialRemark), transactionImportResultExpiration)

		if found {
			var previousResult *models.TransactionImportResponse

			if json.Unmarshal([]byte(remark), &previousResult) == nil && previousResult != nil && previousResult.Finished {
				return a.getTransactionImportResponse(previousResult)
			}

			return nil, errs.ErrRepeatedRequest
		}
	}

	for i := 0; i < len(newTransactions); i++ {
		err = a.transactions.CreateTransaction(c, newTransactions[i], newTransactionTagIdsMap[i], nil)

		if err != nil {
			result.Finished = true
			a.setTransactionImportResult(uid, transactionImportReq.ClientSessionId, result)
			log.Errorf(c, "[transactions.TransactionImportHandler] failed to import transactions for user \"uid:%d\" after %d of %d transactions, because %s", uid, result.ImportedCount, result.TotalCount, err.Error())

			return nil, errs.NewErrorWithContext(errs.Or(err, errs.ErrOperationFailed), map[string]any{
				"importedCount": result.ImportedCount,
				"totalCount":    result.TotalCount,
			})
		}

		result.ImportedCount = i + 1
		a.setTransactionImportResult(uid, transactionImportReq.ClientSessionId, result)
	}

	result.Finished = true
	a.setTransactionImportResult(uid, transactionImportReq.ClientSessionId, result)

	log.Infof(c, "[transactions.TransactionImportHandler] user \"uid:%d\" has imported %d of %d transactions", uid, result.ImportedCount, result.TotalCount)

	return result, nil
}

// TransactionImportProcessHandler returns the process of specified transaction import task by request parameters for current user
func (a *TransactionsApi) TransactionImportProcessHandler(c *core.WebContext) (any, *errs.Error) {
	var transactionImportProcessReq models.TransactionImportProcessRequest
	err := c.ShouldBindQuery(&transactionImportProcessReq)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionImportProcessHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	uid := c.GetCurrentUid()

	if !a.CurrentConfig().EnableDuplicateSubmissionsCheck {
		return nil, nil
	}

	found, remark := a.GetSubmissionRemark(duplicatechecker.DUPLICATE_CHECKER_TYPE_IMPORT_TRANSACTIONS, uid, transactionImportProcessReq.ClientSessionId)

	if !found {
		return nil, nil
	}

	var result *models.TransactionImportResponse
	err = json.Unmarshal([]byte(remark), &result)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionImportProcessHandler] parse process failed, because %s", err.Error())
		return nil, nil
	}

	if result == nil {
		return nil, nil
	}

	return a.getTransactionImportResponse(result)
}

func (a *TransactionsApi) getTransactionImportResponse(result *models.TransactionImportResponse) (any, *errs.Error) {
	if !result.Finished || result.ImportedCount == result.TotalCount {
		return result, nil
	}

	return nil, errs.NewErrorWithContext(errs.ErrOperationFailed, map[string]any{
		"importedCount": result.ImportedCount,
		"totalCount":    result.TotalCount,
	})
}

func (a *TransactionsApi) setTransactionImportResult(uid int64, clientSessionId string, result *models.TransactionImportResponse) {
	if !a.CurrentConfig().EnableDuplicateSubmissionsCheck || clientSessionId == "" {
		return
	}

	remark, err := json.Marshal(result)

	if err == nil {
		a.SetSubmissionRemarkWithCustomExpiration(duplicatechecker.DUPLICATE_CHECKER_TYPE_IMPORT_TRANSACTIONS, uid, clientSessionId, string(remark), transactionImportResultExpiration)
	}
}
