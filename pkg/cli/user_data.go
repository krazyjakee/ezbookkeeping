package cli

import (
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/services"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
	"github.com/mayswind/ezbookkeeping/pkg/validators"
)

// UserDataCli represents user data cli
type UserDataCli struct {
	CliUsingConfig
	accounts                *services.AccountService
	transactions            *services.TransactionService
	categories              *services.TransactionCategoryService
	tags                    *services.TransactionTagService
	users                   *services.UserService
	twoFactorAuthorizations *services.TwoFactorAuthorizationService
	tokens                  *services.TokenService
	forgetPasswords         *services.ForgetPasswordService
}

// Initialize a user data cli singleton instance
var (
	UserData = &UserDataCli{
		CliUsingConfig: CliUsingConfig{
			container: settings.Container,
		},
		accounts:                services.Accounts,
		transactions:            services.Transactions,
		categories:              services.TransactionCategories,
		tags:                    services.TransactionTags,
		users:                   services.Users,
		twoFactorAuthorizations: services.TwoFactorAuthorizations,
		tokens:                  services.Tokens,
		forgetPasswords:         services.ForgetPasswords,
	}
)

// AddNewUser adds a new user according to specified info
func (l *UserDataCli) AddNewUser(c *core.CliContext, username string, email string, nickname string, password string, defaultCurrency string) (*models.User, error) {
	if username == "" {
		log.CliErrorf(c, "[user_data.AddNewUser] user name is empty")
		return nil, errs.ErrUsernameIsEmpty
	}

	if email == "" {
		log.CliErrorf(c, "[user_data.AddNewUser] user email is empty")
		return nil, errs.ErrEmailIsEmpty
	}

	if nickname == "" {
		log.CliErrorf(c, "[user_data.AddNewUser] user nickname is empty")
		return nil, errs.ErrNicknameIsEmpty
	}

	if password == "" {
		log.CliErrorf(c, "[user_data.AddNewUser] user password is empty")
		return nil, errs.ErrPasswordIsEmpty
	}

	if defaultCurrency == "" {
		log.CliErrorf(c, "[user_data.AddNewUser] user default currency is empty")
		return nil, errs.ErrUserDefaultCurrencyIsEmpty
	}

	if _, ok := validators.AllCurrencyNames[defaultCurrency]; !ok {
		log.CliErrorf(c, "[user_data.AddNewUser] user default currency is invalid")
		return nil, errs.ErrUserDefaultCurrencyIsInvalid
	}

	user := &models.User{
		Username:             username,
		Email:                email,
		Nickname:             nickname,
		Password:             password,
		DefaultCurrency:      defaultCurrency,
		FirstDayOfWeek:       core.WEEKDAY_SUNDAY,
		TransactionEditScope: models.TRANSACTION_EDIT_SCOPE_ALL,
		FeatureRestriction:   l.CurrentConfig().DefaultFeatureRestrictions,
	}

	err := l.users.CreateUser(c, user, false)

	if err != nil {
		log.CliErrorf(c, "[user_data.AddNewUser] failed to create user \"%s\", because %s", user.Username, err.Error())
		return nil, err
	}

	log.CliInfof(c, "[user_data.AddNewUser] user \"%s\" has add successfully, uid is %d", user.Username, user.Uid)

	return user, nil
}

// GetUserByUsername returns user by user name
func (l *UserDataCli) GetUserByUsername(c *core.CliContext, username string) (*models.User, error) {
	if username == "" {
		log.CliErrorf(c, "[user_data.GetUserByUsername] user name is empty")
		return nil, errs.ErrUsernameIsEmpty
	}

	user, err := l.users.GetUserByUsername(c, username)

	if err != nil {
		log.CliErrorf(c, "[user_data.GetUserByUsername] failed to get user by user name \"%s\", because %s", username, err.Error())
		return nil, err
	}

	return user, nil
}

