package errors

// Code types. Server error should be >10000. Client error should be <10000.
// Client errors are intended to be parsed and displayed by front end
// Server errors are intended for debugging and all masked to 10000(serverinternalerror) in production
const (
	CodeOK CodeType = 0 // 0

	// ========== client error (1001-10000) ==========
	CodePermissionDenied          = 1001 // Permission Denied.
	CodeFailedToAuthenticateToken = 1002 // Failed to authenticate token.
	CodeFailedToGrantDeveloper    = 1003 // Failed to grant DLiveTV post permission.
	CodeJWTTokenExpired           = 1004 // Login token expired.
	CodeNotLoggedIn               = 1005 // You must login first.
	CodePermissionNotGranted      = 1006 // DLiveTV does not have your permission.
	CodeInvalidArgument           = 1007 // Invalid argument.
	CodeEmailInvalid              = 1008 // Email is invalid.
	CodeRequestTimeout            = 1009 // Request timeout.

	// Barrage (1101-1200)
	CodeBarrageNotFound = 1101 // Barrage is not found.

	// Key (1201-1300)
	CodeKeyNotFound = 1201 // You have to login with wallet.

	// Donation (1301-1400)
	CodeNotEnoughPreauth = 1301 // Not enough preauthorization.
	CodeNotEnoughBalance = 1302 // Insufficient balance.
	CodeEzalorFailed     = 1303 // Ezalor failed.

	// User (1401-1500)
	CodeUserNotFound                  = 1401 // User not found.
	CodeEmptyUsername                 = 1402 // Username is empty.
	CodeAuthenticationMismatch        = 1403 // Authentication mismatch.
	CodeEmailAlreadyRegistered        = 1404 // Your email is already registered.
	CodeFailedToLoginWithFacebook     = 1405 // Failed to login with Facebook.
	CodeFailedToVerifyRecaptcha       = 1406 // Failed to verify your recaptcha.
	CodeFailedToLoginWithWallet       = 1407 // Failed to login with wallet.
	CodeAboutTooLong                  = 1408 // Your about is too long.
	CodeWrongVerifyCode               = 1409 // Verification code incorrect.
	CodeFailedToLoginWithGoogle       = 1411 // Failed to login with Google.
	CodeFailedToLoginWithTwitch       = 1412 // Failed to login with Twitch.
	CodePhoneAlreadyRegistered        = 1414 // Your phone is already registered.
	CodePhoneNotVerified              = 1415 // Your phone is not verified.
	CodeDisplayNameChangeCooldown     = 1416 // You cannot change your display name yet.
	CodeFailedToLoginWithSteemConnect = 1417 // Failed to login with steemconnect.
	CodePasswordTooWeak               = 1418 // Your password is too weak.
	CodeUsernameInvalid               = 1419 // Your username is invalid.
	CodeDisplayNameInvalid            = 1420 // Your displayname is invalid.
	CodeNotRegisteredWithEmail        = 1421 // Your account is not registered by password.
	CodeAlreadyBackedUp               = 1422 // You have already got your recovery phrase.
	CodeAlreadyClaimedAndroidBonus    = 1423 // You have already claimed android bonus.
	CodeBannedFromLivestream          = 1424 // You have been banned from streaming.
	CodePartnerStatusInvalid          = 1425 // DLive partner status is invalid
	CodeRegisterTooFast               = 1426 // You registered too fast.
	CodeReferrerDoesNotExist          = 1427 // Referrer does not exist.
	CodeAlreadyClaimedOnThisDevice    = 1428 // Already claimed bonus on this device.
	CodeAccountSuspended              = 1429 // Your account has been suspended.
	CodeClaimAccountFailed            = 1430 // Claim account from registration failed
	CodeFollowLimitExceeded           = 1431 // Follow limit exceeded.
	CodePartnerOnly                   = 1432 // Partner only.
	CodeRecaptchaRequired             = 1433 // Require recaptcha.

	// UserView (1501-1600)

	// Video (1601-1700)
	CodeVideoNotFound         = 1601 // Video not found.
	CodePermlinkIsEmpty       = 1602 // Permlink is empty.
	CodeEmptyPostIDOrUsername = 1603 // PostID or username is empty.
	CodePermlinkInvalid       = 1604 // Your permlink is invalid.
	CodeAlreadyReposted       = 1605 // You have already reposted.
	CodeCannotRepostARepost   = 1606 // You cannot repost a repost.
	CodeExceedTagLimit        = 1607 // You have too many tags.
	CodePermlinkUserMismatch  = 1608 // Your permlink and user mismatch.

	// Relationship (1701-1800)
	CodeCannotFollowSelf     = 1701 // You cannot follow yourself.
	CodeRelationshipNotFound = 1702 // Relationship not found.

	// Tag (1801-1900)
	CodeTagNotFound = 1801 // Tag not found.

	// Comment (1901-2000)
	CodeCommentNotFound       = 1901 // Comment not found.
	CodeParentPermlinkInvalid = 1902 // The video you trying to comment does not exist.

	// Post (2001-2100)
	CodePostNotFound = 2001 // Post not found.

	// Donation (210100)
	CodeDonationNotFound            = 2101 // Donation not found.
	CodeContributorNotFound         = 2102 // Contributor not found.
	CodeDonationIDNotDefined        = 2103 // ContributorID is not defined.
	CodeDoubleSpendPrevention       = 2104 // Your just made the exact same transaction. Please wait a sec.
	CodeSumOfDonationsNotFound      = 2105 // Contributor not found.
	CodeDonationBlockNotFound       = 2106 // Donation block not found
	CodeRecentTotalDonationNotFound = 2107 //RecentTotalDonation not found

	// Livestream (2201-2300)
	CodeLivestreamNotFound         = 2201 // Livestream not found.
	CodeStreamTemplateNotFound     = 2202 // Stream template not found.
	CodeStreamTemplateNotCompleted = 2203 // Stream template is incomplete.
	CodeLivestreamNSFW             = 2204 // livestream nsfw

	// Category (2301-2320)
	CodeCategoryNotFound                     = 2301 // Category not found.
	CodeCategoryDuplicateFound               = 2302 // Duplicated Category found
	CodeCategoryUpdateImgButImgOverrideIsSet = 2303 // Update category img but imgOverride is set

	CodeLanguageNotFound = 2221 // Language not found.

	// Carousel (2321-2330)
	CodeDuplicateCarousel = 2321 // Duplicate carousel.

	// HappyHour (2331-2340)
	CodeHappyHourNotFound         = 2331 // Closest happy hour is not found
	CodeHappyHourRankingNotFound  = 2332 // Happy hour ranking is not found
	CodeInvalidTimeFormat         = 2333 // Invalid time format
	CodeInvalidHappyHourTimeRange = 2334 // Invalid happy hour time range
	CodeHappyHourWinnerNotFound   = 2335 // Happy hour winner is not found
	CodeHappyHourTicketNotFound   = 2336 // Happy hour ticket is not found
	CodeFreeTicketAlreadyClaimed  = 2337 // Happy hour free ticket is already claimed

	// Emotes (2341-2350)
	CodeEmoteNotFound      = 2341
	CodeEmotesReachedLimit = 2342

	// Chain (3001-4000)
	CodeLinoAPIError = 3001 // Blockchain error.

	// Host (4001-4010)
	CodeHostNotFound            = 4001
	CodeHostFeatureNotAvailable = 4002

	// GiftCode (4011-4030)
	CodeGiftCodeNotFound    = 4011 // Gift code not found
	CodeGiftCodeExpired     = 4012 // Gift code has expired
	CodeGiftCodeAlreadyUsed = 4013 // Gift code has been used

	// StreamerReferral (4031-4050)
	CodeStreamerReferralNotFound = 4031

	// Announcement (4051-4070)
	CodeAnnouncementNotFound = 4051

	// ========== Famchat error (5001-5100) =========
	CodeFamchatNotFound      = 5001 // Fanbase not found.
	CodeFamchatExists        = 5002 // Fanbase already exists.
	CodeBannedFromFamchat    = 5003 // You are banned from this fanbase.
	CodeUserAlreadyBanned    = 5004 // You have already banned this user.
	CodeNotAuthorized        = 5005 // Permission denied.
	CodeUserNotExists        = 5006 // User not found.
	CodeRoomNotFound         = 5007 // Room not found.
	CodeMessageNotFound      = 5008 // Message not found.
	CodeChatExists           = 5009 // Chat already exists.
	CodeUserNotInRoom        = 5010 // User is not in the room.
	CodeBadRequest           = 5011 // Bad request.
	CodeReactionExists       = 5012 // The reaction already exists.
	CodeReactionNotFound     = 5013 // Reaction not found.
	CodeOwnerLeaveFamchat    = 5014 // Owner has left the fanbase.
	CodeUserAlreadyInChannel = 5015 // User is already in the channel.
	CodeUserAlreadyInFanbase = 5016 // User is already in the fanbase.
	CodeExceededChannelLimit = 5017 // You have too many channels.
	CodeModeratorExists      = 5018
	CodeModeratorNotExists   = 5019

	// ========== Highlight error (5101-5200) =========
	CodeHighlightNotFount      = 5101
	CodeHighlightDeleted       = 5102
	CodeHighlightLenghtTooLong = 5103

	// ========== SugarMommy error (5201-6000) =========
	CodeSugarMommyDailyLimitReached               = 5201
	CodeSugarMommyLastOrderStillProcessing        = 5202
	CodeSugarMommyInsufficientCoinPurseToGiveaway = 5203
	CodeSugarMommyNotOnClaimingList               = 5204
	CodeSugarMommyActiveGiveawayOngoing           = 5205
	CodeSugarMommyGiveawayEnded                   = 5206
	CodeSugarMommyNoOngoingGiveaway               = 5207
	CodeSugarMommyValidationExpired               = 5208
	CodeSugarMommyAlraedyOnClaimingList           = 5209

	// ========== Streamchat error (6001-7000) =========
	CodeBannedFromStreamchat     = 6001
	CodeStreamchatInterval       = 6002
	CodeChatContainsFilteredWord = 6003
	CodeTempBanned               = 6004
	CodeGlobalBanned             = 6005
	CodeChatMode                 = 6006
	CodeInvalidMessage           = 6007
	CodeStreamchatTimeout        = 6008
	CodeEmoteBanned              = 6009
	CodeModGuardianOrStaff       = 6010
	CodeNoEmoteMode              = 6011

	// ========== Subscription error (7001-9999) =========
	CodeSubscriptionNotFound               = 7001 // You are not subscribed.
	CodeAlreadySubscribed                  = 7002 // You have already subscribed.
	CodeStreamerNoPost                     = 7003 // The streamer has no post.
	CodeCannotSubscribeSelf                = 7004 // You cannot subscribe yourself.
	CodeSubscriptionSettingNotFound        = 7005 // Subscription setting not found.
	CodeSubscriptionColorInvalid           = 7006 // Color is invalid.
	CodeSubscriptionIsPending              = 7007 // Cannot unsubscribe when subscription is pending.
	CodeSubscriptionBadgeTextLengthInvalid = 7008 // Badge text has to be 1-8 characters.
	CodeStreamerNotVerified                = 7009 // Streamer is not verified.
	CodeSubscriptionBenefitLengthInvalid   = 7010 // Subscription benefit length is invalid.

	// ========== server error (10000-) ==========
	CodeServerInternalError          = 10000 // Server internal error.
	CodeDBUnavailable                = 10001
	CodeFailedToPrepareStatement     = 10002
	CodeFailedToBeginTx              = 10003
	CodeFailedToCommitTx             = 10004
	CodeFailedToRollbackTx           = 10005
	CodeFailedToExecStatement        = 10006
	CodeFailedToGetRowsAffected      = 10007
	CodeMoreRowsAffectedThanExpected = 10008
	CodeFailedToCloseStatement       = 10009
	CodeFailedToQueryStatement       = 10010
	CodeFailedToScan                 = 10011
	CodeFailedToGetLastInsertedID    = 10012
	CodeFailedToConvertTime          = 10013
	CodeFailedToSignJwt              = 10014
	CodeFailedToDecryptJwt           = 10015
	CodeMetadataNotFound             = 10016
	CodeFailedToGetDBManager         = 10017
	CodeFailedToReadConfigYaml       = 10018
	CodeFailedToEncrypt              = 10019
	CodeFailedToDecrypt              = 10020
	CodeFailedToMarshalJSON          = 10021
	CodeFailedToUnmarshalJSON        = 10022
	CodeFailedToMakeHTTPRequest      = 10023
	CodeFailedToSendSMS              = 10024
	CodeFailedToSendEmail            = 10025
	CodeFailedToHashPassword         = 10026
	CodeFailedToPollSqs              = 10027
	CodeFailedToDeleteSqs            = 10028
	CodeRegexMatchError              = 10029

	// RateLimit (10101-10200)
	CodeRateLimitExceeded = 10101 // Server is at capacity. Please try again later.

	// Emotion (10201-10300)
	CodeEmoteAddFailed    = 10201
	CodeEmoteDeleteFaield = 10202
	CodeGetEmoteFailed    = 10203
	CodeGetEmotesFailed   = 10204

	// UserEmotion (10301-10400)

	// User (10401-10500)

	// UserView (10501-10600)
	CodeFailedToInsertUserview = 10501
	CodeFailedToListUserview   = 10502
	// Video (10601-10700)

	// Redis (10701-10800)
	CodeRedisFailedToSet   = 10701
	CodeRedisFailedToGet   = 10702
	CodeRedisFailedToWatch = 10703
	CodeRedisFailedToExec  = 10704
	CodeRedisFailedToDel   = 10705
	CodeRedisFailedToPub   = 10706

	// Post (10801-10900)
	CodeFailedToPostToSQS       = 10801
	CodeFailedToGenerateShortID = 10802

	// Chain (20001-30000)
	CodeFailedToDecodeJSONMeta = 20002
	CodeFailedToEncodeJSONMeta = 20003
	CodeFailedToUpdateAccount  = 20004

	// ElasticSearch (30001-40000)
	CodeESFailedToSearch      = 30001
	CodeESFailedToAddVideo    = 30002
	CodeESFailedToAddUser     = 30003
	CodeESFailedToDeleteVideo = 30004
	CodeESFailedToDeleteUser  = 30005
	CodeESFailedToUpdateVideo = 30006
	CodeESFailedToUpdateUser  = 30007

	// Payment (40001-41000)
	CodeSquareAPIError                = 40000
	CodeAPIError                      = 40001
	CodeSquareInternalError           = 40002
	CodeSquareUnauthorized            = 40003
	CodeSquareForbidden               = 40004
	CodeSquareIncorrectType           = 40005
	CodeSquareInvalidValue            = 40006
	CodeSquareCardExpired             = 40007
	CodeSquareInvalidExpiration       = 40008
	CodeSquareUnsupportedCardBrand    = 40009
	CodeSquareUnsupportedEEntryMethod = 40010
	CodeSquareInvalidCard             = 40011
	CodeSquareInvalidEmailAddr        = 40012
	CodeSquareInvalidPhone            = 40013
	CodeSquareCardDeclined            = 40014
	CodeSquareVerifyCVVFailed         = 40015
	CodeSquareVerifyAVSFailed         = 40016
	CodeSquareNotFound                = 40017
	CodeSquareConflict                = 40018
	CodeSquareRateLimited             = 40019
	CodeSquareServiceUnavailable      = 40020
	CodeSquareGatewayTimeout          = 40021
	CodeSquareBadCertificate          = 40022
	CodeSquareUnexpectedValue         = 40023
	CodeSquareInvalidCardData         = 40024
	CodeDupTx                         = 40025
	// CodeSquare             = 40003
	// CodeSquare             = 40003
	// CodeSquare             = 40003
	// CodeSquare             = 40003
	// CodeSquare             = 40003
	// CodeSquare             = 40003
	// CodeSquare             = 40003
	// CodeSquare             = 40003
	// CodeSquare             = 40003

	// Referral (41001-41500)
	CodeNotValidStreamerReferrer = 41001

	// Kafka (41501-41600)
	CodeKafkaFailedToPush    = 41501
	CodeKafkaFailedToConsume = 41502

	// Registration (41601-41700)
	CodeAccountNotAvailable = 41601
	CodeGetAccountFailed    = 41602

	// ========== SugarMommy server error (41701-41800) =========
	CodeSugarMommyCoinPurseNotFound      = 41701
	CodeSugarMommyGiveawayOrderNotFound  = 41702
	CodeSugarMommyUserEngagementNotFound = 41703
	CodeSugarMommyPayoutNotFound         = 41704
	CodeSugarMommyPayoutNotValidated     = 41705
	CodeSugarMommyPayoutAlreadyPaid      = 41706
	CodeSugarMommyCurrentOrderNotFound   = 41707
	CodeSugarMommyPayoutTransferError    = 41708
)
