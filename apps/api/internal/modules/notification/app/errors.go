package app

import "errors"

// ErrRecipientNotFound 代表查無指定的通知收件人。
var ErrRecipientNotFound = errors.New("notification recipient not found")

// ErrNoNotificationRecipients 代表通知主題沒有任何啟用中的收件人。
var ErrNoNotificationRecipients = errors.New("notification has no recipients")