// ModifyUserPassword modifies user password
func (l *UserDataCli) ModifyUserPassword(c *core.CliContext, username string, password string) error {
	if username == "" {
		log.CliErrorf(c, "[user_data.ModifyUserPassword] user name is empty")
		return errs.ErrUsernameIsEmpty
	}

	if password == "" {
		log.CliErrorf(c, "[user_data.ModifyUserPassword] user password is empty")
		return errs.ErrPasswordIsEmpty
	}

	user, err := l.users.GetUserByUsername(c, username)

	if err != nil {
		log.CliErrorf(c, "[user_data.ModifyUserPassword] failed to get user by user name \"%s\", because %s", username, err.Error())
		return err
	}

	if l.users.IsPasswordEqualsUserPassword(password, user) {
		return errs.ErrNothingWillBeUpdated
	}

	userNew := &models.User{
		Uid:      user.Uid,
		Salt:     user.Salt,
		Password: password,
	}

	err = l.users.UpdateUserPassword(c, userNew)

	if err != nil {
		log.CliErrorf(c, "[user_data.ModifyUserPassword] failed to update user \"%s\" password, because %s", user.Username, err.Error())
		return err
	}

	now := time.Now().Unix()
	err = l.tokens.DeleteTokensBeforeTime(c, user.Uid, now)

	if err == nil {
		log.CliInfof(c, "[user_data.ModifyUserPassword] revoke old tokens before unix time \"%d\" for user \"%s\"", now, user.Username)
	} else {
		log.CliWarnf(c, "[user_data.ModifyUserPassword] failed to revoke old tokens for user \"%s\", because %s", user.Username, err.Error())
	}

	return nil
}

// SendPasswordResetMail sends an email with password reset link
func (l *UserDataCli) SendPasswordResetMail(c *core.CliContext, username string) error {
	if username == "" {
		log.CliErrorf(c, "[user_data.SendPasswordResetMail] user name is empty")
		return errs.ErrUsernameIsEmpty
	}

	user, err := l.users.GetUserByUsername(c, username)

	if err != nil {
		log.CliErrorf(c, "[user_data.SendPasswordResetMail] failed to get user by user name \"%s\", because %s", username, err.Error())
		return err
	}

	if l.CurrentConfig().ForgetPasswordRequireVerifyEmail && !user.EmailVerified {
		log.CliWarnf(c, "[user_data.SendPasswordResetMail] user \"uid:%d\" has not verified email", user.Uid)
		return errs.ErrEmailIsNotVerified
	}

	token, _, err := l.tokens.CreatePasswordResetTokenWithoutUserAgent(c, user)

	if err != nil {
		log.CliErrorf(c, "[user_data.SendPasswordResetMail] failed to create token for user \"uid:%d\", because %s", user.Uid, err.Error())
		return err
	}

	err = l.forgetPasswords.SendPasswordResetEmail(c, user, token, "")

	if err != nil {
		log.CliWarnf(c, "[user_data.SendPasswordResetMail] cannot send email to \"%s\", because %s", user.Email, err.Error())
		return err
	}

	return nil
}

// EnableUser sets user enabled according to the specified user name
func (l *UserDataCli) EnableUser(c *core.CliContext, username string) error {
	if username == "" {
		log.CliErrorf(c, "[user_data.EnableUser] user name is empty")
		return errs.ErrUsernameIsEmpty
	}

	user, err := l.users.GetUserByUsername(c, username)

	if err != nil {
		log.CliErrorf(c, "[user_data.EnableUser] failed to get user by user name \"%s\", because %s", username, err.Error())
		return err
	}

	if !user.Disabled {
		log.CliErrorf(c, "[user_data.EnableUser] user \"%s\" is already enabled", username)
		return errs.ErrNothingWillBeUpdated
	}

	err = l.users.EnableUser(c, username)

	if err != nil {
		log.CliErrorf(c, "[user_data.EnableUser] failed to set user enabled by user name \"%s\", because %s", username, err.Error())
		return err
	}

	return nil
}

