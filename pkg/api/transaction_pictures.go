package api

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"mime/multipart"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/duplicatechecker"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/services"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// TransactionPicturesApi represents transaction pictures api
type TransactionPicturesApi struct {
	ApiUsingConfig
	ApiUsingDuplicateChecker
	users    *services.UserService
	tokens   *services.TokenService
	pictures *services.TransactionPictureService
}

// Initialize a transaction api singleton instance
var (
	TransactionPictures = &TransactionPicturesApi{
		ApiUsingConfig: ApiUsingConfig{
			container: settings.Container,
		},
		ApiUsingDuplicateChecker: ApiUsingDuplicateChecker{
			ApiUsingConfig: ApiUsingConfig{
				container: settings.Container,
			},
			duplicateChecker: duplicatechecker.Container,
		},
		users:    services.Users,
		tokens:   services.Tokens,
		pictures: services.TransactionPictures,
	}
)

// TransactionPictureUploadHandler saves transaction picture by request parameters for current user
func (a *TransactionPicturesApi) TransactionPictureUploadHandler(c *core.WebContext) (any, *errs.Error) {
	uid := c.GetCurrentUid()
	form, err := c.MultipartForm()

	if err != nil {
		log.Errorf(c, "[transaction_pictures.TransactionPictureUploadHandler] failed to get multi-part form data for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.ErrParameterInvalid
	}

	pictureFiles := form.File["picture"]

	if len(pictureFiles) < 1 {
		log.Warnf(c, "[transaction_pictures.TransactionPictureUploadHandler] there is no transaction picture in request for user \"uid:%d\"", uid)
		return nil, errs.ErrNoTransactionPicture
	}

	if pictureFiles[0].Size < 1 {
		log.Warnf(c, "[transaction_pictures.TransactionPictureUploadHandler] the size of transaction picture in request is zero for user \"uid:%d\"", uid)
		return nil, errs.ErrTransactionPictureIsEmpty
	}

	if pictureFiles[0].Size > int64(a.CurrentConfig().MaxTransactionPictureFileSize) {
		log.Warnf(c, "[transaction_pictures.TransactionPictureUploadHandler] the upload file size \"%d\" exceeds the maximum size \"%d\" of transaction picture for user \"uid:%d\"", pictureFiles[0].Size, a.CurrentConfig().MaxTransactionPictureFileSize, uid)
		return nil, errs.ErrExceedMaxTransactionPictureFileSize
	}

	fileExtension := utils.GetFileNameExtension(pictureFiles[0].Filename)

	if utils.GetTransactionAttachmentContentType(fileExtension) == "" {
		log.Warnf(c, "[transaction_pictures.TransactionPictureUploadHandler] the file extension \"%s\" of transaction picture in request is not supported for user \"uid:%d\"", fileExtension, uid)
		return nil, errs.ErrTransactionPictureTypeNotSupported
	}

	pictureFile, err := pictureFiles[0].Open()

	if err != nil {
		log.Errorf(c, "[transaction_pictures.TransactionPictureUploadHandler] failed to get transaction picture file from request for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.ErrOperationFailed
	}

	err = checkTransactionPictureFileContent(pictureFile, fileExtension)

	if err != nil {
		pictureFile.Close()
		log.Warnf(c, "[transaction_pictures.TransactionPictureUploadHandler] the content of transaction picture in request does not match the file extension \"%s\" for user \"uid:%d\", because %s", fileExtension, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	fileSha256, fileSize, err := computeTransactionPictureFileChecksum(pictureFile)

	if err != nil {
		pictureFile.Close()
		log.Errorf(c, "[transaction_pictures.TransactionPictureUploadHandler] failed to compute checksum of transaction picture file for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.ErrOperationFailed
	}

	pictureInfo := a.createNewPictureInfoModel(uid, fileExtension, c.ClientIP())
	pictureInfo.OriginalFileName = models.SanitizeTransactionPictureOriginalFileName(pictureFiles[0].Filename)
	pictureInfo.FileSize = fileSize
	pictureInfo.FileSha256 = fileSha256

	clientSessionIds := form.Value["clientSessionId"]
	clientSessionId := ""

	if len(clientSessionIds) > 0 {
		clientSessionId = clientSessionIds[0]
	}

	if a.CurrentConfig().EnableDuplicateSubmissionsCheck && clientSessionId != "" {
		found, remark := a.GetSubmissionRemark(duplicatechecker.DUPLICATE_CHECKER_TYPE_NEW_PICTURE, uid, clientSessionId)

		if found {
			log.Infof(c, "[transaction_pictures.TransactionPictureUploadHandler] another transaction picture \"id:%s\" has been uploaded for user \"uid:%d\"", remark, uid)
			pictureId, err := utils.StringToInt64(remark)

			if err == nil {
				pictureInfo, err = a.pictures.GetPictureInfoByPictureId(c, uid, pictureId)

				if err != nil {
					log.Errorf(c, "[transaction_pictures.TransactionPictureUploadHandler] failed to get existed transaction picture \"id:%d\" for user \"uid:%d\", because %s", pictureId, uid, err.Error())
					return nil, errs.Or(err, errs.ErrOperationFailed)
				}

				pictureToken, err := a.GetTransactionPictureToken(c, uid)

				if err != nil {
					log.Errorf(c, "[transaction_pictures.TransactionPictureUploadHandler] failed to get transaction picture token for user \"uid:%d\", because %s", uid, err.Error())
					return nil, errs.Or(err, errs.ErrTokenGenerating)
				}

				pictureInfoResp := a.GetTransactionPictureInfoResponse(pictureInfo, pictureToken)

				return pictureInfoResp, nil
			}
		}
	}

	err = a.pictures.UploadPicture(c, pictureInfo, pictureFile)

	if err != nil {
		log.Errorf(c, "[transaction_pictures.TransactionPictureUploadHandler] failed to update transaction picture for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	a.SetSubmissionRemarkIfEnable(duplicatechecker.DUPLICATE_CHECKER_TYPE_NEW_PICTURE, uid, clientSessionId, utils.Int64ToString(pictureInfo.PictureId))

	pictureToken, err := a.GetTransactionPictureToken(c, uid)

	if err != nil {
		log.Errorf(c, "[transaction_pictures.TransactionPictureUploadHandler] failed to get transaction picture token for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrTokenGenerating)
	}

	pictureInfoResp := a.GetTransactionPictureInfoResponse(pictureInfo, pictureToken)

	return pictureInfoResp, nil
}

// TransactionPictureGetHandler returns transaction picture data for current user
func (a *TransactionPicturesApi) TransactionPictureGetHandler(c *core.WebContext) ([]byte, string, *errs.Error) {
	fileName := c.Param("fileName")
	fileExtension := utils.GetFileNameExtension(fileName)
	contentType := utils.GetTransactionAttachmentContentType(fileExtension)

	if contentType == "" {
		return nil, "", errs.ErrTransactionPictureTypeNotSupported
	}

	fileBaseName := utils.GetFileNameWithoutExtension(fileName)
	pictureId, err := utils.StringToInt64(fileBaseName)

	if err != nil {
		return nil, "", errs.ErrTransactionPictureIdInvalid
	}

	uid := c.GetCurrentUid()
	pictureData, err := a.pictures.GetPictureByPictureId(c, uid, pictureId, fileExtension)

	if err != nil {
		if !errs.IsCustomError(err) {
			log.Errorf(c, "[transaction_pictures.TransactionPictureUploadHandler] failed to get transaction picture, because %s", err.Error())
		}

		return nil, "", errs.Or(err, errs.ErrOperationFailed)
	}

	setTransactionPictureResponseHeaders(c, pictureId, fileExtension, c.Query("download") == "true")

	return pictureData, contentType, nil
}

// TransactionPictureRemoveUnusedHandler removes unused transaction picture by request parameters for current user
func (a *TransactionPicturesApi) TransactionPictureRemoveUnusedHandler(c *core.WebContext) (any, *errs.Error) {
	var pictureDeleteReq models.TransactionPictureUnusedDeleteRequest
	err := c.ShouldBindJSON(&pictureDeleteReq)

	if err != nil {
		log.Warnf(c, "[transaction_pictures.TransactionPictureRemoveUnusedHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	uid := c.GetCurrentUid()
	err = a.pictures.RemoveUnusedTransactionPicture(c, uid, pictureDeleteReq.Id)

	if err != nil {
		log.Errorf(c, "[transaction_pictures.TransactionPictureRemoveUnusedHandler] failed to remove unused transaction picture for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	return true, nil
}

func (a *TransactionPicturesApi) GetTransactionPictureToken(c *core.WebContext, uid int64) (string, error) {
	claims := c.GetTokenClaims()

	if claims == nil {
		log.Warnf(c, "[transaction_pictures.GetTransactionPictureToken] token is invalid, because claims is nil or token type is not normal")
		return "", errs.ErrInvalidToken
	}

	if claims.Type != core.USER_TOKEN_TYPE_NORMAL {
		return "", nil
	}

	pictureToken, err := a.tokens.DeriveTransactionPictureToken(c, claims, c.GetTokenRecord())

	if err != nil {
		log.Errorf(c, "[transaction_pictures.GetTransactionPictureToken] failed to derive transaction picture token for user \"uid:%d\", because %s", uid, err.Error())
		return "", errs.Or(err, errs.ErrTokenGenerating)
	}

	return pictureToken, nil
}

func (a *TransactionPicturesApi) createNewPictureInfoModel(uid int64, fileExtension string, clientIp string) *models.TransactionPictureInfo {
	return &models.TransactionPictureInfo{
		Uid:              uid,
		TransactionId:    models.TransactionPictureNewPictureTransactionId,
		PictureExtension: fileExtension,
		CreatedIp:        clientIp,
	}
}

// checkTransactionPictureFileContent checks whether the beginning of the uploaded file matches the magic bytes of the file extension, and rewinds the file to the beginning
func checkTransactionPictureFileContent(pictureFile multipart.File, fileExtension string) error {
	if utils.GetTransactionAttachmentContentType(fileExtension) == "" {
		return errs.ErrTransactionPictureTypeNotSupported
	}

	headerLength := utils.GetTransactionAttachmentMagicBytesLength(fileExtension)

	if headerLength < 1 {
		return nil
	}

	fileHeader := make([]byte, headerLength)
	readLength, err := io.ReadFull(pictureFile, fileHeader)

	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return err
	}

	if !utils.IsTransactionAttachmentContentValid(fileExtension, fileHeader[:readLength]) {
		return errs.ErrTransactionPictureContentInvalid
	}

	_, err = pictureFile.Seek(0, io.SeekStart)

	return err
}

// computeTransactionPictureFileChecksum returns the hex encoded SHA-256 checksum and the size of the uploaded file, and rewinds the file to the beginning
func computeTransactionPictureFileChecksum(pictureFile multipart.File) (string, int64, error) {
	hash := sha256.New()
	size, err := io.Copy(hash, pictureFile)

	if err != nil {
		return "", 0, err
	}

	_, err = pictureFile.Seek(0, io.SeekStart)

	if err != nil {
		return "", 0, err
	}

	return hex.EncodeToString(hash.Sum(nil)), size, nil
}

// setTransactionPictureResponseHeaders sets the security related response headers for the transaction picture or document
func setTransactionPictureResponseHeaders(c *core.WebContext, pictureId int64, fileExtension string, download bool) {
	c.Header("X-Content-Type-Options", "nosniff")

	if !utils.IsPdfFileExtension(fileExtension) {
		return
	}

	dispositionType := "inline"

	if download {
		dispositionType = "attachment"
	}

	// pdf documents can carry active content, so restrict everything except the browser built-in pdf viewer and disallow embedding into other pages
	c.Header("Content-Disposition", dispositionType+"; filename=\""+utils.Int64ToString(pictureId)+"."+utils.PdfFileExtension+"\"")
	c.Header("Content-Security-Policy", "default-src 'none'; object-src 'self'; img-src 'self' data: blob:; style-src 'unsafe-inline'; frame-ancestors 'none'; form-action 'none'; base-uri 'none'")
}
