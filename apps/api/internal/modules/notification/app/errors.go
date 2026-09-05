package app

import "errors"

// ErrRecipientNotFound 代表查無指定的通知收件人。
var ErrRecipientNotFound = errors.New("notification recipient not found")