// DisableUser sets user disabled according to the specified user name
func (l *UserDataCli) DisableUser(c *core.CliContext, username string, remainSessions bool) error {
	if username == "" {
		log.CliErrorf(c, "[user_data.DisableUser] user name is empty")
		return errs.ErrUsernameIsEmpty
	}

	user, err := l.users.GetUserByUsername(c, username)

	if err != nil {
		log.CliErrorf(c, "[user_data.DisableUser] failed to get user by user name \"%s\", because %s", username, err.Error())
		return err
	}

	if user.Disabled {
		log.CliErrorf(c, "[user_data.DisableUser] user \"%s\" is already disabled", username)
		return errs.ErrNothingWillBeUpdated
	}

	err = l.users.DisableUser(c, username)

	if err != nil {
		log.CliErrorf(c, "[user_data.DisableUser] failed to set user disabled by user name \"%s\", because %s", username, err.Error())
		return err
	}

	if !remainSessions {
		err = l.tokens.DeleteAllTokens(c, user.Uid)

		if err != nil {
			log.CliErrorf(c, "[user_data.DisableUser] failed to revoke all tokens for user \"%s\", because %s", username, err.Error())
		}
	}

	return nil
}

// SetUserFeatureRestrictions sets user feature restrictions according to the specified user name
func (l *UserDataCli) SetUserFeatureRestrictions(c *core.CliContext, username string, featureRestriction core.UserFeatureRestrictions) error {
	if username == "" {
		log.CliErrorf(c, "[user_data.SetUserFeatureRestrictions] user name is empty")
		return errs.ErrUsernameIsEmpty
	}

	user, err := l.users.GetUserByUsername(c, username)

	if err != nil {
		log.CliErrorf(c, "[user_data.SetUserFeatureRestrictions] failed to get user by user name \"%s\", because %s", username, err.Error())
		return err
	}

	if user.FeatureRestriction == featureRestriction {
		log.CliErrorf(c, "[user_data.SetUserFeatureRestrictions] user \"%s\" already set the specified feature restrictions", username)
		return errs.ErrNothingWillBeUpdated
	}

	err = l.users.UpdateUserFeatureRestriction(c, username, featureRestriction)

	if err != nil {
		log.CliErrorf(c, "[user_data.SetUserFeatureRestrictions] failed to set user feature restrictions by user name \"%s\", because %s", username, err.Error())
		return err
	}

	return nil
}

// AddUserFeatureRestrictions adds user feature restrictions according to the specified user name
func (l *UserDataCli) AddUserFeatureRestrictions(c *core.CliContext, username string, featureRestriction core.UserFeatureRestrictions) error {
	if username == "" {
		log.CliErrorf(c, "[user_data.AddUserFeatureRestrictions] user name is empty")
		return errs.ErrUsernameIsEmpty
	}

	user, err := l.users.GetUserByUsername(c, username)

	if err != nil {
		log.CliErrorf(c, "[user_data.AddUserFeatureRestrictions] failed to get user by user name \"%s\", because %s", username, err.Error())
		return err
	}

	if user.FeatureRestriction.ContainsAll(featureRestriction) {
		log.CliErrorf(c, "[user_data.AddUserFeatureRestrictions] user \"%s\" already has the specified feature restrictions", username)
		return errs.ErrNothingWillBeUpdated
	}

	err = l.users.AddUserFeatureRestriction(c, username, featureRestriction)

	if err != nil {
		log.CliErrorf(c, "[user_data.AddUserFeatureRestrictions] failed to add user feature restrictions by user name \"%s\", because %s", username, err.Error())
		return err
	}

	return nil
}

