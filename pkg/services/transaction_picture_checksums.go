package services

import (
	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/models"
)

// transactionPictureChecksumQueryChunkSize is the maximum count of checksums in one "IN" condition when querying transaction pictures
const transactionPictureChecksumQueryChunkSize = 500

// GetAttachedPictureInfosBySha256 returns the transaction picture infos which are attached to transactions and whose SHA-256 checksum recorded at upload is one of the given checksums.
// Pictures uploaded before the checksum was recorded are not returned
func (s *TransactionPictureService) GetAttachedPictureInfosBySha256(c core.Context, uid int64, checksums []string) ([]*models.TransactionPictureInfo, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	allPictureInfos := make([]*models.TransactionPictureInfo, 0)

	for start := 0; start < len(checksums); start += transactionPictureChecksumQueryChunkSize {
		end := start + transactionPictureChecksumQueryChunkSize

		if end > len(checksums) {
			end = len(checksums)
		}

		var pictureInfos []*models.TransactionPictureInfo
		err := s.UserDataDB(uid).NewSession(c).Where("uid=? AND deleted=? AND transaction_id<>?", uid, false, models.TransactionPictureNewPictureTransactionId).In("file_sha256", checksums[start:end]).Find(&pictureInfos)

		if err != nil {
			return nil, err
		}

		allPictureInfos = append(allPictureInfos, pictureInfos...)
	}

	return allPictureInfos, nil
}