// RemoveUserFeatureRestrictions removes user feature restrictions according to the specified user name
func (l *UserDataCli) RemoveUserFeatureRestrictions(c *core.CliContext, username string, featureRestriction core.UserFeatureRestrictions) error {
	if username == "" {
		log.CliErrorf(c, "[user_data.RemoveUserFeatureRestrictions] user name is empty")
		return errs.ErrUsernameIsEmpty
	}

	user, err := l.users.GetUserByUsername(c, username)

	if err != nil {
		log.CliErrorf(c, "[user_data.RemoveUserFeatureRestrictions] failed to get user by user name \"%s\", because %s", username, err.Error())
		return err
	}

	if !user.FeatureRestriction.ContainsAny(featureRestriction) {
		log.CliErrorf(c, "[user_data.RemoveUserFeatureRestrictions] user \"%s\" does not have the specified feature restrictions", username)
		return errs.ErrNothingWillBeUpdated
	}

	err = l.users.RemoveUserFeatureRestriction(c, username, featureRestriction)

	if err != nil {
		log.CliErrorf(c, "[user_data.RemoveUserFeatureRestrictions] failed to remove user feature restrictions by user name \"%s\", because %s", username, err.Error())
		return err
	}

	return nil
}

// ResendVerifyEmail resends an email with account activation link
func (l *UserDataCli) ResendVerifyEmail(c *core.CliContext, username string) error {
	if !l.CurrentConfig().EnableUserVerifyEmail {
		return errs.ErrEmailValidationNotAllowed
	}

	if username == "" {
		log.CliErrorf(c, "[user_data.ResendVerifyEmail] user name is empty")
		return errs.ErrUsernameIsEmpty
	}

	user, err := l.users.GetUserByUsername(c, username)

	if err != nil {
		log.CliErrorf(c, "[user_data.ResendVerifyEmail] failed to get user by user name \"%s\", because %s", username, err.Error())
		return err
	}

	if user.EmailVerified {
		log.CliWarnf(c, "[user_data.ResendVerifyEmail] user \"uid:%d\" email has been verified", user.Uid)
		return errs.ErrEmailIsVerified
	}

	token, _, err := l.tokens.CreateEmailVerifyTokenWithoutUserAgent(c, user)

	if err != nil {
		log.CliErrorf(c, "[user_data.ResendVerifyEmail] failed to create token for user \"uid:%d\", because %s", user.Uid, err.Error())
		return errs.ErrTokenGenerating
	}

	err = l.users.SendVerifyEmail(user, token, "")

	if err != nil {
		log.CliErrorf(c, "[user_data.ResendVerifyEmail] cannot send email to \"%s\", because %s", user.Email, err.Error())
		return err
	}

	return nil
}

// SetUserEmailVerified sets user email address verified
func (l *UserDataCli) SetUserEmailVerified(c *core.CliContext, username string) error {
	if username == "" {
		log.CliErrorf(c, "[user_data.SetUserEmailVerified] user name is empty")
		return errs.ErrUsernameIsEmpty
	}

	user, err := l.users.GetUserByUsername(c, username)

	if err != nil {
		log.CliErrorf(c, "[user_data.SetUserEmailVerified] failed to get user by user name \"%s\", because %s", username, err.Error())
		return err
	}

	if user.EmailVerified {
		log.CliErrorf(c, "[user_data.SetUserEmailVerified] user \"%s\" is already verified", username)
		return errs.ErrNothingWillBeUpdated
	}

	err = l.users.SetUserEmailVerified(c, username)

	if err != nil {
		log.CliErrorf(c, "[user_data.SetUserEmailVerified] failed to set user email address verified by user name \"%s\", because %s", username, err.Error())
		return err
	}

	return nil
}

// SetUserEmailUnverified sets user email address unverified
func (l *UserDataCli) SetUserEmailUnverified(c *core.CliContext, username string) error {
	if username == "" {
		log.CliErrorf(c, "[user_data.SetUserEmailUnverified] user name is empty")
		return errs.ErrUsernameIsEmpty
	}

	user, err := l.users.GetUserByUsername(c, username)

	if err != nil {
		log.CliErrorf(c, "[user_data.SetUserEmailUnverified] failed to get user by user name \"%s\", because %s", username, err.Error())
		return err
	}

	if !user.EmailVerified {
		log.CliErrorf(c, "[user_data.SetUserEmailUnverified] user \"%s\" is already unverified", username)
		return errs.ErrNothingWillBeUpdated
	}

	err = l.users.SetUserEmailUnverified(c, username)

	if err != nil {
		log.CliErrorf(c, "[user_data.SetUserEmailUnverified] failed to set user email address unverified by user name \"%s\", because %s", username, err.Error())
		return err
	}

	return nil
}

// DeleteUser deletes user according to the specified user name
func (l *UserDataCli) DeleteUser(c *core.CliContext, username string) error {
	if username == "" {
		log.CliErrorf(c, "[user_data.DeleteUser] user name is empty")
		return errs.ErrUsernameIsEmpty
	}

	uid, err := l.getUserIdByUsername(c, username)

	if err != nil {
		log.CliErrorf(c, "[user_data.DeleteUser] error occurs when getting user id by user name")
		return err
	}

	err = l.users.DeleteUser(c, username)

	if err != nil {
		log.CliErrorf(c, "[user_data.DeleteUser] failed to delete user by user name \"%s\", because %s", username, err.Error())
		return err
	}

	err = l.tokens.DeleteAllTokens(c, uid)

	if err != nil {
		log.CliErrorf(c, "[user_data.DeleteUser] failed to revoke all tokens for user \"%s\", because %s", username, err.Error())
	}

	return nil
}

// ListUserTokens returns all tokens of the specified user
func (l *UserDataCli) ListUserTokens(c *core.CliContext, username string) ([]*models.TokenRecord, error) {
	if username == "" {
		log.CliErrorf(c, "[user_data.ListUserTokens] user name is empty")
		return nil, errs.ErrUsernameIsEmpty
	}

	uid, err := l.getUserIdByUsername(c, username)

	if err != nil {
		log.CliErrorf(c, "[user_data.ListUserTokens] error occurs when getting user id by user name")
		return nil, err
	}

	tokens, err := l.tokens.GetAllUnexpiredNormalAndMCPTokensByUid(c, uid)

	if err != nil {
		log.CliErrorf(c, "[user_data.ListUserTokens] failed to get tokens of user \"%s\", because %s", username, err.Error())
		return nil, err
	}

	return tokens, nil
}

// CreateNewUserToken returns a new token for the specified user
func (l *UserDataCli) CreateNewUserToken(c *core.CliContext, username string, tokenType string, expiresInSeconds int64) (*models.TokenRecord, string, error) {
	if username == "" {
		log.CliErrorf(c, "[user_data.CreateNewUserToken] user name is empty")
		return nil, "", errs.ErrUsernameIsEmpty
	}

	user, err := l.GetUserByUsername(c, username)

	if err != nil {
		log.CliErrorf(c, "[user_data.CreateNewUserToken] error occurs when getting user by user name")
		return nil, "", err
	}

	var token string
	var tokenRecord *models.TokenRecord

	if tokenType == "api" {
		if !l.CurrentConfig().EnableAPIToken {
			return nil, "", errs.ErrAPITokenNotEnabled
		}

		if user.FeatureRestriction.Contains(core.USER_FEATURE_RESTRICTION_TYPE_GENERATE_API_TOKEN) {
			return nil, "", errs.ErrNotPermittedToPerformThisAction
		}

		token, tokenRecord, err = l.tokens.CreateAPITokenViaCli(c, user, expiresInSeconds)
	} else if tokenType == "mcp" {
		if !l.CurrentConfig().EnableMCPServer {
			return nil, "", errs.ErrMCPServerNotEnabled
		}

		if user.FeatureRestriction.Contains(core.USER_FEATURE_RESTRICTION_TYPE_MCP_ACCESS) {
			return nil, "", errs.ErrNotPermittedToPerformThisAction
		}

		token, tokenRecord, err = l.tokens.CreateMCPTokenViaCli(c, user, expiresInSeconds)
	} else {
		return nil, "", errs.ErrParameterInvalid
	}

	if err != nil {
		log.CliErrorf(c, "[user_data.CreateNewUserToken] failed to create token for user \"%s\", because %s", username, err.Error())
		return nil, "", err
	}

	return tokenRecord, token, nil
}

// RevokeUserToken revokes the specified token of the user
func (l *UserDataCli) RevokeUserToken(c *core.CliContext, token string) error {
	claims, _, err := l.tokens.ParseToken(c, token)

	if err != nil {
		log.CliErrorf(c, "[user_data.RevokeUserToken] failed to parse token, because %s", err.Error())
		return err
	}

	userTokenId, err := utils.StringToInt64(claims.UserTokenId)

	if err != nil {
		log.CliErrorf(c, "[user_data.RevokeUserToken] failed to get user token id, because %s", err.Error())
		return err
	}

	tokenRecord := &models.TokenRecord{
		Uid:             claims.Uid,
		UserTokenId:     userTokenId,
		CreatedUnixTime: claims.IssuedAt,
	}

	tokenId := l.tokens.GenerateTokenId(tokenRecord)
	err = l.tokens.DeleteToken(c, tokenRecord)

	if err != nil {
		log.Errorf(c, "[user_data.RevokeUserToken] failed to revoke token \"id:%s\" for user \"uid:%d\", because %s", tokenId, claims.Uid, err.Error())
		return err
	}

	return nil
}

// ClearUserTokens clears all tokens of the specified user
func (l *UserDataCli) ClearUserTokens(c *core.CliContext, username string) error {
	if username == "" {
		log.CliErrorf(c, "[user_data.ClearUserTokens] user name is empty")
		return errs.ErrUsernameIsEmpty
	}

	uid, err := l.getUserIdByUsername(c, username)

	if err != nil {
		log.CliErrorf(c, "[user_data.ClearUserTokens] error occurs when getting user id by user name")
		return err
	}

	err = l.tokens.DeleteAllTokens(c, uid)

	if err != nil {
		log.CliErrorf(c, "[user_data.ClearUserTokens] failed to delete tokens of user \"%s\", because %s", username, err.Error())
		return err
	}

	return nil
}

// DisableUserTwoFactorAuthorization disables 2fa for the specified user
func (l *UserDataCli) DisableUserTwoFactorAuthorization(c *core.CliContext, username string) error {
	if username == "" {
		log.CliErrorf(c, "[user_data.DisableUserTwoFactorAuthorization] user name is empty")
		return errs.ErrUsernameIsEmpty
	}

	uid, err := l.getUserIdByUsername(c, username)

	if err != nil {
		log.CliErrorf(c, "[user_data.DisableUserTwoFactorAuthorization] error occurs when getting user id by user name")
		return err
	}

	enableTwoFactor, err := l.twoFactorAuthorizations.ExistsTwoFactorSetting(c, uid)

	if err != nil {
		log.CliErrorf(c, "[user_data.DisableUserTwoFactorAuthorization] failed to check two-factor setting, because %s", err.Error())
		return err
	}

	if !enableTwoFactor {
		return errs.ErrTwoFactorIsNotEnabled
	}

	err = l.twoFactorAuthorizations.DeleteTwoFactorRecoveryCodes(c, uid)

	if err != nil {
		log.CliErrorf(c, "[user_data.DisableUserTwoFactorAuthorization] failed to delete two-factor recovery codes for user \"%s\"", username)
		return err
	}

	err = l.twoFactorAuthorizations.DeleteTwoFactorSetting(c, uid)

	if err != nil {
		log.CliErrorf(c, "[user_data.DisableUserTwoFactorAuthorization] failed to delete two-factor setting for user \"%s\"", username)
		return err
	}

	return nil
}

func (l *UserDataCli) getUserIdByUsername(c *core.CliContext, username string) (int64, error) {
	user, err := l.GetUserByUsername(c, username)

	if err != nil {
		log.CliErrorf(c, "[user_data.getUserIdByUsername] failed to get user by user name \"%s\", because %s", username, err.Error())
		return 0, err
	}

	return user.Uid, nil
}
